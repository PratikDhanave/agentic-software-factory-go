// Package store manages the single SQLite WAL database (.asf/factory.db)
// backing agents, runs, logs/usage, triggers, and persisted gateway settings.
package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/PratikDhanave/agentic-software-factory-go/internal/state"
)

type AgentRecord struct {
	Slug         string   `json:"slug"`
	Name         string   `json:"name"`
	Role         string   `json:"role"`
	Status       string   `json:"status"`
	Model        string   `json:"model"`
	Authority    string   `json:"authority"`
	Description  string   `json:"description"`
	Capabilities []string `json:"capabilities"`
	SystemPrompt string   `json:"system_prompt"`
	CreatedAt    string   `json:"created_at"`
}

type FileArtifact struct {
	Path    string `json:"path"`
	Lines   int    `json:"lines"`
	Content string `json:"content"`
}

type StepEvent struct {
	Stage      string `json:"stage"`
	Status     string `json:"status"`
	Message    string `json:"message"`
	DurationMs int64  `json:"duration_ms,omitempty"`
	Timestamp  string `json:"timestamp"`
}

type RunRecord struct {
	ID           string             `json:"id"`
	Title        string             `json:"title"`
	Prompt       string             `json:"prompt"`
	SpecID       string             `json:"spec_id"`
	AgentSlug    string             `json:"agent_slug"`
	Stage        string             `json:"stage"`
	Status       string             `json:"status"`
	Autonomy     string             `json:"autonomy"`
	WorkspaceDir string             `json:"workspace_dir"`
	PromotedDir  string             `json:"promoted_dir,omitempty"`
	Promotion    string             `json:"promotion_state"`
	Score        int                `json:"score"`
	Rubric       map[string]int     `json:"rubric,omitempty"`
	Files        []FileArtifact     `json:"files"`
	Steps        []StepEvent        `json:"steps"`
	BuildPassed  bool               `json:"build_passed"`
	VetPassed    bool               `json:"vet_passed"`
	TestsPassed  bool               `json:"tests_passed"`
	TestCases    []string           `json:"test_cases,omitempty"`
	TestOutput   string             `json:"test_output"`
	Attempts     int                `json:"attempts"`
	Error        string             `json:"error,omitempty"`
	CreatedAt    string             `json:"created_at"`
	UpdatedAt    string             `json:"updated_at"`
}

type LogEntry struct {
	ID         int64  `json:"id"`
	RunID      string `json:"run_id,omitempty"`
	AgentSlug  string `json:"agent_slug,omitempty"`
	Provider   string `json:"provider,omitempty"`
	Model      string `json:"model,omitempty"`
	Tokens     int    `json:"tokens"`
	DurationMs int64  `json:"duration_ms"`
	Level      string `json:"level"`
	Message    string `json:"message"`
	CreatedAt  string `json:"created_at"`
}

type TriggerRecord struct {
	ID        string `json:"id"`
	Source    string `json:"source"`
	EventType string `json:"event_type"`
	Status    string `json:"status"`
	RunID     string `json:"run_id,omitempty"`
	Summary   string `json:"summary"`
	CreatedAt string `json:"created_at"`
}

type Store struct {
	path string
	db   *sql.DB
}

