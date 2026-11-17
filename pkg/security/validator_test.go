package security

import (
	"testing"
)

func TestValidateWorkflowID(t *testing.T) {
	v := &Validator{}

	testCases := []struct {
		id    string
		valid bool
	}{
		{"workflow-123", true},
		{"workflow_456", true},
		{"workflow@789", false},
		{"", false},
	}

	for _, tc := range testCases {
		err := v.ValidateWorkflowID(tc.id)
		if (err == nil) != tc.valid {
			t.Errorf("ValidateWorkflowID(%s): expected valid=%v, got err=%v", tc.id, tc.valid, err)
		}
	}
}

func TestValidateRepositoryName(t *testing.T) {
	v := &Validator{}

	testCases := []struct {
		repo  string
		valid bool
	}{
		{"owner/repo", true},
		{"github/cli", true},
		{"owner", false},
		{"owner/repo/extra", false},
	}

	for _, tc := range testCases {
		err := v.ValidateRepositoryName(tc.repo)
		if (err == nil) != tc.valid {
			t.Errorf("ValidateRepositoryName(%s): expected valid=%v, got err=%v", tc.repo, tc.valid, err)
		}
	}
}

func TestRateLimiter(t *testing.T) {
	rl := NewRateLimiter(5) // 5 requests per second

	// Should allow requests within limit
	for i := 0; i < 5; i++ {
		if !rl.Allow("test-ip") {
			t.Errorf("Request %d should be allowed", i+1)
		}
	}

	// Should reject 6th request
	if rl.Allow("test-ip") {
		t.Error("6th request should be rejected")
	}

	// Different IP should have separate limit
	if !rl.Allow("other-ip") {
		t.Error("Different IP should have its own limit")
	}
}

func TestIPWhitelist(t *testing.T) {
	w := NewIPWhitelist()

	w.Add("192.168.1.1")
	
	if !w.IsAllowed("192.168.1.1") {
		t.Error("Whitelisted IP should be allowed")
	}

	if w.IsAllowed("192.168.1.2") {
		t.Error("Non-whitelisted IP should not be allowed")
	}

	w.Remove("192.168.1.1")

	if w.IsAllowed("192.168.1.1") {
		t.Error("Removed IP should not be allowed")
	}
}

func TestAuditLog(t *testing.T) {
	al := NewAuditLog(10)

	al.Log("ACCESS", "user1", "workflow-1", "execute", "success", "Workflow executed")
	al.Log("ERROR", "system", "api", "query", "failed", "Database error")

	events := al.GetEvents(10)
	if len(events) != 2 {
		t.Errorf("Expected 2 events, got %d", len(events))
	}

	if events[0].EventType != "ACCESS" {
		t.Error("First event type incorrect")
	}
}
