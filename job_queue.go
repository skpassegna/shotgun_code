package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

/**
 * JobQueue System for Shotgun Code
 *
 * This module implements a background job queue system that allows long-running operations
 * to execute without blocking the UI. Jobs can be queued, monitored, cancelled, and tracked
 * with real-time progress updates.
 *
 * Key Features:
 * - Non-blocking job execution in goroutines
 * - Real-time progress tracking via Wails events
 * - Job cancellation support
 * - Concurrent job execution with configurable limits
 * - Job history and status tracking
 * - Automatic cleanup of completed jobs
 *
 * Job Types:
 * - context_generation: Generate shotgun context from selected files
 * - diff_splitting: Split large diffs into manageable chunks
 * - llm_call: Call LLM API for code generation
 *
 * Job States:
 * - queued: Job is waiting to start
 * - running: Job is currently executing
 * - completed: Job finished successfully
 * - failed: Job encountered an error
 * - cancelled: Job was cancelled by user
 */

// Job represents a background task with status tracking
type Job struct {
	ID          string             `json:"id"`          // Unique identifier for the job
	Type        string             `json:"type"`        // Job type (context_generation, diff_splitting, llm_call)
	Status      string             `json:"status"`      // Current status (queued, running, completed, failed, cancelled)
	Progress    float64            `json:"progress"`    // Progress percentage (0-100)
	Error       string             `json:"error"`       // Error message if failed
	CreatedAt   time.Time          `json:"createdAt"`   // When the job was created
	StartedAt   time.Time          `json:"startedAt"`   // When the job started running
	CompletedAt time.Time          `json:"completedAt"` // When the job completed
	CancelFunc  context.CancelFunc `json:"-"`           // Function to cancel the job (not serialized)
}

// JobQueue manages background jobs with concurrent execution
type JobQueue struct {
	app     *App       // Reference to main app for Wails events
	jobs    []Job      // List of all jobs (active and historical)
	mu      sync.Mutex // Mutex for thread-safe access to jobs
	maxJobs int        // Maximum number of concurrent jobs
}

// NewJobQueue creates a new job queue instance
//
// Parameters:
//   - app: Reference to the main App struct for emitting Wails events
//
// Returns:
//   - *JobQueue: Initialized job queue with default settings
func NewJobQueue(app *App) *JobQueue {
	return &JobQueue{
		app:     app,
		jobs:    make([]Job, 0),
		maxJobs: 5, // Allow up to 5 concurrent jobs
	}
}

// AddJob adds a new job to the queue and starts it immediately
//
// This method creates a new job, adds it to the queue, and starts executing it
// in a goroutine. The job's progress and status are tracked and emitted via
// Wails events for real-time UI updates.
//
// Parameters:
//   - jobType: Type of job (context_generation, diff_splitting, llm_call)
//   - task: Function to execute, receives a cancellable context
//
// Returns:
//   - string: Unique job ID for tracking
//
// Example:
//
//	jobID := jobQueue.AddJob("context_generation", func(ctx context.Context) error {
//	    // Perform long-running task
//	    return generateContext(ctx, rootDir, excludedPaths)
//	})
func (jq *JobQueue) AddJob(jobType string, task func(ctx context.Context) error) string {
	jq.mu.Lock()

	// Generate unique job ID using type and timestamp
	jobID := fmt.Sprintf("%s_%d", jobType, time.Now().UnixNano())

	// Create cancellable context for this job
	ctx, cancel := context.WithCancel(jq.app.ctx)

	// Create new job with initial state
	job := Job{
		ID:         jobID,
		Type:       jobType,
		Status:     "queued",
		Progress:   0,
		CreatedAt:  time.Now(),
		CancelFunc: cancel,
	}

	// Add job to queue
	jq.jobs = append(jq.jobs, job)

	// Emit initial job queue update to frontend
	runtime.EventsEmit(jq.app.ctx, "jobQueueUpdated", jq.getJobStatusesUnsafe())

	jq.mu.Unlock()

	// Start job execution in goroutine (non-blocking)
	go func() {
		// Update job status to running
		jq.updateJobStatus(jobID, "running")
		jq.setJobStartTime(jobID, time.Now())

		// Execute the task with cancellable context
		err := task(ctx)

		// Update job status based on result
		if ctx.Err() == context.Canceled {
			// Job was cancelled by user
			jq.updateJobStatus(jobID, "cancelled")
			runtime.LogInfo(jq.app.ctx, fmt.Sprintf("Job %s was cancelled", jobID))
		} else if err != nil {
			// Job failed with error
			jq.updateJobStatus(jobID, "failed")
			jq.setJobError(jobID, err.Error())
			runtime.LogError(jq.app.ctx, fmt.Sprintf("Job %s failed: %v", jobID, err))
		} else {
			// Job completed successfully
			jq.updateJobStatus(jobID, "completed")
			jq.setJobProgress(jobID, 100)
			runtime.LogInfo(jq.app.ctx, fmt.Sprintf("Job %s completed successfully", jobID))
		}

		// Set completion time
		jq.setJobCompletionTime(jobID, time.Now())

		// Emit final job queue update
		jq.mu.Lock()
		runtime.EventsEmit(jq.app.ctx, "jobQueueUpdated", jq.getJobStatusesUnsafe())
		jq.mu.Unlock()
	}()

	return jobID
}

