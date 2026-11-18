package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// PersistenceManager handles durable storage for the factory
type PersistenceManager struct {
	db *sql.DB
}

// TaskRecord represents a task persisted to storage
type TaskRecord struct {
	ID          string     `json:"id"`
	Type        string     `json:"type"`
	SpecID      string     `json:"spec_id"`
	Priority    int        `json:"priority"`
	Status      string     `json:"status"`
	AssignedTo  string     `json:"assigned_to"`
	CreatedAt   time.Time  `json:"created_at"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	Result      []byte     `json:"result"`
	Error       string     `json:"error"`
	Retries     int        `json:"retries"`
}

// WorkerRecord represents a worker persisted to storage
type WorkerRecord struct {
	ID           string    `json:"id"`
	Role         string    `json:"role"`
	Capabilities string    `json:"capabilities"`
	Status       string    `json:"status"`
	LastActive   time.Time `json:"last_active"`
}

// NewPersistenceManager creates a new persistence manager
func NewPersistenceManager(dataDir string) (*PersistenceManager, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("creating data directory: %w", err)
	}

	dbPath := filepath.Join(dataDir, "factory.db")
	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	return NewPersistenceManagerWithDB(db)
}

// NewPersistenceManagerWithDB wraps an existing SQLite *sql.DB handle so
// FactoryOrchestrator and Clean Architecture repositories share one store.
func NewPersistenceManagerWithDB(db *sql.DB) (*PersistenceManager, error) {
	pm := &PersistenceManager{db: db}
	if err := pm.initSchema(); err != nil {
		return nil, fmt.Errorf("initializing schema: %w", err)
	}
	return pm, nil
}

// initSchema creates the necessary tables
func (pm *PersistenceManager) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS tasks (
		id TEXT PRIMARY KEY,
		type TEXT NOT NULL,
		spec_id TEXT NOT NULL,
		priority INTEGER NOT NULL,
		status TEXT NOT NULL,
		assigned_to TEXT,
		created_at DATETIME NOT NULL,
		started_at DATETIME,
		completed_at DATETIME,
		result BLOB,
		error TEXT,
		retries INTEGER NOT NULL DEFAULT 0
	);

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

	CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);
	CREATE INDEX IF NOT EXISTS idx_tasks_spec_id ON tasks(spec_id);
	`

	_, err := pm.db.Exec(schema)
	return err
}

// SaveTask saves or updates a task
func (pm *PersistenceManager) SaveTask(ctx context.Context, task *TaskRecord) error {
	resultJSON, _ := json.Marshal(task.Result)

	_, err := pm.db.ExecContext(ctx, `
		INSERT OR REPLACE INTO tasks
		(id, type, spec_id, priority, status, assigned_to, created_at, started_at, completed_at, result, error, retries)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, task.ID, task.Type, task.SpecID, task.Priority, task.Status, task.AssignedTo,
		task.CreatedAt, task.StartedAt, task.CompletedAt, resultJSON, task.Error, task.Retries)

	return err
}

// GetTask retrieves a task by ID
func (pm *PersistenceManager) GetTask(ctx context.Context, id string) (*TaskRecord, error) {
	row := pm.db.QueryRowContext(ctx, `
		SELECT id, type, spec_id, priority, status, assigned_to, created_at, started_at, completed_at, result, error, retries
		FROM tasks WHERE id = ?
	`, id)

	var task TaskRecord
	var resultJSON []byte
	var startedAt, completedAt sql.NullTime

	err := row.Scan(&task.ID, &task.Type, &task.SpecID, &task.Priority, &task.Status,
		&task.AssignedTo, &task.CreatedAt, &startedAt, &completedAt, &resultJSON, &task.Error, &task.Retries)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if startedAt.Valid {
		task.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		task.CompletedAt = &completedAt.Time
	}

	json.Unmarshal(resultJSON, &task.Result)
	return &task, nil
}

// GetTasksByStatus retrieves tasks by status
func (pm *PersistenceManager) GetTasksByStatus(ctx context.Context, status string) ([]*TaskRecord, error) {
	rows, err := pm.db.QueryContext(ctx, `
		SELECT id, type, spec_id, priority, status, assigned_to, created_at, started_at, completed_at, result, error, retries
		FROM tasks WHERE status = ?
	`, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []*TaskRecord
	for rows.Next() {
		var task TaskRecord
		var resultJSON []byte
		var startedAt, completedAt sql.NullTime

		err := rows.Scan(&task.ID, &task.Type, &task.SpecID, &task.Priority, &task.Status,
			&task.AssignedTo, &task.CreatedAt, &startedAt, &completedAt, &resultJSON, &task.Error, &task.Retries)
		if err != nil {
			return nil, err
		}

		if startedAt.Valid {
			task.StartedAt = &startedAt.Time
		}
		if completedAt.Valid {
			task.CompletedAt = &completedAt.Time
		}
		json.Unmarshal(resultJSON, &task.Result)
		tasks = append(tasks, &task)
	}
	return tasks, rows.Err()
}

// SaveWorker saves or updates a worker
func (pm *PersistenceManager) SaveWorker(ctx context.Context, worker *WorkerRecord) error {
	capsJSON, _ := json.Marshal(worker.Capabilities)

	_, err := pm.db.ExecContext(ctx, `
		INSERT OR REPLACE INTO workers (id, role, capabilities, status, last_active)
		VALUES (?, ?, ?, ?, ?)
	`, worker.ID, worker.Role, string(capsJSON), worker.Status, worker.LastActive)

	return err
}

// GetWorker retrieves a worker by ID
func (pm *PersistenceManager) GetWorker(ctx context.Context, id string) (*WorkerRecord, error) {
	row := pm.db.QueryRowContext(ctx, `
		SELECT id, role, capabilities, status, last_active FROM workers WHERE id = ?
	`, id)

	var worker WorkerRecord
	var capsJSON string
	err := row.Scan(&worker.ID, &worker.Role, &capsJSON, &worker.Status, &worker.LastActive)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	json.Unmarshal([]byte(capsJSON), &worker.Capabilities)
	return &worker, nil
}

// GetAllWorkers retrieves all workers
func (pm *PersistenceManager) GetAllWorkers(ctx context.Context) ([]*WorkerRecord, error) {
	rows, err := pm.db.QueryContext(ctx, `SELECT id, role, capabilities, status, last_active FROM workers`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var workers []*WorkerRecord
	for rows.Next() {
		var worker WorkerRecord
		var capsJSON string
		err := rows.Scan(&worker.ID, &worker.Role, &capsJSON, &worker.Status, &worker.LastActive)
		if err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(capsJSON), &worker.Capabilities)
		workers = append(workers, &worker)
	}
	return workers, rows.Err()
}

// SaveState saves a key-value state
func (pm *PersistenceManager) SaveState(ctx context.Context, key, value string) error {
	_, err := pm.db.ExecContext(ctx, `
		INSERT OR REPLACE INTO factory_state (key, value, updated_at)
		VALUES (?, ?, ?)
	`, key, value, time.Now())
	return err
}

// GetState retrieves a state value
func (pm *PersistenceManager) GetState(ctx context.Context, key string) (string, error) {
	var value string
	err := pm.db.QueryRowContext(ctx, `SELECT value FROM factory_state WHERE key = ?`, key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return value, err
}

// Close closes the database connection
func (pm *PersistenceManager) Close() error {
	return pm.db.Close()
}
