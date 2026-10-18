package opsdashboard

import (
	"testing"
	"time"
)

func TestBuildLifecycleAndBatches(t *testing.T) {
	tests := []struct {
		name                 string
		op                   TenantOperation
		wantLifecycle        string
		wantFirstMetric      string
		wantLicenseValue     float64
		wantSummaryLifecycle any
	}{
		{
			name: "tenant onboarded enters trial and publishes seats",
			op: TenantOperation{
				AccountID:  "acct-acme",
				TenantID:   "acme-co",
				TenantName: "Acme Co",
				Plan:       "pro",
				Seats:      12,
				AdminUser:  "ops-analyst",
				Action:     ActionTenantOnboarded,
				At:         time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC),
			},
			wantLifecycle:        "trialing",
			wantFirstMetric:      "saas.tenant.onboarded",
			wantLicenseValue:     12,
			wantSummaryLifecycle: "trialing",
		},
		{
			name: "activation moves tenant to active",
			op: TenantOperation{
				AccountID:  "acct-acme",
				TenantID:   "acme-co",
				TenantName: "Acme Co",
				Plan:       "pro",
				Seats:      20,
				AdminUser:  "ops-analyst",
				Action:     ActionSubscriptionActivated,
				At:         time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC),
			},
			wantLifecycle:        "active",
			wantFirstMetric:      "saas.subscription.activated",
			wantLicenseValue:     20,
			wantSummaryLifecycle: "active",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snapshot := BuildLifecycle(tt.op)
			if snapshot.Lifecycle != tt.wantLifecycle {
				t.Fatalf("lifecycle = %q, want %q", snapshot.Lifecycle, tt.wantLifecycle)
			}

			metricsReq := buildMetricsBatch(tt.op, snapshot)
			if len(metricsReq.Points) != 4 {
				t.Fatalf("metrics count = %d, want 4", len(metricsReq.Points))
			}
			if metricsReq.Points[0].Name != tt.wantFirstMetric {
				t.Fatalf("first metric = %q, want %q", metricsReq.Points[0].Name, tt.wantFirstMetric)
			}
			if metricsReq.Points[3].Value != tt.wantLicenseValue {
				t.Fatalf("license metric = %v, want %v", metricsReq.Points[3].Value, tt.wantLicenseValue)
			}

			realtimeReq := buildRealtimeBatch("ops-dashboard-live", tt.op, snapshot)
			if len(realtimeReq.Messages) != 2 {
				t.Fatalf("events count = %d, want 2", len(realtimeReq.Messages))
			}
			if got := realtimeReq.Messages[0].Data["lifecycle"]; got != tt.wantSummaryLifecycle {
				t.Fatalf("summary lifecycle = %v, want %v", got, tt.wantSummaryLifecycle)
			}
		})
	}
}
