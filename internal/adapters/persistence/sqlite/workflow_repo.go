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

// WorkflowRepository implements repositories.WorkflowRepository backed by SQLite.
type WorkflowRepository struct {
	db *sql.DB
}

var _ repositories.WorkflowRepository = (*WorkflowRepository)(nil)

func NewWorkflowRepository(store *Store) *WorkflowRepository {
	return &WorkflowRepository{db: store.DB()}
}

func (r *WorkflowRepository) SaveDefinition(ctx context.Context, def *entities.WorkflowDefinition) error {
	jobsJSON, err := json.Marshal(def.JobsMap)
	if err != nil {
		return err
	}
	now := time.Now()
	if def.CreatedAt.IsZero() {
		def.CreatedAt = now
	}
	def.UpdatedAt = now

	_, err = r.db.ExecContext(ctx, `
		INSERT INTO workflow_definitions (id, name, version, description, jobs_json, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name,
			version = excluded.version,
			description = excluded.description,
			jobs_json = excluded.jobs_json,
			status = excluded.status,
			updated_at = excluded.updated_at
	`, def.ID, def.Name, def.Version, def.Description, string(jobsJSON), string(def.Status), def.CreatedAt, def.UpdatedAt)
	return err
}

func (r *WorkflowRepository) GetDefinition(ctx context.Context, id string) (*entities.WorkflowDefinition, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, name, version, description, jobs_json, status, created_at, updated_at
		FROM workflow_definitions WHERE id = ?
	`, id)
	return scanDefinition(row)
}

func (r *WorkflowRepository) GetDefinitionByName(ctx context.Context, name string) (*entities.WorkflowDefinition, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, name, version, description, jobs_json, status, created_at, updated_at
		FROM workflow_definitions WHERE name = ? ORDER BY created_at DESC LIMIT 1
	`, name)
	return scanDefinition(row)
}

func (r *WorkflowRepository) ListDefinitions(ctx context.Context, limit int, offset int) ([]*entities.WorkflowDefinition, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, name, version, description, jobs_json, status, created_at, updated_at
		FROM workflow_definitions ORDER BY created_at DESC LIMIT ? OFFSET ?
	`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*entities.WorkflowDefinition
	for rows.Next() {
		var def entities.WorkflowDefinition
		var jobsJSON, status string
		if err := rows.Scan(&def.ID, &def.Name, &def.Version, &def.Description, &jobsJSON, &status, &def.CreatedAt, &def.UpdatedAt); err != nil {
			return nil, err
		}
		def.Status = entities.WorkflowDefinitionStatus(status)
		def.JobsMap = map[string]*entities.JobDef{}
		_ = json.Unmarshal([]byte(jobsJSON), &def.JobsMap)
		out = append(out, &def)
	}
	return out, rows.Err()
}

func (r *WorkflowRepository) DeleteDefinition(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM workflow_definitions WHERE id = ?`, id)
	return err
}

func (r *WorkflowRepository) SaveExecution(ctx context.Context, exec *entities.WorkflowExecution) error {
	triggerJSON, _ := json.Marshal(exec.TriggerEvent)
	nodesJSON, _ := json.Marshal(exec.NodesMap)
	resultsJSON, _ := json.Marshal(exec.Results)
	if exec.CreatedAt.IsZero() {
		exec.CreatedAt = time.Now()
	}

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO workflow_executions (id, definition_id, trigger_json, nodes_json, results_json, status, started_at, completed_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			nodes_json = excluded.nodes_json,
			results_json = excluded.results_json,
			status = excluded.status,
			started_at = excluded.started_at,
			completed_at = excluded.completed_at
	`, exec.ID, exec.DefinitionID, string(triggerJSON), string(nodesJSON), string(resultsJSON), string(exec.Status), exec.StartedAt, exec.CompletedAt, exec.CreatedAt)
	return err
}

func (r *WorkflowRepository) GetExecution(ctx context.Context, id string) (*entities.WorkflowExecution, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, definition_id, trigger_json, nodes_json, results_json, status, started_at, completed_at, created_at
		FROM workflow_executions WHERE id = ?
	`, id)
	return scanExecution(row)
}

func (r *WorkflowRepository) UpdateExecutionStatus(ctx context.Context, id string, status entities.WorkflowExecutionStatus) error {
	var completedAt interface{}
	if status == entities.WorkflowExecCompleted || status == entities.WorkflowExecFailed {
		completedAt = time.Now()
	}
	_, err := r.db.ExecContext(ctx, `
		UPDATE workflow_executions SET status = ?, completed_at = COALESCE(?, completed_at) WHERE id = ?
	`, string(status), completedAt, id)
	return err
}

