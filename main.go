// Agentic Software Factory — self-contained Go Control Plane + Embedded UI + CLI.
//
// Usage:
//
//	./factory                        Start Control Plane & UI on http://localhost:8080
//	./factory run "Build ..."        Run a synchronous verified Go build from the CLI
//	./factory states [--mermaid]     Inspect all explicit finite state machines
//	./factory specs                  List YAML blueprints in specs/
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/PratikDhanave/agentic-software-factory-go/internal/engine"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/gateway"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/sandbox"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/server"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/state"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/store"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "states":
			runStatesCLI(os.Args[2:])
			return
		case "specs":
			runSpecsCLI()
			return
		case "run":
			runBuildCLI(os.Args[2:])
			return
		case "help", "-h", "--help":
			printUsage()
			return
		}
	}
	runServer(os.Args[1:])
}

func printUsage() {
	fmt.Println(`Agentic Software Factory (Go)

Commands:
  factory [-addr :8080]                Start Control Plane + Embedded UI on http://localhost:8080
  factory run [-spec id] [-promote]    Synthesize, go test -race verify, and optionally promote a Go package
  factory states [-json | -mermaid]    Inspect all 8 explicit finite state machines
  factory specs                        List YAML specifications in specs/`)
}

func openStack() (string, *store.Store, *gateway.Gateway, *engine.Engine) {
	rootDir, err := os.Getwd()
	if err != nil {
		rootDir = "."
	}
	dbPath := filepath.Join(rootDir, ".asf", "factory.db")
	st, err := store.Open(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error opening store %s: %v\n", dbPath, err)
		os.Exit(1)
	}
	gw := gateway.New(st)
	eng := engine.New(rootDir, st, gw)
	return rootDir, st, gw, eng
}

func runServer(args []string) {
	fs := flag.NewFlagSet("factory", flag.ExitOnError)
	defaultAddr := ":8080"
	if p := strings.TrimSpace(os.Getenv("PORT")); p != "" {
		defaultAddr = ":" + strings.TrimPrefix(p, ":")
	}
	if a := strings.TrimSpace(os.Getenv("ADDR")); a != "" {
		defaultAddr = a
	}
	addr := fs.String("addr", defaultAddr, "HTTP listen address")
	_ = fs.Parse(args)

	rootDir, st, gw, eng := openStack()
	defer st.Close()

	srv := server.New(rootDir, st, gw, eng)
	httpSrv := &http.Server{
		Addr:              *addr,
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	displayURL := "http://localhost" + *addr
	if !strings.HasPrefix(*addr, ":") {
		displayURL = "http://" + *addr
	}
	prov, model := gw.ActiveProvider()

	fmt.Println("╭──────────────────────────────────────────────────────────────╮")
	fmt.Println("│  Agentic Software Factory — Go Control Plane                 │")
	fmt.Printf("│  UI & API   : %-46s │\n", displayURL)
	fmt.Printf("│  Autonomy   : %-46s │\n", string(eng.Autonomy()))
	fmt.Printf("│  Gateway    : %-46s │\n", fmt.Sprintf("%s (%s)", prov, model))
	fmt.Printf("│  SQLite DB  : %-46s │\n", ".asf/factory.db")
	fmt.Println("╰──────────────────────────────────────────────────────────────╯")

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = httpSrv.Shutdown(ctx)
	}()

	if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}

func runStatesCLI(args []string) {
	fs := flag.NewFlagSet("states", flag.ExitOnError)
	asJSON := fs.Bool("json", false, "Output JSON")
	asMermaid := fs.Bool("mermaid", false, "Output Mermaid stateDiagram-v2")
	_ = fs.Parse(args)

	if *asJSON {
		b, _ := json.MarshalIndent(state.Exports(), "", "  ")
		fmt.Println(string(b))
		return
	}
	for _, m := range state.All() {
		fmt.Printf("=== %s [%s] (initial: %s) ===\n", m.Name(), m.Category(), m.Initial())
		fmt.Printf("  %s\n", m.Description())
		if *asMermaid {
			fmt.Println(m.Mermaid())
			continue
		}
		for _, st := range m.States() {
			next := m.AllowedFrom(st)
			if len(next) == 0 {
				fmt.Printf("  %-20s -> [terminal]\n", st)
			} else {
				fmt.Printf("  %-20s -> %v\n", st, next)
			}
		}
		fmt.Println()
	}
}

func runSpecsCLI() {
	specs, err := sandbox.LoadSpecs("specs")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading specs: %v\n", err)
		os.Exit(1)
	}
	for _, sp := range specs {
		fmt.Printf("%-24s  %s (%s)\n", sp.ID, sp.Title, sp.Filename)
	}
}

func runBuildCLI(args []string) {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	specID := fs.String("spec", "", "Blueprint spec ID (e.g. spec-token-bucket)")
	promote := fs.Bool("promote", false, "Promote verified Go package into generated/")
	_ = fs.Parse(args)

	prompt := strings.TrimSpace(strings.Join(fs.Args(), " "))
	_, st, _, eng := openStack()
	defer st.Close()

	// Ensure CLI run executes end-to-end without pausing unless Crawl was explicitly requested
	if eng.Autonomy() == state.AutonomyCrawl {
		_, _ = eng.SetAutonomy(context.Background(), string(state.AutonomyWalk))
	}

	ctx := context.Background()
	rec, err := eng.StartRun(ctx, engine.StartRequest{
		Prompt: prompt,
		SpecID: *specID,
		Sync:   true,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "run error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Run ID      : %s\n", rec.ID)
	fmt.Printf("Status      : %s (score %d/100)\n", rec.Status, rec.Score)
	fmt.Printf("Workspace   : %s\n", rec.WorkspaceDir)
	fmt.Printf("Tests (%d)   : %v\n", len(rec.TestCases), rec.TestCases)
	if rec.Status != string(state.RunCompleted) {
		fmt.Fprintf(os.Stderr, "Error:\n%s\n", rec.Error)
		os.Exit(1)
	}
	if *promote {
		promoted, err := eng.PromoteRun(ctx, rec.ID, "")
		if err != nil {
			fmt.Fprintf(os.Stderr, "promote error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Promoted To : %s/\n", promoted.PromotedDir)
	}
}
