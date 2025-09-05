package domain

import "slices"

// Role represents a user role in the system
type Role string

const (
	// RoleAdmin has full access to all features and can manage users, tenants, and system settings
	RoleAdmin Role = "admin"

	// RoleUser has basic access to create audit logs and view their own tenant's data
	RoleUser Role = "user"

	// RoleAuditor has read-only access to audit logs and can generate reports
	RoleAuditor Role = "auditor"
)

// ValidRoles contains all valid roles in the system
var ValidRoles = []Role{RoleAdmin, RoleUser, RoleAuditor}

// IsValidRole checks if a given role is valid
func IsValidRole(role string) bool {
	return slices.Contains(ValidRoles, Role(role))
}

// HasRole checks if a slice of roles contains a specific role
func HasRole(roles []string, role Role) bool {
	return slices.Contains(roles, string(role))
}

// HasAnyRole checks if a slice of roles contains any of the specified roles
func HasAnyRole(roles []string, requiredRoles ...Role) bool {
	for _, required := range requiredRoles {
		if HasRole(roles, required) {
			return true
		}
	}
	return false
}

// HasAllRoles checks if a slice of roles contains all of the specified roles
func HasAllRoles(roles []string, requiredRoles ...Role) bool {
	for _, required := range requiredRoles {
		if !HasRole(roles, required) {
			return false
		}
	}
	return true
}
