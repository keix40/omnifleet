package auth

import "fmt"

type Role string

const (
	RoleAdmin      Role = "admin"
	RoleDispatcher Role = "dispatcher"
	RoleDriver     Role = "driver"
	RoleCustomer   Role = "customer"
)

var roleHierarchy = map[Role]int{
	RoleAdmin:      100,
	RoleDispatcher: 80,
	RoleDriver:     50,
	RoleCustomer:   10,
}

// Permission represents a coarse-grained capability checked at the gateway.
type Permission string

const (
	PermLogin           Permission = "auth:login"
	PermIngestPosition  Permission = "tracking:ingest"
	PermViewFleet       Permission = "fleet:view"
	PermManageGeofences Permission = "geofences:manage"
	PermDispatchJobs    Permission = "dispatch:manage"
	PermBillingAdmin    Permission = "billing:admin"
)

var rolePermissions = map[Role]map[Permission]bool{
	RoleAdmin: {
		PermLogin:           true,
		PermIngestPosition:  true,
		PermViewFleet:       true,
		PermManageGeofences: true,
		PermDispatchJobs:    true,
		PermBillingAdmin:    true,
	},
	RoleDispatcher: {
		PermLogin:           true,
		PermViewFleet:       true,
		PermManageGeofences: true,
		PermDispatchJobs:    true,
	},
	RoleDriver: {
		PermLogin:          true,
		PermIngestPosition: true,
		PermViewFleet:      true,
	},
	RoleCustomer: {
		PermLogin:     true,
		PermViewFleet: true,
	},
}

func ParseRole(s string) (Role, error) {
	r := Role(s)
	if _, ok := roleHierarchy[r]; !ok {
		return "", fmt.Errorf("unknown role: %s", s)
	}
	return r, nil
}

func HasPermission(role Role, perm Permission) bool {
	perms, ok := rolePermissions[role]
	if !ok {
		return false
	}
	return perms[perm]
}

func AtLeast(role Role, minimum Role) bool {
	return roleHierarchy[role] >= roleHierarchy[minimum]
}
