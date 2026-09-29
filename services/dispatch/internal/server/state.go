package server

import dispatchv1 "github.com/keix40/omnifleet/gen/go/dispatch/v1"

var allowedTransitions = map[dispatchv1.JobStatus][]dispatchv1.JobStatus{
	dispatchv1.JobStatus_JOB_STATUS_CREATED: {
		dispatchv1.JobStatus_JOB_STATUS_ASSIGNED,
		dispatchv1.JobStatus_JOB_STATUS_CANCELLED,
	},
	dispatchv1.JobStatus_JOB_STATUS_ASSIGNED: {
		dispatchv1.JobStatus_JOB_STATUS_EN_ROUTE,
		dispatchv1.JobStatus_JOB_STATUS_CANCELLED,
	},
	dispatchv1.JobStatus_JOB_STATUS_EN_ROUTE: {
		dispatchv1.JobStatus_JOB_STATUS_PICKED_UP,
		dispatchv1.JobStatus_JOB_STATUS_CANCELLED,
	},
	dispatchv1.JobStatus_JOB_STATUS_PICKED_UP: {
		dispatchv1.JobStatus_JOB_STATUS_DELIVERED,
		dispatchv1.JobStatus_JOB_STATUS_CANCELLED,
	},
}

func canTransition(from, to dispatchv1.JobStatus) bool {
	if from == to {
		return true
	}
	next, ok := allowedTransitions[from]
	if !ok {
		return false
	}
	for _, s := range next {
		if s == to {
			return true
		}
	}
	return false
}

func statusToDB(s dispatchv1.JobStatus) string {
	switch s {
	case dispatchv1.JobStatus_JOB_STATUS_CREATED:
		return "created"
	case dispatchv1.JobStatus_JOB_STATUS_ASSIGNED:
		return "assigned"
	case dispatchv1.JobStatus_JOB_STATUS_EN_ROUTE:
		return "en_route"
	case dispatchv1.JobStatus_JOB_STATUS_PICKED_UP:
		return "picked_up"
	case dispatchv1.JobStatus_JOB_STATUS_DELIVERED:
		return "delivered"
	case dispatchv1.JobStatus_JOB_STATUS_CANCELLED:
		return "cancelled"
	default:
		return "created"
	}
}

func statusFromDB(s string) dispatchv1.JobStatus {
	switch s {
	case "created":
		return dispatchv1.JobStatus_JOB_STATUS_CREATED
	case "assigned":
		return dispatchv1.JobStatus_JOB_STATUS_ASSIGNED
	case "en_route":
		return dispatchv1.JobStatus_JOB_STATUS_EN_ROUTE
	case "picked_up":
		return dispatchv1.JobStatus_JOB_STATUS_PICKED_UP
	case "delivered":
		return dispatchv1.JobStatus_JOB_STATUS_DELIVERED
	case "cancelled":
		return dispatchv1.JobStatus_JOB_STATUS_CANCELLED
	default:
		return dispatchv1.JobStatus_JOB_STATUS_UNSPECIFIED
	}
}
