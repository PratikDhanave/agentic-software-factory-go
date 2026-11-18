package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/PratikDhanave/agentic-software-factory-go/internal/adapters/persistence/sqlite"
)

var (
	sqliteStore  *sqlite.Store
	workflowRepo *sqlite.WorkflowRepository
)

// InitDatabase initializes or attaches the unified SQLite store.
func InitDatabase(dbPath string) error {
	if dbPath == "" {
		dbPath = filepath.Join(".asf", "factory.db")
	}
	store, err := sqlite.Open(dbPath)
	if err != nil {
		return err
	}
	SetSQLiteStore(store)
	log.Printf("[Handler] SQLite database initialized at %s", store.Path())
	return nil
}

// SetSQLiteStore injects an already-opened sqlite.Store from bootstrap.
func SetSQLiteStore(store *sqlite.Store) {
	sqliteStore = store
	if store != nil {
		workflowRepo = sqlite.NewWorkflowRepository(store)
	}
}

func GetSQLiteStore() *sqlite.Store {
	return sqliteStore
}

// HandleDatabaseStats returns comprehensive SQLite database statistics.
func HandleDatabaseStats(w http.ResponseWriter, r *http.Request) {
	if sqliteStore == nil {
		http.Error(w, "Database not initialized", http.StatusServiceUnavailable)
		return
	}

	stats, err := sqliteStore.Stats(r.Context())
	if err != nil {
		log.Printf("[Handler] Failed to get stats: %v", err)
		http.Error(w, "Failed to retrieve statistics", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// HandleWorkflowHistory returns historical workflow execution data from SQLite.
func HandleWorkflowHistory(w http.ResponseWriter, r *http.Request) {
	if workflowRepo == nil {
		http.Error(w, "Database not initialized", http.StatusServiceUnavailable)
		return
	}

	limit := parseBoundedInt(r.URL.Query().Get("limit"), 20, 1, 100)
	offset := parseBoundedInt(r.URL.Query().Get("offset"), 0, 0, 10000)

	runs, err := workflowRepo.ListExecutions(r.Context(), "", limit, offset)
	if err != nil {
		log.Printf("[Handler] Failed to get workflow history: %v", err)
		http.Error(w, "Failed to retrieve workflow history", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"workflows": runs,
		"limit":     limit,
		"offset":    offset,
	})
}

// HandleWorkflowDetails returns detailed information about a specific workflow execution.
func HandleWorkflowDetails(w http.ResponseWriter, r *http.Request) {
	if workflowRepo == nil {
		http.Error(w, "Database not initialized", http.StatusServiceUnavailable)
		return
	}

	workflowID := r.URL.Query().Get("id")
	if workflowID == "" {
		http.Error(w, "Missing workflow ID", http.StatusBadRequest)
		return
	}

	details, err := workflowRepo.GetExecution(r.Context(), workflowID)
	if err != nil {
		http.Error(w, "Workflow not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(details)
}

// HandleDatabaseHealth returns SQLite health status.
func HandleDatabaseHealth(w http.ResponseWriter, r *http.Request) {
	if sqliteStore == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"engine": "sqlite3",
			"status": "disconnected",
		})
		return
	}

	health := sqliteStore.Health(r.Context())
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

// HandleSearchWorkflows searches workflow definitions/executions in SQLite.
func HandleSearchWorkflows(w http.ResponseWriter, r *http.Request) {
	HandleWorkflowHistory(w, r)
}

func parseBoundedInt(raw string, defVal, minVal, maxVal int) int {
	if raw == "" {
		return defVal
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v < minVal || v > maxVal {
		return defVal
	}
	return v
}
