package logger

import (
	"batch-dispatcher/models"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
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

		// Add timestamp to log filename
		timestampedLogPath := addTimestampToFilename(logFilePath)

		logFile, err = os.OpenFile(timestampedLogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
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

// WritePerFileLog writes a detailed execution log for a completed job
// Returns the log file path or empty string if per-file logging is disabled
func (l *Logger) WritePerFileLog(result models.JobResult, execConfig *models.ExecutableConfig) (string, error) {
	if !l.perFileLogs {
		// Per-file logging disabled
		return "", nil
	}

	// Determine status subdirectory
	var statusDir string
	if result.TimedOut {
		statusDir = "timeout"
	} else if result.Success {
		statusDir = "success"
	} else {
		statusDir = "failed"
	}

	// Create status subdirectory if it doesn't exist
	logDir := filepath.Join(l.perFileLogDir, statusDir)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create log status directory: %w", err)
	}

	// Generate log filename: <filename>_<timestamp>.log
	timestamp := result.StartTime.Format("20060102_150405")
	baseFileName := filepath.Base(result.Job.FileName)
	// Remove extension
	ext := filepath.Ext(baseFileName)
	nameWithoutExt := baseFileName[:len(baseFileName)-len(ext)]

	logFileName := fmt.Sprintf("%s_%s.log", nameWithoutExt, timestamp)
	logPath := filepath.Join(logDir, logFileName)

	// Create log file
	logFile, err := os.Create(logPath)
	if err != nil {
		return "", fmt.Errorf("failed to create per-file log: %w", err)
	}
	defer logFile.Close()

	// Write detailed execution log
	fmt.Fprintf(logFile, "=================================================================================\n")
	fmt.Fprintf(logFile, "Execution Log - %s\n", result.Job.FileName)
	fmt.Fprintf(logFile, "=================================================================================\n")
	fmt.Fprintf(logFile, "Worker: %02d\n", result.WorkerID)
	fmt.Fprintf(logFile, "Start Time: %s\n", result.StartTime.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(logFile, "File: %s\n\n", result.Job.FilePath)

	// Write command
	fmt.Fprintf(logFile, "COMMAND:\n")
	fmt.Fprintf(logFile, "%s", execConfig.Path)
	for _, arg := range execConfig.DefaultArgs {
		// Replace {input} placeholder for display
		displayArg := strings.ReplaceAll(arg, "{input}", result.Job.FilePath)
		fmt.Fprintf(logFile, " %s", displayArg)
	}
	fmt.Fprintf(logFile, "\n\n")

	// Write environment variables
	if len(execConfig.EnvironmentExpanded) > 0 {
		fmt.Fprintf(logFile, "ENVIRONMENT VARIABLES:\n")
		for key, value := range execConfig.EnvironmentExpanded {
			fmt.Fprintf(logFile, "%s=%s\n", key, value)
		}
		fmt.Fprintf(logFile, "\n")
	}

	// Write standard output
	fmt.Fprintf(logFile, "STANDARD OUTPUT:\n")
	if result.Stdout != "" {
		fmt.Fprintf(logFile, "%s\n", result.Stdout)
	} else {
		fmt.Fprintf(logFile, "(empty)\n")
	}
	fmt.Fprintf(logFile, "\n")

	// Write standard error
	fmt.Fprintf(logFile, "STANDARD ERROR:\n")
	if result.Stderr != "" {
		fmt.Fprintf(logFile, "%s\n", result.Stderr)
	} else {
		fmt.Fprintf(logFile, "(empty)\n")
	}
	fmt.Fprintf(logFile, "\n")

	// Write result
	fmt.Fprintf(logFile, "RESULT:\n")
	if result.TimedOut {
		fmt.Fprintf(logFile, "Status: TIMEOUT\n")
	} else if result.Success {
		fmt.Fprintf(logFile, "Status: SUCCESS\n")
	} else {
		fmt.Fprintf(logFile, "Status: FAILED\n")
		if result.ErrorMessage != "" {
			fmt.Fprintf(logFile, "Error: %s\n", result.ErrorMessage)
		}
	}
	fmt.Fprintf(logFile, "Exit Code: %d\n", result.ExitCode)
	fmt.Fprintf(logFile, "Duration: %v\n", result.Duration.Round(100*time.Millisecond))

	return logPath, nil
}

// addTimestampToFilename inserts a timestamp before the file extension
// e.g., "dispatcher.log" becomes "dispatcher_20251104-112530.log"
func addTimestampToFilename(filePath string) string {
	timestamp := time.Now().Format("20060102-150405")

	ext := filepath.Ext(filePath)
	nameWithoutExt := filePath[:len(filePath)-len(ext)]

	return fmt.Sprintf("%s_%s%s", nameWithoutExt, timestamp, ext)
}
