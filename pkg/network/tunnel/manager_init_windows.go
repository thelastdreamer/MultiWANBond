// +build windows

package tunnel

// newPlatformManager creates the Windows-specific tunnel manager
func newPlatformManager() (Manager, error) {
	return newWindowsManager()
}
