package logger

import (
	"batch-dispatcher/models"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Logger handles structured logging for the dispatcher
type Logger struct {
	centralLog    *log.Logger
	logFile       *os.File
	perFileLogs   bool
	perFileLogDir string
	mu            sync.Mutex
}

// New creates a new Logger instance
// If logFilePath is empty, logs only to stdout
// If perFileLogDir is non-empty, creates individual log files for each execution
func New(logFilePath string, perFileLogDir string) (*Logger, error) {
	var writers []io.Writer
	var logFile *os.File
	var err error

	// Always write to stdout
	writers = append(writers, os.Stdout)

	// Optionally write to file
	if logFilePath != "" {
		// Create directory if needed
		logDir := filepath.Dir(logFilePath)
		if err := os.MkdirAll(logDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create log directory: %w", err)
		}

		logFile, err = os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file: %w", err)
		}
		writers = append(writers, logFile)
	}

	// Create multi-writer for both stdout and file
	multiWriter := io.MultiWriter(writers...)
	centralLog := log.New(multiWriter, "", log.LstdFlags)

	// Create per-file log directory if needed
	if perFileLogDir != "" {
		if err := os.MkdirAll(perFileLogDir, 0755); err != nil {
			if logFile != nil {
				logFile.Close()
			}
			return nil, fmt.Errorf("failed to create per-file log directory: %w", err)
		}
	}

	return &Logger{
		centralLog:    centralLog,
		logFile:       logFile,
		perFileLogs:   perFileLogDir != "",
		perFileLogDir: perFileLogDir,
	}, nil
}

// Close closes the log file
func (l *Logger) Close() error {
	if l.logFile != nil {
		return l.logFile.Close()
	}
	return nil
}

// Info logs an info-level message
func (l *Logger) Info(format string, args ...interface{}) {
	l.log("INFO", format, args...)
}

// Error logs an error-level message
func (l *Logger) Error(format string, args ...interface{}) {
	l.log("ERROR", format, args...)
}

// log is the internal logging function
func (l *Logger) log(level string, format string, args ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()

	message := fmt.Sprintf(format, args...)
	l.centralLog.Printf("[%s] %s", level, message)
}

// LogJobStart logs the start of job execution
func (l *Logger) LogJobStart(job *models.Job, workerID int) {
	l.Info("Worker %02d: Starting execution of %s", workerID, job.FileName)
}

// LogJobComplete logs successful job completion
func (l *Logger) LogJobComplete(result models.JobResult) {
	l.Info("Worker %02d: Completed %s in %v (exit code: %d)",
		result.WorkerID,
		result.Job.FileName,
		result.Duration,
		result.ExitCode)
}

// LogJobError logs job execution failure
func (l *Logger) LogJobError(result models.JobResult) {
	if result.TimedOut {
		l.Error("Worker %02d: TIMEOUT - %s after %v",
			result.WorkerID,
			result.Job.FileName,
			result.Duration)
	} else {
		l.Error("Worker %02d: FAILED - %s (exit code: %d): %s",
			result.WorkerID,
			result.Job.FileName,
			result.ExitCode,
			result.ErrorMessage)
	}
}

// CreatePerFileLog creates an individual log file for a job execution
// Returns the log file path and a closer function
// If per-file logging is disabled, returns empty string and no-op closer
func (l *Logger) CreatePerFileLog(job *models.Job, workerID int) (string, func() error, error) {
	if !l.perFileLogs {
		// Per-file logging disabled
		return "", func() error { return nil }, nil
	}

	// Generate log filename: <filename>_<timestamp>_worker<id>.log
	timestamp := time.Now().Format("20060102_150405")
	baseFileName := filepath.Base(job.FileName)
	// Remove extension
	ext := filepath.Ext(baseFileName)
	nameWithoutExt := baseFileName[:len(baseFileName)-len(ext)]

	logFileName := fmt.Sprintf("%s_%s_worker%02d.log", nameWithoutExt, timestamp, workerID)
	logPath := filepath.Join(l.perFileLogDir, logFileName)

	// Create log file
	logFile, err := os.Create(logPath)
	if err != nil {
		return "", nil, fmt.Errorf("failed to create per-file log: %w", err)
	}

	// Write header
	fmt.Fprintf(logFile, "=== Execution Log ===\n")
	fmt.Fprintf(logFile, "File: %s\n", job.FilePath)
	fmt.Fprintf(logFile, "Worker: %02d\n", workerID)
	fmt.Fprintf(logFile, "Started: %s\n", time.Now().Format(time.RFC3339))
	fmt.Fprintf(logFile, "====================\n\n")

	closer := func() error {
		// Write footer
		fmt.Fprintf(logFile, "\n====================\n")
		fmt.Fprintf(logFile, "Completed: %s\n", time.Now().Format(time.RFC3339))
		return logFile.Close()
	}

	return logPath, closer, nil
}
