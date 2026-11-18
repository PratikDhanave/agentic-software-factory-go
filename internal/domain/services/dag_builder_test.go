package services

import (
	"testing"

	"github.com/PratikDhanave/agentic-software-factory-go/internal/domain/entities"
)

type MockLogger struct{}

func (m *MockLogger) Info(msg string, fields ...interface{})            {}
func (m *MockLogger) Error(msg string, err error, fields ...interface{}) {}
func (m *MockLogger) Debug(msg string, fields ...interface{})           {}

func TestDAGBuilder_BuildSimpleLinearWorkflow(t *testing.T) {
	logger := &MockLogger{}
	builder := NewDAGBuilder(logger)

	def := &entities.WorkflowDefinition{
		Name: "test-linear",
		Version: "1.0",
		JobsMap: map[string]*entities.JobDef{
			"job1": {
				Task: "task1",
				DependsOn: []string{},
			},
			"job2": {
				Task: "task2",
				DependsOn: []string{"job1"},
			},
			"job3": {
				Task: "task3",
				DependsOn: []string{"job2"},
			},
		},
	}

	dag, err := builder.Build(def)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if dag.Size() != 3 {
		t.Errorf("Expected 3 nodes, got %d", dag.Size())
	}

	levels := dag.GetExecutionLevels()
	if len(levels) != 3 {
		t.Errorf("Expected 3 levels, got %d", len(levels))
	}

	// Check level structure
	if len(levels[0]) != 1 || levels[0][0] != "job1" {
		t.Errorf("Level 0 should be [job1], got %v", levels[0])
	}
	if len(levels[1]) != 1 || levels[1][0] != "job2" {
		t.Errorf("Level 1 should be [job2], got %v", levels[1])
	}
	if len(levels[2]) != 1 || levels[2][0] != "job3" {
		t.Errorf("Level 2 should be [job3], got %v", levels[2])
	}
}

func TestDAGBuilder_BuildParallelWorkflow(t *testing.T) {
	logger := &MockLogger{}
	builder := NewDAGBuilder(logger)

	def := &entities.WorkflowDefinition{
		Name: "test-parallel",
		Version: "1.0",
		JobsMap: map[string]*entities.JobDef{
			"job1": {
				Task: "task1",
				DependsOn: []string{},
			},
			"job2": {
				Task: "task2",
				DependsOn: []string{},
			},
			"job3": {
				Task: "task3",
				DependsOn: []string{"job1", "job2"},
			},
		},
	}

	dag, err := builder.Build(def)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	levels := dag.GetExecutionLevels()
	if len(levels) != 2 {
		t.Errorf("Expected 2 levels, got %d", len(levels))
	}

	// Level 0 should have 2 jobs (job1, job2)
	if len(levels[0]) != 2 {
		t.Errorf("Level 0 should have 2 jobs, got %d", len(levels[0]))
	}

	// Level 1 should have job3
	if len(levels[1]) != 1 || levels[1][0] != "job3" {
		t.Errorf("Level 1 should be [job3], got %v", levels[1])
	}
}

func TestDAGBuilder_BuildComplexWorkflow(t *testing.T) {
	logger := &MockLogger{}
	builder := NewDAGBuilder(logger)

	// GitHub Issue Triage workflow
	def := &entities.WorkflowDefinition{
		Name: "issue-triage",
		Version: "1.0",
		JobsMap: map[string]*entities.JobDef{
			"extract": {
				Task: "extract-issue",
				DependsOn: []string{},
			},
			"triage": {
				Task: "triage",
				DependsOn: []string{"extract"},
			},
			"analyze": {
				Task: "analyze",
				DependsOn: []string{"extract"},
			},
			"classify": {
				Task: "classify",
				DependsOn: []string{"extract"},
			},
			"assign": {
				Task: "assign",
				DependsOn: []string{"analyze", "classify", "triage"},
			},
			"notify": {
				Task: "notify",
				DependsOn: []string{"assign"},
			},
		},
	}

	dag, err := builder.Build(def)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	levels := dag.GetExecutionLevels()
	if len(levels) != 4 {
		t.Errorf("Expected 4 levels, got %d", len(levels))
	}

	// Level 0: extract (1 job)
	if len(levels[0]) != 1 {
		t.Errorf("Level 0 should have 1 job, got %d", len(levels[0]))
	}

	// Level 1: triage, analyze, classify (3 jobs in parallel)
	if len(levels[1]) != 3 {
		t.Errorf("Level 1 should have 3 jobs, got %d", len(levels[1]))
	}

	// Level 2: assign (1 job)
	if len(levels[2]) != 1 {
		t.Errorf("Level 2 should have 1 job, got %d", len(levels[2]))
	}

	// Level 3: notify (1 job)
	if len(levels[3]) != 1 {
		t.Errorf("Level 3 should have 1 job, got %d", len(levels[3]))
	}
}

func TestDAGBuilder_DetectSimpleCycle(t *testing.T) {
	logger := &MockLogger{}
	builder := NewDAGBuilder(logger)

	def := &entities.WorkflowDefinition{
		Name: "cyclic",
		Version: "1.0",
		JobsMap: map[string]*entities.JobDef{
			"job1": {
				Task: "task1",
				DependsOn: []string{"job2"},
			},
			"job2": {
				Task: "task2",
				DependsOn: []string{"job1"},
			},
		},
	}

	_, err := builder.Build(def)
	if err == nil {
		t.Error("Expected cycle detection error, got nil")
	}
}

