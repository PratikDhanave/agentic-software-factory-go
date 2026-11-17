package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/PratikDhanave/agentic-software-factory-go/pkg/db"
)

var database *db.Database

// InitDatabase initializes the database connection
func InitDatabase(dsn string) error {
	var err error
	database, err = db.NewDatabase(dsn)
	if err != nil {
		return err
	}

	// Run migrations
	if err := database.Migrate(); err != nil {
		log.Printf("[DB] Migration error (continuing): %v", err)
		// Don't fail on migration errors - table might already exist
	}

	log.Println("[Handler] Database initialized")
	return nil
}

// GetDatabase returns the database instance
func GetDatabase() *db.Database {
	return database
}

// HandleDatabaseStats returns comprehensive database statistics
func HandleDatabaseStats(w http.ResponseWriter, r *http.Request) {
	if database == nil {
		http.Error(w, "Database not initialized", http.StatusServiceUnavailable)
		return
	}

	stats, err := database.GetDashboardStats()
	if err != nil {
		log.Printf("[Handler] Failed to get stats: %v", err)
		http.Error(w, "Failed to retrieve statistics", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(stats)
}

// HandleWorkflowHistory returns historical workflow data
func HandleWorkflowHistory(w http.ResponseWriter, r *http.Request) {
	if database == nil {
		http.Error(w, "Database not initialized", http.StatusServiceUnavailable)
		return
	}

	limit := 20
	offset := 0

	if l := r.URL.Query().Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 100 {
			limit = v
		}
	}

	if o := r.URL.Query().Get("offset"); o != "" {
		if v, err := strconv.Atoi(o); err == nil && v >= 0 {
			offset = v
		}
	}

	runs, err := database.GetWorkflowHistory(limit, offset)
	if err != nil {
		log.Printf("[Handler] Failed to get workflow history: %v", err)
		http.Error(w, "Failed to retrieve workflow history", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"workflows": runs,
		"limit":     limit,
		"offset":    offset,
	})
}

// HandleWorkflowDetails returns detailed information about a specific workflow
func HandleWorkflowDetails(w http.ResponseWriter, r *http.Request) {
	if database == nil {
		http.Error(w, "Database not initialized", http.StatusServiceUnavailable)
		return
	}

	workflowID := r.URL.Query().Get("id")
	if workflowID == "" {
		http.Error(w, "Missing workflow ID", http.StatusBadRequest)
		return
	}

	details, err := database.GetWorkflowDetails(workflowID)
	if err != nil {
		log.Printf("[Handler] Failed to get workflow details: %v", err)
		http.Error(w, "Workflow not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(details)
}

// HandleDatabaseHealth returns database health status
func HandleDatabaseHealth(w http.ResponseWriter, r *http.Request) {
	if database == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "disconnected",
		})
		return
	}

	health := database.Health()
	status := "healthy"
	if health["error"] != nil {
		status = "unhealthy"
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": status,
		"error":  health["error"],
	})
}

// HandleSearchWorkflows searches for workflows by type
func HandleSearchWorkflows(w http.ResponseWriter, r *http.Request) {
	if database == nil {
		http.Error(w, "Database not initialized", http.StatusServiceUnavailable)
		return
	}

	wfType := r.URL.Query().Get("type")
	limit := 20
	offset := 0

	if l := r.URL.Query().Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 100 {
			limit = v
		}
	}

	if o := r.URL.Query().Get("offset"); o != "" {
		if v, err := strconv.Atoi(o); err == nil && v >= 0 {
			offset = v
		}
	}

	var runs interface{}
	var err error

	if wfType != "" {
		runs, err = database.GetWorkflowByType(wfType, limit, offset)
	} else {
		runs, err = database.GetWorkflowHistory(limit, offset)
	}

	if err != nil {
		log.Printf("[Handler] Failed to search workflows: %v", err)
		http.Error(w, "Failed to search workflows", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"results": runs,
		"type":    wfType,
		"limit":   limit,
		"offset":  offset,
	})
}
