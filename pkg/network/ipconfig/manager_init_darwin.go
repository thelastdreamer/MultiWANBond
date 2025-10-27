//go:build darwin
// +build darwin

package ipconfig

// newPlatformManager creates the Darwin-specific IP configuration manager
func newPlatformManager() (Manager, error) {
	return newDarwinManager()
}
