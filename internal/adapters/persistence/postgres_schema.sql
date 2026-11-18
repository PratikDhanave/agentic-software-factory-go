-- PostgreSQL Schema for Factory Stack

-- Workflow Definitions
CREATE TABLE IF NOT EXISTS workflow_definitions (
    id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    trigger_type VARCHAR(100),
    agents TEXT[], -- Array of agent roles
    skills TEXT[], -- Array of skills to use
    status VARCHAR(50) DEFAULT 'active',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_by VARCHAR(255)
);

-- Workflow Executions (Sessions)
CREATE TABLE IF NOT EXISTS workflow_executions (
    id UUID PRIMARY KEY,
    definition_id UUID REFERENCES workflow_definitions(id),
    issue_title VARCHAR(255),
    issue_description TEXT,
    trigger_type VARCHAR(100),
    trigger_source VARCHAR(50), -- 'github', 'slack', 'manual'
    trigger_id VARCHAR(255), -- PR#, Issue#, etc
    status VARCHAR(50) DEFAULT 'active', -- active, completed, failed, approved
    approval_required BOOLEAN DEFAULT FALSE,
    approver_id VARCHAR(255),
    approval_timestamp TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_by VARCHAR(255)
);

-- Agent Decisions
CREATE TABLE IF NOT EXISTS agent_decisions (
    id UUID PRIMARY KEY,
    execution_id UUID REFERENCES workflow_executions(id),
    agent_role VARCHAR(100),
    decision TEXT NOT NULL,
    reasoning TEXT,
    confidence DECIMAL(3, 2), -- 0.00 to 1.00
    alternatives TEXT[], -- JSON array
    requires_approval BOOLEAN DEFAULT FALSE,
    approval_status VARCHAR(50), -- pending, approved, rejected
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_by VARCHAR(255)
);

-- Approvals
CREATE TABLE IF NOT EXISTS approvals (
    id UUID PRIMARY KEY,
    execution_id UUID REFERENCES workflow_executions(id),
    decision_id UUID REFERENCES agent_decisions(id),
    agent_role VARCHAR(100),
    status VARCHAR(50) DEFAULT 'pending', -- pending, approved, rejected
    approver_id VARCHAR(255),
    approval_comments TEXT,
    approval_timestamp TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Skill Executions
CREATE TABLE IF NOT EXISTS skill_executions (
    id UUID PRIMARY KEY,
    execution_id UUID REFERENCES workflow_executions(id),
    skill_name VARCHAR(255) NOT NULL,
    input JSONB,
    output JSONB,
    status VARCHAR(50) DEFAULT 'completed', -- pending, completed, failed
    error_message TEXT,
    duration_ms INTEGER,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Agent Performance Metrics (for self-improvement)
CREATE TABLE IF NOT EXISTS agent_metrics (
    id UUID PRIMARY KEY,
    agent_role VARCHAR(100),
    metric_type VARCHAR(100), -- 'accuracy', 'speed', 'approval_rate', etc
    metric_value DECIMAL(5, 3),
    execution_id UUID REFERENCES workflow_executions(id),
    period_start TIMESTAMP,
    period_end TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Learning Patterns (for self-improving agents)
CREATE TABLE IF NOT EXISTS learned_patterns (
    id UUID PRIMARY KEY,
    agent_role VARCHAR(100),
    pattern_type VARCHAR(100), -- 'issue_classification', 'decision_pattern', etc
    pattern_description TEXT,
    pattern_data JSONB,
    confidence DECIMAL(3, 2),
    times_applied INTEGER DEFAULT 1,
    success_rate DECIMAL(3, 2),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- GitHub Integration State
CREATE TABLE IF NOT EXISTS github_state (
    id UUID PRIMARY KEY,
    repository VARCHAR(255),
    pull_request_number INTEGER,
    execution_id UUID REFERENCES workflow_executions(id),
    status VARCHAR(50),
    pr_url VARCHAR(500),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Audit Log
CREATE TABLE IF NOT EXISTS audit_log (
    id UUID PRIMARY KEY,
    action VARCHAR(100),
    actor VARCHAR(255),
    resource_type VARCHAR(50),
    resource_id VARCHAR(255),
    details JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexes
CREATE INDEX idx_workflow_executions_status ON workflow_executions(status);
CREATE INDEX idx_workflow_executions_created_at ON workflow_executions(created_at);
CREATE INDEX idx_agent_decisions_execution_id ON agent_decisions(execution_id);
CREATE INDEX idx_approvals_status ON approvals(status);
CREATE INDEX idx_agent_metrics_agent_role ON agent_metrics(agent_role);
CREATE INDEX idx_learned_patterns_agent_role ON learned_patterns(agent_role);
CREATE INDEX idx_audit_log_created_at ON audit_log(created_at);
