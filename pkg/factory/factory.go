package factory

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	asfContext "github.com/PratikDhanave/agentic-software-factory-go/pkg/context"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/harness"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/persistence"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/spec"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/verify"
	"github.com/google/uuid"
)

// Worker represents an agent worker in the factory
type Worker struct {
	ID           string
	Role         string
	Capabilities []string
	Status       string // "idle", "busy", "error"
	CurrentTask  *Task
	LastActive   time.Time
}

// Task represents a unit of work in the factory
type Task struct {
	ID          string
	Type        string // "generate", "verify", "review", "refactor"
	SpecID      string
	Priority    int
	Status      string // "pending", "running", "completed", "failed"
	AssignedTo  string
	CreatedAt   time.Time
	StartedAt   *time.Time
	CompletedAt *time.Time
	Result      interface{}
	Error       string
	Retries     int
}

// FactoryOrchestrator manages the distributed agent factory
type FactoryOrchestrator struct {
	workers         map[string]*Worker
	taskQueue       chan *Task
	results         map[string]*Task
	harness         *harness.CompleteHarness
	contextEngine   *asfContext.ContextEngine
	specEngine      *spec.SpecEngine
	verifier        *verify.VerificationEngine
	persistence     *persistence.PersistenceManager
	mu              sync.RWMutex
	workerCount     int
	maxWorkers      int
	shutdown        chan struct{}
	wg              sync.WaitGroup
}

// NewFactoryOrchestrator creates a new factory orchestrator
func NewFactoryOrchestrator(workingDir string, maxWorkers int) (*FactoryOrchestrator, error) {
	// Initialize persistence
	dataDir := filepath.Join(workingDir, ".asf")
	persist, err := persistence.NewPersistenceManager(dataDir)
	if err != nil {
		return nil, fmt.Errorf("initializing persistence: %w", err)
	}

	fo := &FactoryOrchestrator{
		workers:         make(map[string]*Worker),
		taskQueue:       make(chan *Task, 100),
		results:         make(map[string]*Task),
		harness:         harness.NewCompleteHarness(workingDir),
		contextEngine:   asfContext.NewContextEngine(workingDir),
		specEngine:      spec.NewSpecEngine(filepath.Join(workingDir, "specs")),
		verifier:        verify.NewVerificationEngine(workingDir),
		persistence:     persist,
		maxWorkers:      maxWorkers,
		shutdown:        make(chan struct{}),
	}

	// Initialize workers
	for i := 0; i < maxWorkers; i++ {
		fo.addWorker(fmt.Sprintf("worker-%d", i), "agent", []string{"generate", "verify"})
	}

	// Restore persisted tasks
	if err := fo.restoreTasks(); err != nil {
		return nil, fmt.Errorf("restoring tasks: %w", err)
	}

	return fo, nil
}

// addWorker adds a new worker to the factory
func (fo *FactoryOrchestrator) addWorker(id, role string, capabilities []string) {
	fo.mu.Lock()
	defer fo.mu.Unlock()

	fo.workers[id] = &Worker{
		ID:           id,
		Role:         role,
		Capabilities: capabilities,
		Status:       "idle",
		LastActive:   time.Now(),
	}
	fo.workerCount++

	// Persist worker
	ctx := context.Background()
	capsJSON, _ := json.Marshal(capabilities)
	_ = fo.persistence.SaveWorker(ctx, &persistence.WorkerRecord{
		ID:           id,
		Role:         role,
		Capabilities: string(capsJSON),
		Status:       "idle",
		LastActive:   time.Now(),
	})
}

// Start begins the factory orchestration
func (fo *FactoryOrchestrator) Start(ctx context.Context) error {
	// Start worker goroutines
	for _, worker := range fo.workers {
		fo.wg.Add(1)
		go fo.workerLoop(ctx, worker)
	}

	// Start task dispatcher
	fo.wg.Add(1)
	go fo.dispatchLoop(ctx)

	return nil
}

