//go:build windows
// +build windows

package vlan

// newPlatformManager creates the Windows-specific VLAN manager
func newPlatformManager() (Manager, error) {
	return newWindowsManager()
}
