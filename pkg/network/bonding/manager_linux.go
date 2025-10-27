// +build linux

package bonding

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/vishvananda/netlink"
)

// LinuxManager implements bonding management for Linux using netlink
type LinuxManager struct{}

// newLinuxManager creates a new Linux bonding manager
func newLinuxManager() (Manager, error) {
	// Check if bonding module is loaded
	if _, err := os.Stat("/sys/class/net/bonding_masters"); err != nil {
		return nil, &BondError{
			Op:  "init",
			Err: ErrKernelModuleNotLoaded,
		}
	}

	return &LinuxManager{}, nil
}

// Create creates a new bonding interface
func (m *LinuxManager) Create(config *BondConfig) error {
	if err := validateConfig(config); err != nil {
		return &BondError{
			Op:   "Create",
			Bond: config.Name,
			Err:  err,
		}
	}

	// Check if bond already exists
	exists, err := m.Exists(config.Name)
	if err != nil {
		return err
	}
	if exists {
		return &BondError{
			Op:   "Create",
			Bond: config.Name,
			Err:  ErrBondExists,
		}
	}

	// Create bonding interface using netlink
	bond := netlink.NewLinkBond(netlink.LinkAttrs{
		Name: config.Name,
		MTU:  config.MTU,
	})

	// Set bonding mode
	mode, err := bondModeToNetlink(config.Mode)
	if err != nil {
		return &BondError{
			Op:   "Create",
			Bond: config.Name,
			Err:  err,
		}
	}
	bond.Mode = mode

	// Set MII monitoring
	bond.MiimonInterval = config.MIIMonInterval
	bond.UpDelay = config.UpDelay
	bond.DownDelay = config.DownDelay
	if config.UseCarrier {
		bond.UseCarrier = 1
	} else {
		bond.UseCarrier = 0
	}

	// Set ARP monitoring
	bond.ArpInterval = config.ARPInterval
	if len(config.ARPIPTargets) > 0 {
		bond.ArpIpTargets = config.ARPIPTargets
	}
	bond.ArpValidate = arpValidateToNetlink(config.ARPValidate)
	if config.ARPAllTargets {
		bond.ArpAllTargets = netlink.BOND_ARP_ALL_TARGETS_ALL
	} else {
		bond.ArpAllTargets = netlink.BOND_ARP_ALL_TARGETS_ANY
	}

	// Set transmit hash policy (for mode 2 and 4)
	bond.XmitHashPolicy = xmitHashPolicyToNetlink(config.XmitHashPolicy)

	// Set LACP settings (for 802.3ad mode)
	bond.LacpRate = lacpRateToNetlink(config.LACPRate)
	bond.AdSelect = adSelectToNetlink(config.ADSelect)
	bond.MinLinks = config.MinLinks

	// Set failover settings
	bond.PrimaryReselect = primaryReselectToNetlink(config.PrimaryReselect)
	bond.FailOverMac = failOverMacToNetlink(config.FailOverMAC)

	// Set gratuitous ARP/NA
	bond.NumPeerNotif = config.NumGratARPPeer
	bond.NumGratArp = config.NumGratARPPeer

	// Create the link
	if err := netlink.LinkAdd(bond); err != nil {
		return &BondError{
			Op:   "Create",
			Bond: config.Name,
			Err:  fmt.Errorf("netlink.LinkAdd failed: %w", err),
		}
	}

	// Set MAC address if specified
	if config.MACAddress != "" {
		if err := m.SetMACAddress(config.Name, config.MACAddress); err != nil {
			// Try to clean up
			m.Delete(config.Name)
			return err
		}
	}

	// Add slaves
	for _, slaveName := range config.Slaves {
		if err := m.AddSlave(config.Name, slaveName); err != nil {
			// Don't fail if slave addition fails - bond is still created
			// Just log the error (in production, use proper logging)
			fmt.Fprintf(os.Stderr, "Warning: failed to add slave %s to bond %s: %v\n", slaveName, config.Name, err)
		}
	}

	// Set primary slave if specified
	if config.Primary != "" {
		if err := m.SetPrimary(config.Name, config.Primary); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to set primary slave %s: %v\n", config.Primary, err)
		}
	}

	// Bring interface up if enabled
	if config.Enabled {
		if err := m.Enable(config.Name); err != nil {
			return err
		}
	}

	return nil
}

