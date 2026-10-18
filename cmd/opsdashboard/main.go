package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/example/saas-ops-live-dashboard-go/internal/opsdashboard"
)

func main() {
	apiKey := os.Getenv("INFRAI_API_KEY")
	if apiKey == "" {
		log.Fatal("INFRAI_API_KEY is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	infrai := opsdashboard.NewInfraiClient(apiKey)
	service := opsdashboard.NewService(infrai, "ops-dashboard-live", "ops-war-room")

	result, err := service.RunSample(ctx, opsdashboard.TenantOperation{
		AccountID:  "acct-acme",
		TenantID:   "acme-co",
		TenantName: "Acme Co",
		Plan:       "pro",
		Seats:      12,
		AdminUser:  "ops-analyst",
		Action:     opsdashboard.ActionTenantOnboarded,
		At:         time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC),
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("channel: %s\n", result.Channel)
	fmt.Printf("room: %s\n", result.Room)
	fmt.Printf("tenant: %s\n", result.TenantID)
	fmt.Printf("lifecycle: %s\n", result.Lifecycle)
	fmt.Printf("published metrics: %d\n", result.MetricsPublished)
	fmt.Printf("published events: %d\n", result.EventsPublished)
	fmt.Printf("dashboard token issued: %t\n", result.DashboardTokenIssued)
	fmt.Printf("operator token issued: %t\n", result.OperatorTokenIssued)
}
