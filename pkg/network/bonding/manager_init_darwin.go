// +build darwin

package bonding

// newPlatformManager creates the macOS-specific bonding manager
func newPlatformManager() (Manager, error) {
	return newDarwinManager()
}
