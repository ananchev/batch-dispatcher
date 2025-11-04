package main

import (
	"batch-dispatcher/config"
	"batch-dispatcher/dispatcher"
	"batch-dispatcher/filesystem"
	"batch-dispatcher/logger"
	"batch-dispatcher/models"
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
)

func main() {
	// Parse command-line flags
	configPath := flag.String("config", "config.yaml", "Path to configuration file")
	flag.Parse()

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Create logger
	log, err := logger.New(cfg.Logging.LogFile, cfg.Logging.PerFileLogDirectory)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: Failed to create logger: %v\n", err)
		os.Exit(1)
	}
	defer log.Close()

	log.Info("Batch Dispatcher starting")
	log.Info("Configuration loaded from: %s", *configPath)

	// Scan for files
	filePaths, err := filesystem.ScanInputFolder(cfg.Input.SourceDirectory, cfg.Input.FilePattern)
	if err != nil {
		log.Error("Failed to scan for files: %v", err)
		os.Exit(1)
	}

	if len(filePaths) == 0 {
		log.Info("No files found matching pattern '%s' in directory '%s'",
			cfg.Input.FilePattern, cfg.Input.SourceDirectory)
		fmt.Println("No files to process.")
		os.Exit(0)
	}

	// Apply max files limit if set
	if cfg.Input.MaxFiles > 0 && len(filePaths) > cfg.Input.MaxFiles {
		log.Info("Limiting to %d files (found %d)", cfg.Input.MaxFiles, len(filePaths))
		filePaths = filePaths[:cfg.Input.MaxFiles]
	}

	log.Info("Found %d files to process", len(filePaths))

	// Check for dry-run mode
	if cfg.Advanced.DryRun {
		printDryRunInfo(cfg, filePaths)
		os.Exit(0)
	}

	// Create jobs from file paths
	jobs := make([]*models.Job, 0, len(filePaths))
	totalLines := 0
	for _, filePath := range filePaths {
		// Count lines in the file (excluding header)
		lineCount, err := filesystem.CountFileLines(filePath)
		if err != nil {
			log.Error("Warning: Failed to count lines in %s: %v", filepath.Base(filePath), err)
			lineCount = 0 // Continue processing even if line count fails
		}
		totalLines += lineCount

		jobs = append(jobs, &models.Job{
			FilePath:  filePath,
			FileName:  filepath.Base(filePath),
			LineCount: lineCount,
		})
	}

	log.Info("Total data lines across all files: %d", totalLines)

	// Create progress counter
	counter := logger.NewProgressCounter(len(jobs), cfg.Advanced.ShowProgress)

	// Create dispatcher
	disp, err := dispatcher.New(cfg, log, counter)
	if err != nil {
		log.Error("Failed to create dispatcher: %v", err)
		os.Exit(1)
	}

	// Setup context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown on SIGINT/SIGTERM
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		sig := <-sigChan
		log.Info("Received signal %v, initiating graceful shutdown...", sig)
		fmt.Println("\nShutting down gracefully...")
		cancel()
	}()

	// Run dispatcher
	if err := disp.Run(ctx, jobs); err != nil {
		log.Error("Dispatcher error: %v", err)
		os.Exit(1)
	}

	// Determine exit code based on results
	_, successful, failed := counter.GetStats()

	if failed > 0 {
		log.Info("Exiting with code 1 (some files failed)")
		os.Exit(1)
	}

	if successful == 0 {
		log.Info("Exiting with code 2 (no files processed)")
		os.Exit(2)
	}

	log.Info("All files processed successfully")
	os.Exit(0)
}

