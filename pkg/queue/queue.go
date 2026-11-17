package queue

import (
	"fmt"
	"log"
	"sync"
	"time"
)

// Job represents a workflow job to be executed
type Job struct {
	ID           string                 `json:"id"`
	Type         string                 `json:"type"`
	Status       string                 `json:"status"` // pending, running, success, failed
	Input        map[string]interface{} `json:"input"`
	Output       map[string]interface{} `json:"output"`
	Error        string                 `json:"error"`
	CreatedAt    time.Time              `json:"created_at"`
	StartedAt    *time.Time             `json:"started_at"`
	CompletedAt  *time.Time             `json:"completed_at"`
	Retries      int                    `json:"retries"`
	MaxRetries   int                    `json:"max_retries"`
	Priority     int                    `json:"priority"` // Higher number = higher priority
	ExecutionFn  func(*Job) error       `json:"-"`        // Function to execute
}

// Queue manages job execution with retries and concurrency limits
type Queue struct {
	jobs        map[string]*Job
	pending     []*Job // Priority queue
	running     map[string]*Job
	completed   []*Job
	maxConcur   int
	maxRetries  int
	mu          sync.RWMutex
	jobsChan    chan *Job
	done        chan bool
	workers     int
}

// NewQueue creates a new job queue
func NewQueue(maxConcurrency, maxRetries int) *Queue {
	if maxConcurrency < 1 {
		maxConcurrency = 5
	}
	if maxRetries < 0 {
		maxRetries = 3
	}

	return &Queue{
		jobs:       make(map[string]*Job),
		running:    make(map[string]*Job),
		maxConcur:  maxConcurrency,
		maxRetries: maxRetries,
		jobsChan:   make(chan *Job, 256),
		done:       make(chan bool),
	}
}

// Start begins processing jobs
func (q *Queue) Start() {
	go func() {
		for i := 0; i < q.maxConcur; i++ {
			go q.worker(i)
		}
	}()
}

// Stop stops the queue
func (q *Queue) Stop() {
	close(q.jobsChan)
}

// Submit adds a job to the queue
func (q *Queue) Submit(job *Job) error {
	if job.ID == "" {
		return fmt.Errorf("job must have an ID")
	}

	job.CreatedAt = time.Now()
	job.Status = "pending"
	job.MaxRetries = q.maxRetries

	q.mu.Lock()
	q.jobs[job.ID] = job
	q.mu.Unlock()

	q.jobsChan <- job
	log.Printf("[Queue] Job submitted: %s (type: %s, priority: %d)", job.ID, job.Type, job.Priority)

	return nil
}

// Get retrieves a job by ID
func (q *Queue) Get(jobID string) (*Job, bool) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	job, ok := q.jobs[jobID]
	return job, ok
}

// GetByStatus returns all jobs with a specific status
func (q *Queue) GetByStatus(status string) []*Job {
	q.mu.RLock()
	defer q.mu.RUnlock()

	var result []*Job
	for _, job := range q.jobs {
		if job.Status == status {
			result = append(result, job)
		}
	}
	return result
}

// Stats returns queue statistics
func (q *Queue) Stats() map[string]interface{} {
	q.mu.RLock()
	defer q.mu.RUnlock()

	pending := 0
	running := 0
	success := 0
	failed := 0

	for _, job := range q.jobs {
		switch job.Status {
		case "pending":
			pending++
		case "running":
			running++
		case "success":
			success++
		case "failed":
			failed++
		}
	}

	return map[string]interface{}{
		"total":    len(q.jobs),
		"pending":  pending,
		"running":  running,
		"success":  success,
		"failed":   failed,
		"capacity": q.maxConcur,
	}
}

// worker processes jobs from the queue
func (q *Queue) worker(id int) {
	log.Printf("[Queue] Worker %d started", id)

	for job := range q.jobsChan {
		q.executeJob(job)
	}

	log.Printf("[Queue] Worker %d stopped", id)
}

