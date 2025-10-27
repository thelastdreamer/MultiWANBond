//go:build windows

package routing

func newPlatformRuleManager() RuleManager {
	return NewWindowsRuleManager()
}
