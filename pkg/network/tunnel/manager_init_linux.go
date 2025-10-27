// +build linux

package tunnel

// newPlatformManager creates the Linux-specific tunnel manager
func newPlatformManager() (Manager, error) {
	return newLinuxManager()
}
