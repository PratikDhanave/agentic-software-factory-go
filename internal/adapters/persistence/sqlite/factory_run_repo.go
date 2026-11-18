package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// FactoryRunRecord stores the full state and artifacts of an end-to-end Software Factory run.
type FactoryRunRecord struct {
	ID           string                 `json:"id"`
	Title        string                 `json:"title"`
	Description  string                 `json:"description"`
	SpecID       string                 `json:"spec_id"`
	SessionID    string                 `json:"session_id"`
	PhaseMode    string                 `json:"phase_mode"`
	Stage        string                 `json:"stage"`
	Status       string                 `json:"status"`
	WorkspaceDir string                 `json:"workspace_dir"`
	PromotedDir  string                 `json:"promoted_dir,omitempty"`
	Payload      map[string]interface{} `json:"payload"`
	Error        string                 `json:"error,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
	CompletedAt  *time.Time             `json:"completed_at,omitempty"`
}

type FactoryRunRepository struct {
	db *sql.DB
}

func NewFactoryRunRepository(store *Store) *FactoryRunRepository {
	return &FactoryRunRepository{db: store.DB()}
}

func (r *FactoryRunRepository) Save(ctx context.Context, rec *FactoryRunRecord) error {
	now := time.Now()
	if rec.CreatedAt.IsZero() {
		rec.CreatedAt = now
	}
	rec.UpdatedAt = now
	payloadBytes, err := json.Marshal(rec.Payload)
	if err != nil {
		return err
	}

	_, err = r.db.ExecContext(ctx, `
		INSERT INTO factory_runs (
			id, title, description, spec_id, session_id, phase_mode,
			stage, status, workspace_dir, promoted_dir, payload_json,
			error, created_at, updated_at, completed_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			spec_id = excluded.spec_id,
			session_id = excluded.session_id,
			phase_mode = excluded.phase_mode,
			stage = excluded.stage,
			status = excluded.status,
			workspace_dir = excluded.workspace_dir,
			promoted_dir = excluded.promoted_dir,
			payload_json = excluded.payload_json,
			error = excluded.error,
			updated_at = excluded.updated_at,
			completed_at = excluded.completed_at
	`,
		rec.ID, rec.Title, rec.Description, rec.SpecID, rec.SessionID, rec.PhaseMode,
		rec.Stage, rec.Status, rec.WorkspaceDir, rec.PromotedDir, string(payloadBytes),
		rec.Error, rec.CreatedAt, rec.UpdatedAt, rec.CompletedAt,
	)
	return err
}

func (r *FactoryRunRepository) Get(ctx context.Context, id string) (*FactoryRunRecord, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, title, description, spec_id, session_id, phase_mode,
		       stage, status, workspace_dir, promoted_dir, payload_json,
		       error, created_at, updated_at, completed_at
		FROM factory_runs WHERE id = ?
	`, id)
	return scanFactoryRunRow(row)
}

func (r *FactoryRunRepository) GetBySessionID(ctx context.Context, sessionID string) (*FactoryRunRecord, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, title, description, spec_id, session_id, phase_mode,
		       stage, status, workspace_dir, promoted_dir, payload_json,
		       error, created_at, updated_at, completed_at
		FROM factory_runs WHERE session_id = ? ORDER BY created_at DESC LIMIT 1
	`, sessionID)
	return scanFactoryRunRow(row)
}

func (r *FactoryRunRepository) List(ctx context.Context, limit int) ([]*FactoryRunRecord, error) {
	if limit <= 0 {
		limit = 30
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, title, description, spec_id, session_id, phase_mode,
		       stage, status, workspace_dir, promoted_dir, payload_json,
		       error, created_at, updated_at, completed_at
		FROM factory_runs ORDER BY created_at DESC LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*FactoryRunRecord
	for rows.Next() {
		var rec FactoryRunRecord
		var payloadJSON string
		var completed sql.NullTime
		if err := rows.Scan(
			&rec.ID, &rec.Title, &rec.Description, &rec.SpecID, &rec.SessionID, &rec.PhaseMode,
			&rec.Stage, &rec.Status, &rec.WorkspaceDir, &rec.PromotedDir, &payloadJSON,
			&rec.Error, &rec.CreatedAt, &rec.UpdatedAt, &completed,
		); err != nil {
			return nil, err
		}
		if completed.Valid {
			t := completed.Time
			rec.CompletedAt = &t
		}
		rec.Payload = map[string]interface{}{}
		_ = json.Unmarshal([]byte(payloadJSON), &rec.Payload)
		out = append(out, &rec)
	}
	return out, rows.Err()
}

func scanFactoryRunRow(row *sql.Row) (*FactoryRunRecord, error) {
	var rec FactoryRunRecord
	var payloadJSON string
	var completed sql.NullTime
	if err := row.Scan(
		&rec.ID, &rec.Title, &rec.Description, &rec.SpecID, &rec.SessionID, &rec.PhaseMode,
		&rec.Stage, &rec.Status, &rec.WorkspaceDir, &rec.PromotedDir, &payloadJSON,
		&rec.Error, &rec.CreatedAt, &rec.UpdatedAt, &completed,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("factory run not found")
		}
		return nil, err
	}
	if completed.Valid {
		t := completed.Time
		rec.CompletedAt = &t
	}
	rec.Payload = map[string]interface{}{}
	_ = json.Unmarshal([]byte(payloadJSON), &rec.Payload)
	return &rec, nil
}
