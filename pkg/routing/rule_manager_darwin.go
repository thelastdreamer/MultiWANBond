//go:build darwin

package routing

import "fmt"

// DarwinRuleManager manages policy routing rules on macOS
type DarwinRuleManager struct {
	rules map[int]*PolicyRule
}

// NewDarwinRuleManager creates a new macOS rule manager
func NewDarwinRuleManager() *DarwinRuleManager {
	return &DarwinRuleManager{
		rules: make(map[int]*PolicyRule),
	}
}

// AddRule adds a policy rule
func (rm *DarwinRuleManager) AddRule(rule *PolicyRule) error {
	return fmt.Errorf("policy rules not yet implemented on macOS")
}

// DeleteRule deletes a policy rule
func (rm *DarwinRuleManager) DeleteRule(priority int) error {
	return fmt.Errorf("policy rules not yet implemented on macOS")
}

// GetRule retrieves a rule by priority
func (rm *DarwinRuleManager) GetRule(priority int) (*PolicyRule, error) {
	return nil, fmt.Errorf("policy rules not yet implemented on macOS")
}

// ListRules lists all policy rules
func (rm *DarwinRuleManager) ListRules() ([]*PolicyRule, error) {
	return nil, fmt.Errorf("policy rules not yet implemented on macOS")
}

// FlushRules removes all policy rules
func (rm *DarwinRuleManager) FlushRules() error {
	return fmt.Errorf("policy rules not yet implemented on macOS")
}

// EnableRule enables a rule
func (rm *DarwinRuleManager) EnableRule(priority int) error {
	return fmt.Errorf("policy rules not yet implemented on macOS")
}

// DisableRule disables a rule
func (rm *DarwinRuleManager) DisableRule(priority int) error {
	return fmt.Errorf("policy rules not yet implemented on macOS")
}
