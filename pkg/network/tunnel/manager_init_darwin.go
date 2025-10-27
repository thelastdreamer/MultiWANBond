// +build darwin

package tunnel

// newPlatformManager creates the macOS-specific tunnel manager
func newPlatformManager() (Manager, error) {
	return newDarwinManager()
}