func (r *WorkflowRepository) ListExecutions(ctx context.Context, definitionID string, limit int, offset int) ([]*entities.WorkflowExecution, error) {
	if limit <= 0 {
		limit = 50
	}
	var (
		rows *sql.Rows
		err  error
	)
	if definitionID == "" {
		rows, err = r.db.QueryContext(ctx, `
			SELECT id, definition_id, trigger_json, nodes_json, results_json, status, started_at, completed_at, created_at
			FROM workflow_executions ORDER BY created_at DESC LIMIT ? OFFSET ?
		`, limit, offset)
	} else {
		rows, err = r.db.QueryContext(ctx, `
			SELECT id, definition_id, trigger_json, nodes_json, results_json, status, started_at, completed_at, created_at
			FROM workflow_executions WHERE definition_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?
		`, definitionID, limit, offset)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*entities.WorkflowExecution
	for rows.Next() {
		var exec entities.WorkflowExecution
		var triggerJSON, nodesJSON, resultsJSON, status string
		var startedAt, completedAt sql.NullTime
		if err := rows.Scan(&exec.ID, &exec.DefinitionID, &triggerJSON, &nodesJSON, &resultsJSON, &status, &startedAt, &completedAt, &exec.CreatedAt); err != nil {
			return nil, err
		}
		exec.Status = entities.WorkflowExecutionStatus(status)
		if startedAt.Valid {
			t := startedAt.Time
			exec.StartedAt = &t
		}
		if completedAt.Valid {
			t := completedAt.Time
			exec.CompletedAt = &t
		}
		exec.TriggerEvent = map[string]interface{}{}
		exec.NodesMap = map[string]*entities.DAGNode{}
		exec.Results = map[string]interface{}{}
		_ = json.Unmarshal([]byte(triggerJSON), &exec.TriggerEvent)
		_ = json.Unmarshal([]byte(nodesJSON), &exec.NodesMap)
		_ = json.Unmarshal([]byte(resultsJSON), &exec.Results)
		out = append(out, &exec)
	}
	return out, rows.Err()
}

func (r *WorkflowRepository) SaveJobExecution(ctx context.Context, jobExec *entities.DAGNode, executionID string) error {
	depsJSON, _ := json.Marshal(jobExec.DependsOn)
	inputJSON, _ := json.Marshal(jobExec.Input)
	resultJSON, _ := json.Marshal(jobExec.Result)

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO dag_node_runs (execution_id, job_id, task_name, depends_json, input_json, status, result_json, error, started_at, completed_at, duration_ns)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(execution_id, job_id) DO UPDATE SET
			status = excluded.status,
			result_json = excluded.result_json,
			error = excluded.error,
			started_at = excluded.started_at,
			completed_at = excluded.completed_at,
			duration_ns = excluded.duration_ns
	`, executionID, jobExec.JobID, jobExec.TaskName, string(depsJSON), string(inputJSON), string(jobExec.Status), string(resultJSON), jobExec.Error, jobExec.StartedAt, jobExec.CompletedAt, int64(jobExec.Duration))
	return err
}

func (r *WorkflowRepository) GetJobExecution(ctx context.Context, executionID string, jobID string) (*entities.DAGNode, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT job_id, task_name, depends_json, input_json, status, result_json, error, started_at, completed_at, duration_ns
		FROM dag_node_runs WHERE execution_id = ? AND job_id = ?
	`, executionID, jobID)

	var node entities.DAGNode
	var depsJSON, inputJSON, status, resultJSON string
	var errStr sql.NullString
	var startedAt, completedAt sql.NullTime
	var durNS int64

	if err := row.Scan(&node.JobID, &node.TaskName, &depsJSON, &inputJSON, &status, &resultJSON, &errStr, &startedAt, &completedAt, &durNS); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("dag node %s not found", jobID)
		}
		return nil, err
	}
	node.Status = entities.JobExecutionStatus(status)
	node.Duration = time.Duration(durNS)
	if errStr.Valid {
		s := errStr.String
		node.Error = &s
	}
	if startedAt.Valid {
		t := startedAt.Time
		node.StartedAt = &t
	}
	if completedAt.Valid {
		t := completedAt.Time
		node.CompletedAt = &t
	}
	_ = json.Unmarshal([]byte(depsJSON), &node.DependsOn)
	_ = json.Unmarshal([]byte(inputJSON), &node.Input)
	_ = json.Unmarshal([]byte(resultJSON), &node.Result)
	return &node, nil
}

