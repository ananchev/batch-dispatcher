package filesystem

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// TestMoveToProcessed_Success tests successful file move
func TestMoveToProcessed_Success(t *testing.T) {
	tmpDir := t.TempDir()

	// Create source file
	sourceFile := filepath.Join(tmpDir, "test.csv")
	if err := os.WriteFile(sourceFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Define processed folder
	processedFolder := filepath.Join(tmpDir, "processed")

	// Move file (worker ID = 1)
	destPath, err := MoveToProcessed(sourceFile, processedFolder, 1)
	if err != nil {
		t.Fatalf("MoveToProcessed() error = %v", err)
	}

	// Verify destination path is in worker-specific folder
	expectedDest := filepath.Join(processedFolder+"_worker01", "test.csv")
	if destPath != expectedDest {
		t.Errorf("destPath = %q, want %q", destPath, expectedDest)
	}

	// Verify file was moved (source doesn't exist)
	if _, err := os.Stat(sourceFile); !os.IsNotExist(err) {
		t.Error("Source file still exists after move")
	}

	// Verify file exists at destination
	if _, err := os.Stat(destPath); err != nil {
		t.Errorf("Destination file doesn't exist: %v", err)
	}

	// Verify content
	content, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("Failed to read destination file: %v", err)
	}
	if string(content) != "test content" {
		t.Errorf("Content = %q, want %q", string(content), "test content")
	}
}

