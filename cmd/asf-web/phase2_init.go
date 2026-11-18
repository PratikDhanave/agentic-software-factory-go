package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	_ "github.com/lib/pq"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/adapters/persistence"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/application"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/domain/services"
)

// Phase2Manager coordinates Phase 2 (Walk) components
type Phase2Manager struct {
	db                   *sql.DB
	repo                 *persistence.PostgresRepository
	llmClient            services.ClaudeClient
	skillRegistry        *services.SkillRegistry
	selfImprovingService *application.SelfImprovingService
	integrationManager   *IntegrationManager
}

// InitPhase2 initializes all Phase 2 components
func InitPhase2() (*Phase2Manager, error) {
	// 1. Connect to PostgreSQL
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return nil, fmt.Errorf("DATABASE_URL not set")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("database connection failed: %w", err)
	}

	log.Println("✓ PostgreSQL connected")

	// 2. Initialize real Claude API client
	apiKey := os.Getenv("CLAUDE_API_KEY")
	if apiKey == "" || apiKey == "sk-ant-YOUR_API_KEY_HERE" {
		log.Println("⚠ CLAUDE_API_KEY not set, using mock client")
		return nil, fmt.Errorf("CLAUDE_API_KEY must be set in .env")
	}

	model := os.Getenv("CLAUDE_MODEL")
	if model == "" {
		model = "claude-3-5-sonnet-20241022"
	}

	llmClient := services.NewRealClaudeAPIClient(apiKey, model)
	log.Println("✓ Claude API client initialized")

	// 3. Initialize repository
	repo := persistence.NewPostgresRepository(db)
	log.Println("✓ PostgreSQL repository initialized")

	// 4. Initialize skill registry
	skillRegistry := services.NewSkillRegistry()
	skillRegistry.RegisterSkill(services.NewCodeReviewSkill(llmClient))
	skillRegistry.RegisterSkill(services.NewTriageIssueSkill(llmClient))
	skillRegistry.RegisterSkill(services.NewSecurityReviewSkill(llmClient))
	skillRegistry.RegisterSkill(services.NewReviewSpecSkill(llmClient))
	log.Println("✓ Skill registry initialized (4 skills)")

	// 5. Initialize self-improving service
	selfImprovingService := application.NewSelfImprovingService(repo, llmClient)
	log.Println("✓ Self-improving service initialized")

	// 6. Initialize integration manager
	integrationManager := NewIntegrationManager(repo, llmClient, skillRegistry)
	integrationManager.selfImprovingService = selfImprovingService
	log.Println("✓ Integration manager initialized")

	return &Phase2Manager{
		db:                   db,
		repo:                 repo,
		llmClient:            llmClient,
		skillRegistry:        skillRegistry,
		selfImprovingService: selfImprovingService,
		integrationManager:   integrationManager,
	}, nil
}

// RegisterPhase2Routes registers all Phase 2 HTTP routes
func (pm *Phase2Manager) RegisterPhase2Routes(mux *http.ServeMux) {
	// GitHub webhook
	mux.HandleFunc("POST /webhooks/github", pm.integrationManager.handleGitHubWebhook)

	// Workflow outcome reporting
	mux.HandleFunc("POST /webhooks/outcome", pm.integrationManager.handleWorkflowOutcome)

	// Self-improvement metrics
	mux.HandleFunc("GET /metrics/self-improvement", pm.handleSelfImprovementMetrics)

	// Agent metrics
	mux.HandleFunc("GET /metrics/agents/:role", pm.handleAgentMetrics)

	// Patterns
	mux.HandleFunc("GET /patterns", pm.handlePatterns)

	// Admin endpoints
	mux.HandleFunc("POST /admin/learn/:agent", pm.handleManualLearning)

	log.Println("✓ Phase 2 routes registered")
}

// handleSelfImprovementMetrics returns current metrics
func (pm *Phase2Manager) handleSelfImprovementMetrics(w http.ResponseWriter, r *http.Request) {
	metrics, err := pm.selfImprovingService.GetMetrics(r.Context())
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to get metrics: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if err := json.NewEncoder(w).Encode(metrics); err != nil {
		log.Printf("Error encoding metrics: %v", err)
	}
}

