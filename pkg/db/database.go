package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"
)

type Database struct {
	conn *sql.DB
}

func NewDatabase(dsn string) (*Database, error) {
	if dsn == "" {
		// Use environment variables or defaults
		host := os.Getenv("DB_HOST")
		if host == "" {
			host = "localhost"
		}
		port := os.Getenv("DB_PORT")
		if port == "" {
			port = "5432"
		}
		user := os.Getenv("DB_USER")
		if user == "" {
			user = "postgres"
		}
		password := os.Getenv("DB_PASSWORD")
		if password == "" {
			password = "postgres"
		}
		database := os.Getenv("DB_NAME")
		if database == "" {
			database = "asf"
		}

		dsn = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
			host, port, user, password, database)
	}

	conn, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := conn.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Set connection pool settings
	conn.SetMaxOpenConns(25)
	conn.SetMaxIdleConns(5)
	conn.SetConnMaxLifetime(5 * time.Minute)

	db := &Database{conn: conn}
	log.Println("[DB] Connected successfully")
	return db, nil
}

func (db *Database) Close() error {
	return db.conn.Close()
}

func (db *Database) Migrate() error {
	log.Println("[DB] Running migrations...")

	migrations := []string{
		createWorkflowRunsTable,
		createCodeMetricsTable,
		createReviewFindingsTable,
		createBugDetectionsTable,
		createTriageResultsTable,
		createIndexes,
	}

	for _, migration := range migrations {
		if _, err := db.conn.Exec(migration); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
	}

	log.Println("[DB] Migrations completed")
	return nil
}

func (db *Database) Exec(query string, args ...interface{}) (sql.Result, error) {
	return db.conn.Exec(query, args...)
}

func (db *Database) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return db.conn.Query(query, args...)
}

func (db *Database) QueryRow(query string, args ...interface{}) *sql.Row {
	return db.conn.QueryRow(query, args...)
}

func (db *Database) Health() map[string]interface{} {
	err := db.conn.Ping()
	status := "healthy"
	if err != nil {
		status = "unhealthy"
	}

	return map[string]interface{}{
		"status": status,
		"error":  err,
	}
}

const (
	createWorkflowRunsTable = `
		CREATE TABLE IF NOT EXISTS workflow_runs (
			id TEXT PRIMARY KEY,
			type VARCHAR(50) NOT NULL,
			status VARCHAR(50) NOT NULL,
			input JSONB,
			output JSONB,
			error_message TEXT,
			started_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
			completed_at TIMESTAMP WITH TIME ZONE,
			duration_ms INTEGER,
			created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
	`

	createCodeMetricsTable = `
		CREATE TABLE IF NOT EXISTS code_metrics (
			id TEXT PRIMARY KEY,
			workflow_run_id TEXT REFERENCES workflow_runs(id) ON DELETE CASCADE,
			language VARCHAR(50),
			lines_of_code INTEGER,
			code_score INTEGER,
			bugs_found INTEGER,
			bugs_fixed INTEGER,
			test_coverage DECIMAL(5, 2),
			complexity_score INTEGER,
			maintainability_score INTEGER,
			created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
	`

	createReviewFindingsTable = `
		CREATE TABLE IF NOT EXISTS review_findings (
			id TEXT PRIMARY KEY,
			workflow_run_id TEXT REFERENCES workflow_runs(id) ON DELETE CASCADE,
			file_name VARCHAR(255),
			score INTEGER,
			summary TEXT,
			approved BOOLEAN DEFAULT FALSE,
			findings JSONB,
			created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
	`

	createBugDetectionsTable = `
		CREATE TABLE IF NOT EXISTS bug_detections (
			id TEXT PRIMARY KEY,
			workflow_run_id TEXT REFERENCES workflow_runs(id) ON DELETE CASCADE,
			severity VARCHAR(20) NOT NULL,
			bug_type VARCHAR(100),
			location VARCHAR(255),
			description TEXT,
			impact TEXT,
			fix_suggestion TEXT,
			confidence INTEGER,
			fixed BOOLEAN DEFAULT FALSE,
			created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
	`

	createTriageResultsTable = `
		CREATE TABLE IF NOT EXISTS triage_results (
			id TEXT PRIMARY KEY,
			issue_number INTEGER,
			repository VARCHAR(255),
			priority VARCHAR(20),
			category VARCHAR(50),
			title VARCHAR(255),
			description TEXT,
			suggested_action VARCHAR(255),
			analyzed_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
			created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
	`

	createIndexes = `
		CREATE INDEX IF NOT EXISTS idx_workflow_runs_type ON workflow_runs(type);
		CREATE INDEX IF NOT EXISTS idx_workflow_runs_status ON workflow_runs(status);
		CREATE INDEX IF NOT EXISTS idx_workflow_runs_created ON workflow_runs(created_at DESC);
		CREATE INDEX IF NOT EXISTS idx_code_metrics_workflow ON code_metrics(workflow_run_id);
		CREATE INDEX IF NOT EXISTS idx_review_findings_workflow ON review_findings(workflow_run_id);
		CREATE INDEX IF NOT EXISTS idx_bug_detections_workflow ON bug_detections(workflow_run_id);
		CREATE INDEX IF NOT EXISTS idx_bug_detections_severity ON bug_detections(severity);
		CREATE INDEX IF NOT EXISTS idx_triage_results_priority ON triage_results(priority);
		CREATE INDEX IF NOT EXISTS idx_triage_results_analyzed ON triage_results(analyzed_at DESC);
	`
)
