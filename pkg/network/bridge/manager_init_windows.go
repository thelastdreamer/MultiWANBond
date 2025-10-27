// +build windows

package bridge

// newPlatformManager creates the Windows-specific bridge manager
func newPlatformManager() (Manager, error) {
	return newWindowsManager()
}
