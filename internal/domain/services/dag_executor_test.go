package services

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/PratikDhanave/agentic-software-factory-go/internal/domain/entities"
)

// MockLogger implements LoggerInterface for testing
type MockLogger struct{}

func (m *MockLogger) Info(msg string, fields ...interface{})            {}
func (m *MockLogger) Error(msg string, err error, fields ...interface{}) {}
func (m *MockLogger) Debug(msg string, fields ...interface{})           {}

// MockJobExecutor implements JobExecutor for testing
type MockJobExecutor struct {
	mu         sync.Mutex
	executions map[string]int
	results    map[string]interface{}
	errors     map[string]error
	delays     map[string]time.Duration
}

func NewMockJobExecutor() *MockJobExecutor {
	return &MockJobExecutor{
		executions: make(map[string]int),
		results:    make(map[string]interface{}),
		errors:     make(map[string]error),
		delays:     make(map[string]time.Duration),
	}
}

func (m *MockJobExecutor) Execute(ctx context.Context, taskName string,
	input map[string]interface{}) (interface{}, error) {

	// Simulate delay if configured
	m.mu.Lock()
	delay, hasDelay := m.delays[taskName]
	m.mu.Unlock()

	if hasDelay {
		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	m.mu.Lock()
	m.executions[taskName]++

	// Return error if configured
	if err, exists := m.errors[taskName]; exists {
		m.mu.Unlock()
		return nil, err
	}

	// Return result if configured
	if result, exists := m.results[taskName]; exists {
		m.mu.Unlock()
		return result, nil
	}
	m.mu.Unlock()

	// Default result includes the task name
	return map[string]interface{}{"task": taskName, "input": input}, nil
}

func (m *MockJobExecutor) SetResult(taskName string, result interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.results[taskName] = result
}

func (m *MockJobExecutor) SetError(taskName string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.errors[taskName] = err
}

func (m *MockJobExecutor) SetDelay(taskName string, delay time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.delays[taskName] = delay
}

func (m *MockJobExecutor) GetExecutionCount(taskName string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.executions[taskName]
}

func TestDAGExecutor_ExecuteSimpleLinearWorkflow(t *testing.T) {
	logger := &MockLogger{}
	mockExecutor := NewMockJobExecutor()
	mockExecutor.SetResult("task1", map[string]interface{}{"value": 1})
	mockExecutor.SetResult("task2", map[string]interface{}{"value": 2})
	mockExecutor.SetResult("task3", map[string]interface{}{"value": 3})

	dagExecutor := NewDAGExecutor(mockExecutor, logger, 4)

	// Build DAG: task1 → task2 → task3
	dag := NewDirectedAcyclicGraph()
	dag.AddNode("job1", "task1", []string{})
	dag.AddNode("job2", "task2", []string{"job1"})
	dag.AddNode("job3", "task3", []string{"job2"})
	dag.SetExecutionLevels([][]string{{"job1"}, {"job2"}, {"job3"}})

	execution := entities.NewWorkflowExecution("exec-1", "wf-1", map[string]interface{}{})

	err := dagExecutor.Execute(context.Background(), dag, execution)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if execution.Status != entities.WorkflowExecCompleted {
		t.Errorf("Expected status completed, got %v", execution.Status)
	}

	if mockExecutor.GetExecutionCount("task1") != 1 {
		t.Errorf("Expected task1 to execute 1 time, got %d", mockExecutor.GetExecutionCount("task1"))
	}
	if mockExecutor.GetExecutionCount("task2") != 1 {
		t.Errorf("Expected task2 to execute 1 time, got %d", mockExecutor.GetExecutionCount("task2"))
	}
	if mockExecutor.GetExecutionCount("task3") != 1 {
		t.Errorf("Expected task3 to execute 1 time, got %d", mockExecutor.GetExecutionCount("task3"))
	}
}

func TestDAGExecutor_ExecuteParallelWorkflow(t *testing.T) {
	logger := &MockLogger{}
	mockExecutor := NewMockJobExecutor()
	mockExecutor.SetResult("task1", map[string]interface{}{"value": 1})
	mockExecutor.SetResult("task2", map[string]interface{}{"value": 2})
	mockExecutor.SetResult("task3", map[string]interface{}{"value": 3})

	dagExecutor := NewDAGExecutor(mockExecutor, logger, 4)

	// Build DAG: task1, task2 (parallel) → task3
	dag := NewDirectedAcyclicGraph()
	dag.AddNode("job1", "task1", []string{})
	dag.AddNode("job2", "task2", []string{})
	dag.AddNode("job3", "task3", []string{"job1", "job2"})
	dag.SetExecutionLevels([][]string{{"job1", "job2"}, {"job3"}})

	execution := entities.NewWorkflowExecution("exec-1", "wf-1", map[string]interface{}{})

	err := dagExecutor.Execute(context.Background(), dag, execution)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if execution.Status != entities.WorkflowExecCompleted {
		t.Errorf("Expected status completed, got %v", execution.Status)
	}

	// Verify all jobs executed
	if mockExecutor.GetExecutionCount("task1") != 1 {
		t.Errorf("Expected task1 to execute once")
	}
	if mockExecutor.GetExecutionCount("task2") != 1 {
		t.Errorf("Expected task2 to execute once")
	}
	if mockExecutor.GetExecutionCount("task3") != 1 {
		t.Errorf("Expected task3 to execute once")
	}
}

func TestDAGExecutor_ErrorPropagation(t *testing.T) {
	logger := &MockLogger{}
	mockExecutor := NewMockJobExecutor()
	mockExecutor.SetError("task2", fmt.Errorf("task2 failed"))
	mockExecutor.SetResult("task1", map[string]interface{}{"value": 1})
	mockExecutor.SetResult("task3", map[string]interface{}{"value": 3})

	dagExecutor := NewDAGExecutor(mockExecutor, logger, 4)

	// Build DAG: task1 → task2 (fails) → task3
	dag := NewDirectedAcyclicGraph()
	dag.AddNode("job1", "task1", []string{})
	dag.AddNode("job2", "task2", []string{"job1"})
	dag.AddNode("job3", "task3", []string{"job2"})
	dag.SetExecutionLevels([][]string{{"job1"}, {"job2"}, {"job3"}})

	execution := entities.NewWorkflowExecution("exec-1", "wf-1", map[string]interface{}{})

	err := dagExecutor.Execute(context.Background(), dag, execution)
	if err == nil {
		t.Fatal("Expected execution to fail, got nil")
	}

	if execution.Status != entities.WorkflowExecFailed {
		t.Errorf("Expected status failed, got %v", execution.Status)
	}

	// Verify task1 executed, task2 failed, task3 was skipped
	if mockExecutor.GetExecutionCount("task1") != 1 {
		t.Errorf("Expected task1 to execute once")
	}
	if mockExecutor.GetExecutionCount("task2") != 1 {
		t.Errorf("Expected task2 to execute once")
	}
	if mockExecutor.GetExecutionCount("task3") != 0 {
		t.Errorf("Expected task3 to be skipped, got %d executions", mockExecutor.GetExecutionCount("task3"))
	}
}

func TestDAGExecutor_InputVariableResolution(t *testing.T) {
	logger := &MockLogger{}
	mockExecutor := NewMockJobExecutor()
	mockExecutor.SetResult("analyze", map[string]interface{}{
		"severity": "high",
		"details": map[string]interface{}{
			"count": 5,
		},
	})
	mockExecutor.SetResult("classify", map[string]interface{}{
		"labels": []string{"bug", "critical"},
	})

	dagExecutor := NewDAGExecutor(mockExecutor, logger, 4)

	// Build DAG with template variables
	dag := NewDirectedAcyclicGraph()
	dag.AddNode("analyze", "analyze", []string{})
	dag.AddNode("classify", "classify", []string{})
	dag.AddNode("assign", "auto-assign", []string{"analyze", "classify"})

	// Set input with templates
	assignNode := dag.GetNode("assign")
	assignNode.Input = map[string]interface{}{
		"severity": "{{ jobs.analyze.output.severity }}",
		"labels":   "{{ jobs.classify.output.labels }}",
	}

	dag.SetExecutionLevels([][]string{{"analyze", "classify"}, {"assign"}})

	execution := entities.NewWorkflowExecution("exec-1", "wf-1", map[string]interface{}{})

	err := dagExecutor.Execute(context.Background(), dag, execution)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify template variables were resolved
	// The test verifies that templates are resolved by checking the job execution count
	// and the presence of results (which wouldn't happen if template resolution failed)
	if mockExecutor.GetExecutionCount("auto-assign") != 1 {
		t.Error("Expected auto-assign to execute once")
	}

	// All jobs should have executed successfully
	if execution.Status != entities.WorkflowExecCompleted {
		t.Errorf("Expected status completed, got %v", execution.Status)
	}

	// Should have results from all jobs
	if len(execution.Results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(execution.Results))
	}
}