// Delete removes a bonding interface
func (m *LinuxManager) Delete(name string) error {
	link, err := netlink.LinkByName(name)
	if err != nil {
		return &BondError{
			Op:   "Delete",
			Bond: name,
			Err:  ErrBondNotFound,
		}
	}

	// Verify it's a bonding interface
	if _, ok := link.(*netlink.Bond); !ok {
		return &BondError{
			Op:   "Delete",
			Bond: name,
			Err:  fmt.Errorf("interface is not a bond"),
		}
	}

	// Delete the link
	if err := netlink.LinkDel(link); err != nil {
		return &BondError{
			Op:   "Delete",
			Bond: name,
			Err:  fmt.Errorf("netlink.LinkDel failed: %w", err),
		}
	}

	return nil
}

// Get retrieves information about a bonding interface
func (m *LinuxManager) Get(name string) (*BondInfo, error) {
	link, err := netlink.LinkByName(name)
	if err != nil {
		return nil, &BondError{
			Op:   "Get",
			Bond: name,
			Err:  ErrBondNotFound,
		}
	}

	bond, ok := link.(*netlink.Bond)
	if !ok {
		return nil, &BondError{
			Op:   "Get",
			Bond: name,
			Err:  fmt.Errorf("interface is not a bond"),
		}
	}

	// Get bond info
	info := &BondInfo{
		Name:           bond.Name,
		Mode:           bondModeFromNetlink(bond.Mode),
		MACAddress:     bond.HardwareAddr.String(),
		MTU:            bond.MTU,
		MIIMonInterval: bond.MiimonInterval,
		ARPInterval:    bond.ArpInterval,
		ARPIPTargets:   bond.ArpIpTargets,
		XmitHashPolicy: xmitHashPolicyFromNetlink(bond.XmitHashPolicy),
		LACPRate:       lacpRateFromNetlink(bond.LacpRate),
		MinLinks:       bond.MinLinks,
		Created:        time.Now(), // Can't get actual creation time from netlink
		LastModified:   time.Now(),
	}

	// Get bond state
	if bond.Flags&net.FlagUp != 0 {
		info.State = "up"
		info.MIIStatus = "up"
	} else {
		info.State = "down"
		info.MIIStatus = "down"
	}

	// Get slaves
	slaves, err := m.GetSlaves(name)
	if err == nil {
		info.Slaves = slaves

		// Find active slave
		for _, slave := range slaves {
			if slave.IsActive {
				info.ActiveSlave = slave.Name
				break
			}
		}
	}

	return info, nil
}

// List returns all bonding interfaces
func (m *LinuxManager) List() ([]*BondInfo, error) {
	links, err := netlink.LinkList()
	if err != nil {
		return nil, fmt.Errorf("netlink.LinkList failed: %w", err)
	}

	var bonds []*BondInfo
	for _, link := range links {
		if _, ok := link.(*netlink.Bond); ok {
			info, err := m.Get(link.Attrs().Name)
			if err == nil {
				bonds = append(bonds, info)
			}
		}
	}

	return bonds, nil
}

// Exists checks if a bonding interface exists
func (m *LinuxManager) Exists(name string) (bool, error) {
	link, err := netlink.LinkByName(name)
	if err != nil {
		if _, ok := err.(netlink.LinkNotFoundError); ok {
			return false, nil
		}
		return false, fmt.Errorf("netlink.LinkByName failed: %w", err)
	}

	_, ok := link.(*netlink.Bond)
	return ok, nil
}

