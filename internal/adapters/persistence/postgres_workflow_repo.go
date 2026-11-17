package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/PratikDhanave/agentic-software-factory-go/internal/domain/entities"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/domain/repositories"
)

type PostgresWorkflowRepository struct {
	db *sql.DB
}

func NewPostgresWorkflowRepository(db *sql.DB) repositories.WorkflowRepository {
	return &PostgresWorkflowRepository{db: db}
}

func (r *PostgresWorkflowRepository) Save(ctx context.Context, workflow *entities.Workflow) error {
	if workflow == nil {
		return fmt.Errorf("workflow cannot be nil")
	}

	inputJSON, _ := json.Marshal(workflow.Input)
	outputJSON, _ := json.Marshal(workflow.Output)
	jobIDsJSON, _ := json.Marshal(workflow.JobIDs)

	query := `
		INSERT INTO workflows (id, type, status, input, output, error, created_at, score, job_ids)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (id) DO UPDATE
		SET status = $3, output = $5, error = $6, score = $8, job_ids = $9
	`

	_, err := r.db.ExecContext(ctx,
		query,
		workflow.ID, workflow.Type, workflow.Status, inputJSON, outputJSON, workflow.Error,
		workflow.CreatedAt, workflow.Score, jobIDsJSON,
	)

	return err
}

func (r *PostgresWorkflowRepository) GetByID(ctx context.Context, id string) (*entities.Workflow, error) {
	if id == "" {
		return nil, fmt.Errorf("id cannot be empty")
	}

	workflow := &entities.Workflow{}
	var inputJSON, outputJSON, jobIDsJSON sql.NullString
	var startedAt, endedAt sql.NullTime

	query := `
		SELECT id, type, status, input, output, error, created_at, started_at, ended_at, score, job_ids
		FROM workflows
		WHERE id = $1
	`

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&workflow.ID, &workflow.Type, &workflow.Status, &inputJSON, &outputJSON, &workflow.Error,
		&workflow.CreatedAt, &startedAt, &endedAt, &workflow.Score, &jobIDsJSON,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("workflow not found: %s", id)
	}
	if err != nil {
		return nil, err
	}

	if inputJSON.Valid {
		json.Unmarshal([]byte(inputJSON.String), &workflow.Input)
	}
	if outputJSON.Valid {
		json.Unmarshal([]byte(outputJSON.String), &workflow.Output)
	}
	if startedAt.Valid {
		workflow.StartedAt = &startedAt.Time
	}
	if endedAt.Valid {
		workflow.EndedAt = &endedAt.Time
	}
	if jobIDsJSON.Valid {
		json.Unmarshal([]byte(jobIDsJSON.String), &workflow.JobIDs)
	}

	return workflow, nil
}

func (r *PostgresWorkflowRepository) GetByStatus(ctx context.Context, status string) ([]*entities.Workflow, error) {
	query := `
		SELECT id, type, status, input, output, error, created_at, started_at, ended_at, score, job_ids
		FROM workflows
		WHERE status = $1
		ORDER BY created_at DESC
		LIMIT 100
	`

	rows, err := r.db.QueryContext(ctx, query, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var workflows []*entities.Workflow
	for rows.Next() {
		workflow := &entities.Workflow{}
		var inputJSON, outputJSON, jobIDsJSON sql.NullString
		var startedAt, endedAt sql.NullTime

		if err := rows.Scan(
			&workflow.ID, &workflow.Type, &workflow.Status, &inputJSON, &outputJSON, &workflow.Error,
			&workflow.CreatedAt, &startedAt, &endedAt, &workflow.Score, &jobIDsJSON,
		); err != nil {
			return nil, err
		}

		if inputJSON.Valid {
			json.Unmarshal([]byte(inputJSON.String), &workflow.Input)
		}
		if outputJSON.Valid {
			json.Unmarshal([]byte(outputJSON.String), &workflow.Output)
		}
		if startedAt.Valid {
			workflow.StartedAt = &startedAt.Time
		}
		if endedAt.Valid {
			workflow.EndedAt = &endedAt.Time
		}
		if jobIDsJSON.Valid {
			json.Unmarshal([]byte(jobIDsJSON.String), &workflow.JobIDs)
		}

		workflows = append(workflows, workflow)
	}

	return workflows, rows.Err()
}

func (r *PostgresWorkflowRepository) Update(ctx context.Context, workflow *entities.Workflow) error {
	if workflow == nil {
		return fmt.Errorf("workflow cannot be nil")
	}

	outputJSON, _ := json.Marshal(workflow.Output)
	jobIDsJSON, _ := json.Marshal(workflow.JobIDs)

	query := `
		UPDATE workflows
		SET status = $1, output = $2, error = $3, started_at = $4, ended_at = $5, score = $6, job_ids = $7
		WHERE id = $8
	`

	result, err := r.db.ExecContext(ctx,
		query,
		workflow.Status, outputJSON, workflow.Error, workflow.StartedAt, workflow.EndedAt, workflow.Score, jobIDsJSON, workflow.ID,
	)

	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("workflow not found: %s", workflow.ID)
	}

	return nil
}

func (r *PostgresWorkflowRepository) List(ctx context.Context, limit int, offset int) ([]*entities.Workflow, error) {
	if limit <= 0 {
		limit = 10
	}
	if offset < 0 {
		offset = 0
	}

	query := `
		SELECT id, type, status, input, output, error, created_at, started_at, ended_at, score, job_ids
		FROM workflows
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var workflows []*entities.Workflow
	for rows.Next() {
		workflow := &entities.Workflow{}
		var inputJSON, outputJSON, jobIDsJSON sql.NullString
		var startedAt, endedAt sql.NullTime

		if err := rows.Scan(
			&workflow.ID, &workflow.Type, &workflow.Status, &inputJSON, &outputJSON, &workflow.Error,
			&workflow.CreatedAt, &startedAt, &endedAt, &workflow.Score, &jobIDsJSON,
		); err != nil {
			return nil, err
		}

		if inputJSON.Valid {
			json.Unmarshal([]byte(inputJSON.String), &workflow.Input)
		}
		if outputJSON.Valid {
			json.Unmarshal([]byte(outputJSON.String), &workflow.Output)
		}
		if startedAt.Valid {
			workflow.StartedAt = &startedAt.Time
		}
		if endedAt.Valid {
			workflow.EndedAt = &endedAt.Time
		}
		if jobIDsJSON.Valid {
			json.Unmarshal([]byte(jobIDsJSON.String), &workflow.JobIDs)
		}

		workflows = append(workflows, workflow)
	}

	return workflows, rows.Err()
}
