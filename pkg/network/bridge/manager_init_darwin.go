// +build darwin

package bridge

// newPlatformManager creates the macOS-specific bridge manager
func newPlatformManager() (Manager, error) {
	return newDarwinManager()
}