// Update updates the configuration of an existing bond
func (m *LinuxManager) Update(config *BondConfig) error {
	if err := validateConfig(config); err != nil {
		return &BondError{
			Op:   "Update",
			Bond: config.Name,
			Err:  err,
		}
	}

	// Get existing bond
	link, err := netlink.LinkByName(config.Name)
	if err != nil {
		return &BondError{
			Op:   "Update",
			Bond: config.Name,
			Err:  ErrBondNotFound,
		}
	}

	bond, ok := link.(*netlink.Bond)
	if !ok {
		return &BondError{
			Op:   "Update",
			Bond: config.Name,
			Err:  fmt.Errorf("interface is not a bond"),
		}
	}

	// Update modifiable parameters
	// Note: Some parameters (like mode) cannot be changed while bond has slaves

	// Update monitoring intervals
	bond.MiimonInterval = config.MIIMonInterval
	bond.UpDelay = config.UpDelay
	bond.DownDelay = config.DownDelay

	// Update ARP monitoring
	bond.ArpInterval = config.ARPInterval
	bond.ArpIpTargets = config.ARPIPTargets

	// Apply changes
	if err := netlink.LinkModify(bond); err != nil {
		return &BondError{
			Op:   "Update",
			Bond: config.Name,
			Err:  fmt.Errorf("netlink.LinkModify failed: %w", err),
		}
	}

	// Update MTU if changed
	if config.MTU != bond.MTU {
		if err := m.SetMTU(config.Name, config.MTU); err != nil {
			return err
		}
	}

	// Update MAC if specified and different
	if config.MACAddress != "" && config.MACAddress != bond.HardwareAddr.String() {
		if err := m.SetMACAddress(config.Name, config.MACAddress); err != nil {
			return err
		}
	}

	// Update enabled state
	attrs := link.Attrs()
	isUp := attrs.Flags&net.FlagUp != 0
	if config.Enabled && !isUp {
		return m.Enable(config.Name)
	} else if !config.Enabled && isUp {
		return m.Disable(config.Name)
	}

	return nil
}

// AddSlave adds a slave interface to a bond
func (m *LinuxManager) AddSlave(bondName, slaveName string) error {
	// Get bond interface
	bondLink, err := netlink.LinkByName(bondName)
	if err != nil {
		return &BondError{
			Op:   "AddSlave",
			Bond: bondName,
			Err:  ErrBondNotFound,
		}
	}

	// Get slave interface
	slaveLink, err := netlink.LinkByName(slaveName)
	if err != nil {
		return &SlaveError{
			Op:    "AddSlave",
			Bond:  bondName,
			Slave: slaveName,
			Err:   ErrSlaveNotFound,
		}
	}

	// Set slave's master to the bond
	if err := netlink.LinkSetMaster(slaveLink, bondLink); err != nil {
		return &SlaveError{
			Op:    "AddSlave",
			Bond:  bondName,
			Slave: slaveName,
			Err:   fmt.Errorf("netlink.LinkSetMaster failed: %w", err),
		}
	}

	return nil
}

// RemoveSlave removes a slave interface from a bond
func (m *LinuxManager) RemoveSlave(bondName, slaveName string) error {
	// Get slave interface
	slaveLink, err := netlink.LinkByName(slaveName)
	if err != nil {
		return &SlaveError{
			Op:    "RemoveSlave",
			Bond:  bondName,
			Slave: slaveName,
			Err:   ErrSlaveNotFound,
		}
	}

	// Remove master (set to nil)
	if err := netlink.LinkSetNoMaster(slaveLink); err != nil {
		return &SlaveError{
			Op:    "RemoveSlave",
			Bond:  bondName,
			Slave: slaveName,
			Err:   fmt.Errorf("netlink.LinkSetNoMaster failed: %w", err),
		}
	}

	return nil
}

