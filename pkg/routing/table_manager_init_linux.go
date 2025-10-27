//go:build linux

package routing

func newPlatformTableManager() TableManager {
	return NewLinuxTableManager()
}
