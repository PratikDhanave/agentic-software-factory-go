package factory

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"path/filepath"
	"sort"
	"sync"
	"time"

	asfContext "github.com/PratikDhanave/agentic-software-factory-go/pkg/context"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/harness"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/persistence"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/spec"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/state"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/verify"
	"github.com/google/uuid"
)

// Worker represents an agent worker in the factory
type Worker struct {
	ID           string
	Role         string
	Capabilities []string
	Status       state.State // see WorkerMachine
	CurrentTask  *Task
	LastActive   time.Time
	stop         chan struct{} // closed to retire this worker's goroutine
}

// Task represents a unit of work in the factory
type Task struct {
	ID          string
	Type        string // "generate", "verify", "review", "refactor"
	SpecID      string
	Priority    int
	Status      state.State // see TaskMachine
	AssignedTo  string
	CreatedAt   time.Time
	StartedAt   *time.Time
	CompletedAt *time.Time
	Result      interface{}
	Error       string
	Retries     int
	Parameters  map[string]interface{}
}

// setTaskState applies a task state transition, logging any violation of
// TaskMachine. Violations are logged rather than rejected so an undeclared
// transition surfaces loudly in development without halting the factory.
func setTaskState(task *Task, to state.State) {
	if err := TaskMachine.Validate(task.Status, to); err != nil {
		log.Printf("factory: task %s: %v", task.ID, err)
	}
	task.Status = to
}

// setWorkerState applies a worker state transition under the same policy.
func setWorkerState(worker *Worker, to state.State) {
	if err := WorkerMachine.Validate(worker.Status, to); err != nil {
		log.Printf("factory: worker %s: %v", worker.ID, err)
	}
	worker.Status = to
}

// FactoryOrchestrator manages the distributed agent factory
type FactoryOrchestrator struct {
	workers       map[string]*Worker
	taskQueue     chan *Task
	results       map[string]*Task
	harness       *harness.CompleteHarness
	contextEngine *asfContext.ContextEngine
	specEngine    *spec.SpecEngine
	verifier      *verify.VerificationEngine
	persistence   *persistence.PersistenceManager
	mu            sync.RWMutex
	workerCount   int
	maxWorkers    int
	shutdown      chan struct{}
	wg            sync.WaitGroup
	started       bool
	runCtx        context.Context
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
		workers:       make(map[string]*Worker),
		taskQueue:     make(chan *Task, 100),
		results:       make(map[string]*Task),
		harness:       harness.NewCompleteHarness(workingDir),
		contextEngine: asfContext.NewContextEngine(workingDir),
		specEngine:    spec.NewSpecEngine(filepath.Join(workingDir, "specs")),
		verifier:      verify.NewVerificationEngine(workingDir),
		persistence:   persist,
		maxWorkers:    maxWorkers,
		shutdown:      make(chan struct{}),
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
	fo.addWorkerLocked(id, role, capabilities)
}

// addWorkerLocked adds a worker; the caller must hold fo.mu.
func (fo *FactoryOrchestrator) addWorkerLocked(id, role string, capabilities []string) {
	worker := &Worker{
		ID:           id,
		Role:         role,
		Capabilities: capabilities,
		Status:       WorkerMachine.Initial(),
		LastActive:   time.Now(),
		stop:         make(chan struct{}),
	}
	fo.workers[id] = worker
	fo.workerCount++

	// Workers added after Start (scale-up) need their own goroutine;
	// workers created before Start are launched by Start itself.
	if fo.started {
		fo.wg.Add(1)
		go fo.workerLoop(fo.runCtx, worker)
	}

	// Persist worker
	ctx := context.Background()
	capsJSON, _ := json.Marshal(capabilities)
	_ = fo.persistence.SaveWorker(ctx, &persistence.WorkerRecord{
		ID:           id,
		Role:         role,
		Capabilities: string(capsJSON),
		Status:       string(WorkerMachine.Initial()),
		LastActive:   time.Now(),
	})
}

