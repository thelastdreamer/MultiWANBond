//go:build linux

package routing

func newPlatformRuleManager() RuleManager {
	return NewLinuxRuleManager()
}
