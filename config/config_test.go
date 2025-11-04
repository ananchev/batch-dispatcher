package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLoad_ValidConfig(t *testing.T) {
	configContent := `
executable:
  path: "C:/tool.exe"
  timeout: "30m"
input:
  source_directory: "C:/input"
  file_pattern: "*.csv"
output:
  processed_directory: "C:/processed"
  errors_directory: "C:/errors"
workers:
  count: 4
logging:
  log_file: "test.log"
advanced:
  show_progress: true
`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Executable.Path != "C:/tool.exe" {
		t.Errorf("Path = %q", cfg.Executable.Path)
	}
	if cfg.Executable.TimeoutDuration != 30*time.Minute {
		t.Errorf("Timeout = %v", cfg.Executable.TimeoutDuration)
	}
	if cfg.Workers.Count != 4 {
		t.Errorf("Workers = %d", cfg.Workers.Count)
	}
}

// TestLoad_WithFlags tests configuration with flags (no values)
func TestLoad_WithFlags(t *testing.T) {
	configContent := `
executable:
  path: "tool.exe"
  default_args:
    - "--verbose"
    - "--force"
    - "-q"
  timeout: "5m"
input:
  source_directory: "C:/input"
  file_pattern: "*.txt"
output:
  processed_directory: "C:/processed"
  errors_directory: "C:/errors"
workers:
  count: 2
logging:
  log_file: "test.log"
advanced:
  show_progress: false
`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Verify flags
	if len(cfg.Executable.DefaultArgs) != 3 {
		t.Errorf("Expected 3 default args, got %d", len(cfg.Executable.DefaultArgs))
	}
	expectedArgs := []string{"--verbose", "--force", "-q"}
	for i, arg := range expectedArgs {
		if cfg.Executable.DefaultArgs[i] != arg {
			t.Errorf("Arg[%d] = %q, want %q", i, cfg.Executable.DefaultArgs[i], arg)
		}
	}
}

// TestLoad_WithParametersAndValues tests configuration with parameters that have values
func TestLoad_WithParametersAndValues(t *testing.T) {
	configContent := `
executable:
  path: "converter.exe"
  default_args:
    - "--input"
    - "{input}"
    - "--output-format"
    - "json"
    - "--log-level"
    - "debug"
    - "-c"
    - "config.xml"
  timeout: "10m"
input:
  source_directory: "C:/data"
  file_pattern: "*.dat"
  max_files: 100
output:
  processed_directory: "C:/done"
  errors_directory: "C:/failed"
workers:
  count: 8
logging:
  log_file: "converter.log"
  per_file_log_directory: "C:/logs"
advanced:
  show_progress: true
`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Verify parameters with values
	expectedArgs := []string{"--input", "{input}", "--output-format", "json", "--log-level", "debug", "-c", "config.xml"}
	if len(cfg.Executable.DefaultArgs) != len(expectedArgs) {
		t.Errorf("Expected %d args, got %d", len(expectedArgs), len(cfg.Executable.DefaultArgs))
	}
	for i, arg := range expectedArgs {
		if cfg.Executable.DefaultArgs[i] != arg {
			t.Errorf("Arg[%d] = %q, want %q", i, cfg.Executable.DefaultArgs[i], arg)
		}
	}

	// Verify {input} placeholder is preserved
	if cfg.Executable.DefaultArgs[1] != "{input}" {
		t.Errorf("Expected {input} placeholder, got %q", cfg.Executable.DefaultArgs[1])
	}

	// Verify max_files
	if cfg.Input.MaxFiles != 100 {
		t.Errorf("MaxFiles = %d, want 100", cfg.Input.MaxFiles)
	}

	// Verify per_file_log_directory
	if cfg.Logging.PerFileLogDirectory != "C:/logs" {
		t.Errorf("PerFileLogDirectory = %q, want %q", cfg.Logging.PerFileLogDirectory, "C:/logs")
	}
}

