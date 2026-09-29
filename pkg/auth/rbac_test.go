package auth_test

import (
	"testing"

	"github.com/keix40/omnifleet/pkg/auth"
)

func TestRBAC_DriverCannotDispatch(t *testing.T) {
	if auth.HasPermission(auth.RoleDriver, auth.PermDispatchJobs) {
		t.Fatal("driver must not dispatch jobs")
	}
}

func TestRBAC_DispatcherCanViewFleet(t *testing.T) {
	if !auth.HasPermission(auth.RoleDispatcher, auth.PermViewFleet) {
		t.Fatal("dispatcher should view fleet")
	}
}

func TestRBAC_AdminHasBilling(t *testing.T) {
	if !auth.HasPermission(auth.RoleDriver, auth.PermViewDispatch) {
		t.Fatal("driver should view assigned jobs")
	}
	if auth.HasPermission(auth.RoleCustomer, auth.PermDispatchJobs) {
		t.Fatal("customer must not manage dispatch")
	}
	if !auth.HasPermission(auth.RoleAdmin, auth.PermBillingAdmin) {
		t.Fatal("admin should manage billing")
	}
}

func TestRBAC_Hierarchy(t *testing.T) {
	if !auth.AtLeast(auth.RoleAdmin, auth.RoleDispatcher) {
		t.Fatal("admin should satisfy dispatcher level")
	}
	if auth.AtLeast(auth.RoleCustomer, auth.RoleDriver) {
		t.Fatal("customer should not satisfy driver level")
	}
}
