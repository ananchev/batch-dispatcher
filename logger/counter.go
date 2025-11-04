package logger

import (
	"fmt"
	"sync"
	"time"
)

// ProgressCounter tracks and displays execution progress in the console
type ProgressCounter struct {
	total      int
	completed  int
	successful int
	failed     int
	startTime  time.Time
	mu         sync.Mutex
	enabled    bool
}

// NewProgressCounter creates a new progress counter
func NewProgressCounter(totalFiles int, enabled bool) *ProgressCounter {
	return &ProgressCounter{
		total:     totalFiles,
		startTime: time.Now(),
		enabled:   enabled,
	}
}

// Start displays the initial status
func (pc *ProgressCounter) Start() {
	if !pc.enabled {
		return
	}
	fmt.Printf("\nBatch Dispatcher - Processing Started\n")
	fmt.Printf("Total files: %d\n\n", pc.total)
	pc.display()
}

// IncrementSuccess increments successful job counter and updates display
func (pc *ProgressCounter) IncrementSuccess() {
	if !pc.enabled {
		return
	}

	pc.mu.Lock()
	pc.successful++
	pc.completed++
	pc.mu.Unlock()

	pc.display()
}

// IncrementFailure increments failed job counter and updates display
func (pc *ProgressCounter) IncrementFailure() {
	if !pc.enabled {
		return
	}

	pc.mu.Lock()
	pc.failed++
	pc.completed++
	pc.mu.Unlock()

	pc.display()
}

// display shows current progress (updates in place)
func (pc *ProgressCounter) display() {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	elapsed := time.Since(pc.startTime)
	percentage := 0.0
	if pc.total > 0 {
		percentage = float64(pc.completed) / float64(pc.total) * 100
	}

	// Calculate ETA
	var eta string
	if pc.completed > 0 {
		avgTimePerFile := elapsed / time.Duration(pc.completed)
		remaining := pc.total - pc.completed
		etaDuration := avgTimePerFile * time.Duration(remaining)
		eta = formatDuration(etaDuration)
	} else {
		eta = "calculating..."
	}

	// Display progress (without ANSI escape codes for Windows CMD compatibility)
	fmt.Printf("[%3d%%] Completed: %d/%d | Success: %d | Failed: %d\n",
		int(percentage), pc.completed, pc.total, pc.successful, pc.failed)
	fmt.Printf("Elapsed: %s | ETA: %s\n",
		formatDuration(elapsed), eta)
	fmt.Printf("%s\n", pc.buildProgressBar(percentage))
	fmt.Printf("\n") // Empty line for spacing
}

// buildProgressBar creates a visual progress bar
func (pc *ProgressCounter) buildProgressBar(percentage float64) string {
	barWidth := 40
	filled := int(percentage * float64(barWidth) / 100)

	bar := "["
	for i := 0; i < barWidth; i++ {
		if i < filled {
			bar += "="
		} else if i == filled {
			bar += ">"
		} else {
			bar += " "
		}
	}
	bar += "]"

	return bar
}

// Complete displays final summary
func (pc *ProgressCounter) Complete() {
	if !pc.enabled {
		return
	}

	pc.mu.Lock()
	defer pc.mu.Unlock()

	elapsed := time.Since(pc.startTime)

	fmt.Printf("\n")
	fmt.Printf("Batch Dispatcher - Processing Complete\n")
	fmt.Printf("========================================\n")
	fmt.Printf("Total to process: %d\n", pc.total)
	fmt.Printf("Total processed: %d\n", pc.completed)
	fmt.Printf("  Successful: %d\n", pc.successful)
	fmt.Printf("  Failed: %d\n", pc.failed)
	if pc.completed < pc.total {
		fmt.Printf("  Not processed: %d (stopped due to fail-fast)\n", pc.total-pc.completed)
	}
	fmt.Printf("Total time: %s\n", formatDuration(elapsed))

	if pc.completed > 0 {
		avgTime := elapsed / time.Duration(pc.completed)
		fmt.Printf("Average time per file: %s\n", formatDuration(avgTime))
	}
	fmt.Printf("\n")
}

// formatDuration formats a duration in human-readable format
func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)

	hours := d / time.Hour
	d -= hours * time.Hour

	minutes := d / time.Minute
	d -= minutes * time.Minute

	seconds := d / time.Second

	if hours > 0 {
		return fmt.Sprintf("%02dh %02dm %02ds", hours, minutes, seconds)
	} else if minutes > 0 {
		return fmt.Sprintf("%02dm %02ds", minutes, seconds)
	}
	return fmt.Sprintf("%ds", seconds)
}

// GetStats returns current statistics
func (pc *ProgressCounter) GetStats() (completed, successful, failed int) {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	return pc.completed, pc.successful, pc.failed
}