// GetSlaves returns all slaves of a bond
func (m *LinuxManager) GetSlaves(bondName string) ([]SlaveInfo, error) {
	// Get bond link
	bondLink, err := netlink.LinkByName(bondName)
	if err != nil {
		return nil, &BondError{
			Op:   "GetSlaves",
			Bond: bondName,
			Err:  ErrBondNotFound,
		}
	}

	// Get all links
	links, err := netlink.LinkList()
	if err != nil {
		return nil, fmt.Errorf("netlink.LinkList failed: %w", err)
	}

	var slaves []SlaveInfo
	bondIndex := bondLink.Attrs().Index

	// Read active slave from sysfs
	activeSlavePath := filepath.Join("/sys/class/net", bondName, "bonding", "active_slave")
	activeSlaveData, _ := os.ReadFile(activeSlavePath)
	activeSlave := strings.TrimSpace(string(activeSlaveData))

	// Find all interfaces that have this bond as master
	for _, link := range links {
		attrs := link.Attrs()
		if attrs.MasterIndex == bondIndex {
			slave := SlaveInfo{
				Name:       attrs.Name,
				MACAddress: attrs.HardwareAddr.String(),
				IsActive:   attrs.Name == activeSlave,
				IsPrimary:  false, // Will be set later if we detect it
			}

			// Get link state
			if attrs.Flags&net.FlagUp != 0 {
				slave.State = "up"
				slave.LinkStatus = "up"
				slave.MIIStatus = "up"
			} else {
				slave.State = "down"
				slave.LinkStatus = "down"
				slave.MIIStatus = "down"
			}

			// Try to get speed and duplex from ethtool or sysfs
			speedPath := filepath.Join("/sys/class/net", attrs.Name, "speed")
			if speedData, err := os.ReadFile(speedPath); err == nil {
				if speed, err := strconv.Atoi(strings.TrimSpace(string(speedData))); err == nil && speed > 0 {
					slave.Speed = speed
				}
			}

			duplexPath := filepath.Join("/sys/class/net", attrs.Name, "duplex")
			if duplexData, err := os.ReadFile(duplexPath); err == nil {
				slave.Duplex = strings.TrimSpace(string(duplexData))
			}

			slaves = append(slaves, slave)
		}
	}

	return slaves, nil
}

// SetPrimary sets the primary slave for active-backup mode
func (m *LinuxManager) SetPrimary(bondName, slaveName string) error {
	// Write to sysfs
	primaryPath := filepath.Join("/sys/class/net", bondName, "bonding", "primary")
	if err := os.WriteFile(primaryPath, []byte(slaveName), 0644); err != nil {
		return &SlaveError{
			Op:    "SetPrimary",
			Bond:  bondName,
			Slave: slaveName,
			Err:   fmt.Errorf("failed to write to sysfs: %w", err),
		}
	}

	return nil
}

// SetActive manually sets the active slave
func (m *LinuxManager) SetActive(bondName, slaveName string) error {
	// Write to sysfs
	activePath := filepath.Join("/sys/class/net", bondName, "bonding", "active_slave")
	if err := os.WriteFile(activePath, []byte(slaveName), 0644); err != nil {
		return &SlaveError{
			Op:    "SetActive",
			Bond:  bondName,
			Slave: slaveName,
			Err:   fmt.Errorf("failed to write to sysfs: %w", err),
		}
	}

	return nil
}

