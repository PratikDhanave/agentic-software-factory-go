package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
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
	// Load environment
	loadEnv()

	// Initialize Phase 2
	db, llmClient, repo, _, selfImprovingService := initPhase2()
	defer db.Close()

	// Setup routes
	mux := http.NewServeMux()

	// CORS middleware
	mux.HandleFunc("/metrics/self-improvement", corsHandler(func(w http.ResponseWriter, r *http.Request) {
		metricsHandler(w, r, selfImprovingService)
	}))

	mux.HandleFunc("/test/decision", corsHandler(func(w http.ResponseWriter, r *http.Request) {
		testDecisionHandler(w, r, llmClient, repo, selfImprovingService)
	}))

	mux.HandleFunc("/test/approve", corsHandler(func(w http.ResponseWriter, r *http.Request) {
		testApproveHandler(w, r, repo, selfImprovingService)
	}))

	mux.HandleFunc("/test/learn", corsHandler(func(w http.ResponseWriter, r *http.Request) {
		testLearnHandler(w, r, selfImprovingService)
	}))

	mux.HandleFunc("/health", corsHandler(healthHandler))

	// Print status
	printPhase2Status()

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}

	log.Printf("\n🚀 Phase 2 Server starting on http://localhost:%s\n", port)
	log.Printf("Test endpoints:\n")
	log.Printf("  GET  http://localhost:%s/health\n", port)
	log.Printf("  POST http://localhost:%s/test/decision\n", port)
	log.Printf("  POST http://localhost:%s/test/approve\n", port)
	log.Printf("  GET  http://localhost:%s/metrics/self-improvement\n", port)
	log.Printf("  POST http://localhost:%s/test/learn\n", port)
	log.Printf("\n")

	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func initPhase2() (*sql.DB, services.ClaudeClient, *persistence.PostgresRepository, *services.SkillRegistry, *application.SelfImprovingService) {
	// Connect to database
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL not set")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.Ping(); err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}

	log.Println("✓ PostgreSQL connected")

	// Initialize Claude client
	apiKey := os.Getenv("CLAUDE_API_KEY")
	if apiKey == "" || apiKey == "sk-ant-YOUR_API_KEY_HERE" {
		log.Fatal("CLAUDE_API_KEY must be set in .env")
	}

	model := os.Getenv("CLAUDE_MODEL")
	if model == "" {
		model = "claude-3-5-sonnet-20241022"
	}

	llmClient := services.NewRealClaudeAPIClient(apiKey, model)
	log.Println("✓ Claude API client initialized")

	// Initialize repository
	repo := persistence.NewPostgresRepository(db)
	log.Println("✓ PostgreSQL repository initialized")

	// Initialize skill registry
	skillRegistry := services.NewSkillRegistry()
	skillRegistry.RegisterSkill(services.NewCodeReviewSkill(llmClient))
	skillRegistry.RegisterSkill(services.NewTriageIssueSkill(llmClient))
	log.Println("✓ Skill system initialized")

	// Initialize self-improving service
	selfImprovingService := application.NewSelfImprovingService(repo, llmClient)
	log.Println("✓ Self-improving service initialized")

	return db, llmClient, repo, skillRegistry, selfImprovingService
}

func init() {
	// placeholder
}