// TestLoad_WithEnvironmentVariables tests environment variable expansion
func TestLoad_WithEnvironmentVariables(t *testing.T) {
	// Set test environment variables
	os.Setenv("TEST_JAVA_HOME", "C:/Java/jdk-17")
	os.Setenv("TEST_CUSTOM_PATH", "C:/CustomTools")
	defer os.Unsetenv("TEST_JAVA_HOME")
	defer os.Unsetenv("TEST_CUSTOM_PATH")

	configContent := `
executable:
  path: "processor.jar"
  working_directory: "${TEST_JAVA_HOME}/bin"
  environment:
    JAVA_HOME: "${TEST_JAVA_HOME}"
    CUSTOM_PATH: "${TEST_CUSTOM_PATH}"
    COMBINED: "${TEST_JAVA_HOME}/lib:${TEST_CUSTOM_PATH}/bin"
  timeout: "15m"
input:
  source_directory: "C:/input"
  file_pattern: "*.xml"
output:
  processed_directory: "C:/processed"
  errors_directory: "C:/errors"
workers:
  count: 3
logging:
  log_file: "test.log"
advanced:
  show_progress: true
`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Verify environment variable expansion
	if cfg.Executable.EnvironmentExpanded["JAVA_HOME"] != "C:/Java/jdk-17" {
		t.Errorf("JAVA_HOME = %q, want %q", cfg.Executable.EnvironmentExpanded["JAVA_HOME"], "C:/Java/jdk-17")
	}
	if cfg.Executable.EnvironmentExpanded["CUSTOM_PATH"] != "C:/CustomTools" {
		t.Errorf("CUSTOM_PATH = %q, want %q", cfg.Executable.EnvironmentExpanded["CUSTOM_PATH"], "C:/CustomTools")
	}
	if !strings.Contains(cfg.Executable.EnvironmentExpanded["COMBINED"], "C:/Java/jdk-17/lib") {
		t.Errorf("COMBINED should contain expanded JAVA_HOME, got %q", cfg.Executable.EnvironmentExpanded["COMBINED"])
	}
	if !strings.Contains(cfg.Executable.EnvironmentExpanded["COMBINED"], "C:/CustomTools/bin") {
		t.Errorf("COMBINED should contain expanded CUSTOM_PATH, got %q", cfg.Executable.EnvironmentExpanded["COMBINED"])
	}

	// Verify working directory expansion
	if cfg.Executable.WorkingDirectory != "C:/Java/jdk-17/bin" {
		t.Errorf("WorkingDirectory = %q, want %q", cfg.Executable.WorkingDirectory, "C:/Java/jdk-17/bin")
	}
}

// TestLoad_WithInfiniteTimeout tests timeout set to 0 (infinite)
func TestLoad_WithInfiniteTimeout(t *testing.T) {
	configContent := `
executable:
  path: "long-runner.exe"
  timeout: "0"
input:
  source_directory: "C:/input"
  file_pattern: "*.bin"
output:
  processed_directory: "C:/processed"
  errors_directory: "C:/errors"
workers:
  count: 1
logging:
  log_file: "test.log"
advanced:
  show_progress: false
`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Executable.TimeoutDuration != 0 {
		t.Errorf("TimeoutDuration = %v, want 0 (infinite)", cfg.Executable.TimeoutDuration)
	}
}

// NEGATIVE TESTS

// TestLoad_MissingFile tests error when config file doesn't exist
func TestLoad_MissingFile(t *testing.T) {
	_, err := Load("nonexistent-file-12345.yaml")
	if err == nil {
		t.Error("Expected error for missing file, got nil")
	}
}

// TestLoad_InvalidYAML tests error on malformed YAML
func TestLoad_InvalidYAML(t *testing.T) {
	configContent := `
executable:
  bad syntax here
    - no proper yaml structure
  missing: colons
`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "bad.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(configPath)
	if err == nil {
		t.Error("Expected error for invalid YAML, got nil")
	}
}

// TestLoad_MissingExecutablePath tests error when executable path is missing
func TestLoad_MissingExecutablePath(t *testing.T) {
	configContent := `
executable:
  timeout: "5m"
input:
  source_directory: "C:/input"
  file_pattern: "*.csv"
output:
  processed_directory: "C:/processed"
  errors_directory: "C:/errors"
workers:
  count: 1
logging:
  log_file: "test.log"
advanced:
  show_progress: false
`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(configPath)
	if err == nil {
		t.Error("Expected error for missing executable path, got nil")
	}
}

// TestLoad_MissingTimeout tests error when timeout is missing
func TestLoad_MissingTimeout(t *testing.T) {
	configContent := `
executable:
  path: "tool.exe"
input:
  source_directory: "C:/input"
  file_pattern: "*.csv"
output:
  processed_directory: "C:/processed"
  errors_directory: "C:/errors"
workers:
  count: 1
logging:
  log_file: "test.log"
advanced:
  show_progress: false
`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(configPath)
	if err == nil {
		t.Error("Expected error for missing timeout, got nil")
	}
}

// TestLoad_InvalidTimeoutFormat tests error on invalid timeout format
func TestLoad_InvalidTimeoutFormat(t *testing.T) {
	configContent := `
executable:
  path: "tool.exe"
  timeout: "invalid-time"
input:
  source_directory: "C:/input"
  file_pattern: "*.csv"
output:
  processed_directory: "C:/processed"
  errors_directory: "C:/errors"
workers:
  count: 1
logging:
  log_file: "test.log"
advanced:
  show_progress: false
`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(configPath)
	if err == nil {
		t.Error("Expected error for invalid timeout format, got nil")
	}
}

