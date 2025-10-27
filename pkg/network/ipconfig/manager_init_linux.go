//go:build linux
// +build linux

package ipconfig

// newPlatformManager creates the Linux-specific IP configuration manager
func newPlatformManager() (Manager, error) {
	return newLinuxManager()
}
