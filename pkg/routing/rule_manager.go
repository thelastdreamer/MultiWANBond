package routing

// NewRuleManager creates a new platform-specific rule manager
func NewRuleManager() RuleManager {
	return newPlatformRuleManager()
}
