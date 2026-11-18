package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestControlPlaneClosedLoopGovernanceAndExecution(t *testing.T) {
	workDir := t.TempDir()
	t.Setenv("ASF_LLM_PROVIDER", "mock")
	t.Setenv("ASF_DB_PATH", filepath.Join(workDir, ".asf", "factory.db"))
	t.Setenv("ASF_PHASE", "crawl")

	if err := os.MkdirAll(filepath.Join(workDir, "specs"), 0755); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(filepath.Join(workDir, "specs", "example.yaml"), []byte(specYAMLFixture), 0644)

	srv, err := newWebServer(workDir)
	if err != nil {
		t.Fatalf("newWebServer: %v", err)
	}
	defer srv.container.Close()
	router := srv.routes()

	doJSON := func(method, path string, body interface{}) *httptest.ResponseRecorder {
		var buf bytes.Buffer
		if body != nil {
			_ = json.NewEncoder(&buf).Encode(body)
		}
		req := httptest.NewRequest(method, "http://localhost:8080"+path, &buf)
		req.RemoteAddr = "127.0.0.1:54321"
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
	}

	// 1. Overview
	ovRec := doJSON(http.MethodGet, "/api/system/overview", nil)
	if ovRec.Code != http.StatusOK {
		t.Fatalf("overview status=%d body=%s", ovRec.Code, ovRec.Body.String())
	}

	// 2. Coordinate workflow in Crawl phase -> must pause for human approval
	genRec := doJSON(http.MethodPost, "/api/v1/workflows/generate", map[string]interface{}{
		"issue_title":       "Implement OAuth2 PKCE login flow",
		"issue_description": "Add secure OAuth2 token validation and refresh logic",
	})
	if genRec.Code != http.StatusOK {
		t.Fatalf("generate status=%d body=%s", genRec.Code, genRec.Body.String())
	}
	var genResp map[string]interface{}
	_ = json.Unmarshal(genRec.Body.Bytes(), &genResp)
	if genResp["requires_approval"] != true {
		t.Fatalf("expected requires_approval=true in crawl phase, got %+v", genResp)
	}
	sessionID := genResp["session_id"].(string)

	// 3. Pending approvals must list session gates
	pendRec := doJSON(http.MethodGet, "/api/v1/approvals/pending", nil)
	var pendResp struct {
		TotalPending int                      `json:"total_pending"`
		Approvals    []map[string]interface{} `json:"approvals"`
	}
	_ = json.Unmarshal(pendRec.Body.Bytes(), &pendResp)
	if pendResp.TotalPending == 0 {
		t.Fatalf("expected pending approvals > 0, got 0")
	}

	// 4. Approve entire session -> triggers DAG execution and completes session
	appRec := doJSON(http.MethodPost, "/api/v1/approvals/submit", map[string]interface{}{
		"session_id": sessionID,
		"approved":   true,
		"approver":   "lead-architect",
		"comments":   "LGTM, ship it",
	})
	if appRec.Code != http.StatusOK {
		t.Fatalf("approval submit status=%d body=%s", appRec.Code, appRec.Body.String())
	}
	var appResp map[string]interface{}
	_ = json.Unmarshal(appRec.Body.Bytes(), &appResp)
	if appResp["status"] != "approved" || appResp["execution"] == nil {
		t.Fatalf("expected approved status with DAG execution result, got %+v", appResp)
	}

	// 5. Simulate learning batch & verify self-improvement metrics
	simRec := doJSON(http.MethodPost, "/api/v1/learning/simulate", map[string]int{"count": 40})
	if simRec.Code != http.StatusOK {
		t.Fatalf("learning simulate status=%d body=%s", simRec.Code, simRec.Body.String())
	}
	metRec := doJSON(http.MethodGet, "/api/v1/metrics/self-improvement", nil)
	if metRec.Code != http.StatusOK {
		t.Fatalf("metrics status=%d body=%s", metRec.Code, metRec.Body.String())
	}

	// 6. Verify mounted Factory MCP JSON-RPC 2.0 endpoint
	mcpRec := doJSON(http.MethodPost, "/api/mcp/factory", map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/list",
		"params":  map[string]interface{}{},
	})
	if mcpRec.Code != http.StatusOK {
		t.Fatalf("mcp tools/list status=%d body=%s", mcpRec.Code, mcpRec.Body.String())
	}

	// 7. Verify Autonomous 6-Stage Factory Assembly Line (pause at Crawl gate -> approve -> compile & race-test -> Judge -> promote!)
	runRec := doJSON(http.MethodPost, "/api/v1/factory/runs", map[string]interface{}{
		"title":       "Concurrent Token-Bucket Rate Limiter",
		"description": "Implement a thread-safe Go token-bucket rate limiter with unit tests",
		"sync":        true,
	})
	if runRec.Code != http.StatusOK {
		t.Fatalf("start factory run status=%d body=%s", runRec.Code, runRec.Body.String())
	}
	var startedRun map[string]interface{}
	_ = json.Unmarshal(runRec.Body.Bytes(), &startedRun)
	if startedRun["status"] != "awaiting_approval" {
		t.Fatalf("expected Crawl-phase FactoryRun to pause at awaiting_approval, got %+v", startedRun)
	}
	runID := startedRun["id"].(string)

	// Approve and run Stages 3->6 synchronously (real isolated workspace go build + go test + go vet + Judge!)
	resumedRec := doJSON(http.MethodPost, "/api/v1/factory/runs/"+runID+"/approve", map[string]interface{}{
		"approver": "staff-engineer",
		"sync":     true,
	})
	if resumedRec.Code != http.StatusOK {
		t.Fatalf("approve factory run status=%d body=%s", resumedRec.Code, resumedRec.Body.String())
	}
	var completedRun map[string]interface{}
	_ = json.Unmarshal(resumedRec.Body.Bytes(), &completedRun)
	if completedRun["status"] != "completed" {
		t.Fatalf("expected completed FactoryRun after real workspace build & test, got %+v", completedRun)
	}

	// Promote verified package into generated/
	promRec := doJSON(http.MethodPost, "/api/v1/factory/runs/"+runID+"/promote", map[string]interface{}{
		"target_subdir": "limiter",
	})
	if promRec.Code != http.StatusOK {
		t.Fatalf("promote factory run status=%d body=%s", promRec.Code, promRec.Body.String())
	}
	if _, err := os.Stat(filepath.Join(workDir, "generated", "limiter", "limiter.go")); err != nil {
		t.Fatalf("expected promoted limiter.go on disk, got err=%v", err)
	}
}

const specYAMLFixture = `id: "spec-001"
version: "1.0"
title: "Example Spec"
description: "Test spec"
intent:
  goal: "Test goal"
steps:
  - id: "s1"
    name: "Step 1"
    description: "Do step 1"
`