// handleAgentMetrics returns metrics for a specific agent
func (pm *Phase2Manager) handleAgentMetrics(w http.ResponseWriter, r *http.Request) {
	role := r.PathValue("role")

	decisions, err := pm.repo.GetRecentDecisions(r.Context(), role, 100)
	if err != nil {
		http.Error(w, "failed to fetch decisions", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"agent":     role,
		"decisions": decisions,
		"count":     len(decisions),
	})
}

// handlePatterns returns discovered patterns
func (pm *Phase2Manager) handlePatterns(w http.ResponseWriter, r *http.Request) {
	patterns, err := pm.repo.GetTopPatterns(r.Context(), 20)
	if err != nil {
		http.Error(w, "failed to fetch patterns", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"patterns": patterns,
		"count":    len(patterns),
	})
}

// handleManualLearning triggers learning for an agent
func (pm *Phase2Manager) handleManualLearning(w http.ResponseWriter, r *http.Request) {
	agent := r.PathValue("agent")

	if err := pm.selfImprovingService.LearnPatterns(r.Context(), agent); err != nil {
		http.Error(w, fmt.Sprintf("learning failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "learning triggered",
		"agent":  agent,
	})
}

// GetPhaseConfig returns the current phase configuration
func GetPhaseConfig() map[string]interface{} {
	phase := os.Getenv("FACTORY_PHASE")
	if phase == "" {
		phase = "walk"
	}

	threshold := 0.90
	if t, err := strconv.ParseFloat(os.Getenv("APPROVAL_THRESHOLD"), 64); err == nil {
		threshold = t
	}

	window := 100
	if w, err := strconv.Atoi(os.Getenv("LEARNING_WINDOW")); err == nil {
		window = w
	}

	return map[string]interface{}{
		"phase":                   phase,
		"approval_threshold":      threshold,
		"learning_window":         window,
		"pattern_learning":        phase == "walk" || phase == "run",
		"autonomous_execution":    phase == "run",
		"llm_as_judge_enabled":    phase == "run",
	}
}

// PrintPhase2Status prints initialization status
func PrintPhase2Status(pm *Phase2Manager) {
	log.Println("\n" + strings.Repeat("═", 50))
	log.Println("Phase 2: Walk - Initialization Complete ✓")
	log.Println(strings.Repeat("═", 50))

	config := GetPhaseConfig()
	log.Printf("Phase: %v\n", config["phase"])
	log.Printf("Approval Threshold: %v\n", config["approval_threshold"])
	log.Printf("Learning Window: %v decisions\n", config["learning_window"])
	log.Printf("Pattern Learning: %v\n", config["pattern_learning"])
	log.Printf("Autonomous Execution: %v\n", config["autonomous_execution"])

	log.Println("\nComponents Ready:")
	log.Println("  ✓ PostgreSQL Database")
	log.Println("  ✓ Claude API Client")
	log.Println("  ✓ Skill System (4 skills)")
	log.Println("  ✓ Self-Improving Service")
	log.Println("  ✓ GitHub Integration")

	log.Println("\nEndpoints Available:")
	log.Println("  POST /webhooks/github - GitHub events")
	log.Println("  POST /webhooks/outcome - Workflow outcomes")
	log.Println("  GET  /metrics/self-improvement - Dashboard metrics")
	log.Println("  GET  /metrics/agents/:role - Agent metrics")
	log.Println("  GET  /patterns - Discovered patterns")
	log.Println("  POST /admin/learn/:agent - Manual learning trigger")

	log.Println("\nNext Steps:")
	log.Println("  1. Create a GitHub issue in your repo")
	log.Println("  2. Webhook will trigger triage workflow")
	log.Println("  3. Decision saved to database")
	log.Println("  4. View metrics at http://localhost:8000/metrics/self-improvement")

	log.Println(strings.Repeat("═", 50) + "\n")
}

func json(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(data)
}
