-- Workflow Definitions
CREATE TABLE IF NOT EXISTS workflow_definitions (
    id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    version VARCHAR(50) NOT NULL,
    description TEXT,
    definition JSONB NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'created',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_workflow_definitions_name ON workflow_definitions(name);
CREATE INDEX IF NOT EXISTS idx_workflow_definitions_status ON workflow_definitions(status);

-- Workflow Executions
CREATE TABLE IF NOT EXISTS workflow_executions (
    id UUID PRIMARY KEY,
    workflow_definition_id UUID NOT NULL REFERENCES workflow_definitions(id),
    trigger_event JSONB NOT NULL,
    dag JSONB,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    progress JSONB,
    results JSONB,
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_workflow_executions_workflow_id ON workflow_executions(workflow_definition_id);
CREATE INDEX IF NOT EXISTS idx_workflow_executions_status ON workflow_executions(status);
CREATE INDEX IF NOT EXISTS idx_workflow_executions_created_at ON workflow_executions(created_at);

-- Job Executions
CREATE TABLE IF NOT EXISTS job_executions (
    id UUID PRIMARY KEY,
    workflow_execution_id UUID NOT NULL REFERENCES workflow_executions(id),
    job_id VARCHAR(100) NOT NULL,
    task_name VARCHAR(255) NOT NULL,
    input JSONB NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    result JSONB,
    error TEXT,
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    duration_ms INTEGER,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_job_executions_workflow_execution_id ON job_executions(workflow_execution_id);
CREATE INDEX IF NOT EXISTS idx_job_executions_status ON job_executions(status);
CREATE INDEX IF NOT EXISTS idx_job_executions_job_id ON job_executions(job_id);

-- Job Dependencies
CREATE TABLE IF NOT EXISTS job_dependencies (
    workflow_execution_id UUID NOT NULL REFERENCES workflow_executions(id),
    job_id VARCHAR(100) NOT NULL,
    depends_on_job_id VARCHAR(100) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (workflow_execution_id, job_id, depends_on_job_id)
);

CREATE INDEX IF NOT EXISTS idx_job_dependencies_workflow_execution_id ON job_dependencies(workflow_execution_id);
