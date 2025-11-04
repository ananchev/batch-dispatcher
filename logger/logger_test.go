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

	// Verify a timestamped log file was created (test_*.log)
	files, err := filepath.Glob(filepath.Join(tmpDir, "test_*.log"))
	if err != nil {
		t.Fatalf("Failed to list log files: %v", err)
	}
	if len(files) == 0 {
		t.Error("No timestamped log file was created")
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

	// Find the timestamped log file
	files, err := filepath.Glob(filepath.Join(tmpDir, "test_*.log"))
	if err != nil {
		t.Fatalf("Failed to list log files: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("No timestamped log file was created")
	}

	// Verify log content
	content, err := os.ReadFile(files[0])
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

	// Find the timestamped log file
	files, err := filepath.Glob(filepath.Join(tmpDir, "test_*.log"))
	if err != nil {
		t.Fatalf("Failed to list log files: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("No timestamped log file was created")
	}

	// Verify log content
	content, err := os.ReadFile(files[0])
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

	// Find the timestamped log file
	files, err := filepath.Glob(filepath.Join(tmpDir, "test_*.log"))
	if err != nil {
		t.Fatalf("Failed to list log files: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("No timestamped log file was created")
	}

	// Verify log content
	content, err := os.ReadFile(files[0])
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

	// Find the timestamped log file
	files, err := filepath.Glob(filepath.Join(tmpDir, "test_*.log"))
	if err != nil {
		t.Fatalf("Failed to list log files: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("No timestamped log file was created")
	}

	// Verify log content
	content, err := os.ReadFile(files[0])
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "TIMEOUT") {
		t.Error("Log should contain 'TIMEOUT'")
	}
}

func TestWritePerFileLog(t *testing.T) {
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

	result := models.JobResult{
		Job:       job,
		WorkerID:  1,
		Success:   true,
		ExitCode:  0,
		StartTime: time.Now(),
		EndTime:   time.Now().Add(500 * time.Millisecond),
		Duration:  500 * time.Millisecond,
		Stdout:    "test output",
		Stderr:    "",
	}

	execConfig := &models.ExecutableConfig{
		Path:                "test.exe",
		DefaultArgs:         []string{"-input", "{input}"},
		EnvironmentExpanded: map[string]string{"VAR1": "value1"},
	}

	logPath, err := logger.WritePerFileLog(result, execConfig)
	if err != nil {
		t.Fatalf("Failed to write per-file log: %v", err)
	}

	if logPath == "" {
		t.Error("Log path should not be empty")
	}

	// Verify log file exists
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Error("Per-file log was not created")
	}

	// Verify log content
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
	if !strings.Contains(contentStr, "COMMAND:") {
		t.Error("Log should contain command section")
	}
	if !strings.Contains(contentStr, "STANDARD OUTPUT:") {
		t.Error("Log should contain stdout section")
	}
	if !strings.Contains(contentStr, "test output") {
		t.Error("Log should contain stdout content")
	}
	if !strings.Contains(contentStr, "SUCCESS") {
		t.Error("Log should contain success status")
	}

	// Verify file is in success subdirectory
	if !strings.Contains(logPath, string(filepath.Separator)+"success"+string(filepath.Separator)) {
		t.Errorf("Log should be in success subdirectory, got path: %s", logPath)
	}
}

func TestWritePerFileLog_Failed(t *testing.T) {
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

	result := models.JobResult{
		Job:          job,
		WorkerID:     1,
		Success:      false,
		ExitCode:     1,
		ErrorMessage: "execution failed",
		StartTime:    time.Now(),
		EndTime:      time.Now().Add(500 * time.Millisecond),
		Duration:     500 * time.Millisecond,
		Stdout:       "",
		Stderr:       "error output",
	}

	execConfig := &models.ExecutableConfig{
		Path:                "test.exe",
		DefaultArgs:         []string{"-input", "{input}"},
		EnvironmentExpanded: map[string]string{},
	}

	logPath, err := logger.WritePerFileLog(result, execConfig)
	if err != nil {
		t.Fatalf("Failed to write per-file log: %v", err)
	}

	// Verify file is in failed subdirectory
	if !strings.Contains(logPath, string(filepath.Separator)+"failed"+string(filepath.Separator)) {
		t.Errorf("Log should be in failed subdirectory, got path: %s", logPath)
	}

	// Verify log content
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read per-file log: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "FAILED") {
		t.Error("Log should contain FAILED status")
	}
}

func TestWritePerFileLog_Timeout(t *testing.T) {
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

	result := models.JobResult{
		Job:       job,
		WorkerID:  1,
		Success:   false,
		TimedOut:  true,
		ExitCode:  -1,
		StartTime: time.Now(),
		EndTime:   time.Now().Add(5 * time.Second),
		Duration:  5 * time.Second,
	}

	execConfig := &models.ExecutableConfig{
		Path:                "test.exe",
		DefaultArgs:         []string{"-input", "{input}"},
		EnvironmentExpanded: map[string]string{},
	}

	logPath, err := logger.WritePerFileLog(result, execConfig)
	if err != nil {
		t.Fatalf("Failed to write per-file log: %v", err)
	}

	// Verify file is in timeout subdirectory
	if !strings.Contains(logPath, string(filepath.Separator)+"timeout"+string(filepath.Separator)) {
		t.Errorf("Log should be in timeout subdirectory, got path: %s", logPath)
	}

	// Verify log content
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read per-file log: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "TIMEOUT") {
		t.Error("Log should contain TIMEOUT status")
	}
}

