package models

import (
	"fmt"
	"time"
)

// Config represents the root configuration structure from YAML
type Config struct {
	Input      InputConfig      `yaml:"input"`
	Output     OutputConfig     `yaml:"output"`
	Workers    WorkersConfig    `yaml:"workers"`
	Executable ExecutableConfig `yaml:"executable"`
	Logging    LoggingConfig    `yaml:"logging"`
	Advanced   AdvancedConfig   `yaml:"advanced"`
}

// InputConfig defines input file configuration
type InputConfig struct {
	SourceDirectory string `yaml:"source_directory"`
	FilePattern     string `yaml:"file_pattern"`
	MaxFiles        int    `yaml:"max_files"`
}

// OutputConfig defines output directories
type OutputConfig struct {
	ProcessedDirectory string `yaml:"processed_directory"`
	ErrorsDirectory    string `yaml:"errors_directory"`
}

// WorkersConfig defines worker pool settings
type WorkersConfig struct {
	Count int `yaml:"count"`
}

// LoggingConfig defines logging settings
type LoggingConfig struct {
	LogFile             string `yaml:"log_file"`
	PerFileLogDirectory string `yaml:"per_file_log_directory"`
}

// AdvancedConfig defines advanced options
type AdvancedConfig struct {
	ShowProgress    bool `yaml:"show_progress"`
	ContinueOnError bool `yaml:"continue_on_error"` // If true, continue processing all files despite failures; if false, stop on first failure
	DryRun          bool `yaml:"dry_run"`           // Preview execution without running
}

// ExecutableConfig defines configuration for the executable
type ExecutableConfig struct {
	// Raw values from YAML
	Path             string            `yaml:"path"`
	WorkingDirectory string            `yaml:"working_directory"`
	Environment      map[string]string `yaml:"environment"` // Raw with ${VAR}
	DefaultArgs      []string          `yaml:"default_args"`
	Timeout          string            `yaml:"timeout"` // e.g., "30m" or "0" for infinite

	// Computed/expanded values (populated during config loading)
	EnvironmentExpanded map[string]string // After ${VAR} expansion
	TimeoutDuration     time.Duration     // Parsed timeout (0 = infinite)
}

// Job represents a single file processing task
type Job struct {
	FilePath   string    // Absolute path to input file
	FileName   string    // Base name of file (for logging)
	FileIndex  int       // Position in queue (1-based for display)
	TotalFiles int       // Total number of files
	QueuedAt   time.Time // When job was queued
	LineCount  int       // Number of data lines in file (excluding header)
}

// String returns a human-readable representation of the job
func (j *Job) String() string {
	return fmt.Sprintf("%s (%d/%d)", j.FileName, j.FileIndex, j.TotalFiles)
}

// JobResult represents the outcome of processing a job
type JobResult struct {
	Job          *Job
	WorkerID     int           // Which worker processed this
	Success      bool          // true only if ExitCode=0 AND !TimedOut
	ExitCode     int           // Process exit code (-1 if TimedOut)
	TimedOut     bool          // true if killed by timeout context
	ErrorMessage string        // Human-readable error description
	StartTime    time.Time     // When execution started
	EndTime      time.Time     // When execution completed
	Duration     time.Duration // Actual execution time
	Stdout       string        // Standard output from execution
	Stderr       string        // Standard error from execution
	LogFilePath  string        // Path to individual log file
	MovedTo      string        // Where file was moved (processed/ or errors/)
	MoveError    string        // Error moving file, if any
}

// String returns a human-readable representation of the result
func (r *JobResult) String() string {
	status := "SUCCESS"
	if r.TimedOut {
		status = "TIMEOUT"
	} else if !r.Success {
		status = "FAILED"
	}
	return fmt.Sprintf("[Worker-%d] %s: %s (%.1fs)", r.WorkerID, status, r.Job.FileName, r.Duration.Seconds())
}

// ExecutionSummary holds final statistics for the entire run
type ExecutionSummary struct {
	TotalFiles     int
	ProcessedFiles int
	SuccessCount   int
	FailedCount    int
	TimeoutCount   int // How many timed out
	StartTime      time.Time
	EndTime        time.Time
	Duration       time.Duration
	FailedJobs     []*JobResult // List of failed jobs for reporting
	TimedOutJobs   []*JobResult // List of timed-out jobs
}

// SuccessRate returns the success rate as a percentage
func (s *ExecutionSummary) SuccessRate() float64 {
	if s.ProcessedFiles == 0 {
		return 0.0
	}
	return (float64(s.SuccessCount) / float64(s.ProcessedFiles)) * 100.0
}

// AvgTimePerFile returns the average time per file in seconds
func (s *ExecutionSummary) AvgTimePerFile() float64 {
	if s.ProcessedFiles == 0 {
		return 0.0
	}
	return s.Duration.Seconds() / float64(s.ProcessedFiles)
}

// FilesPerMinute returns the throughput in files per minute
func (s *ExecutionSummary) FilesPerMinute() float64 {
	if s.Duration.Minutes() == 0 {
		return 0.0
	}
	return float64(s.ProcessedFiles) / s.Duration.Minutes()
}

// RuntimeParams holds CLI parameters passed at runtime
type RuntimeParams struct {
	ConfigPath      string
	ExecutableName  string
	InputFolder     string
	LogFolder       string
	Workers         int
	FilePattern     string
	ContinueOnError bool
	DryRun          bool
	CustomParams    map[string]string // --param key=value pairs
}

// Validate performs basic validation of runtime parameters
func (p *RuntimeParams) Validate() error {
	if p.ConfigPath == "" {
		return fmt.Errorf("config path is required")
	}
	if p.ExecutableName == "" {
		return fmt.Errorf("executable name is required")
	}
	if p.InputFolder == "" {
		return fmt.Errorf("input folder is required")
	}
	if p.Workers < 1 {
		return fmt.Errorf("workers must be >= 1, got %d", p.Workers)
	}
	return nil
}