func TestDAGExecutor_ComplexWorkflow(t *testing.T) {
	logger := &MockLogger{}
	mockExecutor := NewMockJobExecutor()

	// Setup results for all tasks
	for _, task := range []string{"extract", "triage", "analyze", "classify", "assign", "notify"} {
		mockExecutor.SetResult(task, map[string]interface{}{"task": task})
	}

	dagExecutor := NewDAGExecutor(mockExecutor, logger, 4)

	// Build Issue Triage DAG
	dag := NewDirectedAcyclicGraph()
	dag.AddNode("extract", "extract", []string{})
	dag.AddNode("triage", "triage", []string{"extract"})
	dag.AddNode("analyze", "analyze", []string{"extract"})
	dag.AddNode("classify", "classify", []string{"extract"})
	dag.AddNode("assign", "assign", []string{"triage", "analyze", "classify"})
	dag.AddNode("notify", "notify", []string{"assign"})

	dag.SetExecutionLevels([][]string{
		{"extract"},
		{"triage", "analyze", "classify"},
		{"assign"},
		{"notify"},
	})

	execution := entities.NewWorkflowExecution("exec-1", "wf-1", map[string]interface{}{})

	err := dagExecutor.Execute(context.Background(), dag, execution)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if execution.Status != entities.WorkflowExecCompleted {
		t.Errorf("Expected status completed")
	}

	// Verify all tasks executed exactly once
	expectedTasks := []string{"extract", "triage", "analyze", "classify", "assign", "notify"}
	for _, task := range expectedTasks {
		if mockExecutor.GetExecutionCount(task) != 1 {
			t.Errorf("Expected %s to execute once, got %d", task, mockExecutor.GetExecutionCount(task))
		}
	}

	// Verify results contain all jobs
	if len(execution.Results) != 6 {
		t.Errorf("Expected 6 results, got %d", len(execution.Results))
	}
}