func testDecisionHandler(w http.ResponseWriter, r *http.Request, llmClient services.ClaudeClient, repo *persistence.PostgresRepository, selfImprovingService *application.SelfImprovingService) {
	if r.Method != "POST" {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	// Generate a test decision using Claude
	req := &services.ClaudeDecisionRequest{
		IssueTitle:       "OAuth2 session timeout issue",
		IssueDescription: "Users are being logged out too frequently after 1 hour of inactivity",
		AgentRole:        "Architect",
	}

	resp, err := llmClient.GenerateDecision(r.Context(), req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Claude API error: %v", err), http.StatusBadRequest)
		return
	}

	// Save decision to database
	execID := uuid.New()
	decision := &domain.AgentDecision{
		ID:               uuid.New(),
		ExecutionID:      execID,
		AgentRole:        "Architect",
		Decision:         resp.Decision,
		Reasoning:        resp.Reasoning,
		Confidence:       resp.Confidence,
		Alternatives:     resp.Alternatives,
		RequiresApproval: resp.Confidence < 0.90,
		ApprovalStatus:   "pending",
		CreatedAt:        time.Now(),
	}

	if err := selfImprovingService.RecordDecision(r.Context(), decision, "auth"); err != nil {
		log.Printf("Error recording decision: %v", err)
	}

	jsonResponse(w, map[string]interface{}{
		"decision":          decision.Decision,
		"confidence":        decision.Confidence,
		"requires_approval": decision.RequiresApproval,
		"decision_id":       decision.ID.String(),
		"execution_id":      execID.String(),
	})
}

func testApproveHandler(w http.ResponseWriter, r *http.Request, repo *persistence.PostgresRepository, selfImprovingService *application.SelfImprovingService) {
	if r.Method != "POST" {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	// Get last decision
	decisions, err := repo.GetRecentDecisions(r.Context(), "Architect", 1)
	if err != nil || len(decisions) == 0 {
		http.Error(w, "No recent decisions found", http.StatusNotFound)
		return
	}

	decision := decisions[0]

	// Mark as approved
	approval := &domain.Approval{
		ID:                 uuid.New(),
		DecisionID:         decision.ID,
		Status:             "approved",
		ApprovalComments:   "LGTM from Phase 2 test",
		ApprovalTimestamp:  time.Now(),
		CreatedAt:          time.Now(),
	}

	if err := selfImprovingService.RecordApproval(r.Context(), approval); err != nil {
		http.Error(w, fmt.Sprintf("Error recording approval: %v", err), http.StatusInternalServerError)
		return
	}

	jsonResponse(w, map[string]interface{}{
		"status":      "approved",
		"decision_id": decision.ID.String(),
	})
}

func testLearnHandler(w http.ResponseWriter, r *http.Request, selfImprovingService *application.SelfImprovingService) {
	if r.Method != "POST" {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	// Trigger learning
	if err := selfImprovingService.LearnPatterns(r.Context(), "Architect"); err != nil {
		log.Printf("Warning: learning failed: %v", err)
		// Don't error - just means not enough data
	}

	jsonResponse(w, map[string]interface{}{
		"status": "learning triggered",
		"agent":  "Architect",
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

func healthHandler(w http.ResponseWriter, r *http.Request) {
	jsonResponse(w, map[string]string{
		"status": "healthy",
		"phase":  "walk",
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
	// Try to load .env file
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

	// Set defaults
	if os.Getenv("DATABASE_URL") == "" {
		os.Setenv("DATABASE_URL", "postgresql://praktik.dhanave@localhost:5432/factory")
	}
	if os.Getenv("FACTORY_PHASE") == "" {
		os.Setenv("FACTORY_PHASE", "walk")
	}
	if os.Getenv("APPROVAL_THRESHOLD") == "" {
		os.Setenv("APPROVAL_THRESHOLD", "0.90")
	}
}

func printPhase2Status() {
	threshold, _ := strconv.ParseFloat(os.Getenv("APPROVAL_THRESHOLD"), 64)

	log.Println("\n" + strings.Repeat("═", 60))
	log.Println("⚙️  Phase 2: Walk - Self-Improving Agents")
	log.Println(strings.Repeat("═", 60))
	log.Println("\n Configuration:")
	log.Printf("  Phase: %s\n", os.Getenv("FACTORY_PHASE"))
	log.Printf("  Approval Threshold: %.2f\n", threshold)
	log.Printf("  Learning Window: %s decisions\n", os.Getenv("LEARNING_WINDOW"))
	log.Println("\n Components Ready:")
	log.Println("  ✓ PostgreSQL Database")
	log.Println("  ✓ Claude API (Real)")
	log.Println("  ✓ Self-Improving Service")
	log.Println("  ✓ GitHub Integration")
	log.Println("\n Test Workflow:")
	log.Println("  1. POST /test/decision  → Generate decision using Claude")
	log.Println("  2. POST /test/approve   → Approve last decision")
	log.Println("  3. POST /test/learn     → Trigger pattern learning")
	log.Println("  4. GET  /metrics/self-improvement → View metrics")
	log.Println("\n" + strings.Repeat("═", 60) + "\n")
}