func TestDAGBuilder_DetectComplexCycle(t *testing.T) {
	logger := &MockLogger{}
	builder := NewDAGBuilder(logger)

	def := &entities.WorkflowDefinition{
		Name: "complex-cycle",
		Version: "1.0",
		JobsMap: map[string]*entities.JobDef{
			"job1": {
				Task: "task1",
				DependsOn: []string{"job4"},
			},
			"job2": {
				Task: "task2",
				DependsOn: []string{"job1"},
			},
			"job3": {
				Task: "task3",
				DependsOn: []string{"job2"},
			},
			"job4": {
				Task: "task4",
				DependsOn: []string{"job3"},
			},
		},
	}

	_, err := builder.Build(def)
	if err == nil {
		t.Error("Expected cycle detection error for complex cycle (job1->job2->job3->job4->job1), got nil")
	}
}

func TestDAGBuilder_InvalidDependency(t *testing.T) {
	logger := &MockLogger{}
	builder := NewDAGBuilder(logger)

	def := &entities.WorkflowDefinition{
		Name: "invalid",
		Version: "1.0",
		JobsMap: map[string]*entities.JobDef{
			"job1": {
				Task: "task1",
				DependsOn: []string{"nonexistent"},
			},
		},
	}

	_, err := builder.Build(def)
	if err == nil {
		t.Error("Expected validation error for invalid dependency")
	}
}

func TestDAGBuilder_NilDefinition(t *testing.T) {
	logger := &MockLogger{}
	builder := NewDAGBuilder(logger)

	_, err := builder.Build(nil)
	if err == nil {
		t.Error("Expected error for nil definition")
	}
}

func TestDAGBuilder_MultipleIndependentPaths(t *testing.T) {
	logger := &MockLogger{}
	builder := NewDAGBuilder(logger)

	// Two independent paths that merge at the end
	def := &entities.WorkflowDefinition{
		Name: "independent-paths",
		Version: "1.0",
		JobsMap: map[string]*entities.JobDef{
			"job_a1": {
				Task: "task_a1",
				DependsOn: []string{},
			},
			"job_a2": {
				Task: "task_a2",
				DependsOn: []string{"job_a1"},
			},
			"job_b1": {
				Task: "task_b1",
				DependsOn: []string{},
			},
			"job_b2": {
				Task: "task_b2",
				DependsOn: []string{"job_b1"},
			},
			"merge": {
				Task: "merge",
				DependsOn: []string{"job_a2", "job_b2"},
			},
		},
	}

	dag, err := builder.Build(def)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	levels := dag.GetExecutionLevels()
	if len(levels) != 3 {
		t.Errorf("Expected 3 levels, got %d", len(levels))
	}

	// Level 0: 2 jobs (job_a1, job_b1)
	if len(levels[0]) != 2 {
		t.Errorf("Level 0 should have 2 jobs, got %d", len(levels[0]))
	}

	// Level 1: 2 jobs (job_a2, job_b2)
	if len(levels[1]) != 2 {
		t.Errorf("Level 1 should have 2 jobs, got %d", len(levels[1]))
	}

	// Level 2: 1 job (merge)
	if len(levels[2]) != 1 {
		t.Errorf("Level 2 should have 1 job, got %d", len(levels[2]))
	}
}

func TestDirectedAcyclicGraph_Operations(t *testing.T) {
	dag := NewDirectedAcyclicGraph()

	// Add nodes
	dag.AddNode("job1", "task1", []string{})
	dag.AddNode("job2", "task2", []string{"job1"})

	if dag.Size() != 2 {
		t.Errorf("Expected size 2, got %d", dag.Size())
	}

	deps := dag.GetDependencies("job2")
	if len(deps) != 1 || deps[0] != "job1" {
		t.Errorf("Expected job2 to depend on job1, got %v", deps)
	}

	dependents := dag.GetDependents("job1")
	if len(dependents) != 0 {
		t.Errorf("Expected job1 to have no dependents initially, got %v", dependents)
	}

	// Add edge
	dag.AddEdge("job1", "job2")
	dependents = dag.GetDependents("job1")
	if len(dependents) != 1 || dependents[0] != "job2" {
		t.Errorf("Expected job1 to have job2 as dependent, got %v", dependents)
	}
}

func TestDAGBuilder_RealWorldPRReviewWorkflow(t *testing.T) {
	logger := &MockLogger{}
	builder := NewDAGBuilder(logger)

	def := &entities.WorkflowDefinition{
		Name: "pr-review",
		Version: "1.0",
		JobsMap: map[string]*entities.JobDef{
			"lint": {
				Task: "lint",
				DependsOn: []string{},
			},
			"security": {
				Task: "security",
				DependsOn: []string{},
			},
			"test": {
				Task: "test",
				DependsOn: []string{},
			},
			"perf": {
				Task: "perf",
				DependsOn: []string{"test"},
			},
			"coverage": {
				Task: "coverage",
				DependsOn: []string{"test"},
			},
			"summarize": {
				Task: "summarize",
				DependsOn: []string{"lint", "security", "perf", "coverage"},
			},
			"post": {
				Task: "post",
				DependsOn: []string{"summarize"},
			},
		},
	}

	dag, err := builder.Build(def)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	levels := dag.GetExecutionLevels()

	// Level 0: lint, security, test (3 parallel)
	if len(levels[0]) != 3 {
		t.Errorf("Level 0 should have 3 jobs, got %d", len(levels[0]))
	}

	// Level 1: perf, coverage (2 parallel, depend on test)
	if len(levels[1]) != 2 {
		t.Errorf("Level 1 should have 2 jobs, got %d", len(levels[1]))
	}

	// Level 2: summarize (1 job)
	if len(levels[2]) != 1 {
		t.Errorf("Level 2 should have 1 job, got %d", len(levels[2]))
	}

	// Level 3: post (1 job)
	if len(levels[3]) != 1 {
		t.Errorf("Level 3 should have 1 job, got %d", len(levels[3]))
	}
}
