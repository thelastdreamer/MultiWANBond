// +build linux

package bonding

// newPlatformManager creates the Linux-specific bonding manager
func newPlatformManager() (Manager, error) {
	return newLinuxManager()
}
