//go:build linux

package routing

import (
	"fmt"
	"sync"

	"github.com/vishvananda/netlink"
)

// LinuxRuleManager manages policy routing rules on Linux using netlink
type LinuxRuleManager struct {
	mu    sync.RWMutex
	rules map[int]*PolicyRule // priority -> rule
}

// NewLinuxRuleManager creates a new Linux rule manager
func NewLinuxRuleManager() *LinuxRuleManager {
	return &LinuxRuleManager{
		rules: make(map[int]*PolicyRule),
	}
}

// AddRule adds a policy rule
func (rm *LinuxRuleManager) AddRule(rule *PolicyRule) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if _, exists := rm.rules[rule.Priority]; exists {
		return fmt.Errorf("rule with priority %d already exists", rule.Priority)
	}

	// Create netlink rule
	nlRule := netlink.NewRule()
	nlRule.Priority = rule.Priority
	nlRule.Table = rule.Table

	// Set source network
	if rule.SourceNetwork != nil {
		nlRule.Src = rule.SourceNetwork
	}

	// Set destination network
	if rule.DestNetwork != nil {
		nlRule.Dst = rule.DestNetwork
	}

	// Set input interface
	if rule.InputInterface != "" {
		nlRule.IifName = rule.InputInterface
	}

	// Set output interface
	if rule.OutputInterface != "" {
		nlRule.OifName = rule.OutputInterface
	}

	// Set fwmark
	if rule.Mark != 0 {
		nlRule.Mark = int(rule.Mark)
		if rule.MarkMask != 0 {
			nlRule.Mask = int(rule.MarkMask)
		} else {
			nlRule.Mask = 0xFFFFFFFF // Match all bits by default
		}
	}

	// Set TOS
	if rule.TOS != 0 {
		nlRule.Tos = uint(rule.TOS)
	}

	// Set invert flag
	if rule.Invert {
		nlRule.Invert = true
	}

	// Add rule via netlink
	if err := netlink.RuleAdd(nlRule); err != nil {
		return fmt.Errorf("failed to add rule: %w", err)
	}

	// Store in our map
	rm.rules[rule.Priority] = rule

	return nil
}

// DeleteRule deletes a policy rule
func (rm *LinuxRuleManager) DeleteRule(priority int) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	rule, exists := rm.rules[priority]
	if !exists {
		return fmt.Errorf("rule with priority %d not found", priority)
	}

	// Create netlink rule for deletion
	nlRule := netlink.NewRule()
	nlRule.Priority = priority
	nlRule.Table = rule.Table

	if rule.SourceNetwork != nil {
		nlRule.Src = rule.SourceNetwork
	}
	if rule.DestNetwork != nil {
		nlRule.Dst = rule.DestNetwork
	}
	if rule.Mark != 0 {
		nlRule.Mark = int(rule.Mark)
		if rule.MarkMask != 0 {
			nlRule.Mask = int(rule.MarkMask)
		}
	}

	// Delete rule via netlink
	if err := netlink.RuleDel(nlRule); err != nil {
		return fmt.Errorf("failed to delete rule: %w", err)
	}

	// Remove from our map
	delete(rm.rules, priority)

	return nil
}

// GetRule retrieves a rule by priority
func (rm *LinuxRuleManager) GetRule(priority int) (*PolicyRule, error) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	rule, exists := rm.rules[priority]
	if !exists {
		return nil, fmt.Errorf("rule with priority %d not found", priority)
	}

	return rule, nil
}

// ListRules lists all policy rules
func (rm *LinuxRuleManager) ListRules() ([]*PolicyRule, error) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	// Get rules from kernel
	nlRules, err := netlink.RuleList(netlink.FAMILY_V4)
	if err != nil {
		return nil, fmt.Errorf("failed to list rules: %w", err)
	}

	// Convert to our PolicyRule type
	rules := make([]*PolicyRule, 0, len(nlRules))
	for _, nlRule := range nlRules {
		rule := rm.netlinkToRule(&nlRule)
		rules = append(rules, rule)

		// Update our map
		rm.rules[rule.Priority] = rule
	}

	return rules, nil
}

// FlushRules removes all policy rules
func (rm *LinuxRuleManager) FlushRules() error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	// Get all rules
	nlRules, err := netlink.RuleList(netlink.FAMILY_V4)
	if err != nil {
		return fmt.Errorf("failed to list rules: %w", err)
	}

	// Delete each custom rule (skip system rules)
	for _, nlRule := range nlRules {
		// Skip system rules (priority < 100)
		if nlRule.Priority < 100 {
			continue
		}

		if err := netlink.RuleDel(&nlRule); err != nil {
			// Continue on error to delete as many as possible
			continue
		}
	}

	// Clear our map (keep only system rules)
	newRules := make(map[int]*PolicyRule)
	for priority, rule := range rm.rules {
		if priority < 100 {
			newRules[priority] = rule
		}
	}
	rm.rules = newRules

	return nil
}

// EnableRule enables a rule
func (rm *LinuxRuleManager) EnableRule(priority int) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	rule, exists := rm.rules[priority]
	if !exists {
		return fmt.Errorf("rule with priority %d not found", priority)
	}

	if rule.Enabled {
		return nil // Already enabled
	}

	// Re-add the rule
	rule.Enabled = true
	return rm.AddRule(rule)
}

// DisableRule disables a rule
func (rm *LinuxRuleManager) DisableRule(priority int) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	rule, exists := rm.rules[priority]
	if !exists {
		return fmt.Errorf("rule with priority %d not found", priority)
	}

	if !rule.Enabled {
		return nil // Already disabled
	}

	// Delete the rule from kernel but keep in our map
	nlRule := netlink.NewRule()
	nlRule.Priority = priority
	nlRule.Table = rule.Table

	if err := netlink.RuleDel(nlRule); err != nil {
		return fmt.Errorf("failed to disable rule: %w", err)
	}

	rule.Enabled = false
	return nil
}

// netlinkToRule converts netlink.Rule to PolicyRule
func (rm *LinuxRuleManager) netlinkToRule(nlRule *netlink.Rule) *PolicyRule {
	rule := &PolicyRule{
		Priority:       nlRule.Priority,
		Table:          nlRule.Table,
		SourceNetwork:  nlRule.Src,
		DestNetwork:    nlRule.Dst,
		InputInterface: nlRule.IifName,
		OutputInterface: nlRule.OifName,
		Mark:           uint32(nlRule.Mark),
		MarkMask:       uint32(nlRule.Mask),
		TOS:            uint8(nlRule.Tos),
		Invert:         nlRule.Invert,
		Action:         RuleActionTable, // Default
		Enabled:        true,
	}

	return rule
}