// Start begins the factory orchestration
func (fo *FactoryOrchestrator) Start(ctx context.Context) error {
	fo.mu.Lock()
	fo.started = true
	fo.runCtx = ctx
	workers := make([]*Worker, 0, len(fo.workers))
	for _, worker := range fo.workers {
		workers = append(workers, worker)
	}
	fo.mu.Unlock()

	// Start worker goroutines
	for _, worker := range workers {
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
	task.Status = TaskMachine.Initial()
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
			Status:    string(task.Status),
			CreatedAt: task.CreatedAt,
			Retries:   task.Retries,
		})
		return nil
	default:
		return fmt.Errorf("task queue full")
	}
}

// GetTaskResult retrieves a snapshot of a task. It returns a copy taken
// under the lock: workers mutate task state concurrently, so handing out
// the live pointer would race with readers.
func (fo *FactoryOrchestrator) GetTaskResult(taskID string) (*Task, bool) {
	fo.mu.RLock()
	defer fo.mu.RUnlock()

	task, ok := fo.results[taskID]
	if !ok {
		return nil, false
	}
	snapshot := *task
	return &snapshot, true
}

// taskFromRecord rebuilds an in-memory Task from its persisted record.
func taskFromRecord(t *persistence.TaskRecord) *Task {
	return &Task{
		ID:          t.ID,
		Type:        t.Type,
		SpecID:      t.SpecID,
		Priority:    t.Priority,
		Status:      state.State(t.Status),
		AssignedTo:  t.AssignedTo,
		CreatedAt:   t.CreatedAt,
		StartedAt:   t.StartedAt,
		CompletedAt: t.CompletedAt,
		Error:       t.Error,
		Retries:     t.Retries,
	}
}

// restoreTasks restores persisted tasks from storage
func (fo *FactoryOrchestrator) restoreTasks() error {
	ctx := context.Background()

	for _, status := range TaskMachine.States() {
		records, err := fo.persistence.GetTasksByStatus(ctx, string(status))
		if err != nil {
			return err
		}

		for _, t := range records {
			task := taskFromRecord(t)

			// Crash recovery: a task persisted as "running" was interrupted
			// mid-execution; re-queue it (explicit running -> pending transition).
			if task.Status == TaskRunning {
				setTaskState(task, TaskPending)
			}

			fo.results[task.ID] = task
			if task.Status == TaskPending {
				fo.taskQueue <- task
			}
		}
	}

	// Workers are not restored: they are ephemeral runtime entities that the
	// constructor recreates fresh each boot (a persisted "busy" worker is
	// meaningless after a restart — its interrupted task is re-queued above).
	// SaveWorker persistence remains as an audit trail.
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
		case <-worker.stop:
			return
		case task := <-fo.taskQueue:
			fo.executeTask(ctx, worker, task)
		}
	}
}

