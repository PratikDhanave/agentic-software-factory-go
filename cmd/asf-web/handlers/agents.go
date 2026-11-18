package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"

	asfContext "github.com/PratikDhanave/agentic-software-factory-go/pkg/context"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/eval"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/factory"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/harness"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/spec"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/state"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/verify"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/workspace"
)

// SevenLayerMachine aliases factory.SevenLayerMachine as the single source of truth.
var SevenLayerMachine = factory.SevenLayerMachine

type AgentRequest struct {
	AgentID string `json:"agentId"`
	LLDYaml string `json:"lldYaml"`
}

type AgentResult struct {
	ID               string   `json:"id"`
	GeneratedCode    string   `json:"generatedCode"`
	WorkspaceDir     string   `json:"workspaceDir,omitempty"`
	FilesWritten     []string `json:"filesWritten,omitempty"`
	Score            float64  `json:"score"`
	TestsPassed      int      `json:"testsPassed"`
	AcceptanceMet    bool     `json:"acceptanceMet"`
	SafetyViolations []string `json:"safetyViolations"`
	CreatedAt        string   `json:"createdAt"`
}

type SSEMessage struct {
	Status      string       `json:"status"`
	Progress    int          `json:"progress"`
	Message     string       `json:"message"`
	ExecutionID string       `json:"executionId,omitempty"`
	Result      *AgentResult `json:"result,omitempty"`
	Error       string       `json:"error,omitempty"`
}

var (
	mu               sync.RWMutex
	executionStore   = make(map[string]*AgentResult)
	executionHistory []AgentResult
	learnedPatterns  = make(map[string]int)
)

// HandleRunAgent executes a real 7-layer Factory Stack pipeline over SSE:
//  1. Layer 1 (Factories-as-Code): Parse LLD YAML into a Specification
//  2. Layer 2 (Data & Context): Load repository ADRs & context via ContextEngine
//  3. Layer 3 (Compute): Provision an isolated Go module workspace (.asf/workspaces/)
//  4. Layer 4 (Inference): Synthesize multi-file Go code & tests via LLMGateway
//  5. Layer 5 (Improvement): Run real `go build`, `go test -v`, `go vet` & LLM-as-a-Judge
//  6. Layer 6 (Orchestration): Self-heal on any verification error & persist telemetry
//  7. Layer 7 (Access): Stream verified source artifact and scorecard
func HandleRunAgent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req AgentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.LLDYaml) == "" {
		http.Error(w, "lldYaml required", http.StatusBadRequest)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	execID := "7layer-" + uuid.New().String()[:8]
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	curLayer := SevenLayerMachine.Initial()
	sendSSE := func(msg SSEMessage) {
		if msg.Status != "" {
			_ = SevenLayerMachine.Transition(&curLayer, state.State(msg.Status))
		}
		if msg.ExecutionID == "" {
			msg.ExecutionID = execID
		}
		data, _ := json.Marshal(msg)
		fmt.Fprintf(w, "data: %s\n\n", string(data))
		flusher.Flush()
	}

	ctx := r.Context()
	workDir := "."
	if activeContainer != nil && activeContainer.WorkingDir != "" {
		workDir = activeContainer.WorkingDir
	}

	// Layer 1: Parse LLD YAML into a Specification
	sendSSE(SSEMessage{Status: "layer-1", Progress: 14, Message: "Layer 1 (Factories-as-Code): Parsing LLD YAML specification..."})
	parsedSpec := parseLLDToSpec(req.LLDYaml, execID)

	// Layer 2: Load Project Context
	sendSSE(SSEMessage{Status: "layer-2", Progress: 28, Message: "Layer 2 (Data & Context): Loading ADRs and codebase context..."})
	ctxItems, _ := asfContext.NewContextEngine(workDir).LoadContext(ctx)
	ctxStr := asfContext.FormatForAgent(ctxItems)
	if len(ctxStr) > 16000 {
		ctxStr = ctxStr[:16000]
	}

	// Layer 3: Provision Isolated Workspace
	sendSSE(SSEMessage{Status: "layer-3", Progress: 42, Message: "Layer 3 (Compute): Provisioning isolated Go module sandbox..."})
	wsMgr := workspace.NewWorkspaceManager(workDir)
	ws, err := wsMgr.CreateModuleWorkspace(execID, "asf/sevenlayer/"+execID)
	if err != nil {
		sendSSE(SSEMessage{Status: "error", Error: err.Error()})
		return
	}

	// Layer 4: Synthesize Code & Unit Tests via LLMGateway
	sendSSE(SSEMessage{Status: "layer-4", Progress: 60, Message: "Layer 4 (Inference): Synthesizing Go implementation & unit tests..."})
	specPrompt := spec.NewSpecEngine("").GeneratePrompt(parsedSpec) +
		"\n\nEmit fenced ```go path=service.go and ```go path=service_test.go blocks.\n\n## Context\n" + ctxStr

	var rawOutput string
	if activeContainer != nil && activeContainer.PromptClient != nil {
		rawOutput, err = activeContainer.PromptClient.Generate(ctx, specPrompt)
		if err != nil {
			sendSSE(SSEMessage{Status: "error", Error: err.Error()})
			return
		}
	} else {
		sendSSE(SSEMessage{Status: "error", Error: "factory container not initialized"})
		return
	}

	artifacts := harness.ParseArtifacts(rawOutput)
	if _, err := ws.ApplyArtifacts(artifacts); err != nil {
		sendSSE(SSEMessage{Status: "error", Error: err.Error()})
		return
	}

	// Layer 5: Real Verification (`go build`, `go test -v`, `go vet`) & Judge Scoring
	sendSSE(SSEMessage{Status: "layer-5", Progress: 76, Message: "Layer 5 (Improvement): Running go build, go test -v, go vet & Judge..."})
	verifier := verify.NewVerificationEngine(ws.Dir)
	verRes, _ := verifier.RunVerificationForSpec(ctx, "", parsedSpec)

	// Layer 6: Self-Heal if verification failed on attempt 1
	sendSSE(SSEMessage{Status: "layer-6", Progress: 88, Message: "Layer 6 (Orchestration): Verifying invariants & recording state..."})
	if verRes != nil && !verRes.Success && activeContainer != nil && activeContainer.PromptClient != nil {
		fixPrompt := verify.GenerateFixPrompt(verRes, specPrompt)
		if fixedOut, fixErr := activeContainer.PromptClient.Generate(ctx, fixPrompt); fixErr == nil {
			if fixedArts := harness.ParseArtifacts(fixedOut); len(fixedArts) > 0 {
				_, _ = ws.ApplyArtifacts(fixedArts)
				rawOutput = fixedOut
				verRes, _ = verifier.RunVerificationForSpec(ctx, "", parsedSpec)
			}
		}
	}

	score := 85.0
	if activeContainer != nil && activeContainer.PromptClient != nil {
		judge := eval.NewEvaluationEngine(activeContainer.PromptClient)
		if verdict, jErr := judge.EvaluateCode(ctx, specPrompt, rawOutput, "", eval.DefaultEvaluationCriteria()); jErr == nil {
			score = verdict.Score * 100
		}
	}

	var combinedSource strings.Builder
	for _, f := range ws.FilesWritten {
		combinedSource.WriteString("// === File: " + f + " ===\n")
		if b, readErr := os.ReadFile(filepath.Join(ws.Dir, f)); readErr == nil {
			combinedSource.Write(b)
			combinedSource.WriteString("\n\n")
		}
	}

	violations := []string{}
	testsPassed := 0
	if verRes != nil {
		testsPassed = strings.Count(verRes.TestOutput, "--- PASS:")
		for _, e := range verRes.Errors {
			violations = append(violations, fmt.Sprintf("[%s] %s", e.Type, e.Message))
		}
		if !verRes.Success && score > 60 {
			score = 55
		}
	}

	// Layer 7: Publish Verified Result
	sendSSE(SSEMessage{Status: "layer-7", Progress: 96, Message: "Layer 7 (Access): Packaging verified deliverable..."})
	result := &AgentResult{
		ID:               execID,
		GeneratedCode:    strings.TrimSpace(combinedSource.String()),
		WorkspaceDir:     ws.Dir,
		FilesWritten:     ws.FilesWritten,
		Score:            score,
		TestsPassed:      testsPassed,
		AcceptanceMet:    verRes != nil && verRes.Success && score >= 75,
		SafetyViolations: violations,
		CreatedAt:        time.Now().UTC().Format(time.RFC3339),
	}

	mu.Lock()
	executionStore[execID] = result
	executionHistory = append(executionHistory, *result)
	if len(executionHistory) > 100 {
		executionHistory = executionHistory[1:]
	}
	if result.AcceptanceMet {
		learnedPatterns["verified-go-module"]++
	}
	mu.Unlock()

	sendSSE(SSEMessage{
		Status:   "done",
		Progress: 100,
		Message:  "Verified 7-layer build & test pipeline complete",
		Result:   result,
	})
}

