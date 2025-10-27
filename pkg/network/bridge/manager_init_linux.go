// +build linux

package bridge

// newPlatformManager creates the Linux-specific bridge manager
func newPlatformManager() (Manager, error) {
	return newLinuxManager()
}
