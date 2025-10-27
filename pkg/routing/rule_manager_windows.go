//go:build windows

package routing

import "fmt"

// WindowsRuleManager manages policy routing rules on Windows
type WindowsRuleManager struct {
	rules map[int]*PolicyRule
}

// NewWindowsRuleManager creates a new Windows rule manager
func NewWindowsRuleManager() *WindowsRuleManager {
	return &WindowsRuleManager{
		rules: make(map[int]*PolicyRule),
	}
}

// AddRule adds a policy rule
func (rm *WindowsRuleManager) AddRule(rule *PolicyRule) error {
	return fmt.Errorf("policy rules not yet implemented on Windows")
}

// DeleteRule deletes a policy rule
func (rm *WindowsRuleManager) DeleteRule(priority int) error {
	return fmt.Errorf("policy rules not yet implemented on Windows")
}

// GetRule retrieves a rule by priority
func (rm *WindowsRuleManager) GetRule(priority int) (*PolicyRule, error) {
	return nil, fmt.Errorf("policy rules not yet implemented on Windows")
}

// ListRules lists all policy rules
func (rm *WindowsRuleManager) ListRules() ([]*PolicyRule, error) {
	return nil, fmt.Errorf("policy rules not yet implemented on Windows")
}

// FlushRules removes all policy rules
func (rm *WindowsRuleManager) FlushRules() error {
	return fmt.Errorf("policy rules not yet implemented on Windows")
}

// EnableRule enables a rule
func (rm *WindowsRuleManager) EnableRule(priority int) error {
	return fmt.Errorf("policy rules not yet implemented on Windows")
}

// DisableRule disables a rule
func (rm *WindowsRuleManager) DisableRule(priority int) error {
	return fmt.Errorf("policy rules not yet implemented on Windows")
}