// GetStats retrieves statistics for a bond
func (m *LinuxManager) GetStats(bondName string) (*BondStats, error) {
	link, err := netlink.LinkByName(bondName)
	if err != nil {
		return nil, &BondError{
			Op:   "GetStats",
			Bond: bondName,
			Err:  ErrBondNotFound,
		}
	}

	attrs := link.Attrs()
	stats := &BondStats{
		Name:       bondName,
		RXBytes:    attrs.Statistics.RxBytes,
		RXPackets:  attrs.Statistics.RxPackets,
		RXErrors:   attrs.Statistics.RxErrors,
		RXDropped:  attrs.Statistics.RxDropped,
		TXBytes:    attrs.Statistics.TxBytes,
		TXPackets:  attrs.Statistics.TxPackets,
		TXErrors:   attrs.Statistics.TxErrors,
		TXDropped:  attrs.Statistics.TxDropped,
		SlaveStats: make(map[string]*SlaveStats),
	}

	// Get slave stats
	slaves, err := m.GetSlaves(bondName)
	if err == nil {
		for _, slave := range slaves {
			if slaveLink, err := netlink.LinkByName(slave.Name); err == nil {
				slaveAttrs := slaveLink.Attrs()
				stats.SlaveStats[slave.Name] = &SlaveStats{
					Name:      slave.Name,
					RXBytes:   slaveAttrs.Statistics.RxBytes,
					RXPackets: slaveAttrs.Statistics.RxPackets,
					RXErrors:  slaveAttrs.Statistics.RxErrors,
					TXBytes:   slaveAttrs.Statistics.TxBytes,
					TXPackets: slaveAttrs.Statistics.TxPackets,
					TXErrors:  slaveAttrs.Statistics.TxErrors,
				}
			}
		}
	}

	return stats, nil
}

// Enable brings a bond interface up
func (m *LinuxManager) Enable(name string) error {
	link, err := netlink.LinkByName(name)
	if err != nil {
		return &BondError{
			Op:   "Enable",
			Bond: name,
			Err:  ErrBondNotFound,
		}
	}

	if err := netlink.LinkSetUp(link); err != nil {
		return &BondError{
			Op:   "Enable",
			Bond: name,
			Err:  fmt.Errorf("netlink.LinkSetUp failed: %w", err),
		}
	}

	return nil
}

// Disable brings a bond interface down
func (m *LinuxManager) Disable(name string) error {
	link, err := netlink.LinkByName(name)
	if err != nil {
		return &BondError{
			Op:   "Disable",
			Bond: name,
			Err:  ErrBondNotFound,
		}
	}

	if err := netlink.LinkSetDown(link); err != nil {
		return &BondError{
			Op:   "Disable",
			Bond: name,
			Err:  fmt.Errorf("netlink.LinkSetDown failed: %w", err),
		}
	}

	return nil
}

// SetMTU sets the MTU for a bond interface
func (m *LinuxManager) SetMTU(name string, mtu int) error {
	if mtu < 68 || mtu > 9000 {
		return &BondError{
			Op:   "SetMTU",
			Bond: name,
			Err:  ErrInvalidMTU,
		}
	}

	link, err := netlink.LinkByName(name)
	if err != nil {
		return &BondError{
			Op:   "SetMTU",
			Bond: name,
			Err:  ErrBondNotFound,
		}
	}

	if err := netlink.LinkSetMTU(link, mtu); err != nil {
		return &BondError{
			Op:   "SetMTU",
			Bond: name,
			Err:  fmt.Errorf("netlink.LinkSetMTU failed: %w", err),
		}
	}

	return nil
}

// SetMACAddress sets the MAC address for a bond interface
func (m *LinuxManager) SetMACAddress(name, mac string) error {
	hwAddr, err := net.ParseMAC(mac)
	if err != nil {
		return &BondError{
			Op:   "SetMACAddress",
			Bond: name,
			Err:  ErrInvalidMACAddress,
		}
	}

	link, err := netlink.LinkByName(name)
	if err != nil {
		return &BondError{
			Op:   "SetMACAddress",
			Bond: name,
			Err:  ErrBondNotFound,
		}
	}

	if err := netlink.LinkSetHardwareAddr(link, hwAddr); err != nil {
		return &BondError{
			Op:   "SetMACAddress",
			Bond: name,
			Err:  fmt.Errorf("netlink.LinkSetHardwareAddr failed: %w", err),
		}
	}

	return nil
}

// Helper functions for converting between our types and netlink types

