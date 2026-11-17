package webhooks

import (
	"encoding/json"
	"testing"
)

func TestVerifySignature(t *testing.T) {
	secret := "test-secret"
	processor := NewEventProcessor(secret)

	payload := []byte(`{"action":"opened"}`)
	expectedSig := "sha256=" + processor.computeSignature(payload)

	if !processor.VerifySignature(payload, expectedSig) {
		t.Error("Valid signature rejected")
	}

	if processor.VerifySignature(payload, "sha256=invalid") {
		t.Error("Invalid signature accepted")
	}
}

func TestProcessIssueEvent(t *testing.T) {
	processor := NewEventProcessor("")

	payload := []byte(`{
		"action":"opened",
		"issue":{"number":1,"title":"Test Issue","body":"Description"},
		"repository":{"full_name":"owner/repo","owner":{"login":"owner"},"html_url":"https://github.com/owner/repo"}
	}`)

	var event GitHubEvent
	json.Unmarshal(payload, &event)

	trigger, err := processor.processIssueEvent(&event)
	if err != nil {
		t.Fatalf("Failed to process issue: %v", err)
	}

	if trigger.Type != "issue-triage" {
		t.Errorf("Expected issue-triage, got %s", trigger.Type)
	}

	if trigger.Issue != 1 {
		t.Errorf("Expected issue 1, got %d", trigger.Issue)
	}
}

func TestProcessPullRequestEvent(t *testing.T) {
	processor := NewEventProcessor("")

	payload := []byte(`{
		"action":"opened",
		"pull_request":{"number":1,"title":"Test PR","body":"Description","head":{"sha":"abc123"}},
		"repository":{"full_name":"owner/repo","owner":{"login":"owner"},"html_url":"https://github.com/owner/repo"}
	}`)

	var event GitHubEvent
	json.Unmarshal(payload, &event)

	trigger, err := processor.processPullRequestEvent(&event)
	if err != nil {
		t.Fatalf("Failed to process PR: %v", err)
	}

	if trigger.Type != "pr-review" {
		t.Errorf("Expected pr-review, got %s", trigger.Type)
	}

	if trigger.PR != 1 {
		t.Errorf("Expected PR 1, got %d", trigger.PR)
	}

	if trigger.SHA != "abc123" {
		t.Errorf("Expected SHA abc123, got %s", trigger.SHA)
	}
}

func TestShouldTrigger(t *testing.T) {
	trigger := &WorkflowTrigger{
		Type: "issue-triage",
		Repo: "owner/repo",
	}

	if !trigger.ShouldTrigger() {
		t.Error("Valid trigger returned false")
	}

	trigger.Type = ""
	if trigger.ShouldTrigger() {
		t.Error("Empty type trigger returned true")
	}
}

func TestGetWorkflowInput(t *testing.T) {
	trigger := &WorkflowTrigger{
		Type: "issue-triage",
		Repo: "owner/repo",
		Issue: 1,
		Title: "Test",
	}

	input := trigger.GetWorkflowInput()

	if input["type"] != "issue-triage" {
		t.Error("Type not in input")
	}

	if input["repo"] != "owner/repo" {
		t.Error("Repo not in input")
	}
}
