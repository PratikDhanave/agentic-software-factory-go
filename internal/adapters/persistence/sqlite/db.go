package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Store wraps the unified WAL-enabled SQLite connection for the entire factory.
type Store struct {
	db   *sql.DB
	path string
}

// Open opens (and creates parent directories for) a SQLite database at dbPath
// (or ":memory:" for ephemeral test stores) and runs idempotent schema migrations.
func Open(dbPath string) (*Store, error) {
	if strings.TrimSpace(dbPath) == "" {
		dbPath = filepath.Join(".asf", "factory.db")
	}

	dsn := dbPath
	isMemory := dbPath == ":memory:" || strings.HasPrefix(dbPath, "file::memory:")
	if !isMemory {
		if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
			return nil, fmt.Errorf("creating sqlite directory: %w", err)
		}
		sep := "?"
		if strings.Contains(dbPath, "?") {
			sep = "&"
		}
		dsn = dbPath + sep + "_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=on"
	}

	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("opening sqlite db: %w", err)
	}

	if isMemory {
		// Keep a single connection for :memory: so schema persists across queries.
		db.SetMaxOpenConns(1)
	} else {
		db.SetMaxOpenConns(8)
		db.SetMaxIdleConns(4)
		db.SetConnMaxLifetime(30 * time.Minute)
	}

	s := &Store{db: db, path: dbPath}
	if err := s.Migrate(context.Background()); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("running sqlite migrations: %w", err)
	}
	return s, nil
}

func (s *Store) DB() *sql.DB {
	return s.db
}

func (s *Store) Path() string {
	return s.path
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *Store) Health(ctx context.Context) map[string]interface{} {
	err := s.db.PingContext(ctx)
	res := map[string]interface{}{
		"engine": "sqlite3",
		"path":   s.path,
		"status": "healthy",
	}
	if err != nil {
		res["status"] = "unhealthy"
		res["error"] = err.Error()
	}
	return res
}

func (s *Store) Stats(ctx context.Context) (map[string]interface{}, error) {
	counts := map[string]int{}
	tables := []string{
		"tasks",
		"jobs",
		"factory_runs",
		"workflow_definitions",
		"workflow_executions",
		"coordination_sessions",
		"agent_decisions",
		"approvals",
		"learned_patterns",
	}
	for _, tbl := range tables {
		var n int
		if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+tbl).Scan(&n); err == nil {
			counts[tbl] = n
		}
	}

	var completedExecs, failedExecs int
	_ = s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM workflow_executions WHERE status = 'completed'").Scan(&completedExecs)
	_ = s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM workflow_executions WHERE status = 'failed'").Scan(&failedExecs)

	var pendingApprovals int
	_ = s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM coordination_sessions WHERE status = 'waiting_approval'").Scan(&pendingApprovals)

	return map[string]interface{}{
		"engine":                  "sqlite3",
		"path":                    s.path,
		"tables":                  counts,
		"total_workflows":         counts["workflow_definitions"],
		"total_executions":        counts["workflow_executions"],
		"total_factory_runs":      counts["factory_runs"],
		"completed_executions":    completedExecs,
		"failed_executions":       failedExecs,
		"total_tasks":             counts["tasks"],
		"total_sessions":          counts["coordination_sessions"],
		"pending_approvals":       pendingApprovals,
		"total_agent_decisions":   counts["agent_decisions"],
		"discovered_patterns":     counts["learned_patterns"],
		"updated_at":              time.Now().UTC().Format(time.RFC3339),
	}, nil
}