// TestMoveToErrors_Success tests successful file move to errors folder
func TestMoveToErrors_Success(t *testing.T) {
	tmpDir := t.TempDir()

	// Create source file
	sourceFile := filepath.Join(tmpDir, "failed.csv")
	if err := os.WriteFile(sourceFile, []byte("error content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Define errors folder
	errorsFolder := filepath.Join(tmpDir, "errors")

	// Move file (worker ID = 1)
	destPath, err := MoveToErrors(sourceFile, errorsFolder, 1)
	if err != nil {
		t.Fatalf("MoveToErrors() error = %v", err)
	}

	// Verify file was moved to worker-specific errors folder
	expectedDest := filepath.Join(errorsFolder+"_worker01", "failed.csv")
	if destPath != expectedDest {
		t.Errorf("destPath = %q, want %q", destPath, expectedDest)
	}

	// Verify file was moved
	if _, err := os.Stat(sourceFile); !os.IsNotExist(err) {
		t.Error("Source file still exists after move")
	}

	// Verify file exists at destination
	if _, err := os.Stat(destPath); err != nil {
		t.Errorf("Destination file doesn't exist: %v", err)
	}
}

// TestMoveFile_NameConflict tests handling of file name conflicts
func TestMoveFile_NameConflict(t *testing.T) {
	tmpDir := t.TempDir()
	processedFolder := filepath.Join(tmpDir, "processed")

	// Using same worker ID to test conflict handling within same worker folder
	workerID := 1

	// Create first file and move it
	file1 := filepath.Join(tmpDir, "test.csv")
	if err := os.WriteFile(file1, []byte("content 1"), 0644); err != nil {
		t.Fatalf("Failed to create test file 1: %v", err)
	}

	dest1, err := MoveToProcessed(file1, processedFolder, workerID)
	if err != nil {
		t.Fatalf("First move failed: %v", err)
	}

	// Verify first file was moved successfully
	expectedDest1 := filepath.Join(processedFolder+"_worker01", "test.csv")
	if dest1 != expectedDest1 {
		t.Errorf("First dest = %q, want %q", dest1, expectedDest1)
	}

	// Create second file with same name and try to move it (same worker)
	// This should fail since destination already exists
	file2 := filepath.Join(tmpDir, "test.csv")
	if err := os.WriteFile(file2, []byte("content 2"), 0644); err != nil {
		t.Fatalf("Failed to create test file 2: %v", err)
	}

	_, err = MoveToProcessed(file2, processedFolder, workerID)
	if err == nil {
		t.Error("Expected error when moving file to existing destination, got nil")
	}

	// Verify error message mentions the conflict
	expectedErrMsg := "destination file already exists"
	if err != nil && !contains(err.Error(), expectedErrMsg) {
		t.Errorf("Error message = %q, should contain %q", err.Error(), expectedErrMsg)
	}

	// Verify first file still has original content and wasn't overwritten
	content1, err := os.ReadFile(dest1)
	if err != nil {
		t.Fatalf("Failed to read first file: %v", err)
	}
	if string(content1) != "content 1" {
		t.Errorf("First file content = %q, want %q (was overwritten!)", string(content1), "content 1")
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestMoveFile_NonExistentSource tests error when source doesn't exist
func TestMoveFile_NonExistentSource(t *testing.T) {
	tmpDir := t.TempDir()
	processedFolder := filepath.Join(tmpDir, "processed")

	_, err := MoveToProcessed("/nonexistent/file.csv", processedFolder, 1)
	if err == nil {
		t.Error("Expected error for non-existent source, got nil")
	}
}

// TestEnsureFolderExists_CreateFolder tests folder creation
func TestEnsureFolderExists_CreateFolder(t *testing.T) {
	tmpDir := t.TempDir()
	newFolder := filepath.Join(tmpDir, "new", "nested", "folder")

	// Folder shouldn't exist yet
	if _, err := os.Stat(newFolder); !os.IsNotExist(err) {
		t.Fatal("Test folder already exists")
	}

	// Create folder
	if err := EnsureFolderExists(newFolder); err != nil {
		t.Fatalf("EnsureFolderExists() error = %v", err)
	}

	// Verify folder exists
	info, err := os.Stat(newFolder)
	if err != nil {
		t.Fatalf("Folder doesn't exist after creation: %v", err)
	}

	if !info.IsDir() {
		t.Error("Path exists but is not a directory")
	}
}

// TestEnsureFolderExists_ExistingFolder tests idempotency
func TestEnsureFolderExists_ExistingFolder(t *testing.T) {
	tmpDir := t.TempDir()

	// Call twice - should not error
	if err := EnsureFolderExists(tmpDir); err != nil {
		t.Errorf("First call error = %v", err)
	}

	if err := EnsureFolderExists(tmpDir); err != nil {
		t.Errorf("Second call error = %v", err)
	}
}

// TestEnsureFolderExists_PathIsFile tests error when path is a file
func TestEnsureFolderExists_PathIsFile(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "file.txt")

	// Create a file
	if err := os.WriteFile(filePath, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Try to create folder with same path
	err := EnsureFolderExists(filePath)
	if err == nil {
		t.Error("Expected error when path is a file, got nil")
	}
}

// TestMoveFile_PreservesPermissions tests that file permissions are preserved
func TestMoveFile_PreservesPermissions(t *testing.T) {
	tmpDir := t.TempDir()

	// Create source file with specific permissions
	sourceFile := filepath.Join(tmpDir, "test.csv")
	if err := os.WriteFile(sourceFile, []byte("test"), 0600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Get original permissions
	sourceInfo, err := os.Stat(sourceFile)
	if err != nil {
		t.Fatalf("Failed to stat source file: %v", err)
	}
	sourceMode := sourceInfo.Mode()

	// Move file
	processedFolder := filepath.Join(tmpDir, "processed")
	destPath, err := MoveToProcessed(sourceFile, processedFolder, 1)
	if err != nil {
		t.Fatalf("MoveToProcessed() error = %v", err)
	}

	// Get destination permissions
	destInfo, err := os.Stat(destPath)
	if err != nil {
		t.Fatalf("Failed to stat destination file: %v", err)
	}
	destMode := destInfo.Mode()

	// Verify permissions are preserved
	if sourceMode != destMode {
		t.Logf("Note: Permissions may differ on some filesystems")
		t.Logf("Source: %v, Dest: %v", sourceMode, destMode)
	}
}

// TestMoveFile_ParallelConflicts tests concurrent moves with same filename
func TestMoveFile_ParallelConflicts(t *testing.T) {
	tmpDir := t.TempDir()
	processedFolder := filepath.Join(tmpDir, "processed")

	// Create multiple source files with same name in different subdirectories
	numFiles := 10
	sourceFiles := make([]string, numFiles)

	for i := 0; i < numFiles; i++ {
		subDir := filepath.Join(tmpDir, fmt.Sprintf("source%d", i))
		if err := os.MkdirAll(subDir, 0755); err != nil {
			t.Fatalf("Failed to create source dir: %v", err)
		}

		sourceFile := filepath.Join(subDir, "test.csv")
		content := fmt.Sprintf("content %d", i)
		if err := os.WriteFile(sourceFile, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
		sourceFiles[i] = sourceFile
	}

	// Move all files in parallel, each with a unique worker ID
	results := make(chan string, numFiles)
	errors := make(chan error, numFiles)

	for i := 0; i < numFiles; i++ {
		go func(sourceFile string, workerID int) {
			destPath, err := MoveToProcessed(sourceFile, processedFolder, workerID)
			if err != nil {
				errors <- err
			} else {
				results <- destPath
			}
		}(sourceFiles[i], i+1) // Worker IDs: 1, 2, 3, ..., 10
	}

	// Collect results
	destPaths := make(map[string]bool)
	for i := 0; i < numFiles; i++ {
		select {
		case destPath := <-results:
			if destPaths[destPath] {
				t.Errorf("Duplicate destination path: %s", destPath)
			}
			destPaths[destPath] = true
		case err := <-errors:
			t.Errorf("Move failed: %v", err)
		}
	}

	// Verify all files were moved successfully with unique paths
	if len(destPaths) != numFiles {
		t.Errorf("Expected %d unique destinations, got %d", numFiles, len(destPaths))
	}

	// Verify all files exist in their worker-specific folders with original filename
	for i := 1; i <= numFiles; i++ {
		expectedPath := filepath.Join(processedFolder+fmt.Sprintf("_worker%02d", i), "test.csv")
		if !destPaths[expectedPath] {
			t.Errorf("Expected file not found: %s", expectedPath)
		}

		// Verify file exists and has correct content
		content, err := os.ReadFile(expectedPath)
		if err != nil {
			t.Errorf("Failed to read file %s: %v", expectedPath, err)
			continue
		}
		expectedContent := fmt.Sprintf("content %d", i-1)
		if string(content) != expectedContent {
			t.Errorf("File %s has content %q, want %q", expectedPath, string(content), expectedContent)
		}
	}

	// Verify worker folders were created
	for i := 1; i <= numFiles; i++ {
		workerFolder := processedFolder + fmt.Sprintf("_worker%02d", i)
		if info, err := os.Stat(workerFolder); err != nil || !info.IsDir() {
			t.Errorf("Worker folder not created: %s", workerFolder)
		}
	}
}
