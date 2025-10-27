// +build windows

package bonding

// newPlatformManager creates the Windows-specific bonding manager
func newPlatformManager() (Manager, error) {
	return newWindowsManager()
}