// TestLoad_NegativeTimeout tests error on negative timeout
func TestLoad_NegativeTimeout(t *testing.T) {
	configContent := `
executable:
  path: "tool.exe"
  timeout: "-10m"
input:
  source_directory: "C:/input"
  file_pattern: "*.csv"
output:
  processed_directory: "C:/processed"
  errors_directory: "C:/errors"
workers:
  count: 1
logging:
  log_file: "test.log"
advanced:
  show_progress: false
`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(configPath)
	if err == nil {
		t.Error("Expected error for negative timeout, got nil")
	}
}

// TestLoad_MissingInputSection tests error when input section is missing
func TestLoad_MissingInputSection(t *testing.T) {
	configContent := `
executable:
  path: "tool.exe"
  timeout: "5m"
output:
  processed_directory: "C:/processed"
  errors_directory: "C:/errors"
workers:
  count: 1
logging:
  log_file: "test.log"
advanced:
  show_progress: false
`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(configPath)
	if err == nil {
		t.Error("Expected error for missing input section, got nil")
	}
}

// TestLoad_EmptySourceDirectory tests error when source directory is empty
func TestLoad_EmptySourceDirectory(t *testing.T) {
	configContent := `
executable:
  path: "tool.exe"
  timeout: "5m"
input:
  source_directory: ""
  file_pattern: "*.csv"
output:
  processed_directory: "C:/processed"
  errors_directory: "C:/errors"
workers:
  count: 1
logging:
  log_file: "test.log"
advanced:
  show_progress: false
`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(configPath)
	if err == nil {
		t.Error("Expected error for empty source directory, got nil")
	}
}

// TestLoad_EmptyFilePattern tests error when file pattern is empty
func TestLoad_EmptyFilePattern(t *testing.T) {
	configContent := `
executable:
  path: "tool.exe"
  timeout: "5m"
input:
  source_directory: "C:/input"
  file_pattern: ""
output:
  processed_directory: "C:/processed"
  errors_directory: "C:/errors"
workers:
  count: 1
logging:
  log_file: "test.log"
advanced:
  show_progress: false
`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(configPath)
	if err == nil {
		t.Error("Expected error for empty file pattern, got nil")
	}
}

// TestLoad_MissingOutputSection tests error when output section is missing
func TestLoad_MissingOutputSection(t *testing.T) {
	configContent := `
executable:
  path: "tool.exe"
  timeout: "5m"
input:
  source_directory: "C:/input"
  file_pattern: "*.csv"
workers:
  count: 1
logging:
  log_file: "test.log"
advanced:
  show_progress: false
`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(configPath)
	if err == nil {
		t.Error("Expected error for missing output section, got nil")
	}
}

// TestLoad_EmptyProcessedDirectory tests error when processed directory is empty
func TestLoad_EmptyProcessedDirectory(t *testing.T) {
	configContent := `
executable:
  path: "tool.exe"
  timeout: "5m"
input:
  source_directory: "C:/input"
  file_pattern: "*.csv"
output:
  processed_directory: ""
  errors_directory: "C:/errors"
workers:
  count: 1
logging:
  log_file: "test.log"
advanced:
  show_progress: false
`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(configPath)
	if err == nil {
		t.Error("Expected error for empty processed directory, got nil")
	}
}

// TestLoad_EmptyErrorsDirectory tests error when errors directory is empty
func TestLoad_EmptyErrorsDirectory(t *testing.T) {
	configContent := `
executable:
  path: "tool.exe"
  timeout: "5m"
input:
  source_directory: "C:/input"
  file_pattern: "*.csv"
output:
  processed_directory: "C:/processed"
  errors_directory: ""
workers:
  count: 1
logging:
  log_file: "test.log"
advanced:
  show_progress: false
`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(configPath)
	if err == nil {
		t.Error("Expected error for empty errors directory, got nil")
	}
}

// TestLoad_MissingWorkersSection tests error when workers section is missing
func TestLoad_MissingWorkersSection(t *testing.T) {
	configContent := `
executable:
  path: "tool.exe"
  timeout: "5m"
input:
  source_directory: "C:/input"
  file_pattern: "*.csv"
output:
  processed_directory: "C:/processed"
  errors_directory: "C:/errors"
logging:
  log_file: "test.log"
advanced:
  show_progress: false
`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(configPath)
	if err == nil {
		t.Error("Expected error for missing workers section, got nil")
	}
}