func Open(dbPath string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("creating db directory: %w", err)
	}
	dsn := fmt.Sprintf("file:%s?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=on", dbPath)
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	s := &Store{path: dbPath, db: db}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := s.seedDefaultAgents(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Path() string { return s.path }
func (s *Store) Close() error { return s.db.Close() }

func (s *Store) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS agents (
		slug TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		role TEXT NOT NULL,
		status TEXT NOT NULL,
		model TEXT NOT NULL,
		authority TEXT NOT NULL,
		description TEXT NOT NULL,
		capabilities_json TEXT NOT NULL,
		system_prompt TEXT NOT NULL,
		created_at TEXT NOT NULL
	);
	CREATE TABLE IF NOT EXISTS runs (
		id TEXT PRIMARY KEY,
		title TEXT NOT NULL,
		prompt TEXT NOT NULL,
		spec_id TEXT NOT NULL,
		agent_slug TEXT NOT NULL,
		stage TEXT NOT NULL,
		status TEXT NOT NULL,
		autonomy TEXT NOT NULL,
		workspace_dir TEXT NOT NULL,
		promoted_dir TEXT NOT NULL,
		promotion_state TEXT NOT NULL,
		score INTEGER NOT NULL,
		rubric_json TEXT NOT NULL,
		files_json TEXT NOT NULL,
		steps_json TEXT NOT NULL,
		build_passed INTEGER NOT NULL,
		vet_passed INTEGER NOT NULL,
		tests_passed INTEGER NOT NULL,
		test_cases_json TEXT NOT NULL,
		test_output TEXT NOT NULL,
		attempts INTEGER NOT NULL,
		error TEXT NOT NULL,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL
	);
	CREATE TABLE IF NOT EXISTS logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		run_id TEXT NOT NULL,
		agent_slug TEXT NOT NULL,
		provider TEXT NOT NULL,
		model TEXT NOT NULL,
		tokens INTEGER NOT NULL,
		duration_ms INTEGER NOT NULL,
		level TEXT NOT NULL,
		message TEXT NOT NULL,
		created_at TEXT NOT NULL
	);
	CREATE TABLE IF NOT EXISTS triggers (
		id TEXT PRIMARY KEY,
		source TEXT NOT NULL,
		event_type TEXT NOT NULL,
		status TEXT NOT NULL,
		run_id TEXT NOT NULL,
		summary TEXT NOT NULL,
		created_at TEXT NOT NULL
	);
	CREATE TABLE IF NOT EXISTS settings (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL
	);`
	_, err := s.db.Exec(schema)
	return err
}

func (s *Store) seedDefaultAgents() error {
	defaults := []AgentRecord{
		{
			Slug:         "factory-architect",
			Name:         "Factory Architect",
			Role:         "architect",
			Status:       string(state.AgentHealthy),
			Model:        "google/gemini-2.5-pro",
			Authority:    "authoritative",
			Description:  "Designs Go package boundaries, explicit state machines, and concurrency invariants.",
			Capabilities: []string{"architecture", "concurrency", "state-machines"},
			SystemPrompt: "You are Factory Architect. Design clean, idiomatic, concurrency-safe Go architectures with explicit state machines and zero global state.",
		},
		{
			Slug:         "factory-developer",
			Name:         "Factory Go Developer",
			Role:         "developer",
			Status:       string(state.AgentHealthy),
			Model:        "google/gemini-2.5-pro",
			Authority:    "authoritative",
			Description:  "Synthesizes multi-file Go implementations inside isolated module sandboxes.",
			Capabilities: []string{"codegen", "go-modules", "self-healing"},
			SystemPrompt: "You are Factory Go Developer. Write clean, production-grade Go 1.23+ code with explicit error handling and zero external bloat.",
		},
		{
			Slug:         "factory-spec-verifier",
			Name:         "Factory Spec Verifier",
			Role:         "qa",
			Status:       string(state.AgentHealthy),
			Model:        "google/gemini-2.5-flash",
			Authority:    "authoritative",
			Description:  "Authors table-driven unit tests and executes go test -race verification.",
			Capabilities: []string{"unit-tests", "race-detector", "invariants"},
			SystemPrompt: "You are Factory Spec Verifier. Design thorough table-driven Go tests and multi-goroutine race detector stress suites.",
		},
		{
			Slug:         "factory-security",
			Name:         "Factory Security",
			Role:         "security",
			Status:       string(state.AgentHealthy),
			Model:        "google/gemini-2.5-pro",
			Authority:    "authoritative",
			Description:  "Audits constant-time HMAC checks, path-boundary safety, and input validation.",
			Capabilities: []string{"cryptography", "sandbox-safety", "rbac"},
			SystemPrompt: "You are Factory Security. Audit Go code for timing side-channels, path traversal, unbounded allocations, and secret exposure.",
		},
		{
			Slug:         "factory-sre",
			Name:         "Factory SRE",
			Role:         "performance",
			Status:       string(state.AgentHealthy),
			Model:        "google/gemini-2.5-flash",
			Authority:    "advisory",
			Description:  "Monitors tail latency, lock contention, backpressure, and graceful shutdown.",
			Capabilities: []string{"benchmarks", "backpressure", "resilience"},
			SystemPrompt: "You are Factory SRE. Analyze lock contention, memory allocations, timeouts, and circuit-breaking resilience in Go services.",
		},
		{
			Slug:         "factory-reviewer",
			Name:         "Factory Code Reviewer",
			Role:         "reviewer",
			Status:       string(state.AgentHealthy),
			Model:        "google/gemini-2.5-flash",
			Authority:    "authoritative",
			Description:  "Scores verified packages across correctness, spec compliance, tests, and maintainability.",
			Capabilities: []string{"code-review", "rubric-judge", "idiomatic-go"},
			SystemPrompt: "You are Factory Code Reviewer. Critique Go code for clarity, idiomatic patterns, edge-case coverage, and maintainability.",
		},
		{
			Slug:         "factory-watcher",
			Name:         "Factory Watcher",
			Role:         "watcher",
			Status:       string(state.AgentHealthy),
			Model:        "google/gemini-2.5-flash",
			Authority:    "advisory",
			Description:  "Watches inbound GitHub issues, PRs, and @factory comments to dispatch runs.",
			Capabilities: []string{"webhooks", "triage", "dispatch"},
			SystemPrompt: "You are Factory Watcher. Triage incoming issues and pull requests into actionable Go specifications.",
		},
		{
			Slug:         "factory-researcher",
			Name:         "Factory Researcher",
			Role:         "researcher",
			Status:       string(state.AgentHealthy),
			Model:        "google/gemini-2.5-flash",
			Authority:    "advisory",
			Description:  "Extracts specifications, edge cases, and API contracts from plain-English prompts.",
			Capabilities: []string{"spec-synthesis", "context-analysis"},
			SystemPrompt: "You are Factory Researcher. Translate high-level feature requests into unambiguous Go interface contracts and acceptance criteria.",
		},
	}

	now := time.Now().UTC().Format(time.RFC3339)
	for _, a := range defaults {
		caps, _ := json.Marshal(a.Capabilities)
		_, err := s.db.Exec(`
			INSERT OR IGNORE INTO agents (slug, name, role, status, model, authority, description, capabilities_json, system_prompt, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, a.Slug, a.Name, a.Role, a.Status, a.Model, a.Authority, a.Description, string(caps), a.SystemPrompt, now)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) ListAgents(ctx context.Context) ([]AgentRecord, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT slug, name, role, status, model, authority, description, capabilities_json, system_prompt, created_at
		FROM agents ORDER BY rowid ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []AgentRecord
	for rows.Next() {
		var a AgentRecord
		var caps string
		if err := rows.Scan(&a.Slug, &a.Name, &a.Role, &a.Status, &a.Model, &a.Authority, &a.Description, &caps, &a.SystemPrompt, &a.CreatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(caps), &a.Capabilities)
		out = append(out, a)
	}
	return out, rows.Err()
}