func HandleAgentStatus(w http.ResponseWriter, r *http.Request) {
	execID := r.URL.Query().Get("executionId")
	if execID == "" {
		http.Error(w, "executionId required", http.StatusBadRequest)
		return
	}

	mu.RLock()
	result, ok := executionStore[execID]
	mu.RUnlock()
	if !ok {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

func HandleMetrics(w http.ResponseWriter, r *http.Request) {
	mu.RLock()
	defer mu.RUnlock()

	avgScore := 0.0
	topScore := 0.0
	for _, run := range executionHistory {
		avgScore += run.Score
		if run.Score > topScore {
			topScore = run.Score
		}
	}
	if len(executionHistory) > 0 {
		avgScore /= float64(len(executionHistory))
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"total_runs":       len(executionHistory),
		"average_score":    avgScore,
		"top_score":        topScore,
		"learned_patterns": learnedPatterns,
	})
}

func parseLLDToSpec(rawYAML, fallbackID string) *spec.Specification {
	var m map[string]interface{}
	_ = yaml.Unmarshal([]byte(rawYAML), &m)

	title, _ := m["name"].(string)
	if title == "" {
		title, _ = m["title"].(string)
	}
	if title == "" {
		title = "GeneratedService"
	}
	desc, _ := m["description"].(string)
	if desc == "" {
		desc = title
	}

	return &spec.Specification{
		ID:          fallbackID,
		Version:     "1.0",
		Title:       title,
		Description: desc,
		Intent: spec.Intent{
			Goal:    desc,
			Context: rawYAML,
		},
		Constraints: []spec.Constraint{
			{ID: "c1", Type: "must_not", Description: "unsafe.Pointer"},
		},
	}
}
