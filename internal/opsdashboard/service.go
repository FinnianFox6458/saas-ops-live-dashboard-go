package opsdashboard

import (
	"context"
	"fmt"
)

type Service struct {
	infrai  *InfraiClient
	channel string
	room    string
}

func NewService(infrai *InfraiClient, channel, room string) *Service {
	return &Service{infrai: infrai, channel: channel, room: room}
}

func (s *Service) RunSample(ctx context.Context, op TenantOperation) (DashboardResult, error) {
	if err := s.infrai.RealtimeChannelCreate(ctx, realtimeChannelCreateRequest{
		Channel: s.channel,
		Type:    "public",
		Vendor:  "infrai",
	}); err != nil {
		return DashboardResult{}, err
	}

	if err := s.infrai.RTCRoomCreate(ctx, rtcRoomCreateRequest{
		Name:            s.room,
		MaxParticipants: 10,
		EmptyTimeoutS:   300,
		Region:          "global",
	}); err != nil {
		return DashboardResult{}, err
	}

	var dashboardToken map[string]any
	if err := s.infrai.RealtimeTokenIssue(ctx, realtimeTokenIssueRequest{
		ClientID:     "dashboard-client",
		Channels:     []string{s.channel},
		Capabilities: []string{"subscribe", "presence"},
		TTLSeconds:   3600,
	}, &dashboardToken); err != nil {
		return DashboardResult{}, err
	}

	var operatorToken map[string]any
	if err := s.infrai.RTCTokenIssue(ctx, rtcTokenIssueRequest{
		Room:         s.room,
		Identity:     op.AdminUser,
		DisplayName:  op.AdminUser,
		TTLS:         3600,
		CanPublish:   true,
		CanSubscribe: true,
	}, &operatorToken); err != nil {
		return DashboardResult{}, err
	}

	snapshot := BuildLifecycle(op)
	metricsReq := buildMetricsBatch(op, snapshot)
	publishReq := buildRealtimeBatch(s.channel, op, snapshot)

	var metricsOut map[string]any
	if err := s.infrai.MetricsBatch(ctx, metricsReq, &metricsOut); err != nil {
		return DashboardResult{}, err
	}

	var publishOut map[string]any
	if err := s.infrai.RealtimePublishBatch(ctx, publishReq, &publishOut); err != nil {
		return DashboardResult{}, err
	}

	return DashboardResult{
		Channel:              s.channel,
		Room:                 s.room,
		TenantID:             op.TenantID,
		Lifecycle:            snapshot.Lifecycle,
		MetricsPublished:     len(metricsReq.Points),
		EventsPublished:      len(publishReq.Messages),
		DashboardTokenIssued: dashboardToken != nil,
		OperatorTokenIssued:  operatorToken != nil,
	}, nil
}

func buildMetricsBatch(op TenantOperation, snapshot LifecycleSnapshot) metricsBatchRequest {
	ts := op.At.Unix()
	labels := map[string]any{
		"tenant_id": op.TenantID,
		"plan":      op.Plan,
		"action":    string(op.Action),
	}

	metrics := []metricsBatchItem{
		{
			Name:      metricNameForAction(op.Action),
			Value:     1,
			Timestamp: ts,
			Labels:    labels,
		},
		{
			Name:      "saas.tenants.active",
			Value:     snapshot.ActiveTenants,
			Timestamp: ts,
			Labels:    labels,
		},
		{
			Name:      "saas.tenants.trial",
			Value:     snapshot.TrialTenants,
			Timestamp: ts,
			Labels:    labels,
		},
		{
			Name:      "saas.licenses.active",
			Value:     snapshot.Licenses,
			Timestamp: ts,
			Labels:    labels,
		},
	}

	return metricsBatchRequest{Points: metrics}
}

func buildRealtimeBatch(channel string, op TenantOperation, snapshot LifecycleSnapshot) realtimePublishBatchRequest {
	summary := map[string]any{
		"tenant_id":   op.TenantID,
		"tenant_name": op.TenantName,
		"plan":        op.Plan,
		"lifecycle":   snapshot.Lifecycle,
		"seats":       op.Seats,
		"action":      string(op.Action),
	}

	numbers := map[string]any{
		"active_tenants": snapshot.ActiveTenants,
		"trial_tenants":  snapshot.TrialTenants,
		"suspended":      snapshot.Suspended,
		"licenses":       snapshot.Licenses,
		"metric_family":  fmt.Sprintf("ops/%s", op.TenantID),
	}

	return realtimePublishBatchRequest{
		Messages: []realtimePublishItem{
			{
				Channel:   channel,
				Event:     "tenant.lifecycle.updated",
				Data:      summary,
				AccountID: op.AccountID,
			},
			{
				Channel:   channel,
				Event:     "dashboard.metrics.updated",
				Data:      numbers,
				AccountID: op.AccountID,
			},
		},
	}
}

func metricNameForAction(action Action) string {
	switch action {
	case ActionTenantOnboarded:
		return "saas.tenant.onboarded"
	case ActionSubscriptionActivated:
		return "saas.subscription.activated"
	case ActionSubscriptionCanceled:
		return "saas.subscription.canceled"
	case ActionAdminSuspendedTenant:
		return "saas.admin.tenant_suspended"
	default:
		return "saas.tenant.changed"
	}
}