// executeTask executes a single task
func (fo *FactoryOrchestrator) executeTask(ctx context.Context, worker *Worker, task *Task) {
	fo.mu.Lock()
	setWorkerState(worker, WorkerBusy)
	worker.CurrentTask = task
	now := time.Now()
	task.StartedAt = &now
	setTaskState(task, TaskRunning)
	task.AssignedTo = worker.ID
	fo.mu.Unlock()

	// Persist task state
	_ = fo.persistence.SaveTask(ctx, &persistence.TaskRecord{
		ID:         task.ID,
		Type:       task.Type,
		SpecID:     task.SpecID,
		Priority:   task.Priority,
		Status:     string(task.Status),
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
	setWorkerState(worker, WorkerIdle)
	worker.CurrentTask = nil
	worker.LastActive = time.Now()

	completedAt := time.Now()
	task.CompletedAt = &completedAt

	if err != nil {
		task.Error = err.Error()
		task.Retries++
		if task.Retries < 3 {
			// Retryable failure: re-queue (running -> pending).
			setTaskState(task, TaskPending)
			select {
			case fo.taskQueue <- task:
			default:
				// Queue full; retries exhausted for all practical purposes.
				setTaskState(task, TaskFailed)
			}
		} else {
			// Retries exhausted (running -> failed, terminal).
			setTaskState(task, TaskFailed)
		}
	} else {
		setTaskState(task, TaskCompleted)
	}

	fo.results[task.ID] = task
	fo.mu.Unlock()

	// Persist final state
	resultJSON, _ := json.Marshal(task.Result)
	capsJSON, _ := json.Marshal(worker.Capabilities)
	_ = fo.persistence.SaveTask(ctx, &persistence.TaskRecord{
		ID:          task.ID,
		Type:        task.Type,
		SpecID:      task.SpecID,
		Priority:    task.Priority,
		Status:      string(task.Status),
		AssignedTo:  task.AssignedTo,
		CreatedAt:   task.CreatedAt,
		StartedAt:   task.StartedAt,
		CompletedAt: task.CompletedAt,
		Result:      resultJSON,
		Error:       task.Error,
		Retries:     task.Retries,
	})
	_ = fo.persistence.SaveWorker(ctx, &persistence.WorkerRecord{
		ID:           worker.ID,
		Role:         worker.Role,
		Capabilities: string(capsJSON),
		Status:       string(worker.Status),
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
		if w.Status == WorkerBusy {
			busy++
		}
	}

	// Scale up if all workers busy and under max
	if busy == fo.workerCount && fo.workerCount < fo.maxWorkers {
		fo.addWorkerLocked(fmt.Sprintf("worker-%d", fo.workerCount), "agent", []string{"generate", "verify"})
	}

	// Scale down if too many idle workers: retire one idle worker, stopping
	// its goroutine so map and goroutines cannot drift apart.
	idle := fo.workerCount - busy
	if idle > 2 && fo.workerCount > 2 {
		for id, w := range fo.workers {
			if w.Status == WorkerIdle && w.CurrentTask == nil {
				close(w.stop)
				delete(fo.workers, id)
				fo.workerCount--
				break
			}
		}
	}
}

// WorkerStatus is a point-in-time snapshot of one worker, safe to serialize.
type WorkerStatus struct {
	ID       string      `json:"id"`
	Status   state.State `json:"status"`
	TaskType string      `json:"task_type,omitempty"`
	TaskSpec string      `json:"task_spec,omitempty"`
}

// GetFactoryStatus returns the current factory status
func (fo *FactoryOrchestrator) GetFactoryStatus() map[string]interface{} {
	fo.mu.RLock()
	defer fo.mu.RUnlock()

	busy := 0
	idle := 0
	workers := make([]WorkerStatus, 0, len(fo.workers))
	for _, w := range fo.workers {
		if w.Status == WorkerBusy {
			busy++
		} else {
			idle++
		}
		ws := WorkerStatus{ID: w.ID, Status: w.Status}
		if w.CurrentTask != nil {
			ws.TaskType = w.CurrentTask.Type
			ws.TaskSpec = w.CurrentTask.SpecID
		}
		workers = append(workers, ws)
	}
	sort.Slice(workers, func(i, j int) bool { return workers[i].ID < workers[j].ID })

	pending := len(fo.taskQueue)
	completed := 0
	failed := 0
	for _, t := range fo.results {
		if t.Status == TaskCompleted {
			completed++
		} else if t.Status == TaskFailed {
			failed++
		}
	}

	return map[string]interface{}{
		"workers_total":   fo.workerCount,
		"workers_busy":    busy,
		"workers_idle":    idle,
		"workers":         workers,
		"tasks_pending":   pending,
		"tasks_completed": completed,
		"tasks_failed":    failed,
		"max_workers":     fo.maxWorkers,
	}
}
