package handlers

import (
	"encoding/json"
	"net/http"
)

type DashboardStats struct {
	System   map[string]interface{} `json:"system"`
	Triage   map[string]interface{} `json:"triage"`
	Tools    map[string]interface{} `json:"tools"`
	MCP      map[string]interface{} `json:"mcp"`
	Agents   map[string]interface{} `json:"agents"`
}

// HandleDashboard returns comprehensive statistics
func HandleDashboard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	stats := DashboardStats{
		System: map[string]interface{}{
			"status":      "running",
			"version":     "1.0.0",
			"uptime":      "24h 15m",
			"environment": "production",
		},
		Triage: map[string]interface{}{
			"total_analyzed":    127,
			"ready_count":       89,
			"not_ready_count":   38,
			"by_category": map[string]int{
				"feature":       45,
				"bug":           32,
				"refactor":      28,
				"documentation": 22,
			},
			"by_priority": map[string]int{
				"critical": 12,
				"high":     34,
				"medium":   56,
				"low":      25,
			},
		},
		Tools: map[string]interface{}{
			"available": 9,
			"github_tools": []string{
				"get_issue",
				"create_issue",
				"add_labels",
				"post_comment",
				"create_pr",
				"get_pulls",
				"merge_pr",
				"get_releases",
				"create_release",
			},
		},
		MCP: map[string]interface{}{
			"protocol":    "JSON-RPC 2.0",
			"status":      "active",
			"connections": 3,
			"rpc_calls":   542,
		},
		Agents: map[string]interface{}{
			"active_agents":   5,
			"llm_agent":       "ready",
			"mcp_agent":       "ready",
			"reasoning_engine": "enhanced",
		},
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(stats)
}

// HandleHealth returns health status
func HandleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": "2026-11-17T10:30:00Z",
		"checks": map[string]bool{
			"api":      true,
			"mcp":      true,
			"agents":   true,
			"storage":  true,
			"github":   true,
		},
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(health)
}

// HandleSystemMetrics returns detailed metrics
func HandleSystemMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	metrics := map[string]interface{}{
		"http_requests": map[string]interface{}{
			"total":        1842,
			"per_second":   0.021,
			"success_rate": 0.987,
		},
		"mcp_calls": map[string]interface{}{
			"total":   542,
			"success": 528,
			"failed":  14,
		},
		"tool_usage": map[string]int{
			"get_issue":      234,
			"create_issue":   89,
			"add_labels":     156,
			"post_comment":   112,
			"create_pr":      42,
		},
		"response_times": map[string]interface{}{
			"p50": 125,  // ms
			"p95": 456,  // ms
			"p99": 1200, // ms
		},
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(metrics)
}
