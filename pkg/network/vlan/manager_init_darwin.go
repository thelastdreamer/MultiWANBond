//go:build darwin
// +build darwin

package vlan

// newPlatformManager creates the Darwin-specific VLAN manager
func newPlatformManager() (Manager, error) {
	return newDarwinManager()
}