func bondModeToNetlink(mode BondMode) (netlink.BondMode, error) {
	switch mode {
	case BondModeRoundRobin:
		return netlink.BOND_MODE_BALANCE_RR, nil
	case BondModeActiveBackup:
		return netlink.BOND_MODE_ACTIVE_BACKUP, nil
	case BondModeXOR:
		return netlink.BOND_MODE_BALANCE_XOR, nil
	case BondModeBroadcast:
		return netlink.BOND_MODE_BROADCAST, nil
	case BondMode8023AD:
		return netlink.BOND_MODE_802_3AD, nil
	case BondModeTLB:
		return netlink.BOND_MODE_BALANCE_TLB, nil
	case BondModeALB:
		return netlink.BOND_MODE_BALANCE_ALB, nil
	default:
		return netlink.BOND_MODE_ACTIVE_BACKUP, ErrInvalidMode
	}
}

func bondModeFromNetlink(mode netlink.BondMode) BondMode {
	switch mode {
	case netlink.BOND_MODE_BALANCE_RR:
		return BondModeRoundRobin
	case netlink.BOND_MODE_ACTIVE_BACKUP:
		return BondModeActiveBackup
	case netlink.BOND_MODE_BALANCE_XOR:
		return BondModeXOR
	case netlink.BOND_MODE_BROADCAST:
		return BondModeBroadcast
	case netlink.BOND_MODE_802_3AD:
		return BondMode8023AD
	case netlink.BOND_MODE_BALANCE_TLB:
		return BondModeTLB
	case netlink.BOND_MODE_BALANCE_ALB:
		return BondModeALB
	default:
		return BondModeActiveBackup
	}
}

func xmitHashPolicyToNetlink(policy XmitHashPolicy) netlink.BondXmitHashPolicy {
	switch policy {
	case XmitHashLayer2:
		return netlink.BOND_XMIT_HASH_POLICY_LAYER2
	case XmitHashLayer23:
		return netlink.BOND_XMIT_HASH_POLICY_LAYER2_3
	case XmitHashLayer34:
		return netlink.BOND_XMIT_HASH_POLICY_LAYER3_4
	case XmitHashEncap23:
		return netlink.BOND_XMIT_HASH_POLICY_ENCAP2_3
	case XmitHashEncap34:
		return netlink.BOND_XMIT_HASH_POLICY_ENCAP3_4
	default:
		return netlink.BOND_XMIT_HASH_POLICY_LAYER2
	}
}

func xmitHashPolicyFromNetlink(policy netlink.BondXmitHashPolicy) XmitHashPolicy {
	switch policy {
	case netlink.BOND_XMIT_HASH_POLICY_LAYER2:
		return XmitHashLayer2
	case netlink.BOND_XMIT_HASH_POLICY_LAYER2_3:
		return XmitHashLayer23
	case netlink.BOND_XMIT_HASH_POLICY_LAYER3_4:
		return XmitHashLayer34
	case netlink.BOND_XMIT_HASH_POLICY_ENCAP2_3:
		return XmitHashEncap23
	case netlink.BOND_XMIT_HASH_POLICY_ENCAP3_4:
		return XmitHashEncap34
	default:
		return XmitHashLayer2
	}
}

func lacpRateToNetlink(rate LACPRate) netlink.BondLacpRate {
	if rate == LACPRateFast {
		return netlink.BOND_LACP_RATE_FAST
	}
	return netlink.BOND_LACP_RATE_SLOW
}

func lacpRateFromNetlink(rate netlink.BondLacpRate) LACPRate {
	if rate == netlink.BOND_LACP_RATE_FAST {
		return LACPRateFast
	}
	return LACPRateSlow
}

func adSelectToNetlink(sel ADSelect) netlink.BondAdSelect {
	switch sel {
	case ADSelectBandwidth:
		return netlink.BOND_AD_SELECT_BANDWIDTH
	case ADSelectCount:
		return netlink.BOND_AD_SELECT_COUNT
	default:
		return netlink.BOND_AD_SELECT_STABLE
	}
}