// Stop gracefully shuts down the factory
func (fo *FactoryOrchestrator) Stop() {
	close(fo.shutdown)
	fo.wg.Wait()
	fo.persistence.Close()
}

// SubmitTask submits a new task to the factory
func (fo *FactoryOrchestrator) SubmitTask(task *Task) error {
	if task.ID == "" {
		task.ID = uuid.New().String()
	}
	task.Status = "pending"
	task.CreatedAt = time.Now()

	select {
	case fo.taskQueue <- task:
		// Persist the task
		ctx := context.Background()
		_ = fo.persistence.SaveTask(ctx, &persistence.TaskRecord{
			ID:        task.ID,
			Type:      task.Type,
			SpecID:    task.SpecID,
			Priority:  task.Priority,
			Status:    task.Status,
			CreatedAt: task.CreatedAt,
			Retries:   task.Retries,
		})
		return nil
	default:
		return fmt.Errorf("task queue full")
	}
}

// GetTaskResult retrieves the result of a task
func (fo *FactoryOrchestrator) GetTaskResult(taskID string) (*Task, bool) {
	fo.mu.RLock()
	defer fo.mu.RUnlock()

	task, ok := fo.results[taskID]
	return task, ok
}

// restoreTasks restores persisted tasks from storage
func (fo *FactoryOrchestrator) restoreTasks() error {
	ctx := context.Background()

	// Restore pending tasks
	pendingTasks, err := fo.persistence.GetTasksByStatus(ctx, "pending")
	if err != nil {
		return err
	}

	for _, t := range pendingTasks {
		task := &Task{
			ID:          t.ID,
			Type:        t.Type,
			SpecID:      t.SpecID,
			Priority:    t.Priority,
			Status:      t.Status,
			AssignedTo:  t.AssignedTo,
			CreatedAt:   t.CreatedAt,
			StartedAt:   t.StartedAt,
			CompletedAt: t.CompletedAt,
			Error:       t.Error,
			Retries:     t.Retries,
		}
		fo.results[task.ID] = task
		fo.taskQueue <- task
	}

	// Restore running tasks (mark as pending for retry)
	runningTasks, err := fo.persistence.GetTasksByStatus(ctx, "running")
	if err != nil {
		return err
	}

	for _, t := range runningTasks {
		task := &Task{
			ID:          t.ID,
			Type:        t.Type,
			SpecID:      t.SpecID,
			Priority:    t.Priority,
			Status:      "pending", // Re-queue running tasks
			AssignedTo:  t.AssignedTo,
			CreatedAt:   t.CreatedAt,
			Retries:     t.Retries,
		}
		fo.results[task.ID] = task
		fo.taskQueue <- task
	}

	// Restore completed/failed tasks
	completedTasks, _ := fo.persistence.GetTasksByStatus(ctx, "completed")
	failedTasks, _ := fo.persistence.GetTasksByStatus(ctx, "failed")

	for _, t := range completedTasks {
		task := &Task{
			ID:          t.ID,
			Type:        t.Type,
			SpecID:      t.SpecID,
			Priority:    t.Priority,
			Status:      t.Status,
			AssignedTo:  t.AssignedTo,
			CreatedAt:   t.CreatedAt,
			StartedAt:   t.StartedAt,
			CompletedAt: t.CompletedAt,
			Error:       t.Error,
			Retries:     t.Retries,
		}
		fo.results[task.ID] = task
	}

	for _, t := range failedTasks {
		task := &Task{
			ID:          t.ID,
			Type:        t.Type,
			SpecID:      t.SpecID,
			Priority:    t.Priority,
			Status:      t.Status,
			AssignedTo:  t.AssignedTo,
			CreatedAt:   t.CreatedAt,
			StartedAt:   t.StartedAt,
			CompletedAt: t.CompletedAt,
			Error:       t.Error,
			Retries:     t.Retries,
		}
		fo.results[task.ID] = task
	}

	// Restore workers
	workers, err := fo.persistence.GetAllWorkers(ctx)
	if err != nil {
		return err
	}

	for _, w := range workers {
		var caps []string
		json.Unmarshal([]byte(w.Capabilities), &caps)
		fo.workers[w.ID] = &Worker{
			ID:           w.ID,
			Role:         w.Role,
			Capabilities: caps,
			Status:       w.Status,
			LastActive:   w.LastActive,
		}
		fo.workerCount++
	}

	return nil
}