// TestLoad_ZeroWorkers tests error when worker count is zero
func TestLoad_ZeroWorkers(t *testing.T) {
	configContent := `
executable:
  path: "tool.exe"
  timeout: "5m"
input:
  source_directory: "C:/input"
  file_pattern: "*.csv"
output:
  processed_directory: "C:/processed"
  errors_directory: "C:/errors"
workers:
  count: 0
logging:
  log_file: "test.log"
advanced:
  show_progress: false
`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(configPath)
	if err == nil {
		t.Error("Expected error for zero workers, got nil")
	}
}

// TestLoad_NegativeWorkers tests error when worker count is negative
func TestLoad_NegativeWorkers(t *testing.T) {
	configContent := `
executable:
  path: "tool.exe"
  timeout: "5m"
input:
  source_directory: "C:/input"
  file_pattern: "*.csv"
output:
  processed_directory: "C:/processed"
  errors_directory: "C:/errors"
workers:
  count: -5
logging:
  log_file: "test.log"
advanced:
  show_progress: false
`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(configPath)
	if err == nil {
		t.Error("Expected error for negative workers, got nil")
	}
}

// TestLoad_NegativeMaxFiles tests error when max_files is negative
func TestLoad_NegativeMaxFiles(t *testing.T) {
	configContent := `
executable:
  path: "tool.exe"
  timeout: "5m"
input:
  source_directory: "C:/input"
  file_pattern: "*.csv"
  max_files: -10
output:
  processed_directory: "C:/processed"
  errors_directory: "C:/errors"
workers:
  count: 1
logging:
  log_file: "test.log"
advanced:
  show_progress: false
`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(configPath)
	if err == nil {
		t.Error("Expected error for negative max_files, got nil")
	}
}

// TestLoad_UndefinedEnvironmentVariable tests error when environment variable is not defined
func TestLoad_UndefinedEnvironmentVariable(t *testing.T) {
	// Make sure variable is not set
	os.Unsetenv("UNDEFINED_VAR_XYZ_123")

	configContent := `
executable:
  path: "tool.exe"
  environment:
    BAD_VAR: "${UNDEFINED_VAR_XYZ_123}"
  timeout: "5m"
input:
  source_directory: "C:/input"
  file_pattern: "*.csv"
output:
  processed_directory: "C:/processed"
  errors_directory: "C:/errors"
workers:
  count: 1
logging:
  log_file: "test.log"
advanced:
  show_progress: false
`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(configPath)
	if err == nil {
		t.Error("Expected error for undefined environment variable, got nil")
	}
}

// TestLoad_FailFastMode tests fail-fast mode (default when continue_on_error is false)
func TestLoad_FailFastMode(t *testing.T) {
	configContent := `
executable:
  path: "tool.exe"
  timeout: "5m"
input:
  source_directory: "C:/input"
  file_pattern: "*.csv"
output:
  processed_directory: "C:/processed"
  errors_directory: "C:/errors"
workers:
  count: 4
logging:
  log_file: "test.log"
advanced:
  show_progress: false
  continue_on_error: false
`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Expected no error for fail-fast mode (continue_on_error=false), got: %v", err)
	}
	if cfg.Advanced.ContinueOnError {
		t.Error("Expected ContinueOnError to be false for fail-fast mode")
	}
}

// TestLoad_ContinueOnErrorMode tests continue_on_error mode can be enabled alone
func TestLoad_ContinueOnErrorMode(t *testing.T) {
	configContent := `
executable:
  path: "tool.exe"
  timeout: "5m"
input:
  source_directory: "C:/input"
  file_pattern: "*.csv"
output:
  processed_directory: "C:/processed"
  errors_directory: "C:/errors"
workers:
  count: 4
logging:
  log_file: "test.log"
advanced:
  show_progress: false
  continue_on_error: true
`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Expected no error for continue_on_error alone, got: %v", err)
	}
	if !cfg.Advanced.ContinueOnError {
		t.Error("Expected ContinueOnError to be true")
	}
}

// TestLoad_DryRunMode tests dry_run mode can be enabled
func TestLoad_DryRunMode(t *testing.T) {
	configContent := `
executable:
  path: "tool.exe"
  timeout: "5m"
input:
  source_directory: "C:/input"
  file_pattern: "*.csv"
output:
  processed_directory: "C:/processed"
  errors_directory: "C:/errors"
workers:
  count: 4
logging:
  log_file: "test.log"
advanced:
  show_progress: false
  dry_run: true
`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Expected no error for dry_run, got: %v", err)
	}
	if !cfg.Advanced.DryRun {
		t.Error("Expected DryRun to be true")
	}
}
