package config

import (
	"batch-dispatcher/models"
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Load loads and parses the YAML configuration file
func Load(configPath string) (*models.Config, error) {
	// Read config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse YAML
	var config models.Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Validate required fields
	if config.Executable.Path == "" {
		return nil, fmt.Errorf("executable.path is required")
	}
	if config.Executable.Timeout == "" {
		return nil, fmt.Errorf("executable.timeout is required")
	}
	if config.Input.SourceDirectory == "" {
		return nil, fmt.Errorf("input.source_directory is required")
	}
	if config.Input.FilePattern == "" {
		return nil, fmt.Errorf("input.file_pattern is required")
	}
	if config.Output.ProcessedDirectory == "" {
		return nil, fmt.Errorf("output.processed_directory is required")
	}
	if config.Output.ErrorsDirectory == "" {
		return nil, fmt.Errorf("output.errors_directory is required")
	}
	if config.Workers.Count < 1 {
		return nil, fmt.Errorf("workers.count must be >= 1, got %d", config.Workers.Count)
	}
	if config.Input.MaxFiles < 0 {
		return nil, fmt.Errorf("input.max_files cannot be negative, got %d", config.Input.MaxFiles)
	}

	// Expand environment variables in working_directory
	if config.Executable.WorkingDirectory != "" {
		sysEnv := getSystemEnvironment()
		sysEnvUpper := make(map[string]string)
		for k, v := range sysEnv {
			sysEnvUpper[strings.ToUpper(k)] = v
		}
		expanded, err := expandString(config.Executable.WorkingDirectory, make(map[string]string), sysEnvUpper)
		if err != nil {
			return nil, fmt.Errorf("failed to expand working_directory: %w", err)
		}
		config.Executable.WorkingDirectory = expanded
	}

	// Expand environment variables
	if config.Executable.Environment != nil {
		expanded, err := expandEnvironmentVariables(config.Executable.Environment)
		if err != nil {
			return nil, fmt.Errorf("failed to expand environment variables: %w", err)
		}
		config.Executable.EnvironmentExpanded = expanded
	} else {
		config.Executable.EnvironmentExpanded = make(map[string]string)
	}

	// Parse timeout duration
	timeoutDuration, err := parseDuration(config.Executable.Timeout, "executable.timeout")
	if err != nil {
		return nil, err
	}
	if timeoutDuration < 0 {
		return nil, fmt.Errorf("executable.timeout cannot be negative: got %s", timeoutDuration)
	}
	config.Executable.TimeoutDuration = timeoutDuration

	return &config, nil
}

// expandEnvironmentVariables expands ${VAR} references in environment map
// Uses case-insensitive lookup for Windows compatibility
func expandEnvironmentVariables(envWithPlaceholders map[string]string) (map[string]string, error) {
	// Get system environment
	sysEnv := getSystemEnvironment()

	// Create case-insensitive lookup map for system environment
	sysEnvUpper := make(map[string]string)
	for k, v := range sysEnv {
		sysEnvUpper[strings.ToUpper(k)] = v
	}

	// Create result map and its uppercase lookup
	result := make(map[string]string)
	resultUpper := make(map[string]string)

	// Expand each config variable
	for key, value := range envWithPlaceholders {
		// Expand variables in value
		expandedValue, err := expandString(value, resultUpper, sysEnvUpper)
		if err != nil {
			return nil, err
		}

		// Store result
		result[key] = expandedValue
		resultUpper[strings.ToUpper(key)] = expandedValue
	}

	return result, nil
}

// expandString expands ${VAR} placeholders in a string
func expandString(value string, configEnvUpper map[string]string, sysEnvUpper map[string]string) (string, error) {
	result := value
	undefinedVars := []string{}

	// Use os.Expand for ${VAR} syntax
	result = os.Expand(result, func(varName string) string {
		if varName == "" {
			undefinedVars = append(undefinedVars, "${}")
			return ""
		}

		// Try case-insensitive lookup in config vars first
		varNameUpper := strings.ToUpper(varName)
		if val, ok := configEnvUpper[varNameUpper]; ok {
			return val
		}

		// Try system environment
		if val, ok := sysEnvUpper[varNameUpper]; ok {
			return val
		}

		// Variable not found
		undefinedVars = append(undefinedVars, fmt.Sprintf("${%s}", varName))
		return ""
	})

	// Report undefined variables
	if len(undefinedVars) > 0 {
		return "", fmt.Errorf("undefined variable(s): %s", strings.Join(undefinedVars, ", "))
	}

	return result, nil
}

// parseDuration parses a duration string with helpful error messages
func parseDuration(durationStr string, fieldName string) (time.Duration, error) {
	durationStr = strings.TrimSpace(durationStr)

	// Parse using time.ParseDuration
	duration, err := time.ParseDuration(durationStr)
	if err != nil {
		return 0, fmt.Errorf("invalid duration format for %s: %w", fieldName, err)
	}

	return duration, nil
}

// getSystemEnvironment returns current system environment as a map
func getSystemEnvironment() map[string]string {
	env := make(map[string]string)
	for _, e := range os.Environ() {
		pair := strings.SplitN(e, "=", 2)
		if len(pair) == 2 {
			env[pair[0]] = pair[1]
		}
	}
	return env
}