func TestWritePerFileLog_Disabled(t *testing.T) {
	logger, err := New("", "")
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	job := &models.Job{
		FilePath: "C:\\data\\test.csv",
		FileName: "test.csv",
	}

	result := models.JobResult{
		Job:       job,
		WorkerID:  1,
		Success:   true,
		ExitCode:  0,
		StartTime: time.Now(),
		EndTime:   time.Now().Add(500 * time.Millisecond),
		Duration:  500 * time.Millisecond,
	}

	execConfig := &models.ExecutableConfig{
		Path:        "test.exe",
		DefaultArgs: []string{"-input", "{input}"},
	}

	logPath, err := logger.WritePerFileLog(result, execConfig)
	if err != nil {
		t.Fatalf("Should not error when per-file logging disabled: %v", err)
	}

	if logPath != "" {
		t.Error("Log path should be empty when per-file logging disabled")
	}
}

func TestAddTimestampToFilename(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantBase string
		wantExt  string
	}{
		{
			name:     "simple log file",
			input:    "dispatcher.log",
			wantBase: "dispatcher_",
			wantExt:  ".log",
		},
		{
			name:     "full path with extension",
			input:    "C:\\data\\logs\\batch.log",
			wantBase: "C:\\data\\logs\\batch_",
			wantExt:  ".log",
		},
		{
			name:     "file without extension",
			input:    "logfile",
			wantBase: "logfile_",
			wantExt:  "",
		},
		{
			name:     "multiple dots in name",
			input:    "app.dispatcher.log",
			wantBase: "app.dispatcher_",
			wantExt:  ".log",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := addTimestampToFilename(tt.input)

			// Check that it starts with the expected base
			if !strings.HasPrefix(result, tt.wantBase) {
				t.Errorf("Expected result to start with %q, got %q", tt.wantBase, result)
			}

			// Check that it ends with the expected extension
			if !strings.HasSuffix(result, tt.wantExt) {
				t.Errorf("Expected result to end with %q, got %q", tt.wantExt, result)
			}

			// Check that timestamp is in the middle (format: 20060102-150405)
			// Extract the middle part between base and extension
			middle := result[len(tt.wantBase) : len(result)-len(tt.wantExt)]
			if len(middle) != 15 { // YYYYMMDD-HHMMSS = 15 chars
				t.Errorf("Expected timestamp length 15, got %d in %q", len(middle), middle)
			}

			// Verify timestamp format (digits and dash)
			if len(middle) > 0 && middle[8] != '-' {
				t.Errorf("Expected dash at position 8 in timestamp, got %q", middle)
			}
		})
	}
}

func TestNew_CreatesTimestampedLogFile(t *testing.T) {
	tmpDir := t.TempDir()
	baseLogFile := filepath.Join(tmpDir, "test.log")

	// Create first logger
	logger1, err := New(baseLogFile, "")
	if err != nil {
		t.Fatalf("Failed to create first logger: %v", err)
	}
	logger1.Close()

	// Small delay to ensure different timestamp
	time.Sleep(1 * time.Second)

	// Create second logger with same base name
	logger2, err := New(baseLogFile, "")
	if err != nil {
		t.Fatalf("Failed to create second logger: %v", err)
	}
	logger2.Close()

	// Check that multiple timestamped files were created
	files, err := filepath.Glob(filepath.Join(tmpDir, "test_*.log"))
	if err != nil {
		t.Fatalf("Failed to list log files: %v", err)
	}

	if len(files) < 2 {
		t.Errorf("Expected at least 2 timestamped log files, got %d: %v", len(files), files)
	}

	// Verify no file named exactly "test.log" exists (should all have timestamps)
	if _, err := os.Stat(baseLogFile); !os.IsNotExist(err) {
		t.Error("Base log file without timestamp should not exist")
	}
}

func TestNew_TimestampedLogsAreIndependent(t *testing.T) {
	tmpDir := t.TempDir()
	baseLogFile := filepath.Join(tmpDir, "test.log")

	// Create first logger and write message
	logger1, err := New(baseLogFile, "")
	if err != nil {
		t.Fatalf("Failed to create first logger: %v", err)
	}
	logger1.Info("First message")
	logger1.Close()

	// Small delay to ensure different timestamp
	time.Sleep(1 * time.Second)

	// Create second logger and write different message
	logger2, err := New(baseLogFile, "")
	if err != nil {
		t.Fatalf("Failed to create second logger: %v", err)
	}
	logger2.Info("Second message")
	logger2.Close()

	// Get all timestamped files
	files, err := filepath.Glob(filepath.Join(tmpDir, "test_*.log"))
	if err != nil {
		t.Fatalf("Failed to list log files: %v", err)
	}

	if len(files) != 2 {
		t.Errorf("Expected exactly 2 timestamped log files, got %d: %v", len(files), files)
	}

	// Read and verify first file contains only first message
	content1, err := os.ReadFile(files[0])
	if err != nil {
		t.Fatalf("Failed to read first log file: %v", err)
	}
	content1Str := string(content1)
	if !strings.Contains(content1Str, "First message") {
		t.Error("First log file should contain 'First message'")
	}
	if strings.Contains(content1Str, "Second message") {
		t.Error("First log file should not contain 'Second message'")
	}

	// Read and verify second file contains only second message
	content2, err := os.ReadFile(files[1])
	if err != nil {
		t.Fatalf("Failed to read second log file: %v", err)
	}
	content2Str := string(content2)
	if !strings.Contains(content2Str, "Second message") {
		t.Error("Second log file should contain 'Second message'")
	}
	if strings.Contains(content2Str, "First message") {
		t.Error("Second log file should not contain 'First message'")
	}
}
