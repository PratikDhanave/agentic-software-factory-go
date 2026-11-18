package factory

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	asfContext "github.com/PratikDhanave/agentic-software-factory-go/pkg/context"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/eval"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/harness"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/persistence"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/spec"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/state"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/verify"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/workspace"
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
	Type        string // "generate", "verify", "review", "refactor", "evaluate"
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

// contextBudgetFor returns the maximum characters of project context to
// include in a prompt (ASF_CONTEXT_MAX_CHARS overrides). Local models
// typically run small context windows where a 100K+-token prompt means
// minutes of prefill per call; large-context cloud models get far more.
func contextBudgetFor(provider string) int {
	if v := os.Getenv("ASF_CONTEXT_MAX_CHARS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	switch provider {
	case "ollama", "openai-compatible":
		return 24_000 // ~6K tokens — fits typical local context windows
	default:
		return 400_000 // ~100K tokens for large-context cloud models
	}
}

// llmMaxConcurrent returns the cap on concurrent paid LLM calls
// (ASF_LLM_MAX_CONCURRENT, default 2).
func llmMaxConcurrent() int {
	if v := os.Getenv("ASF_LLM_MAX_CONCURRENT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return 2
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
	agent         harness.AgentClient // nil when no LLM provider is configured
	agentModel    string
	agentProvider string
	llmSlots      chan struct{} // caps concurrent paid LLM calls
	contextBudget int           // max chars of project context per prompt
	evaluator     *eval.EvaluationEngine // LLM-as-a-judge; nil without credentials
	evalCriteria  eval.EvaluationCriteria
	mu            sync.RWMutex
	workerCount   int
	maxWorkers    int
	shutdown      chan struct{}
	wg            sync.WaitGroup
	started       bool
	runCtx        context.Context
}

// FactoryConfig configures FactoryOrchestrator with optional shared SQLite DB
// and multi-provider LLM + judge clients.
type FactoryConfig struct {
	WorkingDir    string
	MaxWorkers    int
	SharedDB      *sql.DB
	Agent         harness.AgentClient
	EvalClient    eval.LLMClient
	AgentProvider string
	AgentModel    string
}

// NewFactoryOrchestrator creates a new factory orchestrator
func NewFactoryOrchestrator(workingDir string, maxWorkers int) (*FactoryOrchestrator, error) {
	return NewFactoryOrchestratorWithConfig(FactoryConfig{
		WorkingDir: workingDir,
		MaxWorkers: maxWorkers,
	})
}

// NewFactoryOrchestratorWithConfig creates a factory orchestrator from explicit config.
func NewFactoryOrchestratorWithConfig(cfg FactoryConfig) (*FactoryOrchestrator, error) {
	if cfg.MaxWorkers <= 0 {
		cfg.MaxWorkers = 4
	}

	var (
		persist *persistence.PersistenceManager
		err     error
	)
	if cfg.SharedDB != nil {
		persist, err = persistence.NewPersistenceManagerWithDB(cfg.SharedDB)
	} else {
		dataDir := filepath.Join(cfg.WorkingDir, ".asf")
		persist, err = persistence.NewPersistenceManager(dataDir)
	}
	if err != nil {
		return nil, fmt.Errorf("initializing persistence: %w", err)
	}

	fo := &FactoryOrchestrator{
		workers:       make(map[string]*Worker),
		taskQueue:     make(chan *Task, 100),
		results:       make(map[string]*Task),
		harness:       harness.NewCompleteHarness(cfg.WorkingDir),
		contextEngine: asfContext.NewContextEngine(cfg.WorkingDir),
		specEngine:    spec.NewSpecEngine(filepath.Join(cfg.WorkingDir, "specs")),
		verifier:      verify.NewVerificationEngine(cfg.WorkingDir),
		persistence:   persist,
		llmSlots:      make(chan struct{}, llmMaxConcurrent()),
		evalCriteria:  eval.DefaultEvaluationCriteria(),
		contextBudget: contextBudgetFor(cfg.AgentProvider),
		maxWorkers:    cfg.MaxWorkers,
		shutdown:      make(chan struct{}),
	}

	if cfg.Agent != nil {
		fo.SetAgent(cfg.Agent, cfg.EvalClient, cfg.AgentProvider, cfg.AgentModel)
	} else {
		log.Printf("factory: running in prompt-only mode (attach AgentClient for closed-loop execution)")
	}

	// Initialize workers
	for i := 0; i < cfg.MaxWorkers; i++ {
		fo.addWorker(fmt.Sprintf("worker-%d", i), "agent", []string{"generate", "verify", "review", "refactor", "evaluate"})
	}

	// Restore persisted tasks
	if err := fo.restoreTasks(); err != nil {
		return nil, fmt.Errorf("restoring tasks: %w", err)
	}

	return fo, nil
}

// SetAgent wires a live or deterministic LLM AgentClient and optional LLM-as-a-judge client.
func (fo *FactoryOrchestrator) SetAgent(agent harness.AgentClient, evalClient eval.LLMClient, provider, model string) {
	fo.mu.Lock()
	defer fo.mu.Unlock()
	fo.agent = agent
	fo.agentProvider = provider
	fo.agentModel = model
	fo.contextBudget = contextBudgetFor(provider)
	if evalClient != nil {
		fo.evaluator = eval.NewEvaluationEngine(evalClient)
	}
}

// SpecEngine exposes the specification engine for MCP and HTTP control planes.
func (fo *FactoryOrchestrator) SpecEngine() *spec.SpecEngine {
	return fo.specEngine
}

// Verifier exposes the verification engine for MCP and HTTP control planes.
func (fo *FactoryOrchestrator) Verifier() *verify.VerificationEngine {
	return fo.verifier
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
		// Register immediately so the task is visible to GetTaskResult /
		// RecentTasks while still pending or running (long-running LLM
		// tasks would otherwise 404 until completion).
		fo.mu.Lock()
		fo.results[task.ID] = task
		fo.mu.Unlock()

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

// RecentTasks returns snapshots of the most recently created tasks,
// newest first.
func (fo *FactoryOrchestrator) RecentTasks(limit int) []*Task {
	fo.mu.RLock()
	defer fo.mu.RUnlock()

	tasks := make([]*Task, 0, len(fo.results))
	for _, t := range fo.results {
		snapshot := *t
		tasks = append(tasks, &snapshot)
	}
	sort.Slice(tasks, func(i, j int) bool { return tasks[i].CreatedAt.After(tasks[j].CreatedAt) })
	if limit > 0 && len(tasks) > limit {
		tasks = tasks[:limit]
	}
	return tasks
}

// taskFromRecord rebuilds an in-memory Task from its persisted record,
// including the serialized result (needed e.g. so evaluate tasks can find
// generate outputs produced before a restart).
func taskFromRecord(t *persistence.TaskRecord) *Task {
	var result interface{}
	if len(t.Result) > 0 {
		_ = json.Unmarshal(t.Result, &result)
	}
	return &Task{
		Result:      result,
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
	case "evaluate":
		err = fo.executeEvaluate(ctx, task)
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

// buildTaskParts loads the spec prompt and project context for a task,
// returned separately so the judge can receive them as distinct sections.
func (fo *FactoryOrchestrator) buildTaskParts(ctx context.Context, specID string) (specPrompt, contextStr string, contextCount int, err error) {
	specs, err := fo.specEngine.LoadAllSpecs()
	if err != nil {
		return "", "", 0, err
	}
	s, ok := specs[specID]
	if !ok {
		return "", "", 0, fmt.Errorf("spec not found: %s", specID)
	}

	contextItems, err := fo.contextEngine.LoadContext(ctx)
	if err != nil {
		return "", "", 0, err
	}

	contextStr = asfContext.FormatForAgent(contextItems)
	if fo.contextBudget > 0 && len(contextStr) > fo.contextBudget {
		contextStr = contextStr[:fo.contextBudget] +
			"\n\n[project context truncated to fit the model's context budget]"
	}
	return fo.specEngine.GeneratePrompt(s), contextStr, len(contextItems), nil
}

// withLLMSlot runs one paid LLM call under the concurrency cap and a
// per-call timeout.
func (fo *FactoryOrchestrator) withLLMSlot(ctx context.Context, fn func(context.Context) error) error {
	select {
	case fo.llmSlots <- struct{}{}:
		defer func() { <-fo.llmSlots }()
	case <-ctx.Done():
		return ctx.Err()
	}

	callCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	return fn(callCtx)
}

// promptOnlyResult records the assembled prompt as the task result when no
// LLM is configured, keeping the factory usable end-to-end without a key.
func promptOnlyResult(task *Task, fullPrompt string, contextCount int) {
	task.Result = map[string]interface{}{
		"prompt":        fullPrompt,
		"spec_id":       task.SpecID,
		"context_items": contextCount,
		"note":          "no LLM provider configured; returning prompt only (see pkg/llm: ASF_LLM_PROVIDER + credentials)",
	}
}

// evalMaxAttempts returns how many generate attempts the judge loop may use
// (ASF_EVAL_MAX_ATTEMPTS, default 2: one generation plus one revision).
func evalMaxAttempts() int {
	if v := os.Getenv("ASF_EVAL_MAX_ATTEMPTS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return 2
}

// evaluationView is the serializable shape of a judge verdict.
func evaluationView(ev *eval.EvaluationResult) map[string]interface{} {
	return map[string]interface{}{
		"score":        ev.Score,
		"passed":       ev.Passed,
		"criteria":     ev.Criteria,
		"feedback":     ev.Feedback,
		"improvements": ev.Improvements,
	}
}

// runAgentTask builds the task prompt and, when an LLM is wired, executes it
// against the model (no judge loop — used by review/refactor).
func (fo *FactoryOrchestrator) runAgentTask(ctx context.Context, task *Task, instruction, outputKey string) error {
	specPrompt, contextStr, contextCount, err := fo.buildTaskParts(ctx, task.SpecID)
	if err != nil {
		return err
	}
	fullPrompt := fmt.Sprintf("%s%s\n\n## Project Context\n%s", instruction, specPrompt, contextStr)

	if fo.agent == nil {
		promptOnlyResult(task, fullPrompt, contextCount)
		return nil
	}

	var output string
	err = fo.withLLMSlot(ctx, func(callCtx context.Context) error {
		output, err = fo.agent.Generate(callCtx, fullPrompt)
		return err
	})
	if err != nil {
		return fmt.Errorf("llm %s task: %w", task.Type, err)
	}

	task.Result = map[string]interface{}{
		outputKey:       output,
		"spec_id":       task.SpecID,
		"context_items": contextCount,
		"model":         fo.agentModel,
	}
	return nil
}

// executeGenerate executes a code generation task through a two-tier
// self-healing loop:
//  1. When the model emits Go source artifacts, materialize them into an
//     isolated workspace and run real build/test/vet/constraint verification,
//     feeding compiler/test errors back before spending judge tokens.
//  2. Run LLM-as-a-judge evaluation against the spec and revise if below
//     threshold, up to evalMaxAttempts.
func (fo *FactoryOrchestrator) executeGenerate(ctx context.Context, task *Task) error {
	specs, err := fo.specEngine.LoadAllSpecs()
	if err != nil {
		return err
	}
	specObj, ok := specs[task.SpecID]
	if !ok {
		return fmt.Errorf("spec not found: %s", task.SpecID)
	}

	specPrompt, contextStr, contextCount, err := fo.buildTaskParts(ctx, task.SpecID)
	if err != nil {
		return err
	}
	fullPrompt := fmt.Sprintf("%s\n\n## Project Context\n%s", specPrompt, contextStr)

	if fo.agent == nil {
		promptOnlyResult(task, fullPrompt, contextCount)
		return nil
	}

	maxAttempts := evalMaxAttempts()
	prompt := fullPrompt
	var (
		output       string
		verdict      *eval.EvaluationResult
		verification *verify.VerificationResult
		ws           *workspace.Workspace
		judgeNote    string
		attempts     int
	)
	wsMgr := workspace.NewWorkspaceManager(fo.verifier.WorkingDir)

	for {
		attempts++

		err = fo.withLLMSlot(ctx, func(callCtx context.Context) error {
			output, err = fo.agent.Generate(callCtx, prompt)
			return err
		})
		if err != nil {
			return fmt.Errorf("llm generate task: %w", err)
		}

		// Tier 1: Materialize & verify compilable Go artifacts inside an isolated workspace.
		if artifacts := harness.ParseArtifacts(output); len(artifacts) > 0 {
			if ws == nil {
				ws, err = wsMgr.CreateModuleWorkspace(task.ID, "")
				if err != nil {
					return fmt.Errorf("provisioning isolated workspace: %w", err)
				}
			}
			if _, applyErr := ws.ApplyArtifacts(artifacts); applyErr != nil {
				return fmt.Errorf("applying workspace artifacts: %w", applyErr)
			}
			verification, _ = fo.verifier.WithWorkingDir(ws.Dir).RunVerificationForSpec(ctx, "", specObj)
			if verification != nil && !verification.Success {
				if attempts < maxAttempts {
					prompt = verify.GenerateFixPrompt(verification, fullPrompt)
					continue
				}
				task.Result = map[string]interface{}{
					"output":        output,
					"spec_id":       task.SpecID,
					"attempts":      attempts,
					"verified":      false,
					"workspace_dir": ws.Dir,
					"files_written": ws.FilesWritten,
					"diff":          ws.DiffSummary(),
					"verification":  verification,
				}
				return fmt.Errorf("workspace verification failed after %d attempt(s)", attempts)
			}
		}

		// Tier 2: LLM-as-a-judge rubric evaluation.
		if fo.evaluator == nil {
			break
		}

		var evalErr error
		evalErr = fo.withLLMSlot(ctx, func(callCtx context.Context) error {
			verdict, evalErr = fo.evaluator.EvaluateCode(callCtx, specPrompt, output, contextStr, fo.evalCriteria)
			return evalErr
		})
		if evalErr != nil {
			judgeNote = fmt.Sprintf("evaluation skipped: %v", evalErr)
			log.Printf("factory: task %s: %s", task.ID, judgeNote)
			break
		}

		if verdict.Passed || attempts >= maxAttempts {
			break
		}

		prompt = fmt.Sprintf(
			"%s\n\n## Previous Attempt (scored %.2f, below the %.2f threshold)\n%s\n\n## Judge Feedback — address every point\n%s\n\nImprovements required:\n- %s\n\nProduce a complete, revised artifact.",
			fullPrompt, verdict.Score, fo.evalCriteria.Threshold, output,
			verdict.Feedback, strings.Join(verdict.Improvements, "\n- "),
		)
	}

	result := map[string]interface{}{
		"output":        output,
		"spec_id":       task.SpecID,
		"context_items": contextCount,
		"provider":      fo.agentProvider,
		"model":         fo.agentModel,
		"attempts":      attempts,
	}
	if ws != nil {
		result["workspace_dir"] = ws.Dir
		result["files_written"] = ws.FilesWritten
		result["diff"] = ws.DiffSummary()
		result["verified"] = verification != nil && verification.Success
		result["verification"] = verification
	}
	if verdict != nil {
		result["evaluation"] = evaluationView(verdict)
	}
	if judgeNote != "" {
		result["note"] = judgeNote
	}
	task.Result = result
	return nil
}

// latestGenerateOutput returns the newest completed generate output for a
// spec, along with the task ID that produced it.
func (fo *FactoryOrchestrator) latestGenerateOutput(specID string) (output, taskID string, ok bool) {
	fo.mu.RLock()
	defer fo.mu.RUnlock()

	var newest time.Time
	for _, t := range fo.results {
		if t.Type != "generate" || t.SpecID != specID || t.Status != TaskCompleted {
			continue
		}
		res, isMap := t.Result.(map[string]interface{})
		if !isMap {
			continue
		}
		out, isStr := res["output"].(string)
		if !isStr || out == "" {
			continue
		}
		if !ok || t.CreatedAt.After(newest) {
			newest = t.CreatedAt
			output, taskID, ok = out, t.ID, true
		}
	}
	return output, taskID, ok
}

// executeEvaluate judges the most recent completed generate output for the
// task's spec (LLM-as-a-judge) and records the verdict.
func (fo *FactoryOrchestrator) executeEvaluate(ctx context.Context, task *Task) error {
	specPrompt, contextStr, contextCount, err := fo.buildTaskParts(ctx, task.SpecID)
	if err != nil {
		return err
	}

	if fo.agent == nil || fo.evaluator == nil {
		promptOnlyResult(task, specPrompt, contextCount)
		return nil
	}

	output, sourceTaskID, ok := fo.latestGenerateOutput(task.SpecID)
	if !ok {
		return fmt.Errorf("no completed generate output found for spec %s — run a generate task first", task.SpecID)
	}

	var verdict *eval.EvaluationResult
	err = fo.withLLMSlot(ctx, func(callCtx context.Context) error {
		verdict, err = fo.evaluator.EvaluateCode(callCtx, specPrompt, output, contextStr, fo.evalCriteria)
		return err
	})
	if err != nil {
		return fmt.Errorf("llm evaluate task: %w", err)
	}

	task.Result = map[string]interface{}{
		"evaluation":     evaluationView(verdict),
		"evaluated_task": sourceTaskID,
		"spec_id":        task.SpecID,
		"model":          fo.agentModel,
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
	instruction := "## Task: Code Review\nReview the project context below against the specification. " +
		"Report every issue you find, including ones you are uncertain about or consider low-severity — " +
		"a downstream step filters by importance. For each finding include file, severity, and a concrete fix.\n\n"
	return fo.runAgentTask(ctx, task, instruction, "review")
}

// executeRefactor executes a refactoring task
func (fo *FactoryOrchestrator) executeRefactor(ctx context.Context, task *Task) error {
	instruction := "## Task: Refactoring Plan\nAnalyze the project context below against the specification and " +
		"produce a prioritized refactoring plan: duplication to collapse, abstractions to simplify, and naming or " +
		"structural cleanups — with concrete before/after sketches for the top items.\n\n"
	return fo.runAgentTask(ctx, task, instruction, "refactor_plan")
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
		"llm_enabled":     fo.agent != nil,
		"llm_model":       fo.agentModel,
		"llm_provider":    fo.agentProvider,
		"tasks_pending":   pending,
		"tasks_completed": completed,
		"tasks_failed":    failed,
		"max_workers":     fo.maxWorkers,
	}
}