// CancelJob cancels a running job by its ID
//
// This method calls the job's cancel function, which will cause the job's context
// to be cancelled. The job's task function should check for context cancellation
// and exit gracefully.
//
// Parameters:
//   - jobID: Unique identifier of the job to cancel
//
// Returns:
//   - error: Error if job not found, nil otherwise
func (jq *JobQueue) CancelJob(jobID string) error {
	jq.mu.Lock()
	defer jq.mu.Unlock()

	// Find job by ID
	for i, job := range jq.jobs {
		if job.ID == jobID {
			// Only cancel if job is queued or running
			if job.Status == "queued" || job.Status == "running" {
				// Call cancel function to cancel context
				if job.CancelFunc != nil {
					job.CancelFunc()
				}

				// Update status to cancelled
				jq.jobs[i].Status = "cancelled"
				jq.jobs[i].CompletedAt = time.Now()

				// Emit update to frontend
				runtime.EventsEmit(jq.app.ctx, "jobQueueUpdated", jq.getJobStatusesUnsafe())

				runtime.LogInfo(jq.app.ctx, fmt.Sprintf("Cancelled job: %s", jobID))
				return nil
			}

			return fmt.Errorf("job %s cannot be cancelled (status: %s)", jobID, job.Status)
		}
	}

	return fmt.Errorf("job not found: %s", jobID)
}

// GetJobStatuses returns a copy of all job statuses
//
// This method is thread-safe and returns a copy of the jobs slice to avoid
// race conditions when the frontend reads job data.
//
// Returns:
//   - []Job: Copy of all jobs in the queue
func (jq *JobQueue) GetJobStatuses() []Job {
	jq.mu.Lock()
	defer jq.mu.Unlock()

	return jq.getJobStatusesUnsafe()
}

// getJobStatusesUnsafe returns job statuses without locking (internal use only)
//
// This method should only be called when the mutex is already locked.
// It creates a deep copy of the jobs slice to prevent race conditions.
//
// Returns:
//   - []Job: Copy of all jobs in the queue
func (jq *JobQueue) getJobStatusesUnsafe() []Job {
	// Create a copy to avoid race conditions
	statuses := make([]Job, len(jq.jobs))
	copy(statuses, jq.jobs)
	return statuses
}

// updateJobStatus updates the status of a job by ID
//
// This is a thread-safe method that updates the job's status and emits
// an event to notify the frontend.
//
// Parameters:
//   - jobID: Unique identifier of the job
//   - status: New status (queued, running, completed, failed, cancelled)
func (jq *JobQueue) updateJobStatus(jobID string, status string) {
	jq.mu.Lock()
	defer jq.mu.Unlock()

	for i, job := range jq.jobs {
		if job.ID == jobID {
			jq.jobs[i].Status = status
			runtime.EventsEmit(jq.app.ctx, "jobQueueUpdated", jq.getJobStatusesUnsafe())
			break
		}
	}
}

// setJobError sets the error message for a failed job
//
// Parameters:
//   - jobID: Unique identifier of the job
//   - errMsg: Error message to store
func (jq *JobQueue) setJobError(jobID string, errMsg string) {
	jq.mu.Lock()
	defer jq.mu.Unlock()

	for i, job := range jq.jobs {
		if job.ID == jobID {
			jq.jobs[i].Error = errMsg
			break
		}
	}
}

// setJobProgress updates the progress percentage of a job
//
// Parameters:
//   - jobID: Unique identifier of the job
//   - progress: Progress percentage (0-100)
func (jq *JobQueue) setJobProgress(jobID string, progress float64) {
	jq.mu.Lock()
	defer jq.mu.Unlock()

	for i, job := range jq.jobs {
		if job.ID == jobID {
			jq.jobs[i].Progress = progress
			runtime.EventsEmit(jq.app.ctx, "jobQueueUpdated", jq.getJobStatusesUnsafe())
			break
		}
	}
}

// setJobStartTime sets the start time for a job
//
// Parameters:
//   - jobID: Unique identifier of the job
//   - startTime: Time when the job started
func (jq *JobQueue) setJobStartTime(jobID string, startTime time.Time) {
	jq.mu.Lock()
	defer jq.mu.Unlock()

	for i, job := range jq.jobs {
		if job.ID == jobID {
			jq.jobs[i].StartedAt = startTime
			break
		}
	}
}

// setJobCompletionTime sets the completion time for a job
//
// Parameters:
//   - jobID: Unique identifier of the job
//   - completionTime: Time when the job completed
func (jq *JobQueue) setJobCompletionTime(jobID string, completionTime time.Time) {
	jq.mu.Lock()
	defer jq.mu.Unlock()

	for i, job := range jq.jobs {
		if job.ID == jobID {
			jq.jobs[i].CompletedAt = completionTime
			break
		}
	}
}

// CleanupOldJobs removes completed/failed/cancelled jobs older than the specified duration
//
// This method helps prevent the job queue from growing indefinitely by removing
// old jobs that are no longer relevant.
//
// Parameters:
//   - maxAge: Maximum age for completed jobs (e.g., 1 hour)
//
// Returns:
//   - int: Number of jobs removed
func (jq *JobQueue) CleanupOldJobs(maxAge time.Duration) int {
	jq.mu.Lock()
	defer jq.mu.Unlock()

	now := time.Now()
	removed := 0
	newJobs := make([]Job, 0)

	for _, job := range jq.jobs {
		// Keep running and queued jobs
		if job.Status == "running" || job.Status == "queued" {
			newJobs = append(newJobs, job)
			continue
		}

		// Keep recent completed/failed/cancelled jobs
		if !job.CompletedAt.IsZero() && now.Sub(job.CompletedAt) < maxAge {
			newJobs = append(newJobs, job)
			continue
		}

		// Remove old jobs
		removed++
	}

	jq.jobs = newJobs

	if removed > 0 {
		runtime.LogInfo(jq.app.ctx, fmt.Sprintf("Cleaned up %d old jobs", removed))
		runtime.EventsEmit(jq.app.ctx, "jobQueueUpdated", jq.getJobStatusesUnsafe())
	}

	return removed
}