// workerLoop runs the main loop for a worker
func (fo *FactoryOrchestrator) workerLoop(ctx context.Context, worker *Worker) {
	defer fo.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case <-fo.shutdown:
			return
		case task := <-fo.taskQueue:
			fo.executeTask(ctx, worker, task)
		}
	}
}

// executeTask executes a single task
func (fo *FactoryOrchestrator) executeTask(ctx context.Context, worker *Worker, task *Task) {
	fo.mu.Lock()
	worker.Status = "busy"
	worker.CurrentTask = task
	now := time.Now()
	task.StartedAt = &now
	task.Status = "running"
	task.AssignedTo = worker.ID
	fo.mu.Unlock()

	// Persist task state
	_ = fo.persistence.SaveTask(ctx, &persistence.TaskRecord{
		ID:         task.ID,
		Type:       task.Type,
		SpecID:     task.SpecID,
		Priority:   task.Priority,
		Status:     task.Status,
		AssignedTo: task.AssignedTo,
		CreatedAt:  task.CreatedAt,
		StartedAt:  task.StartedAt,
		Retries:    task.Retries,
	})

	var err error
	switch task.Type {
	case "generate":
		err = fo.executeGenerate(ctx, task)
	case "verify":
		err = fo.executeVerify(ctx, task)
	case "review":
		err = fo.executeReview(ctx, task)
	case "refactor":
		err = fo.executeRefactor(ctx, task)
	default:
		err = fmt.Errorf("unknown task type: %s", task.Type)
	}

	fo.mu.Lock()
	worker.Status = "idle"
	worker.CurrentTask = nil
	worker.LastActive = time.Now()

	completedAt := time.Now()
	task.CompletedAt = &completedAt

	if err != nil {
		task.Status = "failed"
		task.Error = err.Error()
		task.Retries++
		// Re-queue if retries available
		if task.Retries < 3 {
			task.Status = "pending"
			select {
			case fo.taskQueue <- task:
			default:
				// Queue full, drop task
			}
		}
	} else {
		task.Status = "completed"
	}

	fo.results[task.ID] = task
	fo.mu.Unlock()

	// Persist final state
	resultJSON, _ := json.Marshal(task.Result)
	capsJSON, _ := json.Marshal(worker.Capabilities)
	_ = fo.persistence.SaveTask(ctx, &persistence.TaskRecord{
		ID:           task.ID,
		Type:         task.Type,
		SpecID:       task.SpecID,
		Priority:     task.Priority,
		Status:       task.Status,
		AssignedTo:   task.AssignedTo,
		CreatedAt:    task.CreatedAt,
		StartedAt:    task.StartedAt,
		CompletedAt:  task.CompletedAt,
		Result:       resultJSON,
		Error:        task.Error,
		Retries:      task.Retries,
	})
	_ = fo.persistence.SaveWorker(ctx, &persistence.WorkerRecord{
		ID:           worker.ID,
		Role:         worker.Role,
		Capabilities: string(capsJSON),
		Status:       worker.Status,
		LastActive:   worker.LastActive,
	})
}