func (s *Store) GetAgent(ctx context.Context, slug string) (*AgentRecord, error) {
	var a AgentRecord
	var caps string
	err := s.db.QueryRowContext(ctx, `
		SELECT slug, name, role, status, model, authority, description, capabilities_json, system_prompt, created_at
		FROM agents WHERE slug = ? OR role = ? LIMIT 1
	`, slug, slug).Scan(&a.Slug, &a.Name, &a.Role, &a.Status, &a.Model, &a.Authority, &a.Description, &caps, &a.SystemPrompt, &a.CreatedAt)
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal([]byte(caps), &a.Capabilities)
	return &a, nil
}

func (s *Store) SaveRun(ctx context.Context, r *RunRecord) error {
	now := time.Now().UTC().Format(time.RFC3339)
	if r.CreatedAt == "" {
		r.CreatedAt = now
	}
	r.UpdatedAt = now
	if r.Files == nil {
		r.Files = []FileArtifact{}
	}
	if r.Steps == nil {
		r.Steps = []StepEvent{}
	}
	if r.Rubric == nil {
		r.Rubric = map[string]int{}
	}
	if r.TestCases == nil {
		r.TestCases = []string{}
	}
	rubricB, _ := json.Marshal(r.Rubric)
	filesB, _ := json.Marshal(r.Files)
	stepsB, _ := json.Marshal(r.Steps)
	casesB, _ := json.Marshal(r.TestCases)

	btoi := func(b bool) int {
		if b {
			return 1
		}
		return 0
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO runs (
			id, title, prompt, spec_id, agent_slug, stage, status, autonomy,
			workspace_dir, promoted_dir, promotion_state, score, rubric_json,
			files_json, steps_json, build_passed, vet_passed, tests_passed,
			test_cases_json, test_output, attempts, error, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			title = excluded.title,
			spec_id = excluded.spec_id,
			stage = excluded.stage,
			status = excluded.status,
			workspace_dir = excluded.workspace_dir,
			promoted_dir = excluded.promoted_dir,
			promotion_state = excluded.promotion_state,
			score = excluded.score,
			rubric_json = excluded.rubric_json,
			files_json = excluded.files_json,
			steps_json = excluded.steps_json,
			build_passed = excluded.build_passed,
			vet_passed = excluded.vet_passed,
			tests_passed = excluded.tests_passed,
			test_cases_json = excluded.test_cases_json,
			test_output = excluded.test_output,
			attempts = excluded.attempts,
			error = excluded.error,
			updated_at = excluded.updated_at
	`,
		r.ID, r.Title, r.Prompt, r.SpecID, r.AgentSlug, r.Stage, r.Status, r.Autonomy,
		r.WorkspaceDir, r.PromotedDir, r.Promotion, r.Score, string(rubricB),
		string(filesB), string(stepsB), btoi(r.BuildPassed), btoi(r.VetPassed), btoi(r.TestsPassed),
		string(casesB), r.TestOutput, r.Attempts, r.Error, r.CreatedAt, r.UpdatedAt,
	)
	return err
}

func scanRun(scanner interface{ Scan(dest ...any) error }) (*RunRecord, error) {
	var r RunRecord
	var rubricS, filesS, stepsS, casesS string
	var buildI, vetI, testsI int
	err := scanner.Scan(
		&r.ID, &r.Title, &r.Prompt, &r.SpecID, &r.AgentSlug, &r.Stage, &r.Status, &r.Autonomy,
		&r.WorkspaceDir, &r.PromotedDir, &r.Promotion, &r.Score, &rubricS,
		&filesS, &stepsS, &buildI, &vetI, &testsI,
		&casesS, &r.TestOutput, &r.Attempts, &r.Error, &r.CreatedAt, &r.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	r.BuildPassed = buildI == 1
	r.VetPassed = vetI == 1
	r.TestsPassed = testsI == 1
	_ = json.Unmarshal([]byte(rubricS), &r.Rubric)
	_ = json.Unmarshal([]byte(filesS), &r.Files)
	_ = json.Unmarshal([]byte(stepsS), &r.Steps)
	_ = json.Unmarshal([]byte(casesS), &r.TestCases)
	return &r, nil
}

func (s *Store) GetRun(ctx context.Context, id string) (*RunRecord, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, title, prompt, spec_id, agent_slug, stage, status, autonomy,
		       workspace_dir, promoted_dir, promotion_state, score, rubric_json,
		       files_json, steps_json, build_passed, vet_passed, tests_passed,
		       test_cases_json, test_output, attempts, error, created_at, updated_at
		FROM runs WHERE id = ?
	`, id)
	return scanRun(row)
}

func (s *Store) ListRuns(ctx context.Context, limit int) ([]RunRecord, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, title, prompt, spec_id, agent_slug, stage, status, autonomy,
		       workspace_dir, promoted_dir, promotion_state, score, rubric_json,
		       files_json, steps_json, build_passed, vet_passed, tests_passed,
		       test_cases_json, test_output, attempts, error, created_at, updated_at
		FROM runs ORDER BY created_at DESC LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]RunRecord, 0)
	for rows.Next() {
		r, err := scanRun(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *r)
	}
	return out, rows.Err()
}

