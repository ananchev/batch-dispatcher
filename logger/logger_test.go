package logger

import (
	"batch-dispatcher/models"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")

	logger, err := New(logFile, "")
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	if logger.centralLog == nil {
		t.Error("Central log should not be nil")
	}
	if logger.logFile == nil {
		t.Error("Log file should not be nil")
	}

	// Verify log file was created
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		t.Error("Log file was not created")
	}
}

func TestNew_NoFile(t *testing.T) {
	// Create logger without file (stdout only)
	logger, err := New("", "")
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	if logger.centralLog == nil {
		t.Error("Central log should not be nil")
	}
	if logger.logFile != nil {
		t.Error("Log file should be nil when no path provided")
	}
}

func TestLogJobStart(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")

	logger, err := New(logFile, "")
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	job := &models.Job{
		FilePath: "C:\\data\\test.csv",
		FileName: "test.csv",
	}

	logger.LogJobStart(job, 1)
	logger.Close()

	// Verify log content
	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "Worker 01") {
		t.Error("Log should contain worker ID")
	}
	if !strings.Contains(contentStr, "test.csv") {
		t.Error("Log should contain filename")
	}
}

func TestLogJobComplete(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")

	logger, err := New(logFile, "")
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	result := models.JobResult{
		Job: &models.Job{
			FilePath: "C:\\data\\test.csv",
			FileName: "test.csv",
		},
		WorkerID:  2,
		Success:   true,
		ExitCode:  0,
		Duration:  500 * time.Millisecond,
		StartTime: time.Now(),
		EndTime:   time.Now().Add(500 * time.Millisecond),
	}

	logger.LogJobComplete(result)
	logger.Close()

	// Verify log content
	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "Worker 02") {
		t.Error("Log should contain worker ID")
	}
	if !strings.Contains(contentStr, "Completed") {
		t.Error("Log should contain 'Completed'")
	}
	if !strings.Contains(contentStr, "test.csv") {
		t.Error("Log should contain filename")
	}
}

func TestLogJobError(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")

	logger, err := New(logFile, "")
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	result := models.JobResult{
		Job: &models.Job{
			FilePath: "C:\\data\\test.csv",
			FileName: "test.csv",
		},
		WorkerID:     3,
		Success:      false,
		ExitCode:     1,
		ErrorMessage: "command failed",
		Duration:     500 * time.Millisecond,
		StartTime:    time.Now(),
		EndTime:      time.Now().Add(500 * time.Millisecond),
	}

	logger.LogJobError(result)
	logger.Close()

	// Verify log content
	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "Worker 03") {
		t.Error("Log should contain worker ID")
	}
	if !strings.Contains(contentStr, "FAILED") {
		t.Error("Log should contain 'FAILED'")
	}
	if !strings.Contains(contentStr, "command failed") {
		t.Error("Log should contain error message")
	}
}

func TestLogJobError_Timeout(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")

	logger, err := New(logFile, "")
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	result := models.JobResult{
		Job: &models.Job{
			FilePath: "C:\\data\\test.csv",
			FileName: "test.csv",
		},
		WorkerID:  4,
		Success:   false,
		TimedOut:  true,
		ExitCode:  -1,
		Duration:  5 * time.Second,
		StartTime: time.Now(),
		EndTime:   time.Now().Add(5 * time.Second),
	}

	logger.LogJobError(result)
	logger.Close()

	// Verify log content
	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "TIMEOUT") {
		t.Error("Log should contain 'TIMEOUT'")
	}
}

func TestCreatePerFileLog(t *testing.T) {
	tmpDir := t.TempDir()
	perFileDir := filepath.Join(tmpDir, "per-file-logs")

	logger, err := New("", perFileDir)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	job := &models.Job{
		FilePath: "C:\\data\\test.csv",
		FileName: "test.csv",
	}

	logPath, closer, err := logger.CreatePerFileLog(job, 1)
	if err != nil {
		t.Fatalf("Failed to create per-file log: %v", err)
	}

	if logPath == "" {
		t.Error("Log path should not be empty")
	}

	// Close the log
	if err := closer(); err != nil {
		t.Errorf("Failed to close per-file log: %v", err)
	}

	// Verify log file exists
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Error("Per-file log was not created")
	}

	// Verify log content has header and footer
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read per-file log: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "Execution Log") {
		t.Error("Log should contain header")
	}
	if !strings.Contains(contentStr, "Worker: 01") {
		t.Error("Log should contain worker ID")
	}
	if !strings.Contains(contentStr, "Completed:") {
		t.Error("Log should contain completion footer")
	}
}

func TestCreatePerFileLog_Disabled(t *testing.T) {
	logger, err := New("", "")
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	job := &models.Job{
		FilePath: "C:\\data\\test.csv",
		FileName: "test.csv",
	}

	logPath, closer, err := logger.CreatePerFileLog(job, 1)
	if err != nil {
		t.Fatalf("Should not error when per-file logging disabled: %v", err)
	}

	if logPath != "" {
		t.Error("Log path should be empty when per-file logging disabled")
	}

	// Closer should be no-op
	if err := closer(); err != nil {
		t.Errorf("Closer should not error: %v", err)
	}
}
