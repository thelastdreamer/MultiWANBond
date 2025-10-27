//go:build windows

package routing

func newPlatformTableManager() TableManager {
	return NewWindowsTableManager()
}
