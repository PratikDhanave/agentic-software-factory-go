package storage

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type TriageDecision struct {
	ID              string            `json:"id"`
	Repo            string            `json:"repo"`
	Issue           int               `json:"issue"`
	Title           string            `json:"title"`
	Category        string            `json:"category"`
	Priority        string            `json:"priority"`
	ReadyToImpl      bool              `json:"ready_to_implement"`
	Labels          []string          `json:"labels"`
	ToolsUsed       []string          `json:"tools_used"`
	ReasoningSteps  []string          `json:"reasoning_steps"`
	DecisionTime    time.Time         `json:"decision_time"`
	ExecutionTime   int               `json:"execution_ms"`
	Metadata        map[string]string `json:"metadata,omitempty"`
}

type TriageStore struct {
	storagePath string
	mutex       sync.RWMutex
	cache       map[string]*TriageDecision
}

func NewTriageStore(basePath string) (*TriageStore, error) {
	storagePath := filepath.Join(basePath, "triage_decisions")

	// Create directory if it doesn't exist
	if err := os.MkdirAll(storagePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	store := &TriageStore{
		storagePath: storagePath,
		cache:       make(map[string]*TriageDecision),
	}

	// Load existing decisions
	if err := store.loadAll(); err != nil {
		log.Printf("[TriageStore] Warning: failed to load existing decisions: %v", err)
	}

	return store, nil
}

// Save persists a triage decision
func (s *TriageStore) Save(decision *TriageDecision) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Add to cache
	s.cache[decision.ID] = decision

	// Write to file
	filename := filepath.Join(s.storagePath, fmt.Sprintf("%s.json", decision.ID))
	data, err := json.MarshalIndent(decision, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return err
	}

	log.Printf("[TriageStore] Saved decision %s", decision.ID)
	return nil
}

// Get retrieves a triage decision by ID
func (s *TriageStore) Get(id string) (*TriageDecision, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if decision, ok := s.cache[id]; ok {
		return decision, nil
	}

	return nil, fmt.Errorf("decision not found: %s", id)
}

// ListByRepo returns all decisions for a repository
func (s *TriageStore) ListByRepo(repo string) ([]*TriageDecision, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	var results []*TriageDecision
	for _, decision := range s.cache {
		if decision.Repo == repo {
			results = append(results, decision)
		}
	}

	return results, nil
}

// ListRecent returns N most recent decisions
func (s *TriageStore) ListRecent(n int) []*TriageDecision {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	var results []*TriageDecision
	for _, decision := range s.cache {
		results = append(results, decision)
	}

	// Sort by time descending
	for i := 0; i < len(results); i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].DecisionTime.After(results[i].DecisionTime) {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	if len(results) > n {
		return results[:n]
	}
	return results
}

// GetStats returns usage statistics
func (s *TriageStore) GetStats() map[string]interface{} {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	stats := map[string]interface{}{
		"total_decisions": len(s.cache),
		"by_category":     make(map[string]int),
		"by_priority":     make(map[string]int),
		"ready_count":     0,
		"not_ready_count": 0,
	}

	categories := stats["by_category"].(map[string]int)
	priorities := stats["by_priority"].(map[string]int)

	for _, decision := range s.cache {
		categories[decision.Category]++
		priorities[decision.Priority]++
		if decision.ReadyToImpl {
			stats["ready_count"] = stats["ready_count"].(int) + 1
		} else {
			stats["not_ready_count"] = stats["not_ready_count"].(int) + 1
		}
	}

	return stats
}

// loadAll loads all decisions from disk
func (s *TriageStore) loadAll() error {
	files, err := os.ReadDir(s.storagePath)
	if err != nil {
		return err
	}

	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".json" {
			path := filepath.Join(s.storagePath, file.Name())
			data, err := os.ReadFile(path)
			if err != nil {
				log.Printf("[TriageStore] Warning: failed to read %s: %v", path, err)
				continue
			}

			var decision TriageDecision
			if err := json.Unmarshal(data, &decision); err != nil {
				log.Printf("[TriageStore] Warning: failed to unmarshal %s: %v", path, err)
				continue
			}

			s.cache[decision.ID] = &decision
		}
	}

	log.Printf("[TriageStore] Loaded %d decisions from disk", len(s.cache))
	return nil
}

// Delete removes a decision
func (s *TriageStore) Delete(id string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	delete(s.cache, id)

	filename := filepath.Join(s.storagePath, fmt.Sprintf("%s.json", id))
	if err := os.Remove(filename); err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}
