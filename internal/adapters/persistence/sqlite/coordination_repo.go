package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/PratikDhanave/agentic-software-factory-go/internal/domain/entities"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/domain/repositories"
)

// CoordinationSessionRepository implements repositories.CoordinationSessionRepository in SQLite.
type CoordinationSessionRepository struct {
	db *sql.DB
}

var _ repositories.CoordinationSessionRepository = (*CoordinationSessionRepository)(nil)

func NewCoordinationSessionRepository(store *Store) *CoordinationSessionRepository {
	return &CoordinationSessionRepository{db: store.DB()}
}

func (r *CoordinationSessionRepository) SaveSession(session *entities.CoordinationSession) error {
	triggerJSON, _ := json.Marshal(session.TriggerEvent)
	decisionsJSON, _ := json.Marshal(session.Decisions)
	now := time.Now()
	if session.CreatedAt.IsZero() {
		session.CreatedAt = now
	}
	session.UpdatedAt = now

	_, err := r.db.ExecContext(context.Background(), `
		INSERT INTO coordination_sessions (id, workflow_def_id, trigger_json, decisions_json, status, created_at, updated_at, completed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			workflow_def_id = excluded.workflow_def_id,
			trigger_json = excluded.trigger_json,
			decisions_json = excluded.decisions_json,
			status = excluded.status,
			updated_at = excluded.updated_at,
			completed_at = excluded.completed_at
	`, session.ID, session.WorkflowDefID, string(triggerJSON), string(decisionsJSON), string(session.Status), session.CreatedAt, session.UpdatedAt, session.CompletedAt)
	return err
}

func (r *CoordinationSessionRepository) GetSession(sessionID string) (*entities.CoordinationSession, error) {
	row := r.db.QueryRowContext(context.Background(), `
		SELECT id, workflow_def_id, trigger_json, decisions_json, status, created_at, updated_at, completed_at
		FROM coordination_sessions WHERE id = ?
	`, sessionID)

	var s entities.CoordinationSession
	var triggerJSON, decisionsJSON, status string
	var completedAt sql.NullTime
	if err := row.Scan(&s.ID, &s.WorkflowDefID, &triggerJSON, &decisionsJSON, &status, &s.CreatedAt, &s.UpdatedAt, &completedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("coordination session %s not found", sessionID)
		}
		return nil, err
	}
	s.Status = entities.SessionStatus(status)
	if completedAt.Valid {
		t := completedAt.Time
		s.CompletedAt = &t
	}
	s.TriggerEvent = map[string]interface{}{}
	s.Decisions = map[string]*entities.Decision{}
	_ = json.Unmarshal([]byte(triggerJSON), &s.TriggerEvent)
	_ = json.Unmarshal([]byte(decisionsJSON), &s.Decisions)
	return &s, nil
}

func (r *CoordinationSessionRepository) UpdateSessionStatus(sessionID string, status entities.SessionStatus) error {
	now := time.Now()
	var completedAt interface{}
	if status == entities.SessionCompleted || status == entities.SessionFailed {
		completedAt = now
	}
	_, err := r.db.ExecContext(context.Background(), `
		UPDATE coordination_sessions
		SET status = ?, updated_at = ?, completed_at = COALESCE(?, completed_at)
		WHERE id = ?
	`, string(status), now, completedAt, sessionID)
	return err
}

func (r *CoordinationSessionRepository) UpdateDecision(sessionID string, agentRole string, decision *entities.Decision) error {
	session, err := r.GetSession(sessionID)
	if err != nil {
		return err
	}
	if session.Decisions == nil {
		session.Decisions = make(map[string]*entities.Decision)
	}
	session.Decisions[agentRole] = decision
	return r.SaveSession(session)
}

func (r *CoordinationSessionRepository) ApproveSession(sessionID string, approver string) error {
	session, err := r.GetSession(sessionID)
	if err != nil {
		return err
	}
	for _, d := range session.Decisions {
		if d != nil {
			d.NeedsApproval = false
			d.Approver = approver
		}
	}
	decisionsJSON, _ := json.Marshal(session.Decisions)
	now := time.Now()
	_, err = r.db.ExecContext(context.Background(), `
		UPDATE coordination_sessions
		SET status = ?, approver = ?, decisions_json = ?, updated_at = ?
		WHERE id = ?
	`, string(entities.SessionExecuting), approver, string(decisionsJSON), now, sessionID)
	return err
}

func (r *CoordinationSessionRepository) ListPendingApprovals() ([]*entities.CoordinationSession, error) {
	return r.ListByStatus(entities.SessionWaitingApproval, 100)
}

func (r *CoordinationSessionRepository) ListRecentSessions(limit int) ([]*entities.CoordinationSession, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := r.db.QueryContext(context.Background(), `
		SELECT id, workflow_def_id, trigger_json, decisions_json, status, created_at, updated_at, completed_at
		FROM coordination_sessions ORDER BY created_at DESC LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSessions(rows)
}

func (r *CoordinationSessionRepository) ListByStatus(status entities.SessionStatus, limit int) ([]*entities.CoordinationSession, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := r.db.QueryContext(context.Background(), `
		SELECT id, workflow_def_id, trigger_json, decisions_json, status, created_at, updated_at, completed_at
		FROM coordination_sessions WHERE status = ? ORDER BY created_at DESC LIMIT ?
	`, string(status), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSessions(rows)
}

func (r *CoordinationSessionRepository) DeleteSession(sessionID string) error {
	_, err := r.db.ExecContext(context.Background(), `DELETE FROM coordination_sessions WHERE id = ?`, sessionID)
	return err
}

func scanSessions(rows *sql.Rows) ([]*entities.CoordinationSession, error) {
	var out []*entities.CoordinationSession
	for rows.Next() {
		var s entities.CoordinationSession
		var triggerJSON, decisionsJSON, status string
		var completedAt sql.NullTime
		if err := rows.Scan(&s.ID, &s.WorkflowDefID, &triggerJSON, &decisionsJSON, &status, &s.CreatedAt, &s.UpdatedAt, &completedAt); err != nil {
			return nil, err
		}
		s.Status = entities.SessionStatus(status)
		if completedAt.Valid {
			t := completedAt.Time
			s.CompletedAt = &t
		}
		s.TriggerEvent = map[string]interface{}{}
		s.Decisions = map[string]*entities.Decision{}
		_ = json.Unmarshal([]byte(triggerJSON), &s.TriggerEvent)
		_ = json.Unmarshal([]byte(decisionsJSON), &s.Decisions)
		out = append(out, &s)
	}
	return out, rows.Err()
}
