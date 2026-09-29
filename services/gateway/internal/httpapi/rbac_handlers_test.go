package httpapi_test

import (
	"testing"

	"github.com/keix40/omnifleet/pkg/auth"
)

func TestRBAC_NewPlatformEndpoints(t *testing.T) {
	cases := []struct {
		role auth.Role
		perm auth.Permission
		want bool
	}{
		{auth.RoleCustomer, auth.PermDispatchJobs, false},
		{auth.RoleDispatcher, auth.PermDispatchJobs, true},
		{auth.RoleDriver, auth.PermViewDispatch, true},
		{auth.RoleDriver, auth.PermDispatchJobs, false},
		{auth.RoleDriver, auth.PermUpdateOwnJobs, true},
		{auth.RoleCustomer, auth.PermViewETA, true},
		{auth.RoleCustomer, auth.PermBillingAdmin, false},
		{auth.RoleAdmin, auth.PermManageNotifications, true},
		{auth.RoleDispatcher, auth.PermManageNotifications, false},
	}
	for _, c := range cases {
		got := auth.HasPermission(c.role, c.perm)
		if got != c.want {
			t.Fatalf("role=%s perm=%s got=%v want=%v", c.role, c.perm, got, c.want)
		}
	}
}