func adSelectFromNetlink(sel netlink.BondAdSelect) ADSelect {
	switch sel {
	case netlink.BOND_AD_SELECT_BANDWIDTH:
		return ADSelectBandwidth
	case netlink.BOND_AD_SELECT_COUNT:
		return ADSelectCount
	default:
		return ADSelectStable
	}
}

func primaryReselectToNetlink(resel PrimaryReselect) netlink.BondPrimaryReselect {
	switch resel {
	case PrimaryReselectAlways:
		return netlink.BOND_PRIMARY_RESELECT_ALWAYS
	case PrimaryReselectBetter:
		return netlink.BOND_PRIMARY_RESELECT_BETTER
	default:
		return netlink.BOND_PRIMARY_RESELECT_FAILURE
	}
}

func failOverMacToNetlink(fom FailOverMAC) netlink.BondFailOverMac {
	switch fom {
	case FailOverMACActive:
		return netlink.BOND_FAIL_OVER_MAC_ACTIVE
	case FailOverMACFollow:
		return netlink.BOND_FAIL_OVER_MAC_FOLLOW
	default:
		return netlink.BOND_FAIL_OVER_MAC_NONE
	}
}

func arpValidateToNetlink(validate ARPValidate) netlink.BondArpValidate {
	switch validate {
	case ARPValidateActive:
		return netlink.BOND_ARP_VALIDATE_ACTIVE
	case ARPValidateBackup:
		return netlink.BOND_ARP_VALIDATE_BACKUP
	case ARPValidateAll:
		return netlink.BOND_ARP_VALIDATE_ALL
	default:
		return netlink.BOND_ARP_VALIDATE_NONE
	}
}

// validateConfig validates a bond configuration
func validateConfig(config *BondConfig) error {
	if config.Name == "" {
		return ErrInvalidBondName
	}

	// Validate mode
	validModes := []BondMode{
		BondModeRoundRobin, BondModeActiveBackup, BondModeXOR,
		BondModeBroadcast, BondMode8023AD, BondModeTLB, BondModeALB,
	}
	valid := false
	for _, mode := range validModes {
		if config.Mode == mode {
			valid = true
			break
		}
	}
	if !valid {
		return ErrInvalidMode
	}

	// Validate MII monitoring
	if config.MIIMonInterval < 0 {
		return ErrInvalidMIIMonInterval
	}

	if config.MIIMonInterval > 0 {
		if config.UpDelay%config.MIIMonInterval != 0 {
			return ErrInvalidDelay
		}
		if config.DownDelay%config.MIIMonInterval != 0 {
			return ErrInvalidDelay
		}
	}

	// Validate ARP monitoring
	if config.ARPInterval < 0 {
		return ErrInvalidARPInterval
	}

	// Can't have both MII and ARP monitoring
	if config.MIIMonInterval > 0 && config.ARPInterval > 0 {
		return ErrMIIAndARPBothEnabled
	}

	// If ARP monitoring is enabled, must have targets
	if config.ARPInterval > 0 && len(config.ARPIPTargets) == 0 {
		return ErrNoARPTargets
	}

	// Validate LACP settings are only used with 802.3ad
	if config.Mode != BondMode8023AD {
		if config.LACPRate != LACPRateSlow {
			return ErrLACPRequires8023AD
		}
		if config.ADSelect != ADSelectStable {
			return ErrLACPRequires8023AD
		}
	}

	// Validate primary slave is in slaves list
	if config.Primary != "" && len(config.Slaves) > 0 {
		found := false
		for _, slave := range config.Slaves {
			if slave == config.Primary {
				found = true
				break
			}
		}
		if !found {
			return ErrPrimaryNotInSlaves
		}
	}

	// Validate MTU
	if config.MTU < 68 || config.MTU > 9000 {
		return ErrInvalidMTU
	}

	// Validate MAC address if specified
	if config.MACAddress != "" {
		if _, err := net.ParseMAC(config.MACAddress); err != nil {
			return ErrInvalidMACAddress
		}
	}

	return nil
}