func (s *Store) AddLog(ctx context.Context, entry LogEntry) {
	if entry.CreatedAt == "" {
		entry.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	if entry.Level == "" {
		entry.Level = "info"
	}
	_, _ = s.db.ExecContext(ctx, `
		INSERT INTO logs (run_id, agent_slug, provider, model, tokens, duration_ms, level, message, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, entry.RunID, entry.AgentSlug, entry.Provider, entry.Model, entry.Tokens, entry.DurationMs, entry.Level, entry.Message, entry.CreatedAt)
}

func (s *Store) ListLogs(ctx context.Context, limit int) ([]LogEntry, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, run_id, agent_slug, provider, model, tokens, duration_ms, level, message, created_at
		FROM logs ORDER BY id DESC LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]LogEntry, 0)
	for rows.Next() {
		var l LogEntry
		if err := rows.Scan(&l.ID, &l.RunID, &l.AgentSlug, &l.Provider, &l.Model, &l.Tokens, &l.DurationMs, &l.Level, &l.Message, &l.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

func (s *Store) UsageSummary(ctx context.Context) map[string]any {
	var totalCalls, totalTokens int
	var avgMs float64
	_ = s.db.QueryRowContext(ctx, `
		SELECT COUNT(*), COALESCE(SUM(tokens), 0), COALESCE(AVG(duration_ms), 0)
		FROM logs WHERE tokens > 0
	`).Scan(&totalCalls, &totalTokens, &avgMs)

	byProvider := make([]map[string]any, 0)
	rows, err := s.db.QueryContext(ctx, `
		SELECT provider, model, COUNT(*), COALESCE(SUM(tokens), 0)
		FROM logs WHERE provider != '' GROUP BY provider, model ORDER BY SUM(tokens) DESC
	`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var p, m string
			var c, t int
			if rows.Scan(&p, &m, &c, &t) == nil {
				byProvider = append(byProvider, map[string]any{
					"provider": p,
					"model":    m,
					"calls":    c,
					"tokens":   t,
				})
			}
		}
	}
	return map[string]any{
		"total_calls":    totalCalls,
		"total_tokens":   totalTokens,
		"avg_latency_ms": int64(avgMs),
		"by_provider":    byProvider,
	}
}

func (s *Store) SaveTrigger(ctx context.Context, tr TriggerRecord) error {
	if tr.CreatedAt == "" {
		tr.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT OR REPLACE INTO triggers (id, source, event_type, status, run_id, summary, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, tr.ID, tr.Source, tr.EventType, tr.Status, tr.RunID, tr.Summary, tr.CreatedAt)
	return err
}

func (s *Store) ListTriggers(ctx context.Context, limit int) ([]TriggerRecord, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, source, event_type, status, run_id, summary, created_at
		FROM triggers ORDER BY created_at DESC LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]TriggerRecord, 0)
	for rows.Next() {
		var tr TriggerRecord
		if err := rows.Scan(&tr.ID, &tr.Source, &tr.EventType, &tr.Status, &tr.RunID, &tr.Summary, &tr.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, tr)
	}
	return out, rows.Err()
}

func (s *Store) SetSetting(ctx context.Context, key, val string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO settings (key, value) VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value
	`, key, val)
	return err
}

func (s *Store) GetSetting(ctx context.Context, key, fallback string) string {
	var val string
	if err := s.db.QueryRowContext(ctx, `SELECT value FROM settings WHERE key = ?`, key).Scan(&val); err != nil || val == "" {
		return fallback
	}
	return val
}
