package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	_ "github.com/lib/pq"
	"github.com/google/uuid"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/adapters/persistence"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/application"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/domain"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/domain/services"
)

func main() {
	loadEnv()

	// Connect to database
	db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}

	log.Println("✓ PostgreSQL connected")

	// Initialize with mock Claude client
	llmClient := &services.MockClaudeClient{}
	repo := persistence.NewPostgresRepository(db)
	selfImprovingService := application.NewSelfImprovingService(repo, llmClient)

	log.Println("✓ Self-improving service initialized (Mock mode)")

	// Setup routes
	mux := http.NewServeMux()

	mux.HandleFunc("GET /", corsHandler(homeHandler))
	mux.HandleFunc("POST /simulate/batch", corsHandler(func(w http.ResponseWriter, r *http.Request) {
		simulateBatchHandler(w, r, repo, selfImprovingService)
	}))
	mux.HandleFunc("GET /metrics", corsHandler(func(w http.ResponseWriter, r *http.Request) {
		metricsHandler(w, r, selfImprovingService)
	}))
	mux.HandleFunc("POST /metrics/approve-last", corsHandler(func(w http.ResponseWriter, r *http.Request) {
		approveLastHandler(w, r, repo, selfImprovingService)
	}))
	mux.HandleFunc("POST /metrics/learn", corsHandler(func(w http.ResponseWriter, r *http.Request) {
		learnHandler(w, r, selfImprovingService)
	}))
	mux.HandleFunc("GET /metrics/database", corsHandler(func(w http.ResponseWriter, r *http.Request) {
		databaseStatsHandler(w, r, db)
	}))

	printWelcome()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}

	log.Printf("\n🎬 Phase 2 Simulator running on http://localhost:%s\n\n", port)
	http.ListenAndServe(":"+port, mux)
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	jsonResponse(w, map[string]interface{}{
		"name":    "Phase 2: Walk Simulator",
		"status":  "ready",
		"mode":    "mock (no API key needed)",
		"features": []string{
			"Generate 1-100 simulated decisions",
			"Watch approval rate drop",
			"See patterns emerge at 100 decisions",
			"Real PostgreSQL storage",
		},
		"endpoints": map[string]string{
			"POST /simulate/batch":      "Generate N decisions (body: {\"count\": 10})",
			"GET  /metrics":             "Current metrics",
			"POST /metrics/approve-last": "Approve last decision",
			"POST /metrics/learn":       "Trigger learning",
			"GET  /metrics/database":    "Database stats",
		},
	})
}