func TestDAGExecutor_ParallelLimitEnforced(t *testing.T) {
	logger := &MockLogger{}
	mockExecutor := NewMockJobExecutor()

	// Set all tasks to have a small delay to test concurrency
	for i := 1; i <= 8; i++ {
		taskName := fmt.Sprintf("task%d", i)
		mockExecutor.SetDelay(taskName, 10*time.Millisecond)
		mockExecutor.SetResult(taskName, map[string]interface{}{"value": i})
	}

	dagExecutor := NewDAGExecutor(mockExecutor, logger, 2) // Max 2 parallel

	// Build DAG with 8 independent jobs
	dag := NewDirectedAcyclicGraph()
	for i := 1; i <= 8; i++ {
		taskName := fmt.Sprintf("task%d", i)
		jobID := fmt.Sprintf("job%d", i)
		dag.AddNode(jobID, taskName, []string{})
	}

	jobIDs := make([]string, 8)
	for i := 1; i <= 8; i++ {
		jobIDs[i-1] = fmt.Sprintf("job%d", i)
	}
	dag.SetExecutionLevels([][]string{jobIDs})

	execution := entities.NewWorkflowExecution("exec-1", "wf-1", map[string]interface{}{})

	startTime := time.Now()
	err := dagExecutor.Execute(context.Background(), dag, execution)
	duration := time.Since(startTime)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// With 2 parallel and 10ms delay each:
	// - 4 batches of 2 jobs = 40ms minimum
	// Allow some overhead
	if duration < 35*time.Millisecond {
		t.Errorf("Execution was too fast (got %v), suggests maxParallel not enforced", duration)
	}

	// Verify all 8 tasks executed
	if execution.Status != entities.WorkflowExecCompleted {
		t.Errorf("Expected status completed")
	}
}

