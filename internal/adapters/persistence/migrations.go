package persistence

const (
	createJobsTable = `
		CREATE TABLE IF NOT EXISTS jobs (
			id TEXT PRIMARY KEY,
			type TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending',
			input JSONB,
			output JSONB,
			error TEXT,
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			started_at TIMESTAMP,
			completed_at TIMESTAMP,
			retries INT DEFAULT 0,
			max_retries INT DEFAULT 3,
			priority INT DEFAULT 0,
			updated_at TIMESTAMP DEFAULT NOW()
		);
		CREATE INDEX IF NOT EXISTS idx_jobs_status ON jobs(status);
		CREATE INDEX IF NOT EXISTS idx_jobs_created_at ON jobs(created_at DESC);
		CREATE INDEX IF NOT EXISTS idx_jobs_type ON jobs(type);
	`

	createWorkflowsTable = `
		CREATE TABLE IF NOT EXISTS workflows (
			id TEXT PRIMARY KEY,
			type TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending',
			input JSONB,
			output JSONB,
			error TEXT,
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			started_at TIMESTAMP,
			ended_at TIMESTAMP,
			score FLOAT8 DEFAULT 0.0,
			job_ids JSONB,
			updated_at TIMESTAMP DEFAULT NOW()
		);
		CREATE INDEX IF NOT EXISTS idx_workflows_status ON workflows(status);
		CREATE INDEX IF NOT EXISTS idx_workflows_created_at ON workflows(created_at DESC);
		CREATE INDEX IF NOT EXISTS idx_workflows_type ON workflows(type);
	`
)

func CreateTablesSQL() []string {
	return []string{
		createJobsTable,
		createWorkflowsTable,
	}
}
