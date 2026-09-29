package httpapi

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	billingv1 "github.com/keix40/omnifleet/gen/go/billing/v1"
	commonv1 "github.com/keix40/omnifleet/gen/go/common/v1"
	dispatchv1 "github.com/keix40/omnifleet/gen/go/dispatch/v1"
	etav1 "github.com/keix40/omnifleet/gen/go/eta/v1"
	notificationsv1 "github.com/keix40/omnifleet/gen/go/notifications/v1"
	"github.com/keix40/omnifleet/pkg/auth"
)

type geoPointJSON struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type createJobRequest struct {
	Pickup       geoPointJSON `json:"pickup"`
	Dropoff      geoPointJSON `json:"dropoff"`
	PickupLabel  string       `json:"pickup_label"`
	DropoffLabel string       `json:"dropoff_label"`
	AutoAssign   bool         `json:"auto_assign"`
}

func (s *Server) handleCreateJob(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromContext(r.Context())
	if !auth.HasPermission(claims.Role, auth.PermDispatchJobs) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	var body createJobRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	resp, err := s.dispatchClient.CreateJob(r.Context(), &dispatchv1.CreateJobRequest{
		TenantId:     claims.TenantID,
		Pickup:       &commonv1.GeoPoint{Latitude: body.Pickup.Latitude, Longitude: body.Pickup.Longitude},
		Dropoff:      &commonv1.GeoPoint{Latitude: body.Dropoff.Latitude, Longitude: body.Dropoff.Longitude},
		PickupLabel:  body.PickupLabel,
		DropoffLabel: body.DropoffLabel,
		AutoAssign:   body.AutoAssign,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	writeJSON(w, resp.Job)
}

func (s *Server) handleListJobs(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromContext(r.Context())
	if !auth.HasPermission(claims.Role, auth.PermViewDispatch) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	req := &dispatchv1.ListJobsRequest{TenantId: claims.TenantID}
	if auth.HasPermission(claims.Role, auth.PermUpdateOwnJobs) && !auth.HasPermission(claims.Role, auth.PermDispatchJobs) {
		req.DriverId = claims.UserID
	}
	resp, err := s.dispatchClient.ListJobs(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	writeJSON(w, map[string]any{"jobs": resp.Jobs})
}

func (s *Server) handleGetJob(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromContext(r.Context())
	if !auth.HasPermission(claims.Role, auth.PermViewDispatch) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	jobID := chi.URLParam(r, "jobID")
	resp, err := s.dispatchClient.GetJob(r.Context(), &dispatchv1.GetJobRequest{
		TenantId: claims.TenantID,
		JobId:    jobID,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	writeJSON(w, resp.Job)
}

type assignJobRequest struct {
	VehicleID string `json:"vehicle_id"`
	DriverID  string `json:"driver_id"`
}

func (s *Server) handleAssignJob(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromContext(r.Context())
	if !auth.HasPermission(claims.Role, auth.PermDispatchJobs) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	var body assignJobRequest
	_ = json.NewDecoder(r.Body).Decode(&body)
	jobID := chi.URLParam(r, "jobID")
	resp, err := s.dispatchClient.AssignJob(r.Context(), &dispatchv1.AssignJobRequest{
		TenantId:  claims.TenantID,
		JobId:     jobID,
		VehicleId: body.VehicleID,
		DriverId:  body.DriverID,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	writeJSON(w, resp.Job)
}

func (s *Server) handleAutoAssignJob(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromContext(r.Context())
	if !auth.HasPermission(claims.Role, auth.PermDispatchJobs) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	jobID := chi.URLParam(r, "jobID")
	resp, err := s.dispatchClient.AutoAssignNearest(r.Context(), &dispatchv1.AutoAssignNearestRequest{
		TenantId: claims.TenantID,
		JobId:    jobID,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	writeJSON(w, resp.Job)
}

type updateJobStatusRequest struct {
	Status string `json:"status"`
}

func (s *Server) handleUpdateJobStatus(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromContext(r.Context())
	canDispatch := auth.HasPermission(claims.Role, auth.PermDispatchJobs)
	canDriver := auth.HasPermission(claims.Role, auth.PermUpdateOwnJobs)
	if !canDispatch && !canDriver {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	var body updateJobStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	jobID := chi.URLParam(r, "jobID")
	status := parseJobStatus(body.Status)
	actor := ""
	if canDriver && !canDispatch {
		actor = claims.UserID
	}
	resp, err := s.dispatchClient.UpdateJobStatus(r.Context(), &dispatchv1.UpdateJobStatusRequest{
		TenantId:    claims.TenantID,
		JobId:       jobID,
		Status:      status,
		ActorUserId: actor,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	writeJSON(w, resp.Job)
}

func parseJobStatus(raw string) dispatchv1.JobStatus {
	switch raw {
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

type computeETARequest struct {
	VehicleID     string       `json:"vehicle_id"`
	Destination   geoPointJSON `json:"destination"`
	PublishUpdate bool         `json:"publish_update"`
}

func (s *Server) handleComputeETA(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromContext(r.Context())
	if !auth.HasPermission(claims.Role, auth.PermViewETA) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	var body computeETARequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	resp, err := s.etaClient.ComputeETA(r.Context(), &etav1.ComputeETARequest{
		TenantId:  claims.TenantID,
		VehicleId: body.VehicleID,
		Destination: &commonv1.GeoPoint{
			Latitude:  body.Destination.Latitude,
			Longitude: body.Destination.Longitude,
		},
		PublishUpdate: body.PublishUpdate,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	writeJSON(w, resp)
}

func (s *Server) handleGetSubscription(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromContext(r.Context())
	if !auth.HasPermission(claims.Role, auth.PermViewBilling) && !auth.HasPermission(claims.Role, auth.PermBillingAdmin) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	resp, err := s.billingClient.GetSubscription(r.Context(), &billingv1.GetSubscriptionRequest{TenantId: claims.TenantID})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	writeJSON(w, resp)
}

func (s *Server) handleGetUsage(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromContext(r.Context())
	if !auth.HasPermission(claims.Role, auth.PermViewBilling) && !auth.HasPermission(claims.Role, auth.PermBillingAdmin) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	resp, err := s.billingClient.GetUsage(r.Context(), &billingv1.GetUsageRequest{TenantId: claims.TenantID})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	writeJSON(w, resp)
}

func (s *Server) handleStripeWebhook(w http.ResponseWriter, r *http.Request) {
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "read body", http.StatusBadRequest)
		return
	}
	resp, err := s.billingClient.HandleStripeWebhook(r.Context(), &billingv1.HandleStripeWebhookRequest{
		Payload:         payload,
		StripeSignature: r.Header.Get("Stripe-Signature"),
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	if !resp.Accepted {
		http.Error(w, resp.Message, http.StatusBadRequest)
		return
	}
	writeJSON(w, resp)
}

func (s *Server) handleGetNotificationPrefs(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromContext(r.Context())
	if !auth.HasPermission(claims.Role, auth.PermManageNotifications) &&
		!auth.HasPermission(claims.Role, auth.PermDispatchJobs) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	resp, err := s.notificationsClient.GetPreferences(r.Context(), &notificationsv1.GetPreferencesRequest{
		TenantId: claims.TenantID,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	writeJSON(w, resp.Preferences)
}

func (s *Server) handleUpdateNotificationPrefs(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromContext(r.Context())
	if !auth.HasPermission(claims.Role, auth.PermManageNotifications) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	var prefs notificationsv1.NotificationPreferences
	if err := json.NewDecoder(r.Body).Decode(&prefs); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	prefs.TenantId = claims.TenantID
	resp, err := s.notificationsClient.UpdatePreferences(r.Context(), &notificationsv1.UpdatePreferencesRequest{
		Preferences: &prefs,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	writeJSON(w, resp.Preferences)
}
