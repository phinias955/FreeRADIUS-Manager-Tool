package auth

// Role constants
const (
	RoleSuperAdmin = "super_admin"
	RoleAdmin      = "admin"
	RoleOperator   = "operator"
)

// roleHierarchy maps each role to its numeric level (higher = more permissions).
var roleHierarchy = map[string]int{
	RoleOperator:   1,
	RoleAdmin:      2,
	RoleSuperAdmin: 3,
}

// HasRole returns true if the user's role is in the allowed roles list.
func HasRole(userRole string, allowedRoles ...string) bool {
	for _, r := range allowedRoles {
		if userRole == r {
			return true
		}
	}
	return false
}

// HasMinRole returns true if the user's role is at least as privileged as minRole.
func HasMinRole(userRole, minRole string) bool {
	userLevel := roleHierarchy[userRole]
	minLevel := roleHierarchy[minRole]
	return userLevel >= minLevel
}

// CanManageRole returns true if the actor can manage (create/edit/delete) the target role.
// A user cannot manage peers or higher roles.
func CanManageRole(actorRole, targetRole string) bool {
	return roleHierarchy[actorRole] > roleHierarchy[targetRole]
}
