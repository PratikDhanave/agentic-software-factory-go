package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	asfContext "github.com/PratikDhanave/agentic-software-factory-go/pkg/context"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/factory"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/factorydef"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/harness"
)

// webServer represents the ASF web server
type webServer struct {
	factory   *factory.FactoryOrchestrator
	factoryDef *factorydef.FactoryDefinition
	harness   *harness.CompleteHarness
}

// newWebServer creates a new web server instance
func newWebServer(workingDir string) (*webServer, error) {
	// Initialize factory
	fo, err := factory.NewFactoryOrchestrator(workingDir, 4)
	if err != nil {
		return nil, fmt.Errorf("creating factory: %w", err)
	}

	// Load factory definition
	factoryDefPath, err := factorydef.FindFactoryDefinition(workingDir)
	var factoryDef *factorydef.FactoryDefinition
	if err == nil {
		factoryDef, err = factorydef.LoadFactoryDefinition(factoryDefPath)
		if err != nil {
			log.Printf("Warning: failed to load factory definition: %v", err)
		}
	}

	// Initialize harness
	h := harness.NewCompleteHarness(workingDir)

	return &webServer{
		factory:    fo,
		factoryDef: factoryDef,
		harness:    h,
	}, nil
}

// startHTTPServer starts the HTTP server with routes
func startHTTPServer(w *webServer, addr string) *http.Server {
	mux := http.NewServeMux()

	// Static file serving
	mux.HandleFunc("/static/", func(w http.ResponseWriter, r *http.Request) {
		http.StripPrefix("/static/", http.FileServer(http.Dir("./web/static"))).ServeHTTP(w, r)
	})

	mux.HandleFunc("/templates/", func(w http.ResponseWriter, r *http.Request) {
		http.StripPrefix("/templates/", http.FileServer(http.Dir("./web/templates"))).ServeHTTP(w, r)
	})

	// API routes
	mux.HandleFunc("/api/factory/status", w.handleFactoryStatus)
	mux.HandleFunc("/api/harness/status", w.handleHarnessStatus)
	mux.HandleFunc("/api/specs", w.handleListSpecs)
	mux.HandleFunc("/api/agents", w.handleListAgents)
	// WebSocket handler for task status streaming
func (w *webServer) handleTaskStream(w http.ResponseWriter, r *http.Request) {
	taskID := strings.TrimPrefix(r.URL.Path, "/api/tasks/")
	taskID = strings.TrimSuffix(taskID, "/stream")

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		task, ok := w.factory.GetTaskResult(taskID)
		if !ok {
			continue
		}

		err := conn.WriteJSON(map[string]interface{}{
			"status": task.Status,
			"log": fmt.Sprintf("[%s] Task %s: %s", time.Now().Format("15:04:05"), taskID, task.Status),
		})

		if task.Status == "completed" || task.Status == "failed" {
			conn.WriteJSON(map[string]interface{}{
				"status": task.Status,
				"result": task.Result,
				"error": task.Error,
			})
			break
		}
	}
}
	mux.HandleFunc("/api/tasks/", w.handleTaskStream)

	// Root path serves index.html
	mux.HandleFunc("/", w.handleRoot)

	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	return server
}

// handleFactoryLaunch launches a factory task
func (w *webServer) handleFactoryLaunch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.writeJSON(map[string]string{"error": "method not allowed"})
		return
	}

	var req struct {
		SpecID      string                 `json:"spec_id"`
		TaskType    string                 `json:"task_type"`
		Parameters  map[string]interface{} `json:"parameters"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.writeJSON(map[string]string{"error": "invalid request"})
		return
	}

	if req.SpecID == "" {
		w.writeJSON(map[string]string{"error": "spec_id is required"})
		return
	}

	task := &factory.Task{
		Type:       req.TaskType,
		SpecID:     req.SpecID,
		Priority:   1,
		Parameters: req.Parameters,
	}

	if err := w.factory.SubmitTask(task); err != nil {
		w.writeJSON(map[string]string{"error": err.Error()})
		return
	}

	w.writeJSON(map[string]interface{}{
		"task_id": task.ID,
		"status":  "queued",
		"message": "Factory task launched successfully",
	})
}

func (w *webServer) handleHarnessStatus(w http.ResponseWriter, r *http.Request) {
	// Return harness status
	w.writeJSON(map[string]interface{}{
		"status": "ready",
	})
}

func (w *webServer) handleListSpecs(w http.ResponseWriter, r *http.Request) {
	if w.factoryDef == nil {
		w.writeJSON(map[string]interface{}{
			"specs": []interface{}{},
		})
		return
	}

	specs := make([]map[string]interface{}, 0)
	for _, workflow := range w.factoryDef.Factory.Workflows {
		specs = append(specs, map[string]interface{}{
			"id":   workflow.ID,
			"steps": len(workflow.Steps),
		})
	}

	w.writeJSON(map[string]interface{}{
		"specs": specs,
	})
}

func (w *webServer) handleListAgents(w http.ResponseWriter, r *http.Request) {
	if w.factoryDef == nil {
		w.writeJSON(map[string]interface{}{
			"agents": []interface{}{},
		})
		return
	}

	agents := make([]map[string]interface{}, 0)
	for _, a := range w.factoryDef.Factory.Agents {
		agents = append(agents, map[string]interface{}{
			"id":           a.ID,
			"role":         a.Role,
			"capabilities": a.Capabilities,
			"model":        a.Model,
		})
	}

	w.writeJSON(map[string]interface{}{
		"agents": agents,
	})
}

func (w *webServer) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, "./web/index.html")
}

// writeJSON writes a JSON response
func (w *webServer) writeJSON(data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w.ResponseWriter).Encode(data)
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]
	workingDir, _ := os.Getwd()

	switch command {
	case "init":
		initProject(workingDir)
	case "web":
		runWebServer(workingDir)
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func runWebServer(workingDir string) {
	fmt.Println("Starting ASF Web Server...")

	// Create web server
	server, err := newWebServer(workingDir)
	if err != nil {
		log.Fatalf("Error creating web server: %v", err)
	}

	// Start HTTP server
	addr := "localhost:8080"
	if len(os.Args) > 2 {
		addr = os.Args[2]
	}

	srv := startHTTPServer(server, addr)

	fmt.Printf("ASF Web Server starting on %s...\n", addr)
	fmt.Println("Available endpoints:")
	fmt.Println("  GET  /              - Web interface")
	fmt.Println("  GET  /static/*      - Static files")
	fmt.Println("  GET  /templates/*   - Template files")
	fmt.Println("  GET  /api/factory/status  - Factory status")
	fmt.Println("  GET  /api/harness/status  - Harness status")
	fmt.Println("  GET  /api/specs       - List specifications")
	fmt.Println("  GET  /api/agents      - List agents")
	fmt.Println("  Press Ctrl+C to stop")

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server error: %v", err)
	}
}

func printUsage() {
	fmt.Print(`ASF Web Server - Agentic Software Factory Web Interface

Usage: asf web [address]

Commands:
  init              Initialize a new ASF project structure
  web               Start the web server (default: localhost:8080)
  help              Show this help

Examples:
  asf init
  asf web
  asf web :8080
`)
}