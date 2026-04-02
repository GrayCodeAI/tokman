// Package rbac provides role-based access control
package rbac

import (
	"fmt"
	"sync"
)

// Role represents a user role
type Role string

const (
	RoleAdmin     Role = "admin"
	RoleManager   Role = "manager"
	RoleDeveloper Role = "developer"
	RoleViewer    Role = "viewer"
)

// Permission represents a permission
type Permission string

const (
	PermRead   Permission = "read"
	PermWrite  Permission = "write"
	PermDelete Permission = "delete"
	PermAdmin  Permission = "admin"
)

// Resource represents a resource type
type Resource string

const (
	ResourceBenchmarks  Resource = "benchmarks"
	ResourceStressTests Resource = "stress_tests"
	ResourceCostData    Resource = "cost_data"
	ResourceAlerts      Resource = "alerts"
	ResourceTeams       Resource = "teams"
	ResourceConfig      Resource = "config"
)

// RBAC manages role-based access control
type RBAC struct {
	roles     map[Role]map[Resource][]Permission
	userRoles map[string]Role
	mu        sync.RWMutex
}

// NewRBAC creates a new RBAC manager
func NewRBAC() *RBAC {
	rbac := &RBAC{
		roles:     make(map[Role]map[Resource][]Permission),
		userRoles: make(map[string]Role),
	}

	// Define default role permissions
	rbac.roles[RoleAdmin] = map[Resource][]Permission{
		ResourceBenchmarks:  {PermRead, PermWrite, PermDelete, PermAdmin},
		ResourceStressTests: {PermRead, PermWrite, PermDelete, PermAdmin},
		ResourceCostData:    {PermRead, PermWrite, PermDelete, PermAdmin},
		ResourceAlerts:      {PermRead, PermWrite, PermDelete, PermAdmin},
		ResourceTeams:       {PermRead, PermWrite, PermDelete, PermAdmin},
		ResourceConfig:      {PermRead, PermWrite, PermDelete, PermAdmin},
	}

	rbac.roles[RoleManager] = map[Resource][]Permission{
		ResourceBenchmarks:  {PermRead, PermWrite},
		ResourceStressTests: {PermRead, PermWrite},
		ResourceCostData:    {PermRead, PermWrite},
		ResourceAlerts:      {PermRead, PermWrite},
		ResourceTeams:       {PermRead, PermWrite},
		ResourceConfig:      {PermRead},
	}

	rbac.roles[RoleDeveloper] = map[Resource][]Permission{
		ResourceBenchmarks:  {PermRead, PermWrite},
		ResourceStressTests: {PermRead, PermWrite},
		ResourceCostData:    {PermRead},
		ResourceAlerts:      {PermRead},
		ResourceTeams:       {PermRead},
		ResourceConfig:      {PermRead},
	}

	rbac.roles[RoleViewer] = map[Resource][]Permission{
		ResourceBenchmarks:  {PermRead},
		ResourceStressTests: {PermRead},
		ResourceCostData:    {PermRead},
		ResourceAlerts:      {PermRead},
		ResourceTeams:       {PermRead},
		ResourceConfig:      {PermRead},
	}

	return rbac
}

// AssignRole assigns a role to a user
func (r *RBAC) AssignRole(userID string, role Role) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.roles[role]; !ok {
		return fmt.Errorf("unknown role: %s", role)
	}

	r.userRoles[userID] = role
	return nil
}

// CheckPermission checks if a user has a permission
func (r *RBAC) CheckPermission(userID string, resource Resource, permission Permission) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	role, ok := r.userRoles[userID]
	if !ok {
		return false
	}

	perms, ok := r.roles[role][resource]
	if !ok {
		return false
	}

	for _, p := range perms {
		if p == permission {
			return true
		}
	}

	return false
}

// GetUserRole returns a user's role
func (r *RBAC) GetUserRole(userID string) (Role, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	role, ok := r.userRoles[userID]
	return role, ok
}

// RemoveUser removes a user
func (r *RBAC) RemoveUser(userID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.userRoles, userID)
}

// ListUsers returns all users with their roles
func (r *RBAC) ListUsers() map[string]Role {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]Role)
	for user, role := range r.userRoles {
		result[user] = role
	}

	return result
}