func simulateBatchHandler(w http.ResponseWriter, r *http.Request, repo *persistence.PostgresRepository, selfImprovingService *application.SelfImprovingService) {
	if r.Method != "POST" {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Count int `json:"count"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		req.Count = 1
	}

	if req.Count <= 0 {
		req.Count = 1
	}
	if req.Count > 100 {
		req.Count = 100
	}

	log.Printf("Generating %d simulated decisions...", req.Count)

	var decisions []map[string]interface{}

	for i := 0; i < req.Count; i++ {
		// Random issue type and agent
		issueTypes := []string{"auth", "performance", "security", "data"}
		agents := []string{"Architect", "Developer", "QA", "Security", "Performance"}
		issuetype := issueTypes[rand.Intn(len(issueTypes))]
		agent := agents[rand.Intn(len(agents))]

		// Generate confidence (mostly high, some low)
		var confidence float64
		if rand.Float64() < 0.7 {
			confidence = 0.85 + rand.Float64()*0.15 // 85-100%
		} else {
			confidence = 0.70 + rand.Float64()*0.20 // 70-90%
		}

		execID := uuid.New()
		decision := &domain.AgentDecision{
			ID:               uuid.New(),
			ExecutionID:      execID,
			AgentRole:        agent,
			Decision:         fmt.Sprintf("Fix %s issue: implement solution X", issuetype),
			Reasoning:        "Analyzed the problem and determined best approach",
			Confidence:       confidence,
			RequiresApproval: confidence < 0.90,
			ApprovalStatus:   "pending",
			CreatedAt:        time.Now(),
		}

		// Record the decision
		if err := selfImprovingService.RecordDecision(r.Context(), decision, issuetype); err != nil {
			log.Printf("Error recording decision: %v", err)
		}

		// Auto-approve if high confidence
		if !decision.RequiresApproval {
			approval := &domain.Approval{
				ID:                uuid.New(),
				DecisionID:        decision.ID,
				Status:            "approved",
				ApprovalComments:  "auto-approved (high confidence)",
				ApprovalTimestamp: time.Now(),
				CreatedAt:         time.Now(),
			}
			if err := selfImprovingService.RecordApproval(r.Context(), approval); err != nil {
				log.Printf("Error recording approval: %v", err)
			}
		}

		decisions = append(decisions, map[string]interface{}{
			"agent":              agent,
			"issue_type":         issuetype,
			"confidence":         fmt.Sprintf("%.2f", confidence),
			"requires_approval":  decision.RequiresApproval,
			"decision_id":        decision.ID.String(),
		})
	}

	jsonResponse(w, map[string]interface{}{
		"generated":  req.Count,
		"decisions":  decisions,
		"message":    fmt.Sprintf("Generated %d decisions - check /metrics to see dashboard", req.Count),
	})
}

func metricsHandler(w http.ResponseWriter, r *http.Request, selfImprovingService *application.SelfImprovingService) {
	metrics, err := selfImprovingService.GetMetrics(r.Context())
	if err != nil {
		http.Error(w, fmt.Sprintf("Error: %v", err), http.StatusInternalServerError)
		return
	}

	jsonResponse(w, metrics)
}

func approveLastHandler(w http.ResponseWriter, r *http.Request, repo *persistence.PostgresRepository, selfImprovingService *application.SelfImprovingService) {
	if r.Method != "POST" {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	decisions, err := repo.GetRecentDecisions(r.Context(), "Architect", 1)
	if err != nil || len(decisions) == 0 {
		http.Error(w, "No recent decisions", http.StatusNotFound)
		return
	}

	approval := &domain.Approval{
		ID:                uuid.New(),
		DecisionID:        decisions[0].ID,
		Status:            "approved",
		ApprovalComments:  "Manually approved from simulator",
		ApprovalTimestamp: time.Now(),
		CreatedAt:         time.Now(),
	}

	if err := selfImprovingService.RecordApproval(r.Context(), approval); err != nil {
		http.Error(w, fmt.Sprintf("Error: %v", err), http.StatusInternalServerError)
		return
	}

	jsonResponse(w, map[string]string{
		"status": "approved",
		"id":     decisions[0].ID.String(),
	})
}

func learnHandler(w http.ResponseWriter, r *http.Request, selfImprovingService *application.SelfImprovingService) {
	if r.Method != "POST" {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	if err := selfImprovingService.LearnPatterns(r.Context(), "Architect"); err != nil {
		log.Printf("Warning: learning failed: %v", err)
	}

	jsonResponse(w, map[string]string{
		"status": "learning triggered",
	})
}

func databaseStatsHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	var decisionCount, approvalCount, patternCount int

	db.QueryRow("SELECT COUNT(*) FROM agent_decisions").Scan(&decisionCount)
	db.QueryRow("SELECT COUNT(*) FROM approvals").Scan(&approvalCount)
	db.QueryRow("SELECT COUNT(*) FROM learned_patterns").Scan(&patternCount)

	jsonResponse(w, map[string]interface{}{
		"decisions": decisionCount,
		"approvals": approvalCount,
		"patterns":  patternCount,
		"status":    "Phase 2 database growing",
	})
}

func corsHandler(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		handler(w, r)
	}
}

func jsonResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func loadEnv() {
	if content, err := os.ReadFile(".env"); err == nil {
		for _, line := range strings.Split(string(content), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}

			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				val := strings.TrimSpace(parts[1])
				os.Setenv(key, val)
			}
		}
	}

	if os.Getenv("DATABASE_URL") == "" {
		os.Setenv("DATABASE_URL", "postgresql://praktik.dhanave@localhost:5432/factory")
	}
}

func printWelcome() {
	log.Println("\n" + strings.Repeat("═", 70))
	log.Println("🎬 Phase 2: Walk - SIMULATOR MODE")
	log.Println("   (No Claude API key needed - uses mock decisions)")
	log.Println(strings.Repeat("═", 70))
	log.Println("\n📊 Quick Test:")
	log.Println("  1. Generate 20 decisions:")
	log.Println("     curl -X POST http://localhost:8000/simulate/batch -d '{\"count\": 20}' -H 'Content-Type: application/json'")
	log.Println("")
	log.Println("  2. View metrics:")
	log.Println("     curl http://localhost:8000/metrics | jq .")
	log.Println("")
	log.Println("  3. Trigger learning (after 100 decisions):")
	log.Println("     curl -X POST http://localhost:8000/metrics/learn")
	log.Println("")
	log.Println("  4. Check database:")
	log.Println("     curl http://localhost:8000/metrics/database")
	log.Println("")
	log.Println("📈 Build Full Dataset:")
	log.Println("  for i in {1..10}; do")
	log.Println("    curl -s -X POST http://localhost:8000/simulate/batch \\")
	log.Println("      -d '{\"count\": 10}' -H 'Content-Type: application/json'")
	log.Println("  done")
	log.Println("")
	log.Println("⚡ Then Add Real Claude API:")
	log.Println("  1. Get key from: https://console.anthropic.com/account/keys")
	log.Println("  2. Update .env: CLAUDE_API_KEY=sk-ant-...")
	log.Println("  3. Run: ./bin/phase2-server")
	log.Println("")
	log.Println(strings.Repeat("═", 70) + "\n")
}
