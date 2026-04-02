// Package retention provides data retention policy management
package retention

import (
	"fmt"
	"sync"
	"time"
)

// Policy represents a data retention policy
type Policy struct {
	ID                  string
	Name                string
	Description         string
	RetentionPeriod     time.Duration
	DataType            string
	AutoDelete          bool
	ArchiveBeforeDelete bool
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// RetentionManager manages data retention policies
type RetentionManager struct {
	policies map[string]*Policy
	mu       sync.RWMutex
}

// NewRetentionManager creates a new retention manager
func NewRetentionManager() *RetentionManager {
	return &RetentionManager{
		policies: make(map[string]*Policy),
	}
}

// AddPolicy adds a retention policy
func (rm *RetentionManager) AddPolicy(policy *Policy) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if policy.ID == "" {
		policy.ID = fmt.Sprintf("ret-%d", time.Now().UnixNano())
	}

	rm.policies[policy.ID] = policy
	return nil
}

// GetPolicy returns a policy by ID
func (rm *RetentionManager) GetPolicy(id string) (*Policy, error) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	policy, ok := rm.policies[id]
	if !ok {
		return nil, fmt.Errorf("policy not found: %s", id)
	}

	return policy, nil
}

// IsExpired checks if data is expired based on policy
func (rm *RetentionManager) IsExpired(policyID string, createdAt time.Time) (bool, error) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	policy, ok := rm.policies[policyID]
	if !ok {
		return false, fmt.Errorf("policy not found: %s", policyID)
	}

	expiryDate := createdAt.Add(policy.RetentionPeriod)
	return time.Now().After(expiryDate), nil
}

// GetExpiredItems returns items that are expired
func (rm *RetentionManager) GetExpiredItems(items []RetentionItem) []RetentionItem {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	expired := make([]RetentionItem, 0)

	for _, item := range items {
		policy, ok := rm.policies[item.PolicyID]
		if !ok {
			continue
		}

		expiryDate := item.CreatedAt.Add(policy.RetentionPeriod)
		if time.Now().After(expiryDate) {
			expired = append(expired, item)
		}
	}

	return expired
}

// RetentionItem represents an item subject to retention
type RetentionItem struct {
	ID        string
	PolicyID  string
	CreatedAt time.Time
	Data      interface{}
}

// StandardPolicies returns standard retention policies
func StandardPolicies() []*Policy {
	return []*Policy{
		{
			Name:                "Benchmark Results",
			Description:         "Retain benchmark results for 90 days",
			RetentionPeriod:     90 * 24 * time.Hour,
			DataType:            "benchmark",
			AutoDelete:          true,
			ArchiveBeforeDelete: true,
		},
		{
			Name:                "Stress Test Results",
			Description:         "Retain stress test results for 30 days",
			RetentionPeriod:     30 * 24 * time.Hour,
			DataType:            "stress_test",
			AutoDelete:          true,
			ArchiveBeforeDelete: false,
		},
		{
			Name:                "Audit Logs",
			Description:         "Retain audit logs for 1 year",
			RetentionPeriod:     365 * 24 * time.Hour,
			DataType:            "audit_log",
			AutoDelete:          false,
			ArchiveBeforeDelete: true,
		},
		{
			Name:                "Cost Data",
			Description:         "Retain cost data for 2 years",
			RetentionPeriod:     730 * 24 * time.Hour,
			DataType:            "cost_data",
			AutoDelete:          false,
			ArchiveBeforeDelete: true,
		},
	}
}
