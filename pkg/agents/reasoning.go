package agents

import (
	"log"
	"strings"
)

type ReasoningEngine struct {
	patterns map[string][]string
	weights  map[string]float32
}

func NewReasoningEngine() *ReasoningEngine {
	return &ReasoningEngine{
		patterns: map[string][]string{
			// Get/Read operations
			"get_issue": {
				"get", "fetch", "read", "show", "display", "view", "retrieve",
				"what is", "tell me", "find out", "look at", "check",
			},
			// Create operations
			"create_issue": {
				"create", "new", "open", "file", "report", "submit",
				"add issue", "make", "write", "post", "raise",
			},
			// Label operations
			"add_labels": {
				"label", "tag", "mark", "classify", "categorize",
				"add label", "apply label", "set priority", "set type",
				"priority-", "category-", "ready-to-implement",
			},
			// Comment operations
			"post_comment": {
				"comment", "reply", "respond", "feedback", "note",
				"add comment", "post message", "write comment", "say",
				"respond to", "notify", "mention",
			},
			// PR operations
			"create_pr": {
				"pull request", "pr", "merge", "merge request",
				"create pr", "open pr", "submit pr", "make pr",
				"branch", "changes", "code change",
			},
		},
		weights: map[string]float32{
			"get_issue":      1.0,
			"create_issue":   1.0,
			"add_labels":     0.9,
			"post_comment":   0.9,
			"create_pr":      0.95,
		},
	}
}

// AnalyzeTask determines which tools to use for a task
func (r *ReasoningEngine) AnalyzeTask(task string) []string {
	log.Printf("[Reasoning] Analyzing: %s", task)

	taskLower := strings.ToLower(task)
	scores := make(map[string]float32)

	// Score each tool based on keyword matches
	for tool, keywords := range r.patterns {
		score := 0.0
		matchCount := 0

		for _, keyword := range keywords {
			if strings.Contains(taskLower, keyword) {
				score++
				matchCount++
			}
		}

		if matchCount > 0 {
			normalizedScore := float32(score) * r.weights[tool]
			scores[tool] = normalizedScore
			log.Printf("[Reasoning] %s: score=%.2f matches=%d", tool, normalizedScore, matchCount)
		}
	}

	// Sort and return top tools
	return rankToolsByScore(scores)
}

// rankToolsByScore returns tools sorted by score descending
func rankToolsByScore(scores map[string]float32) []string {
	type kv struct {
		tool  string
		score float32
	}

	var sorted []kv
	for tool, score := range scores {
		if score > 0 {
			sorted = append(sorted, kv{tool, score})
		}
	}

	// Simple bubble sort (scores are small)
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j].score > sorted[i].score {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	var result []string
	for _, kv := range sorted {
		result = append(result, kv.tool)
	}

	return result
}

// SuggestParams recommends parameters for a tool based on context
func (r *ReasoningEngine) SuggestParams(tool, context string) map[string]interface{} {
	// Extract common patterns from context
	contextLower := strings.ToLower(context)

	params := make(map[string]interface{})

	// Extract owner/repo if present
	if strings.Contains(contextLower, "owner:") || strings.Contains(contextLower, "owner =") {
		// Would parse more carefully in production
		params["owner"] = extractValue(context, "owner")
	}

	if strings.Contains(contextLower, "repo:") || strings.Contains(contextLower, "repo =") {
		params["repo"] = extractValue(context, "repo")
	}

	if strings.Contains(contextLower, "issue:") || strings.Contains(contextLower, "issue =") {
		params["issue"] = extractValue(context, "issue")
	}

	return params
}

// extractValue extracts a value after a key (simple implementation)
func extractValue(text, key string) string {
	lower := strings.ToLower(text)
	idx := strings.Index(lower, strings.ToLower(key))
	if idx == -1 {
		return ""
	}

	// Find start of value (after ":" or "=")
	start := idx + len(key)
	for start < len(text) && (text[start] == ':' || text[start] == '=' || text[start] == ' ') {
		start++
	}

	// Find end of value (space or punctuation)
	end := start
	for end < len(text) && text[end] != ' ' && text[end] != '\n' && text[end] != ',' {
		end++
	}

	return strings.TrimSpace(text[start:end])
}
