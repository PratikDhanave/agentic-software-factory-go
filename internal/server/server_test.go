package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/PratikDhanave/agentic-software-factory-go/internal/engine"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/gateway"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/state"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/store"
)

func newTestServer(t *testing.T) (*Server, string) {
	t.Helper()
	root := t.TempDir()
	specsDir := filepath.Join(root, "specs")
	if err := os.MkdirAll(specsDir, 0755); err != nil {
		t.Fatal(err)
	}
	specYAML := `id: "spec-token-bucket"
version: "1.0"
title: "Concurrent Token-Bucket Rate Limiter"
description: "Thread-safe token-bucket rate limiter"
intent:
  goal: "Implement a zero-dependency Go token-bucket rate limiter"
  success_criteria:
    - "Compiles cleanly"
constraints:
  - id: "no-unsafe"
    type: "must_not"
    description: "unsafe.Pointer"
steps:
  - id: "step-1"
    name: "Implement & Test"
    description: "Build limiter and race tests"
`
	if err := os.WriteFile(filepath.Join(specsDir, "token-bucket-limiter.yaml"), []byte(specYAML), 0644); err != nil {
		t.Fatal(err)
	}

	st, err := store.Open(filepath.Join(root, ".asf", "factory.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })

	gw := gateway.New(st)
	eng := engine.New(root, st, gw)
	return New(root, st, gw, eng), root
}

func doReq(t *testing.T, h http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatal(err)
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	req.RemoteAddr = "127.0.0.1:54321"
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestControlPlaneEndToEnd(t *testing.T) {
	srv, root := newTestServer(t)
	h := srv.Handler()

	// 1. Embedded UI root serves HTML
	uiRec := doReq(t, h, http.MethodGet, "/", nil)
	if uiRec.Code != http.StatusOK || !strings.Contains(uiRec.Header().Get("Content-Type"), "text/html") {
		t.Fatalf("expected 200 text/html on GET /, got %d (%s)", uiRec.Code, uiRec.Header().Get("Content-Type"))
	}

	// 2. Non-loopback requests to /api/* are blocked when ASF_API_KEY is unset
	extReq := httptest.NewRequest(http.MethodGet, "/api/overview", nil)
	extReq.RemoteAddr = "203.0.113.42:40000"
	extRec := httptest.NewRecorder()
	h.ServeHTTP(extRec, extReq)
	if extRec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for non-loopback request, got %d", extRec.Code)
	}

	// 3. State Machines endpoint + transition validator
	smRec := doReq(t, h, http.MethodGet, "/api/state-machines", nil)
	if smRec.Code != http.StatusOK {
		t.Fatalf("expected 200 on /api/state-machines, got %d", smRec.Code)
	}
	valRec := doReq(t, h, http.MethodPost, "/api/state-machines/validate", map[string]string{
		"machine": "factory-run",
		"from":    "queued",
		"to":      "completed",
	})
	var valBody struct {
		Valid bool `json:"valid"`
	}
	_ = json.Unmarshal(valRec.Body.Bytes(), &valBody)
	if valBody.Valid {
		t.Fatal("expected queued -> completed direct jump to be rejected by factory-run state machine")
	}

	// 4. Switch to Crawl mode -> Start Run -> Verify pause at awaiting_approval -> Approve -> Complete -> Promote
	autRec := doReq(t, h, http.MethodPost, "/api/autonomy", map[string]string{"mode": "crawl"})
	if autRec.Code != http.StatusOK {
		t.Fatalf("expected 200 setting crawl mode, got %d: %s", autRec.Code, autRec.Body.String())
	}

	runRec := doReq(t, h, http.MethodPost, "/api/runs", map[string]any{
		"spec_id": "spec-token-bucket",
		"prompt":  "Build token bucket rate limiter",
	})
	if runRec.Code != http.StatusCreated {
		t.Fatalf("expected 201 starting run, got %d: %s", runRec.Code, runRec.Body.String())
	}
	var created store.RunRecord
	_ = json.Unmarshal(runRec.Body.Bytes(), &created)

	// Wait briefly for async planning goroutine to pause at awaiting_approval
	deadline := time.Now().Add(5 * time.Second)
	var cur store.RunRecord
	for time.Now().Before(deadline) {
		r := doReq(t, h, http.MethodGet, "/api/runs/"+created.ID, nil)
		_ = json.Unmarshal(r.Body.Bytes(), &cur)
		if cur.Status == string(state.RunAwaitingApproval) {
			break
		}
		time.Sleep(30 * time.Millisecond)
	}
	if cur.Status != string(state.RunAwaitingApproval) {
		t.Fatalf("expected run to pause in awaiting_approval under Crawl mode, got %s", cur.Status)
	}

	// Approve checkpoint -> watch synthesis + real go build / go vet / go test -race complete!
	appRec := doReq(t, h, http.MethodPost, "/api/runs/"+created.ID+"/approve", map[string]any{})
	if appRec.Code != http.StatusOK {
		t.Fatalf("approve failed: %d %s", appRec.Code, appRec.Body.String())
	}

	verifyDeadline := time.Now().Add(40 * time.Second)
	for time.Now().Before(verifyDeadline) {
		r := doReq(t, h, http.MethodGet, "/api/runs/"+created.ID, nil)
		_ = json.Unmarshal(r.Body.Bytes(), &cur)
		if cur.Status == string(state.RunCompleted) || cur.Status == string(state.RunFailed) {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if cur.Status != string(state.RunCompleted) {
		t.Fatalf("expected run to complete, got %s (err: %s)", cur.Status, cur.Error)
	}
	if !cur.BuildPassed || !cur.VetPassed || !cur.TestsPassed || cur.Score < 90 {
		t.Fatalf("unexpected verification flags: build=%v vet=%v tests=%v score=%d", cur.BuildPassed, cur.VetPassed, cur.TestsPassed, cur.Score)
	}

	// Promote verified artifacts to generated/
	promRec := doReq(t, h, http.MethodPost, "/api/runs/"+created.ID+"/promote", map[string]string{})
	if promRec.Code != http.StatusOK {
		t.Fatalf("promote failed: %d %s", promRec.Code, promRec.Body.String())
	}
	var promoted store.RunRecord
	_ = json.Unmarshal(promRec.Body.Bytes(), &promoted)
	if promoted.Promotion != string(state.PromoteCompleted) {
		t.Fatalf("expected promotion_state=promoted, got %s", promoted.Promotion)
	}
	if _, err := os.Stat(filepath.Join(root, promoted.PromotedDir, "limiter.go")); err != nil {
		t.Fatalf("expected promoted limiter.go on disk: %v", err)
	}

	// 5. MCP JSON-RPC 2.0 tools/list & tools/call
	mcpRec := doReq(t, h, http.MethodPost, "/api/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "validate_transition",
			"arguments": map[string]any{
				"machine": "autonomy-mode",
				"from":    "crawl",
				"to":      "run",
			},
		},
	})
	if mcpRec.Code != http.StatusOK || !strings.Contains(mcpRec.Body.String(), `\"valid\": true`) {
		t.Fatalf("unexpected MCP response: %d %s", mcpRec.Code, mcpRec.Body.String())
	}
}
