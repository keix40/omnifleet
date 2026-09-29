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
	PermViewDispatch    Permission = "dispatch:view"
	PermUpdateOwnJobs   Permission = "dispatch:driver_update"
	PermViewETA         Permission = "eta:view"
	PermBillingAdmin    Permission = "billing:admin"
	PermViewBilling     Permission = "billing:view"
	PermManageNotifications Permission = "notifications:manage"
)

var rolePermissions = map[Role]map[Permission]bool{
	RoleAdmin: {
		PermLogin:               true,
		PermIngestPosition:      true,
		PermViewFleet:           true,
		PermManageGeofences:     true,
		PermDispatchJobs:        true,
		PermViewDispatch:        true,
		PermUpdateOwnJobs:       true,
		PermViewETA:             true,
		PermBillingAdmin:        true,
		PermViewBilling:         true,
		PermManageNotifications: true,
	},
	RoleDispatcher: {
		PermLogin:           true,
		PermViewFleet:       true,
		PermManageGeofences: true,
		PermDispatchJobs:    true,
		PermViewDispatch:    true,
		PermViewETA:         true,
		PermViewBilling:     true,
	},
	RoleDriver: {
		PermLogin:          true,
		PermIngestPosition: true,
		PermViewFleet:      true,
		PermViewDispatch:   true,
		PermUpdateOwnJobs:  true,
		PermViewETA:        true,
	},
	RoleCustomer: {
		PermLogin:       true,
		PermViewFleet:   true,
		PermViewDispatch: true,
		PermViewETA:     true,
		PermViewBilling: true,
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