// executeJob runs a job with retry logic
func (q *Queue) executeJob(job *Job) {
	q.mu.Lock()
	job.Status = "running"
	now := time.Now()
	job.StartedAt = &now
	q.running[job.ID] = job
	q.mu.Unlock()

	log.Printf("[Queue] Executing job: %s (attempt %d/%d)", job.ID, job.Retries+1, job.MaxRetries+1)

	var err error
	if job.ExecutionFn != nil {
		err = job.ExecutionFn(job)
	}

	q.mu.Lock()
	now = time.Now()
	job.CompletedAt = &now
	delete(q.running, job.ID)

	if err != nil {
		job.Error = err.Error()

		// Retry logic
		if job.Retries < job.MaxRetries {
			job.Retries++
			job.Status = "pending"
			job.StartedAt = nil
			job.CompletedAt = nil

			q.mu.Unlock()
			log.Printf("[Queue] Job failed, retrying: %s (attempt %d)", job.ID, job.Retries+1)

			// Resubmit after delay
			go func(j *Job) {
				time.Sleep(time.Duration(job.Retries) * time.Second) // Exponential backoff
				q.jobsChan <- j
			}(job)
			return
		}

		job.Status = "failed"
		log.Printf("[Queue] Job failed permanently: %s - %v", job.ID, err)
	} else {
		job.Status = "success"
		log.Printf("[Queue] Job completed: %s", job.ID)
	}

	q.completed = append(q.completed, job)
	q.mu.Unlock()
}

// Scheduler schedules jobs to run at specific times
type Scheduler struct {
	queue    *Queue
	jobs     map[string]*ScheduledJob
	mu       sync.RWMutex
	done     chan bool
	ticker   *time.Ticker
}

// ScheduledJob represents a job scheduled for later execution
type ScheduledJob struct {
	ID          string
	RunAt       time.Time
	Interval    time.Duration // 0 = run once, >0 = repeat
	Job         *Job
	Enabled     bool
	LastRunAt   *time.Time
}

// NewScheduler creates a new job scheduler
func NewScheduler(queue *Queue) *Scheduler {
	return &Scheduler{
		queue:  queue,
		jobs:   make(map[string]*ScheduledJob),
		done:   make(chan bool),
		ticker: time.NewTicker(10 * time.Second), // Check every 10 seconds
	}
}

// Start begins the scheduler
func (s *Scheduler) Start() {
	go func() {
		for {
			select {
			case <-s.ticker.C:
				s.checkJobs()
			case <-s.done:
				s.ticker.Stop()
				return
			}
		}
	}()
}

// Stop stops the scheduler
func (s *Scheduler) Stop() {
	s.done <- true
}

// Schedule adds a job to the scheduler
func (s *Scheduler) Schedule(id string, runAt time.Time, job *Job) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.jobs[id] = &ScheduledJob{
		ID:      id,
		RunAt:   runAt,
		Job:     job,
		Enabled: true,
	}

	log.Printf("[Scheduler] Job scheduled: %s to run at %v", id, runAt)
	return nil
}

// ScheduleRecurring schedules a recurring job
func (s *Scheduler) ScheduleRecurring(id string, job *Job, interval time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.jobs[id] = &ScheduledJob{
		ID:       id,
		RunAt:    time.Now().Add(interval),
		Interval: interval,
		Job:      job,
		Enabled:  true,
	}

	log.Printf("[Scheduler] Recurring job scheduled: %s (interval: %v)", id, interval)
	return nil
}

// Cancel removes a scheduled job
func (s *Scheduler) Cancel(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if sj, ok := s.jobs[id]; ok {
		sj.Enabled = false
		delete(s.jobs, id)
		log.Printf("[Scheduler] Job cancelled: %s", id)
	}
}

// checkJobs checks for jobs that should run
func (s *Scheduler) checkJobs() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for _, sj := range s.jobs {
		if !sj.Enabled {
			continue
		}

		if now.After(sj.RunAt) || now.Equal(sj.RunAt) {
			// Submit job
			s.queue.Submit(sj.Job)

			// Update last run time
			sj.LastRunAt = &now

			// If recurring, schedule next run
			if sj.Interval > 0 {
				sj.RunAt = now.Add(sj.Interval)
			} else {
				// One-time job, disable it
				sj.Enabled = false
			}
		}
	}
}

// ListScheduled returns all scheduled jobs
func (s *Scheduler) ListScheduled() []*ScheduledJob {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*ScheduledJob
	for _, sj := range s.jobs {
		if sj.Enabled {
			result = append(result, sj)
		}
	}
	return result
}
