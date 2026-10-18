package opsdashboard

import "time"

type Action string

const (
	ActionTenantOnboarded Action = "tenant_onboarded"
	ActionSubscriptionActivated Action = "subscription_activated"
	ActionSubscriptionCanceled Action = "subscription_canceled"
	ActionAdminSuspendedTenant Action = "admin_suspended_tenant"
)

type TenantOperation struct {
	AccountID  string
	TenantID   string
	TenantName string
	Plan       string
	Seats      int
	AdminUser  string
	Action     Action
	At         time.Time
}

type LifecycleSnapshot struct {
	Lifecycle     string
	ActiveTenants float64
	TrialTenants  float64
	Suspended     float64
	Licenses      float64
}

type DashboardResult struct {
	Channel              string
	Room                 string
	TenantID             string
	Lifecycle            string
	MetricsPublished     int
	EventsPublished      int
	DashboardTokenIssued bool
	OperatorTokenIssued  bool
}

func BuildLifecycle(op TenantOperation) LifecycleSnapshot {
	switch op.Action {
	case ActionTenantOnboarded:
		return LifecycleSnapshot{
			Lifecycle:     "trialing",
			ActiveTenants: 0,
			TrialTenants:  1,
			Suspended:     0,
			Licenses:      float64(op.Seats),
		}
	case ActionSubscriptionActivated:
		return LifecycleSnapshot{
			Lifecycle:     "active",
			ActiveTenants: 1,
			TrialTenants:  0,
			Suspended:     0,
			Licenses:      float64(op.Seats),
		}
	case ActionSubscriptionCanceled:
		return LifecycleSnapshot{
			Lifecycle:     "canceled",
			ActiveTenants: 0,
			TrialTenants:  0,
			Suspended:     0,
			Licenses:      0,
		}
	case ActionAdminSuspendedTenant:
		return LifecycleSnapshot{
			Lifecycle:     "suspended",
			ActiveTenants: 0,
			TrialTenants:  0,
			Suspended:     1,
			Licenses:      float64(op.Seats),
		}
	default:
		return LifecycleSnapshot{Lifecycle: "unknown"}
	}
}
