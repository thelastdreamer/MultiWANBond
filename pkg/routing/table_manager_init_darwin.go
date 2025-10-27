//go:build darwin

package routing

func newPlatformTableManager() TableManager {
	return NewDarwinTableManager()
}
