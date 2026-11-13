package main

import (
	"context"
	"fmt"
	"os"

	asfContext "github.com/PratikDhanave/agentic-software-factory-go/pkg/context"
)

func main() {
	workingDir := "."
	if len(os.Args) > 1 {
		workingDir = os.Args[1]
	}

	fmt.Println("=== Context Engineering Demo (Chapter 3) ===")
	fmt.Println("Demonstrating open-loop vs closed-loop agent behavior")
	fmt.Println()

	ctx := context.Background()
	engine := asfContext.NewContextEngine(workingDir)

	// Load context
	items, err := engine.LoadContext(ctx)
	if err != nil {
		fmt.Printf("Error loading context: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Loaded %d context items\n\n", len(items))

	// Show context by category
	categories := map[string][]asfContext.ContextItem{}
	for _, item := range items {
		categories[item.Category] = append(categories[item.Category], item)
	}

	for cat, catItems := range categories {
		fmt.Printf("=== %s (%d items) ===\n", cat, len(catItems))
		for _, item := range catItems {
			fmt.Printf("  %s (priority: %d, ~%d tokens)\n", item.Path, item.Priority, item.TokenEstimate)
		}
		fmt.Println()
	}

	// Demonstrate open-loop vs closed-loop
	fmt.Println("=== OPEN-LOOP BEHAVIOR (No Context) ===")
	fmt.Println("Prompt: 'Build a basic HTTP server in Go that listens on port 8080'")
	fmt.Println()
	fmt.Println("Expected agent output (statistical baseline):")
	fmt.Println(`  http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintf(w, "Hello, World!")
  })
  http.ListenAndServe(":8080", nil)`)
	fmt.Println()
	fmt.Println("Issues: No structured logging, no health endpoint, no graceful shutdown")
	fmt.Println()

	// Check for ADRs
	adrs := categories["adr"]
	if len(adrs) > 0 {
		fmt.Println("=== CLOSED-LOOP BEHAVIOR (With ADR Context) ===")
		fmt.Println("Prompt: 'Build a basic HTTP server in Go that listens on port 8080'")
		fmt.Println("Context: ADR-001-Logging.md requires structured JSON logging with zap")
		fmt.Println()
		fmt.Println("Expected agent output (constrained by ADR):")
		fmt.Println(`  logger, _ := zap.NewProduction()
  defer logger.Sync()

  http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
  })

  http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
    logger.Info("request received",
      zap.String("method", r.Method),
      zap.String("path", r.URL.Path),
      zap.String("remote", r.RemoteAddr),
    )
    fmt.Fprintf(w, "Hello, World!")
  })

  server := &http.Server{Addr: ":8080"}
  go func() { server.ListenAndServe() }()
  // Graceful shutdown handling...`)
		fmt.Println()
		fmt.Println("Benefits: Structured logging, health endpoint, graceful shutdown, observability")
	} else {
		fmt.Println("=== ADD ADR CONTEXT FOR CLOSED-LOOP DEMO ===")
		fmt.Println("Create docs/ADR-001-Logging.md with structured logging requirements")
		fmt.Println("Then re-run to see closed-loop behavior")
	}

	// Show formatted context for agent
	fmt.Println("=== FORMATTED CONTEXT FOR AGENT INJECTION ===")
	formatted := asfContext.FormatForAgent(items)
	fmt.Println(formatted)
}