package filesystem

import (
	"fmt"
	"os"
	"path/filepath"
)

// MoveToProcessed moves a file to the processed folder
// workerID is used to create a worker-specific subfolder: processed_worker01, processed_worker02, etc.
// Returns the final destination path
func MoveToProcessed(filePath string, processedFolder string, workerID int) (string, error) {
	// Create worker-specific subfolder
	workerFolder := fmt.Sprintf("%s_worker%02d", processedFolder, workerID)
	return moveFile(filePath, workerFolder)
}

// MoveToErrors moves a file to the errors folder
// workerID is used to create a worker-specific subfolder: errors_worker01, errors_worker02, etc.
// Returns the final destination path
func MoveToErrors(filePath string, errorsFolder string, workerID int) (string, error) {
	// Create worker-specific subfolder
	workerFolder := fmt.Sprintf("%s_worker%02d", errorsFolder, workerID)
	return moveFile(filePath, workerFolder)
}

// EnsureFolderExists creates a folder if it doesn't exist
func EnsureFolderExists(folderPath string) error {
	// Check if folder already exists
	info, err := os.Stat(folderPath)
	if err == nil {
		// Path exists, verify it's a directory
		if !info.IsDir() {
			return fmt.Errorf("path exists but is not a directory: %s", folderPath)
		}
		return nil
	}

	// Check if error is "not exists"
	if !os.IsNotExist(err) {
		return fmt.Errorf("failed to check folder '%s': %w", folderPath, err)
	}

	// Folder doesn't exist, create it (with parent directories)
	if err := os.MkdirAll(folderPath, 0755); err != nil {
		return fmt.Errorf("failed to create folder '%s': %w", folderPath, err)
	}

	return nil
}

// moveFile is the common implementation for moving files
// Files are moved to worker-specific folders, keeping original filenames
func moveFile(filePath string, destFolder string) (string, error) {
	// Verify source file exists
	if _, err := os.Stat(filePath); err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("source file does not exist: %s", filePath)
		}
		return "", fmt.Errorf("failed to access source file '%s': %w", filePath, err)
	}

	// Ensure destination folder exists
	if err := EnsureFolderExists(destFolder); err != nil {
		return "", err
	}

	// Build destination path with original filename
	fileName := filepath.Base(filePath)
	destPath := filepath.Join(destFolder, fileName)

	// Check if destination already exists (should be rare with worker-specific folders)
	if _, err := os.Stat(destPath); err == nil {
		return "", fmt.Errorf("destination file already exists: %s", destPath)
	}

	// Attempt atomic move with os.Rename
	err := os.Rename(filePath, destPath)
	if err != nil {
		return "", fmt.Errorf("failed to move file from '%s' to '%s': %w", filePath, destPath, err)
	}

	return destPath, nil
}
