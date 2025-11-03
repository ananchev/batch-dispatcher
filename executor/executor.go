package executor

import (
	"batch-dispatcher/models"
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"time"
)

// ExecuteJob runs an external executable with the given job configuration
// Returns a JobResult with execution details, output, and any errors
func ExecuteJob(job *models.Job, execConfig *models.ExecutableConfig, workerID int) models.JobResult {
	startTime := time.Now()
	result := models.JobResult{
		Job:       job,
		WorkerID:  workerID,
		StartTime: startTime,
		Success:   false,
		TimedOut:  false,
	}

	// Create context with timeout (0 means infinite)
	var ctx context.Context
	var cancel context.CancelFunc
	if execConfig.TimeoutDuration > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), execConfig.TimeoutDuration)
	} else {
		ctx, cancel = context.WithCancel(context.Background())
	}
	defer cancel()

	// Build command with executable and arguments
	// Replace {input} placeholder with actual input file path
	args := buildArguments(execConfig.DefaultArgs, job.FilePath)
	cmd := exec.CommandContext(ctx, execConfig.Path, args...)

	// Set working directory if specified
	if execConfig.WorkingDirectory != "" {
		cmd.Dir = execConfig.WorkingDirectory
	}

	// Set up environment variables (use expanded environment)
	if len(execConfig.EnvironmentExpanded) > 0 {
		cmd.Env = buildEnvironment(execConfig.EnvironmentExpanded)
	}

	// Capture stdout and stderr
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Execute the command
	err := cmd.Run()

	// Calculate duration and set end time
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	// Determine result status
	if ctx.Err() == context.DeadlineExceeded {
		// Timeout occurred
		result.TimedOut = true
		result.ErrorMessage = fmt.Sprintf("execution timed out after %v", execConfig.TimeoutDuration)
		result.ExitCode = -1
		return result
	}

	if err != nil {
		// Command failed
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.ExitCode = -1
		}
		// Combine error message with stderr output
		if stderr.Len() > 0 {
			result.ErrorMessage = fmt.Sprintf("%s\nstderr: %s", err.Error(), stderr.String())
		} else {
			result.ErrorMessage = err.Error()
		}
		return result
	}

	// Success
	result.Success = true
	result.ExitCode = 0
	return result
}

// buildArguments replaces {input} placeholder in arguments with actual input file path
// Example: ["--input", "{input}", "--output", "result.txt"] with inputFile "C:\data\test.csv"
//
//	produces: ["--input", "C:\data\test.csv", "--output", "result.txt"]
func buildArguments(argTemplates []string, inputFile string) []string {
	// Create new slice with same length (no growing needed)
	args := make([]string, len(argTemplates))
	for i, arg := range argTemplates {
		// Exact match: if argument equals "{input}", replace with file path
		if arg == "{input}" {
			args[i] = inputFile
		} else {
			// Otherwise copy argument unchanged
			args[i] = arg
		}
	}
	return args
}

// buildEnvironment creates environment variable array from map
// Returns array in "KEY=VALUE" format required by exec.Cmd
func buildEnvironment(envMap map[string]string) []string {
	env := make([]string, 0, len(envMap))
	for key, value := range envMap {
		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}
	return env
}
