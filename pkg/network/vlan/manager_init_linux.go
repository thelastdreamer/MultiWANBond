//go:build linux
// +build linux

package vlan

// newPlatformManager creates the Linux-specific VLAN manager
func newPlatformManager() (Manager, error) {
	return newLinuxManager()
}
