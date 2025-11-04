package executor

import (
	"batch-dispatcher/models"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

// TestExecuteJob_Success tests successful command execution
func TestExecuteJob_Success(t *testing.T) {
	// Create a simple command that succeeds
	var execPath, arg string
	if runtime.GOOS == "windows" {
		execPath = "cmd.exe"
		arg = "/c"
	} else {
		execPath = "sh"
		arg = "-c"
	}

	execConfig := &models.ExecutableConfig{
		Path:                execPath,
		DefaultArgs:         []string{arg, "echo hello"},
		EnvironmentExpanded: map[string]string{},
		TimeoutDuration:     5 * time.Second,
	}

	job := &models.Job{
		FilePath: "test.csv",
		FileName: "test.csv",
	}

	result := ExecuteJob(job, execConfig, 1)

	if !result.Success {
		t.Errorf("Expected success, got failure: %v", result.ErrorMessage)
	}
	if result.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", result.ExitCode)
	}
	if result.TimedOut {
		t.Error("Command should not have timed out")
	}
	if result.Duration <= 0 {
		t.Error("Duration should be positive")
	}
	if result.WorkerID != 1 {
		t.Errorf("WorkerID = %d, want 1", result.WorkerID)
	}
}

// TestExecuteJob_Timeout tests command timeout handling
func TestExecuteJob_Timeout(t *testing.T) {
	// Create a command that sleeps longer than timeout
	var execPath string
	var args []string
	if runtime.GOOS == "windows" {
		execPath = "cmd.exe"
		args = []string{"/c", "timeout", "10"}
	} else {
		execPath = "sleep"
		args = []string{"10"}
	}

	execConfig := &models.ExecutableConfig{
		Path:                execPath,
		DefaultArgs:         args,
		EnvironmentExpanded: map[string]string{},
		TimeoutDuration:     100 * time.Millisecond,
	}

	job := &models.Job{
		FilePath: "test.csv",
		FileName: "test.csv",
	}

	result := ExecuteJob(job, execConfig, 1)

	if result.Success {
		t.Error("Command should have failed due to timeout")
	}
	if !result.TimedOut {
		t.Error("TimedOut should be true")
	}
	if result.ExitCode != -1 {
		t.Errorf("ExitCode = %d, want -1 for timeout", result.ExitCode)
	}
}

// TestExecuteJob_NonZeroExit tests handling of non-zero exit codes
func TestExecuteJob_NonZeroExit(t *testing.T) {
	var execPath string
	var args []string
	if runtime.GOOS == "windows" {
		execPath = "cmd.exe"
		args = []string{"/c", "exit", "42"}
	} else {
		execPath = "sh"
		args = []string{"-c", "exit 42"}
	}

	execConfig := &models.ExecutableConfig{
		Path:                execPath,
		DefaultArgs:         args,
		EnvironmentExpanded: map[string]string{},
		TimeoutDuration:     5 * time.Second,
	}

	job := &models.Job{
		FilePath: "test.csv",
		FileName: "test.csv",
	}

	result := ExecuteJob(job, execConfig, 1)

	if result.Success {
		t.Error("Command should have failed with non-zero exit")
	}
	if result.ExitCode != 42 {
		t.Errorf("ExitCode = %d, want 42", result.ExitCode)
	}
	if result.TimedOut {
		t.Error("Command should not have timed out")
	}
}

// TestExecuteJob_CapturesOutput tests that command runs (we don't capture stdout/stderr separately in JobResult)
func TestExecuteJob_CapturesOutput(t *testing.T) {
	var execPath string
	var args []string
	if runtime.GOOS == "windows" {
		execPath = "cmd.exe"
		args = []string{"/c", "echo test output"}
	} else {
		execPath = "echo"
		args = []string{"test output"}
	}

	execConfig := &models.ExecutableConfig{
		Path:                execPath,
		DefaultArgs:         args,
		EnvironmentExpanded: map[string]string{},
		TimeoutDuration:     5 * time.Second,
	}

	job := &models.Job{
		FilePath: "test.csv",
		FileName: "test.csv",
	}

	result := ExecuteJob(job, execConfig, 1)

	if !result.Success {
		t.Errorf("Command failed: %v", result.ErrorMessage)
	}
}

// TestExecuteJob_InputPlaceholder tests {input} placeholder replacement
func TestExecuteJob_InputPlaceholder(t *testing.T) {
	tmpDir := t.TempDir()
	inputFile := filepath.Join(tmpDir, "test.csv")

	// Create test input file
	if err := os.WriteFile(inputFile, []byte("test data"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	var execPath string
	var args []string
	if runtime.GOOS == "windows" {
		execPath = "cmd.exe"
		args = []string{"/c", "type", "{input}"}
	} else {
		execPath = "cat"
		args = []string{"{input}"}
	}

	execConfig := &models.ExecutableConfig{
		Path:                execPath,
		DefaultArgs:         args,
		EnvironmentExpanded: map[string]string{},
		TimeoutDuration:     5 * time.Second,
	}

	job := &models.Job{
		FilePath: inputFile,
		FileName: "test.csv",
	}

	result := ExecuteJob(job, execConfig, 1)

	if !result.Success {
		t.Errorf("Command failed: %v", result.ErrorMessage)
	}
}

// TestBuildArguments tests argument template replacement
func TestBuildArguments(t *testing.T) {
	tests := []struct {
		name     string
		template []string
		input    string
		expected []string
	}{
		{
			name:     "No placeholder",
			template: []string{"arg1", "arg2"},
			input:    "/path/to/file.csv",
			expected: []string{"arg1", "arg2"},
		},
		{
			name:     "Single placeholder",
			template: []string{"{input}"},
			input:    "/path/to/file.csv",
			expected: []string{"/path/to/file.csv"},
		},
		{
			name:     "Multiple args with placeholder",
			template: []string{"--input", "{input}", "--output", "result.txt"},
			input:    "/path/to/file.csv",
			expected: []string{"--input", "/path/to/file.csv", "--output", "result.txt"},
		},
		{
			name:     "Embedded placeholder with equals",
			template: []string{"-inp={input}", "--flag"},
			input:    "/path/to/file.csv",
			expected: []string{"-inp=/path/to/file.csv", "--flag"},
		},
		{
			name:     "Multiple embedded placeholders",
			template: []string{"-inp={input}", "-out={input}.out"},
			input:    "/path/to/file.csv",
			expected: []string{"-inp=/path/to/file.csv", "-out=/path/to/file.csv.out"},
		},
		{
			name:     "Windows path with embedded placeholder",
			template: []string{"-file={input}"},
			input:    "C:\\data\\test.csv",
			expected: []string{"-file=C:\\data\\test.csv"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildArguments(tt.template, tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("Length = %d, want %d", len(result), len(tt.expected))
				return
			}
			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("Arg[%d] = %q, want %q", i, result[i], tt.expected[i])
				}
			}
		})
	}
}

// TestBuildEnvironment tests environment variable array creation
func TestBuildEnvironment(t *testing.T) {
	envMap := map[string]string{
		"VAR1": "value1",
		"VAR2": "value2",
	}

	result := buildEnvironment(envMap)

	if len(result) != 2 {
		t.Errorf("Length = %d, want 2", len(result))
	}

	// Check that both key-value pairs exist (order may vary due to map)
	found := make(map[string]bool)
	for _, env := range result {
		if env == "VAR1=value1" {
			found["VAR1"] = true
		}
		if env == "VAR2=value2" {
			found["VAR2"] = true
		}
	}

	if !found["VAR1"] || !found["VAR2"] {
		t.Errorf("Missing expected environment variables in %v", result)
	}
}