// executeGenerate executes a code generation task
func (fo *FactoryOrchestrator) executeGenerate(ctx context.Context, task *Task) error {
	// Load spec
	specs, err := fo.specEngine.LoadAllSpecs()
	if err != nil {
		return err
	}

	s, ok := specs[task.SpecID]
	if !ok {
		return fmt.Errorf("spec not found: %s", task.SpecID)
	}

	// Load context
	contextItems, err := fo.contextEngine.LoadContext(ctx)
	if err != nil {
		return err
	}

	// Generate prompt
	prompt := fo.specEngine.GeneratePrompt(s)
	contextStr := asfContext.FormatForAgent(contextItems)
	fullPrompt := fmt.Sprintf("%s\n\n## Project Context\n%s", prompt, contextStr)

	// In real implementation, call agent here
	// For now, store the prompt as result
	task.Result = map[string]interface{}{
		"prompt":        fullPrompt,
		"spec_id":       task.SpecID,
		"context_items": len(contextItems),
	}

	return nil
}

// executeVerify executes a verification task
func (fo *FactoryOrchestrator) executeVerify(ctx context.Context, task *Task) error {
	result, err := fo.verifier.RunVerification(ctx, "")
	if err != nil {
		return err
	}

	task.Result = result
	if !result.Success {
		return fmt.Errorf("verification failed with %d errors", len(result.Errors))
	}

	return nil
}

// executeReview executes a code review task
func (fo *FactoryOrchestrator) executeReview(ctx context.Context, task *Task) error {
	// Placeholder for code review logic
	task.Result = map[string]interface{}{
		"reviewed": true,
		"comments": []string{"Code looks good", "Consider adding more tests"},
	}
	return nil
}

// executeRefactor executes a refactoring task
func (fo *FactoryOrchestrator) executeRefactor(ctx context.Context, task *Task) error {
	// Placeholder for refactoring logic
	task.Result = map[string]interface{}{
		"refactored": true,
		"changes":    []string{"Extracted interface", "Improved error handling"},
	}
	return nil
}

// dispatchLoop manages task dispatching
func (fo *FactoryOrchestrator) dispatchLoop(ctx context.Context) {
	defer fo.wg.Done()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-fo.shutdown:
			return
		case <-ticker.C:
			fo.rebalanceWorkers()
		}
	}
}

// rebalanceWorkers rebalances worker load
func (fo *FactoryOrchestrator) rebalanceWorkers() {
	fo.mu.Lock()
	defer fo.mu.Unlock()

	// Count busy workers
	busy := 0
	for _, w := range fo.workers {
		if w.Status == "busy" {
			busy++
		}
	}

	// Scale up if all workers busy and under max
	if busy == fo.workerCount && fo.workerCount < fo.maxWorkers {
		fo.addWorker(fmt.Sprintf("worker-%d", fo.workerCount), "agent", []string{"generate", "verify"})
	}

	// Scale down if too many idle workers
	idle := fo.workerCount - busy
	if idle > 2 && fo.workerCount > 2 {
		// Find and remove an idle worker
		for id, w := range fo.workers {
			if w.Status == "idle" && w.CurrentTask == nil {
				delete(fo.workers, id)
				fo.workerCount--
				break
			}
		}
	}
}

// GetFactoryStatus returns the current factory status
func (fo *FactoryOrchestrator) GetFactoryStatus() map[string]interface{} {
	fo.mu.RLock()
	defer fo.mu.RUnlock()

	busy := 0
	idle := 0
	for _, w := range fo.workers {
		if w.Status == "busy" {
			busy++
		} else {
			idle++
		}
	}

	pending := len(fo.taskQueue)
	completed := 0
	failed := 0
	for _, t := range fo.results {
		if t.Status == "completed" {
			completed++
		} else if t.Status == "failed" {
			failed++
		}
	}

	return map[string]interface{}{
		"workers_total":   fo.workerCount,
		"workers_busy":    busy,
		"workers_idle":    idle,
		"tasks_pending":   pending,
		"tasks_completed": completed,
		"tasks_failed":    failed,
		"max_workers":     fo.maxWorkers,
	}
}