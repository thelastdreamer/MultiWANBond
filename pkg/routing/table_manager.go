package routing

// NewTableManager creates a new platform-specific table manager
func NewTableManager() TableManager {
	return newPlatformTableManager()
}