func (s *Store) Migrate(ctx context.Context) error {
	schema := `
	PRAGMA foreign_keys = ON;

	CREATE TABLE IF NOT EXISTS tasks (
		id TEXT PRIMARY KEY,
		type TEXT NOT NULL,
		spec_id TEXT NOT NULL,
		priority INTEGER NOT NULL DEFAULT 0,
		status TEXT NOT NULL,
		assigned_to TEXT DEFAULT '',
		created_at DATETIME NOT NULL,
		started_at DATETIME,
		completed_at DATETIME,
		result BLOB,
		error TEXT DEFAULT '',
		retries INTEGER NOT NULL DEFAULT 0
	);
	CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);
	CREATE INDEX IF NOT EXISTS idx_tasks_spec_id ON tasks(spec_id);

	CREATE TABLE IF NOT EXISTS workers (
		id TEXT PRIMARY KEY,
		role TEXT NOT NULL,
		capabilities TEXT NOT NULL,
		status TEXT NOT NULL,
		last_active DATETIME NOT NULL
	);

	CREATE TABLE IF NOT EXISTS factory_state (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL,
		updated_at DATETIME NOT NULL
	);

	CREATE TABLE IF NOT EXISTS jobs (
		id TEXT PRIMARY KEY,
		type TEXT NOT NULL,
		status TEXT NOT NULL,
		input_json TEXT NOT NULL DEFAULT '{}',
		output_json TEXT NOT NULL DEFAULT '{}',
		error TEXT NOT NULL DEFAULT '',
		created_at DATETIME NOT NULL,
		started_at DATETIME,
		completed_at DATETIME,
		retries INTEGER NOT NULL DEFAULT 0,
		max_retries INTEGER NOT NULL DEFAULT 3,
		priority INTEGER NOT NULL DEFAULT 0
	);
	CREATE INDEX IF NOT EXISTS idx_jobs_status ON jobs(status);

	CREATE TABLE IF NOT EXISTS workflow_definitions (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		version TEXT NOT NULL,
		description TEXT NOT NULL DEFAULT '',
		jobs_json TEXT NOT NULL DEFAULT '{}',
		status TEXT NOT NULL,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_wf_def_name ON workflow_definitions(name);

	CREATE TABLE IF NOT EXISTS workflow_executions (
		id TEXT PRIMARY KEY,
		definition_id TEXT NOT NULL,
		trigger_json TEXT NOT NULL DEFAULT '{}',
		nodes_json TEXT NOT NULL DEFAULT '{}',
		results_json TEXT NOT NULL DEFAULT '{}',
		pr_url TEXT NOT NULL DEFAULT '',
		status TEXT NOT NULL,
		started_at DATETIME,
		completed_at DATETIME,
		created_at DATETIME NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_wf_exec_def ON workflow_executions(definition_id);
	CREATE INDEX IF NOT EXISTS idx_wf_exec_status ON workflow_executions(status);

	CREATE TABLE IF NOT EXISTS dag_node_runs (
		execution_id TEXT NOT NULL,
		job_id TEXT NOT NULL,
		task_name TEXT NOT NULL,
		depends_json TEXT NOT NULL DEFAULT '[]',
		input_json TEXT NOT NULL DEFAULT '{}',
		status TEXT NOT NULL,
		result_json TEXT NOT NULL DEFAULT 'null',
		error TEXT,
		started_at DATETIME,
		completed_at DATETIME,
		duration_ns INTEGER NOT NULL DEFAULT 0,
		PRIMARY KEY (execution_id, job_id)
	);

	CREATE TABLE IF NOT EXISTS coordination_sessions (
		id TEXT PRIMARY KEY,
		workflow_def_id TEXT NOT NULL DEFAULT '',
		trigger_json TEXT NOT NULL DEFAULT '{}',
		decisions_json TEXT NOT NULL DEFAULT '{}',
		status TEXT NOT NULL,
		approver TEXT NOT NULL DEFAULT '',
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL,
		completed_at DATETIME
	);
	CREATE INDEX IF NOT EXISTS idx_coord_status ON coordination_sessions(status);

	CREATE TABLE IF NOT EXISTS agent_decisions (
		id TEXT PRIMARY KEY,
		execution_id TEXT NOT NULL,
		agent_role TEXT NOT NULL,
		issue_type TEXT NOT NULL DEFAULT 'general',
		issue_number INTEGER NOT NULL DEFAULT 0,
		decision TEXT NOT NULL,
		reasoning TEXT NOT NULL DEFAULT '',
		confidence REAL NOT NULL DEFAULT 0,
		alternatives_json TEXT NOT NULL DEFAULT '[]',
		requires_approval INTEGER NOT NULL DEFAULT 0,
		approval_status TEXT NOT NULL DEFAULT 'pending',
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_decisions_role ON agent_decisions(agent_role, created_at DESC);

	CREATE TABLE IF NOT EXISTS approvals (
		id TEXT PRIMARY KEY,
		execution_id TEXT NOT NULL DEFAULT '',
		decision_id TEXT NOT NULL,
		agent_role TEXT NOT NULL DEFAULT '',
		status TEXT NOT NULL,
		approver_id TEXT NOT NULL DEFAULT '',
		comments TEXT NOT NULL DEFAULT '',
		approved_at DATETIME NOT NULL,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL
	);

	CREATE TABLE IF NOT EXISTS agent_metrics (
		id TEXT PRIMARY KEY,
		agent_role TEXT NOT NULL,
		execution_id TEXT NOT NULL,
		decision_correct INTEGER NOT NULL DEFAULT 0,
		confidence REAL NOT NULL DEFAULT 0,
		approval_required INTEGER NOT NULL DEFAULT 0,
		approved_by_human INTEGER NOT NULL DEFAULT 0,
		issue_type TEXT NOT NULL DEFAULT 'general',
		created_at DATETIME NOT NULL
	);

	CREATE TABLE IF NOT EXISTS learned_patterns (
		id TEXT PRIMARY KEY,
		agent_role TEXT NOT NULL,
		pattern_type TEXT NOT NULL,
		issue_type TEXT NOT NULL,
		description TEXT NOT NULL DEFAULT '',
		success_rate REAL NOT NULL DEFAULT 0,
		times_applied INTEGER NOT NULL DEFAULT 0,
		confidence_base REAL NOT NULL DEFAULT 0,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL,
		UNIQUE(agent_role, pattern_type, issue_type)
	);

	CREATE TABLE IF NOT EXISTS confidence_calibrations (
		agent_role TEXT NOT NULL,
		confidence_bin TEXT NOT NULL,
		calibration REAL NOT NULL DEFAULT 1.0,
		sample_count INTEGER NOT NULL DEFAULT 0,
		actual_rate REAL NOT NULL DEFAULT 0,
		predicted_rate REAL NOT NULL DEFAULT 0,
		updated_at DATETIME NOT NULL,
		PRIMARY KEY (agent_role, confidence_bin)
	);

	CREATE TABLE IF NOT EXISTS agent_specializations (
		agent TEXT NOT NULL,
		specialty_type TEXT NOT NULL,
		success_rate REAL NOT NULL DEFAULT 0,
		avg_confidence REAL NOT NULL DEFAULT 0,
		decision_count INTEGER NOT NULL DEFAULT 0,
		updated_at DATETIME NOT NULL,
		PRIMARY KEY (agent, specialty_type)
	);

	CREATE TABLE IF NOT EXISTS factory_runs (
		id TEXT PRIMARY KEY,
		title TEXT NOT NULL,
		description TEXT NOT NULL DEFAULT '',
		spec_id TEXT NOT NULL DEFAULT '',
		session_id TEXT NOT NULL DEFAULT '',
		phase_mode TEXT NOT NULL DEFAULT 'walk',
		stage TEXT NOT NULL,
		status TEXT NOT NULL,
		workspace_dir TEXT NOT NULL DEFAULT '',
		promoted_dir TEXT NOT NULL DEFAULT '',
		payload_json TEXT NOT NULL DEFAULT '{}',
		error TEXT NOT NULL DEFAULT '',
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL,
		completed_at DATETIME
	);
	CREATE INDEX IF NOT EXISTS idx_factory_runs_created ON factory_runs(created_at DESC);
	CREATE INDEX IF NOT EXISTS idx_factory_runs_status ON factory_runs(status);
	`
	_, err := s.db.ExecContext(ctx, schema)
	return err
}
