package server

import (
	"testing"

	dispatchv1 "github.com/keix40/omnifleet/gen/go/dispatch/v1"
)

func TestCanTransition(t *testing.T) {
	if !canTransition(dispatchv1.JobStatus_JOB_STATUS_CREATED, dispatchv1.JobStatus_JOB_STATUS_ASSIGNED) {
		t.Fatal("expected created->assigned")
	}
	if canTransition(dispatchv1.JobStatus_JOB_STATUS_DELIVERED, dispatchv1.JobStatus_JOB_STATUS_EN_ROUTE) {
		t.Fatal("delivered should not go en_route")
	}
}