func (r *WorkflowRepository) UpdateJobStatus(ctx context.Context, executionID string, jobID string, status entities.JobExecutionStatus) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE dag_node_runs SET status = ? WHERE execution_id = ? AND job_id = ?
	`, string(status), executionID, jobID)
	return err
}

func scanDefinition(row *sql.Row) (*entities.WorkflowDefinition, error) {
	var def entities.WorkflowDefinition
	var jobsJSON, status string
	if err := row.Scan(&def.ID, &def.Name, &def.Version, &def.Description, &jobsJSON, &status, &def.CreatedAt, &def.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("workflow definition not found")
		}
		return nil, err
	}
	def.Status = entities.WorkflowDefinitionStatus(status)
	def.JobsMap = map[string]*entities.JobDef{}
	_ = json.Unmarshal([]byte(jobsJSON), &def.JobsMap)
	return &def, nil
}

func scanExecution(row *sql.Row) (*entities.WorkflowExecution, error) {
	var exec entities.WorkflowExecution
	var triggerJSON, nodesJSON, resultsJSON, status string
	var startedAt, completedAt sql.NullTime
	if err := row.Scan(&exec.ID, &exec.DefinitionID, &triggerJSON, &nodesJSON, &resultsJSON, &status, &startedAt, &completedAt, &exec.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("workflow execution not found")
		}
		return nil, err
	}
	exec.Status = entities.WorkflowExecutionStatus(status)
	if startedAt.Valid {
		t := startedAt.Time
		exec.StartedAt = &t
	}
	if completedAt.Valid {
		t := completedAt.Time
		exec.CompletedAt = &t
	}
	exec.TriggerEvent = map[string]interface{}{}
	exec.NodesMap = map[string]*entities.DAGNode{}
	exec.Results = map[string]interface{}{}
	_ = json.Unmarshal([]byte(triggerJSON), &exec.TriggerEvent)
	_ = json.Unmarshal([]byte(nodesJSON), &exec.NodesMap)
	_ = json.Unmarshal([]byte(resultsJSON), &exec.Results)
	return &exec, nil
}

// JobRepository implements repositories.JobRepository backed by SQLite.
type JobRepository struct {
	db *sql.DB
}

var _ repositories.JobRepository = (*JobRepository)(nil)

func NewJobRepository(store *Store) *JobRepository {
	return &JobRepository{db: store.DB()}
}

func (r *JobRepository) Save(ctx context.Context, job *entities.Job) error {
	inputJSON, _ := json.Marshal(job.Input)
	outputJSON, _ := json.Marshal(job.Output)
	if job.CreatedAt.IsZero() {
		job.CreatedAt = time.Now()
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO jobs (id, type, status, input_json, output_json, error, created_at, started_at, completed_at, retries, max_retries, priority)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			status = excluded.status,
			output_json = excluded.output_json,
			error = excluded.error,
			started_at = excluded.started_at,
			completed_at = excluded.completed_at,
			retries = excluded.retries
	`, job.ID, job.Type, job.Status, string(inputJSON), string(outputJSON), job.Error, job.CreatedAt, job.StartedAt, job.CompletedAt, job.Retries, job.MaxRetries, job.Priority)
	return err
}

func (r *JobRepository) Update(ctx context.Context, job *entities.Job) error {
	return r.Save(ctx, job)
}

func (r *JobRepository) GetByID(ctx context.Context, id string) (*entities.Job, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, type, status, input_json, output_json, error, created_at, started_at, completed_at, retries, max_retries, priority
		FROM jobs WHERE id = ?
	`, id)
	var j entities.Job
	var inStr, outStr string
	var started, completed sql.NullTime
	if err := row.Scan(&j.ID, &j.Type, &j.Status, &inStr, &outStr, &j.Error, &j.CreatedAt, &started, &completed, &j.Retries, &j.MaxRetries, &j.Priority); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("job %s not found", id)
		}
		return nil, err
	}
	if started.Valid {
		t := started.Time
		j.StartedAt = &t
	}
	if completed.Valid {
		t := completed.Time
		j.CompletedAt = &t
	}
	_ = json.Unmarshal([]byte(inStr), &j.Input)
	_ = json.Unmarshal([]byte(outStr), &j.Output)
	return &j, nil
}

func (r *JobRepository) GetByStatus(ctx context.Context, status string) ([]*entities.Job, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, type, status, input_json, output_json, error, created_at, started_at, completed_at, retries, max_retries, priority
		FROM jobs WHERE status = ? ORDER BY priority DESC, created_at ASC
	`, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanJobs(rows)
}

func (r *JobRepository) List(ctx context.Context, limit int, offset int) ([]*entities.Job, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, type, status, input_json, output_json, error, created_at, started_at, completed_at, retries, max_retries, priority
		FROM jobs ORDER BY created_at DESC LIMIT ? OFFSET ?
	`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanJobs(rows)
}

func scanJobs(rows *sql.Rows) ([]*entities.Job, error) {
	var out []*entities.Job
	for rows.Next() {
		var j entities.Job
		var inStr, outStr string
		var started, completed sql.NullTime
		if err := rows.Scan(&j.ID, &j.Type, &j.Status, &inStr, &outStr, &j.Error, &j.CreatedAt, &started, &completed, &j.Retries, &j.MaxRetries, &j.Priority); err != nil {
			return nil, err
		}
		if started.Valid {
			t := started.Time
			j.StartedAt = &t
		}
		if completed.Valid {
			t := completed.Time
			j.CompletedAt = &t
		}
		_ = json.Unmarshal([]byte(inStr), &j.Input)
		_ = json.Unmarshal([]byte(outStr), &j.Output)
		out = append(out, &j)
	}
	return out, rows.Err()
}