// printDryRunInfo prints execution plan without running
func printDryRunInfo(cfg *models.Config, filePaths []string) {
	fmt.Println("================================================================================")
	fmt.Println("                     BATCH DISPATCHER - DRY RUN MODE")
	fmt.Println("================================================================================")
	fmt.Println()
	fmt.Println("NO FILES WILL BE PROCESSED - THIS IS A PREVIEW")
	fmt.Println()

	fmt.Println("CONFIGURATION:")
	fmt.Println("--------------------------------------------------------------------------------")
	fmt.Printf("Executable: %s\n", cfg.Executable.Path)
	if cfg.Executable.WorkingDirectory != "" {
		fmt.Printf("Working Directory: %s\n", cfg.Executable.WorkingDirectory)
	}
	fmt.Printf("Timeout: %s (%s)\n", cfg.Executable.Timeout, cfg.Executable.TimeoutDuration)
	fmt.Printf("Workers: %d parallel workers\n", cfg.Workers.Count)
	fmt.Println()

	fmt.Println("INPUT:")
	fmt.Println("--------------------------------------------------------------------------------")
	fmt.Printf("Source Directory: %s\n", cfg.Input.SourceDirectory)
	fmt.Printf("File Pattern: %s\n", cfg.Input.FilePattern)
	fmt.Printf("Files Found: %d\n", len(filePaths))
	if cfg.Input.MaxFiles > 0 {
		fmt.Printf("Max Files Limit: %d\n", cfg.Input.MaxFiles)
	}
	fmt.Println()

	fmt.Println("OUTPUT:")
	fmt.Println("--------------------------------------------------------------------------------")
	fmt.Printf("Processed Directory: %s\n", cfg.Output.ProcessedDirectory)
	fmt.Printf("  (Worker folders will be created: processed_worker01, processed_worker02, ...)\n")
	fmt.Printf("Errors Directory: %s\n", cfg.Output.ErrorsDirectory)
	fmt.Printf("  (Worker folders will be created: errors_worker01, errors_worker02, ...)\n")
	fmt.Println()

	fmt.Println("ERROR HANDLING:")
	fmt.Println("--------------------------------------------------------------------------------")
	if cfg.Advanced.ContinueOnError {
		fmt.Println("Mode: CONTINUE-ON-ERROR (process all files despite failures)")
	} else {
		fmt.Println("Mode: FAIL-FAST (stop on first failure)")
	}
	fmt.Println()

	if cfg.Executable.Environment != nil && len(cfg.Executable.EnvironmentExpanded) > 0 {
		fmt.Println("ENVIRONMENT VARIABLES:")
		fmt.Println("--------------------------------------------------------------------------------")
		for key, value := range cfg.Executable.EnvironmentExpanded {
			fmt.Printf("%s=%s\n", key, value)
		}
		fmt.Println()
	}

	fmt.Println("COMMAND TEMPLATE:")
	fmt.Println("--------------------------------------------------------------------------------")
	fmt.Printf("%s", cfg.Executable.Path)
	for _, arg := range cfg.Executable.DefaultArgs {
		if strings.Contains(arg, "{input}") {
			// Show placeholder in template
			fmt.Printf(" %s", arg)
		} else {
			fmt.Printf(" %s", arg)
		}
	}
	fmt.Println()
	fmt.Println()

	fmt.Println("EXAMPLE COMMAND FOR FIRST FILE:")
	fmt.Println("--------------------------------------------------------------------------------")
	if len(filePaths) > 0 {
		fmt.Printf("%s", cfg.Executable.Path)
		for _, arg := range cfg.Executable.DefaultArgs {
			if strings.Contains(arg, "{input}") {
				// Replace {input} placeholder with actual file path
				exampleArg := strings.ReplaceAll(arg, "{input}", filePaths[0])
				fmt.Printf(" %s", exampleArg)
			} else {
				fmt.Printf(" %s", arg)
			}
		}
		fmt.Println()
	}
	fmt.Println()

	fmt.Println("FILES TO PROCESS:")
	fmt.Println("--------------------------------------------------------------------------------")
	maxDisplay := 20
	for i, path := range filePaths {
		if i >= maxDisplay {
			fmt.Printf("... and %d more files\n", len(filePaths)-maxDisplay)
			break
		}
		fmt.Printf("%3d. %s\n", i+1, filepath.Base(path))
	}
	fmt.Println()

	fmt.Println("================================================================================")
	fmt.Println("DRY RUN COMPLETE - No files were processed")
	fmt.Println("To execute for real, set 'dry_run: false' in your config file")
	fmt.Println("================================================================================")
}
