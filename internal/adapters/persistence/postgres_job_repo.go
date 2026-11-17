package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/PratikDhanave/agentic-software-factory-go/internal/domain/entities"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/domain/repositories"
)

type PostgresJobRepository struct {
	db *sql.DB
}

func NewPostgresJobRepository(db *sql.DB) repositories.JobRepository {
	return &PostgresJobRepository{db: db}
}

func (r *PostgresJobRepository) Save(ctx context.Context, job *entities.Job) error {
	if job == nil {
		return fmt.Errorf("job cannot be nil")
	}

	inputJSON, _ := json.Marshal(job.Input)
	outputJSON, _ := json.Marshal(job.Output)

	query := `
		INSERT INTO jobs (id, type, status, input, output, error, created_at, retries, max_retries, priority)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (id) DO UPDATE
		SET status = $3, output = $5, error = $6, retries = $8
	`

	_, err := r.db.ExecContext(ctx,
		query,
		job.ID, job.Type, job.Status, inputJSON, outputJSON, job.Error,
		job.CreatedAt, job.Retries, job.MaxRetries, job.Priority,
	)

	return err
}

func (r *PostgresJobRepository) GetByID(ctx context.Context, id string) (*entities.Job, error) {
	if id == "" {
		return nil, fmt.Errorf("id cannot be empty")
	}

	job := &entities.Job{}
	var inputJSON, outputJSON sql.NullString
	var startedAt, completedAt sql.NullTime

	query := `
		SELECT id, type, status, input, output, error, created_at, started_at, completed_at, retries, max_retries, priority
		FROM jobs
		WHERE id = $1
	`

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&job.ID, &job.Type, &job.Status, &inputJSON, &outputJSON, &job.Error,
		&job.CreatedAt, &startedAt, &completedAt, &job.Retries, &job.MaxRetries, &job.Priority,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("job not found: %s", id)
	}
	if err != nil {
		return nil, err
	}

	if inputJSON.Valid {
		json.Unmarshal([]byte(inputJSON.String), &job.Input)
	}
	if outputJSON.Valid {
		json.Unmarshal([]byte(outputJSON.String), &job.Output)
	}
	if startedAt.Valid {
		job.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		job.CompletedAt = &completedAt.Time
	}

	return job, nil
}

func (r *PostgresJobRepository) GetByStatus(ctx context.Context, status string) ([]*entities.Job, error) {
	query := `
		SELECT id, type, status, input, output, error, created_at, started_at, completed_at, retries, max_retries, priority
		FROM jobs
		WHERE status = $1
		ORDER BY created_at DESC
		LIMIT 100
	`

	rows, err := r.db.QueryContext(ctx, query, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []*entities.Job
	for rows.Next() {
		job := &entities.Job{}
		var inputJSON, outputJSON sql.NullString
		var startedAt, completedAt sql.NullTime

		if err := rows.Scan(
			&job.ID, &job.Type, &job.Status, &inputJSON, &outputJSON, &job.Error,
			&job.CreatedAt, &startedAt, &completedAt, &job.Retries, &job.MaxRetries, &job.Priority,
		); err != nil {
			return nil, err
		}

		if inputJSON.Valid {
			json.Unmarshal([]byte(inputJSON.String), &job.Input)
		}
		if outputJSON.Valid {
			json.Unmarshal([]byte(outputJSON.String), &job.Output)
		}
		if startedAt.Valid {
			job.StartedAt = &startedAt.Time
		}
		if completedAt.Valid {
			job.CompletedAt = &completedAt.Time
		}

		jobs = append(jobs, job)
	}

	return jobs, rows.Err()
}

func (r *PostgresJobRepository) Update(ctx context.Context, job *entities.Job) error {
	if job == nil {
		return fmt.Errorf("job cannot be nil")
	}

	outputJSON, _ := json.Marshal(job.Output)

	query := `
		UPDATE jobs
		SET status = $1, output = $2, error = $3, started_at = $4, completed_at = $5, retries = $6
		WHERE id = $7
	`

	result, err := r.db.ExecContext(ctx,
		query,
		job.Status, outputJSON, job.Error, job.StartedAt, job.CompletedAt, job.Retries, job.ID,
	)

	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("job not found: %s", job.ID)
	}

	return nil
}

func (r *PostgresJobRepository) List(ctx context.Context, limit int, offset int) ([]*entities.Job, error) {
	if limit <= 0 {
		limit = 10
	}
	if offset < 0 {
		offset = 0
	}

	query := `
		SELECT id, type, status, input, output, error, created_at, started_at, completed_at, retries, max_retries, priority
		FROM jobs
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []*entities.Job
	for rows.Next() {
		job := &entities.Job{}
		var inputJSON, outputJSON sql.NullString
		var startedAt, completedAt sql.NullTime

		if err := rows.Scan(
			&job.ID, &job.Type, &job.Status, &inputJSON, &outputJSON, &job.Error,
			&job.CreatedAt, &startedAt, &completedAt, &job.Retries, &job.MaxRetries, &job.Priority,
		); err != nil {
			return nil, err
		}

		if inputJSON.Valid {
			json.Unmarshal([]byte(inputJSON.String), &job.Input)
		}
		if outputJSON.Valid {
			json.Unmarshal([]byte(outputJSON.String), &job.Output)
		}
		if startedAt.Valid {
			job.StartedAt = &startedAt.Time
		}
		if completedAt.Valid {
			job.CompletedAt = &completedAt.Time
		}

		jobs = append(jobs, job)
	}

	return jobs, rows.Err()
}
