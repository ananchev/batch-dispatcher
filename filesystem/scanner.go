package filesystem

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// ScanInputFolder scans the input folder for files matching the pattern
// Returns sorted list of absolute file paths
// Returns empty list (not error) if no files match pattern
func ScanInputFolder(inputFolder string, pattern string) ([]string, error) {
	// Validate input folder exists and is a directory
	if err := validateInputFolder(inputFolder); err != nil {
		return nil, err
	}

	// Build full glob pattern
	globPattern := filepath.Join(inputFolder, pattern)

	// Find matching files
	matches, err := filepath.Glob(globPattern)
	if err != nil {
		return nil, fmt.Errorf("invalid file pattern '%s': %w", pattern, err)
	}

	// Filter to only include files (not directories)
	var files []string
	for _, match := range matches {
		info, err := os.Stat(match)
		if err != nil {
			// File might have been deleted between Glob and Stat, skip it
			continue
		}
		if !info.IsDir() {
			// Convert to absolute path
			absPath, err := filepath.Abs(match)
			if err != nil {
				return nil, fmt.Errorf("failed to get absolute path for '%s': %w", match, err)
			}
			files = append(files, absPath)
		}
	}

	// Sort alphabetically for predictable processing order
	sort.Strings(files)

	return files, nil
}

// validateInputFolder checks if folder exists and is accessible
func validateInputFolder(inputFolder string) error {
	// Check if path exists
	info, err := os.Stat(inputFolder)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("input folder does not exist: %s", inputFolder)
		}
		return fmt.Errorf("failed to access input folder '%s': %w", inputFolder, err)
	}

	// Check if path is a directory
	if !info.IsDir() {
		return fmt.Errorf("input folder is not a directory: %s", inputFolder)
	}

	return nil
}

// CountFileLines counts the number of lines in a file, excluding the header (first line)
// Returns the count of data lines (total lines - 1 for header)
// Returns 0 if file is empty or has only a header line
func CountFileLines(filePath string) (int, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, fmt.Errorf("failed to open file for line counting: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineCount := 0

	for scanner.Scan() {
		lineCount++
	}

	if err := scanner.Err(); err != nil {
		return 0, fmt.Errorf("error reading file for line counting: %w", err)
	}

	// Subtract 1 for header, but don't go below 0
	dataLines := lineCount - 1
	if dataLines < 0 {
		dataLines = 0
	}

	return dataLines, nil
}
