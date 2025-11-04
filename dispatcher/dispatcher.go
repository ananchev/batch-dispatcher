package dispatcher

import (
	"batch-dispatcher/executor"
	"batch-dispatcher/filesystem"
	"batch-dispatcher/logger"
	"batch-dispatcher/models"
	"context"
	"fmt"
	"sync"
	"time"
)

// Dispatcher coordinates workers and job execution
type Dispatcher struct {
	config     *models.Config
	logger     *logger.Logger
	counter    *logger.ProgressCounter
	jobQueue   chan *models.Job
	results    chan models.JobResult
	wg         sync.WaitGroup
	failSignal chan struct{} // Signal to stop workers on first failure
	failOnce   sync.Once     // Ensure fail signal sent only once
}

// New creates a new Dispatcher
func New(cfg *models.Config, log *logger.Logger, counter *logger.ProgressCounter) (*Dispatcher, error) {
	return &Dispatcher{
		config:     cfg,
		logger:     log,
		counter:    counter,
		jobQueue:   make(chan *models.Job, cfg.Workers.Count*2), // Buffer for smoother flow
		results:    make(chan models.JobResult, cfg.Workers.Count*2),
		failSignal: make(chan struct{}), // Unbuffered - signal only
	}, nil
}

// Run starts the dispatcher and processes all jobs
func (d *Dispatcher) Run(ctx context.Context, jobs []*models.Job) error {
	startTime := time.Now()
	totalFiles := len(jobs)

	d.logger.Info("Starting dispatcher with %d workers", d.config.Workers.Count)
	d.logger.Info("Processing %d files", totalFiles)

	// Log error handling mode
	if d.config.Advanced.ContinueOnError {
		d.logger.Info("Error handling: CONTINUE-ON-ERROR mode (process all files despite failures)")
	} else {
		d.logger.Info("Error handling: FAIL-FAST mode (stop on first failure)")
	}

	// Start progress counter
	d.counter.Start()

	// Start result collector
	collectorDone := make(chan struct{})
	go d.collectResults(collectorDone)

	// Start workers
	for i := 1; i <= d.config.Workers.Count; i++ {
		d.wg.Add(1)
		go d.worker(ctx, i)
	}

	// Queue all jobs
	go func() {
		for _, job := range jobs {
			select {
			case d.jobQueue <- job:
			case <-ctx.Done():
				d.logger.Error("Context cancelled while queuing jobs")
				return
			case <-d.failSignal:
				d.logger.Error("Fail-fast triggered, stopping job queue")
				return
			}
		}
		close(d.jobQueue)
	}()

	// Wait for all workers to finish
	d.wg.Wait()
	close(d.results)

	// Wait for result collector to finish
	<-collectorDone

	// Display final summary
	d.counter.Complete()

	completed, successful, failed := d.counter.GetStats()
	totalDuration := time.Since(startTime)

	d.logger.Info("Processing complete:")
	d.logger.Info("  Total to process: %d", totalFiles)
	d.logger.Info("  Total processed: %d", completed)
	d.logger.Info("  Successful: %d", successful)
	d.logger.Info("  Failed: %d", failed)
	if completed < totalFiles {
		d.logger.Info("  Not processed: %d (stopped due to fail-fast)", totalFiles-completed)
	}
	d.logger.Info("Total processing time: %v", totalDuration.Round(time.Second))

	return nil
}

// worker processes jobs from the queue
func (d *Dispatcher) worker(ctx context.Context, workerID int) {
	defer d.wg.Done()

	for {
		select {
		case <-ctx.Done():
			d.logger.Info("Worker %02d: Shutting down (context cancelled)", workerID)
			return
		case <-d.failSignal:
			d.logger.Info("Worker %02d: Shutting down (fail-fast triggered)", workerID)
			return
		case job, ok := <-d.jobQueue:
			if !ok {
				// Queue closed, no more jobs
				d.logger.Info("Worker %02d: No more jobs, shutting down", workerID)
				return
			}

			// Process the job
			result := d.processJob(ctx, job, workerID)

			// Check if job failed and fail-fast is enabled (continue_on_error is false)
			if !result.Success && !d.config.Advanced.ContinueOnError {
				d.logger.Error("Worker %02d: Job failed, triggering fail-fast shutdown", workerID)
				d.signalFailure()
				d.results <- result
				return
			}

			d.results <- result
		}
	}
}

// processJob executes a single job
func (d *Dispatcher) processJob(ctx context.Context, job *models.Job, workerID int) models.JobResult {
	d.logger.LogJobStart(job, workerID)

	// Execute the job
	result := executor.ExecuteJob(job, &d.config.Executable, workerID)

	// Write per-file execution log
	if logPath, err := d.logger.WritePerFileLog(result, &d.config.Executable); err != nil {
		d.logger.Error("Worker %02d: Failed to write per-file log: %v", workerID, err)
	} else if logPath != "" {
		result.LogFilePath = logPath
	}

	// Log result to central log
	if result.Success {
		d.logger.LogJobComplete(result)
	} else {
		d.logger.LogJobError(result)
	}

	// Move file to appropriate directory
	var finalPath string
	var moveErr error
	if result.Success {
		finalPath, moveErr = filesystem.MoveToProcessed(job.FilePath, d.config.Output.ProcessedDirectory, workerID)
	} else {
		finalPath, moveErr = filesystem.MoveToErrors(job.FilePath, d.config.Output.ErrorsDirectory, workerID)
	}

	if moveErr != nil {
		d.logger.Error("Worker %02d: Failed to move file %s: %v", workerID, job.FileName, moveErr)
		// If file move fails after successful execution, mark as failed
		if result.Success {
			result.Success = false
			result.ErrorMessage = fmt.Sprintf("Execution succeeded but file move failed: %v", moveErr)
		}
	} else {
		d.logger.Info("Worker %02d: Moved file to %s", workerID, finalPath)
	}

	return result
}

// collectResults collects and processes job results
func (d *Dispatcher) collectResults(done chan struct{}) {
	defer close(done)

	for result := range d.results {
		if result.Success {
			d.counter.IncrementSuccess()
		} else {
			d.counter.IncrementFailure()
		}
	}
}

// signalFailure triggers the fail-fast shutdown signal
// Uses sync.Once to ensure signal is only sent once
func (d *Dispatcher) signalFailure() {
	d.failOnce.Do(func() {
		close(d.failSignal)
	})
}