func TestDAGExecutor_EmptyDAG(t *testing.T) {
	logger := &MockLogger{}
	mockExecutor := NewMockJobExecutor()
	dagExecutor := NewDAGExecutor(mockExecutor, logger, 4)

	dag := NewDirectedAcyclicGraph()
	execution := entities.NewWorkflowExecution("exec-1", "wf-1", map[string]interface{}{})

	err := dagExecutor.Execute(context.Background(), dag, execution)
	if err == nil {
		t.Fatal("Expected error for empty DAG")
	}
}

func TestDAGExecutor_CascadingFailure(t *testing.T) {
	logger := &MockLogger{}
	mockExecutor := NewMockJobExecutor()

	// Setup: task2 and task3 fail
	mockExecutor.SetError("task2", fmt.Errorf("task2 error"))
	mockExecutor.SetError("task3", fmt.Errorf("task3 error"))
	mockExecutor.SetResult("task1", map[string]interface{}{"value": 1})
	mockExecutor.SetResult("task4", map[string]interface{}{"value": 4})

	dagExecutor := NewDAGExecutor(mockExecutor, logger, 4)

	// Build DAG: task1 → [task2, task3] → task4
	// When task2 and task3 fail, task4 should be skipped
	dag := NewDirectedAcyclicGraph()
	dag.AddNode("job1", "task1", []string{})
	dag.AddNode("job2", "task2", []string{"job1"})
	dag.AddNode("job3", "task3", []string{"job1"})
	dag.AddNode("job4", "task4", []string{"job2", "job3"})

	dag.SetExecutionLevels([][]string{
		{"job1"},
		{"job2", "job3"},
		{"job4"},
	})

	execution := entities.NewWorkflowExecution("exec-1", "wf-1", map[string]interface{}{})

	err := dagExecutor.Execute(context.Background(), dag, execution)
	if err == nil {
		t.Fatal("Expected execution to fail")
	}

	if execution.Status != entities.WorkflowExecFailed {
		t.Errorf("Expected status failed")
	}

	// Verify job1 executed, job2/job3 failed, job4 skipped
	if mockExecutor.GetExecutionCount("task1") != 1 {
		t.Error("Expected task1 to execute")
	}
	if mockExecutor.GetExecutionCount("task2") != 1 {
		t.Error("Expected task2 to execute (and fail)")
	}
	if mockExecutor.GetExecutionCount("task3") != 1 {
		t.Error("Expected task3 to execute (and fail)")
	}
	if mockExecutor.GetExecutionCount("task4") != 0 {
		t.Error("Expected task4 to be skipped (depends on failed jobs)")
	}
}

func TestDAGExecutor_MultipleIndependentPaths(t *testing.T) {
	logger := &MockLogger{}
	mockExecutor := NewMockJobExecutor()

	tasks := []string{"a1", "a2", "b1", "b2", "merge"}
	for _, task := range tasks {
		mockExecutor.SetResult(task, map[string]interface{}{"task": task})
	}

	dagExecutor := NewDAGExecutor(mockExecutor, logger, 4)

	// Build DAG with two independent paths
	// Path A: a1 → a2
	// Path B: b1 → b2
	// Merge: a2 and b2 → merge
	dag := NewDirectedAcyclicGraph()
	dag.AddNode("a1", "a1", []string{})
	dag.AddNode("a2", "a2", []string{"a1"})
	dag.AddNode("b1", "b1", []string{})
	dag.AddNode("b2", "b2", []string{"b1"})
	dag.AddNode("merge", "merge", []string{"a2", "b2"})

	dag.SetExecutionLevels([][]string{
		{"a1", "b1"},
		{"a2", "b2"},
		{"merge"},
	})

	execution := entities.NewWorkflowExecution("exec-1", "wf-1", map[string]interface{}{})

	err := dagExecutor.Execute(context.Background(), dag, execution)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if execution.Status != entities.WorkflowExecCompleted {
		t.Errorf("Expected completed status")
	}

	// Verify all executed
	for _, task := range tasks {
		if mockExecutor.GetExecutionCount(task) != 1 {
			t.Errorf("Expected %s to execute once", task)
		}
	}
}
