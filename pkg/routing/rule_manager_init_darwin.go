//go:build darwin

package routing

func newPlatformRuleManager() RuleManager {
	return NewDarwinRuleManager()
}
