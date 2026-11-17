package factory

import (
	"testing"

	"github.com/PratikDhanave/agentic-software-factory-go/pkg/metrics"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/queue"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/realtime"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/security"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/webhooks"
)

func TestIntegration(t *testing.T) {
	// Test webhooks
	processor := webhooks.NewEventProcessor("secret")
	if processor == nil {
		t.Fatal("Failed to create webhook processor")
	}

	// Test queue
	q := queue.NewQueue(2, 3)
	q.Start()
	defer q.Stop()

	// Test scheduler
	sched := queue.NewScheduler(q)
	sched.Start()
	defer sched.Stop()

	// Test metrics
	mc := metrics.NewMetricsCollector(100)
	if mc == nil {
		t.Fatal("Failed to create metrics collector")
	}

	// Test security
	v := &security.Validator{}
	rl := security.NewRateLimiter(10)
	al := security.NewAuditLog(50)

	// Test realtime hub
	hub := realtime.NewHub()
	hub.Start()

	// Test that all components work together
	if err := v.ValidateRepositoryName("owner/repo"); err != nil {
		t.Fatal("Validation failed")
	}

	if !rl.Allow("test-ip") {
		t.Fatal("Rate limiter rejected allowed request")
	}

	al.Log("TEST", "user", "resource", "action", "success", "test event")

	hub.BroadcastWorkflowStarted("test-wf-1", "test")

	if hub.ClientCount() == 0 {
		t.Log("Note: No connected clients for hub broadcast (expected)")
	}
}
