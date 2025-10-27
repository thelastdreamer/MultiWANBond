//go:build windows
// +build windows

package ipconfig

// newPlatformManager creates the Windows-specific IP configuration manager
func newPlatformManager() (Manager, error) {
	return newWindowsManager()
}
