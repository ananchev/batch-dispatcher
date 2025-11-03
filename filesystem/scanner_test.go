package filesystem

import (
	"os"
	"path/filepath"
	"testing"
)

// TestScanInputFolder_ValidFolder tests scanning a folder with matching files
func TestScanInputFolder_ValidFolder(t *testing.T) {
	// Create temp directory with test files
	tmpDir := t.TempDir()

	// Create test files
	testFiles := []string{
		"eol_2025-12-31.csv",
		"eol_2026-12-31.csv",
		"eol_2027-12-31.csv",
		"other_file.txt",
	}

	for _, filename := range testFiles {
		filePath := filepath.Join(tmpDir, filename)
		if err := os.WriteFile(filePath, []byte("test content"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	// Scan for eol_*.csv files
	files, err := ScanInputFolder(tmpDir, "eol_*.csv")
	if err != nil {
		t.Fatalf("ScanInputFolder() error = %v", err)
	}

	// Should find 3 matching files
	if len(files) != 3 {
		t.Errorf("Expected 3 files, got %d", len(files))
	}

	// Verify files are sorted
	expected := []string{
		filepath.Join(tmpDir, "eol_2025-12-31.csv"),
		filepath.Join(tmpDir, "eol_2026-12-31.csv"),
		filepath.Join(tmpDir, "eol_2027-12-31.csv"),
	}

	for i, file := range files {
		absExpected, _ := filepath.Abs(expected[i])
		if file != absExpected {
			t.Errorf("File[%d] = %q, want %q", i, file, absExpected)
		}
	}
}

// TestScanInputFolder_EmptyFolder tests scanning folder with no matching files
func TestScanInputFolder_EmptyFolder(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a file that doesn't match pattern
	filePath := filepath.Join(tmpDir, "other.txt")
	if err := os.WriteFile(filePath, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Scan for non-existent pattern
	files, err := ScanInputFolder(tmpDir, "eol_*.csv")
	if err != nil {
		t.Fatalf("ScanInputFolder() error = %v, expected nil", err)
	}

	// Should return empty list, not error
	if len(files) != 0 {
		t.Errorf("Expected 0 files, got %d", len(files))
	}
}

// TestScanInputFolder_NonExistentFolder tests error on non-existent folder
func TestScanInputFolder_NonExistentFolder(t *testing.T) {
	_, err := ScanInputFolder("/nonexistent/folder/path", "*.csv")
	if err == nil {
		t.Error("Expected error for non-existent folder, got nil")
	}
}

// TestScanInputFolder_FileNotFolder tests error when path is a file
func TestScanInputFolder_FileNotFolder(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "file.txt")
	if err := os.WriteFile(filePath, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	_, err := ScanInputFolder(filePath, "*.csv")
	if err == nil {
		t.Error("Expected error when path is a file, got nil")
	}
}

// TestScanInputFolder_InvalidPattern tests error on invalid glob pattern
func TestScanInputFolder_InvalidPattern(t *testing.T) {
	tmpDir := t.TempDir()

	// Invalid glob pattern with unmatched bracket
	_, err := ScanInputFolder(tmpDir, "[invalid")
	if err == nil {
		t.Error("Expected error for invalid pattern, got nil")
	}
}

// TestScanInputFolder_SkipsDirectories tests that directories are not included
func TestScanInputFolder_SkipsDirectories(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a file
	filePath := filepath.Join(tmpDir, "test.csv")
	if err := os.WriteFile(filePath, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create a subdirectory that matches pattern
	subDir := filepath.Join(tmpDir, "test_dir.csv")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	// Scan
	files, err := ScanInputFolder(tmpDir, "*.csv")
	if err != nil {
		t.Fatalf("ScanInputFolder() error = %v", err)
	}

	// Should only find the file, not the directory
	if len(files) != 1 {
		t.Errorf("Expected 1 file, got %d", len(files))
	}

	absFile, _ := filepath.Abs(filePath)
	if len(files) > 0 && files[0] != absFile {
		t.Errorf("Expected %q, got %q", absFile, files[0])
	}
}

// TestScanInputFolder_AlphabeticalOrder tests that files are sorted alphabetically
func TestScanInputFolder_AlphabeticalOrder(t *testing.T) {
	tmpDir := t.TempDir()

	// Create files in non-alphabetical order
	testFiles := []string{
		"z_last.csv",
		"a_first.csv",
		"m_middle.csv",
	}

	for _, filename := range testFiles {
		filePath := filepath.Join(tmpDir, filename)
		if err := os.WriteFile(filePath, []byte("test"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	// Scan
	files, err := ScanInputFolder(tmpDir, "*.csv")
	if err != nil {
		t.Fatalf("ScanInputFolder() error = %v", err)
	}

	// Verify alphabetical order
	expectedOrder := []string{"a_first.csv", "m_middle.csv", "z_last.csv"}
	for i, file := range files {
		if filepath.Base(file) != expectedOrder[i] {
			t.Errorf("File[%d] = %q, want %q", i, filepath.Base(file), expectedOrder[i])
		}
	}
}
