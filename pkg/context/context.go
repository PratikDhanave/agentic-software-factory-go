package context

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ContextEngine handles context curation and retrieval for agent operations
type ContextEngine struct {
	rootPath     string
	adrPath      string
	specsPath    string
	maxFileSize  int64
	priorityDirs []string
}

// ContextItem represents a single piece of context loaded into the agent
type ContextItem struct {
	Path        string
	Content     string
	Priority    int
	Category    string // "adr", "spec", "code", "config", "doc"
	TokenEstimate int
}

// NewContextEngine creates a new context engine
func NewContextEngine(rootPath string) *ContextEngine {
	return &ContextEngine{
		rootPath:     rootPath,
		adrPath:      filepath.Join(rootPath, "docs"),
		specsPath:    filepath.Join(rootPath, "specs"),
		maxFileSize:  50 * 1024, // 50KB
		priorityDirs: []string{"docs", "specs", "internal", "pkg", "cmd"},
	}
}

// LoadContext loads all relevant context items for the agent
func (e *ContextEngine) LoadContext(ctx context.Context) ([]ContextItem, error) {
	var items []ContextItem

	// Load ADRs (highest priority)
	adrs, err := e.loadADRs(ctx)
	if err != nil {
		return nil, fmt.Errorf("loading ADRs: %w", err)
	}
	items = append(items, adrs...)

	// Load specifications
	specs, err := e.loadSpecs(ctx)
	if err != nil {
		return nil, fmt.Errorf("loading specs: %w", err)
	}
	items = append(items, specs...)

	// Load source code (lower priority, but essential)
	code, err := e.loadSourceCode(ctx)
	if err != nil {
		return nil, fmt.Errorf("loading source code: %w", err)
	}
	items = append(items, code...)

	// Sort by priority (higher first)
	// We'll use a simple sort - in production, use sort.Slice
	return items, nil
}

// loadADRs loads Architectural Decision Records
func (e *ContextEngine) loadADRs(ctx context.Context) ([]ContextItem, error) {
	var items []ContextItem

	entries, err := os.ReadDir(e.adrPath)
	if err != nil {
		// ADRs directory might not exist
		return items, nil
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		path := filepath.Join(e.adrPath, entry.Name())
		content, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		if int64(len(content)) > e.maxFileSize {
			continue
		}

		items = append(items, ContextItem{
			Path:         path,
			Content:      string(content),
			Priority:     100,
			Category:     "adr",
			TokenEstimate: len(content) / 4,
		})
	}

	return items, nil
}

// loadSpecs loads specification files
func (e *ContextEngine) loadSpecs(ctx context.Context) ([]ContextItem, error) {
	var items []ContextItem

	entries, err := os.ReadDir(e.specsPath)
	if err != nil {
		return items, nil
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") && !strings.HasSuffix(entry.Name(), ".yml") {
			continue
		}

		path := filepath.Join(e.specsPath, entry.Name())
		content, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		if int64(len(content)) > e.maxFileSize {
			continue
		}

		items = append(items, ContextItem{
			Path:         path,
			Content:      string(content),
			Priority:     90,
			Category:     "spec",
			TokenEstimate: len(content) / 4,
		})
	}

	return items, nil
}

// loadSourceCode loads relevant source code files
func (e *ContextEngine) loadSourceCode(ctx context.Context) ([]ContextItem, error) {
	var items []ContextItem

	for _, dir := range e.priorityDirs {
		fullPath := filepath.Join(e.rootPath, dir)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			continue
		}

		filepath.Walk(fullPath, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}

			// Only include Go files
			if !strings.HasSuffix(info.Name(), ".go") {
				return nil
			}

			// Skip test files for context
			if strings.HasSuffix(info.Name(), "_test.go") {
				return nil
			}

			content, err := os.ReadFile(path)
			if err != nil {
				return nil
			}

			if int64(len(content)) > e.maxFileSize {
				return nil
			}

			relPath, _ := filepath.Rel(e.rootPath, path)
			items = append(items, ContextItem{
				Path:         relPath,
				Content:      string(content),
				Priority:     50,
				Category:     "code",
				TokenEstimate: len(content) / 4,
			})

			return nil
		})
	}

	return items, nil
}

// FormatForAgent formats context items for injection into agent prompt
func FormatForAgent(items []ContextItem) string {
	var sb strings.Builder

	for _, item := range items {
		sb.WriteString(fmt.Sprintf("=== %s [%s] (priority: %d) ===\n", item.Path, item.Category, item.Priority))
		sb.WriteString(item.Content)
		sb.WriteString("\n\n")
	}

	return sb.String()
}