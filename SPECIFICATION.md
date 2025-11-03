# Go Batch Dispatcher - Complete Specification

**Version:** 1.0  
**Date:** November 3, 2025  
**Purpose:** Generic executable dispatcher for batch processing of files with parallel execution support

---

## 1. Core Components

```
batch-dispatcher/
├── main.go                    # Entry point, CLI parsing
├── config/
│   ├── config.go             # Configuration structures & YAML loading
│   └── validator.go          # Config validation
├── dispatcher/
│   ├── dispatcher.go         # Core dispatching logic
│   └── worker_pool.go        # Fixed worker pool management
├── executor/
│   ├── executor.go           # Executable invocation wrapper
│   └── env_builder.go        # Environment variable setup
├── filesystem/
│   ├── scanner.go            # Input folder scanning
│   └── handler.go            # File moving (processed/errors folders)
├── logger/
│   ├── logger.go             # Simple text logging
│   └── counter.go            # Simple progress counters
└── models/
    └── types.go              # Shared data structures
```

---

## 2. Configuration File Structure

**File:** `dispatcher-config.yaml`

```yaml
# Registry of available executables
executables:
  bnl_bulk_edit:
    path: "C:/Path/To/BNL_BULK_EDIT.exe"
    working_directory: ""  # Optional, defaults to executable's directory
    
    # Centralized environment variables for this tool
    environment:
      JAVA_HOME: "C:/Apps/Java/zulu-11-jre"
      PATH: "${JAVA_HOME}/bin;${PATH}"  # Variable expansion supported
    
    # Default arguments (no override from CLI)
    default_args:
      - "-host=https://nws-int.siemens.cloud/tc"
      - "-sso=https://nws-int.siemens.cloud/loginservice/sa"
      - "-appID=TCintAWC"
      # Optional: "-debug" for detailed logging in individual logs
    
    # Execution parameters
    delay_between_files: "3s"  # Throttling within same worker
    timeout: "30m"             # Per-file execution timeout
    
  another_tool:
    path: "C:/Path/To/AnotherTool.exe"
    environment:
      CUSTOM_VAR: "value"
    default_args:
      - "--flag1=value1"
      - "--flag2=value2"
    delay_between_files: "1s"
    timeout: "10m"

# Global dispatcher settings
dispatcher:
  default_file_pattern: "*.csv"
  default_workers: 1
```

**Configuration Notes:**
- Uses `gopkg.in/yaml.v3` for parsing (vendored)
- Environment variable expansion: `${VAR_NAME}` syntax
- All paths support both forward slash and backslash (cross-platform)
- Timeouts: Valid Go duration format (e.g., "30s", "5m", "1h")

---

## 3. CLI Interface

### Commands & Flags

```bash
# REQUIRED FLAGS
--config=<path>              # Path to dispatcher-config.yaml
--executable=<name>          # Which tool from config (e.g., "bnl_bulk_edit")
--input-folder=<path>        # Input files directory

# OPTIONAL FLAGS
--workers=<N>                # Parallel workers (N ≥ 1, no upper limit)
                             # Default: from config or 1
                             # If files < workers: spawn only file count
--file-pattern=<pattern>     # File filter (default: *.csv or from config)
--log-folder=<path>          # Log output location (default: <input-folder>/logs)

# RUNTIME PARAMETERS (passed to executable)
--param key=value            # Runtime parameters (e.g., --param group=dba)
                             # Can specify multiple times
                             # Appended to default_args from config

# ERROR HANDLING
--fail-fast                  # Stop all workers on first error (DEFAULT)
--continue-on-error          # Log errors, move to errors/ folder, continue

# SPECIAL MODES
--dry-run                    # Print execution plan without running
```

### Usage Examples

**Basic execution with defaults:**
```bash
batch-dispatcher \
  --config=dispatcher-config.yaml \
  --executable=bnl_bulk_edit \
  --input-folder=./csv
```

**With parallelism:**
```bash
batch-dispatcher \
  --config=dispatcher-config.yaml \
  --executable=bnl_bulk_edit \
  --input-folder=./csv \
  --workers=4
```

**With runtime parameters:**
```bash
batch-dispatcher \
  --config=dispatcher-config.yaml \
  --executable=bnl_bulk_edit \
  --input-folder=./csv \
  --workers=4 \
  --param group=dba \
  --param role=DBA \
  --param u=myuser \
  --param p=mypassword
```

**With environment variable:**
```powershell
# PowerShell
$env:BULK_EDIT_PASSWORD="mysecret"
batch-dispatcher `
  --config=dispatcher-config.yaml `
  --executable=bnl_bulk_edit `
  --input-folder=./csv `
  --workers=4 `
  --param u=myuser `
  --param p=$env:BULK_EDIT_PASSWORD
```

**Specific file pattern, continue on errors:**
```bash
batch-dispatcher \
  --config=dispatcher-config.yaml \
  --executable=bnl_bulk_edit \
  --input-folder=./csv \
  --workers=8 \
  --file-pattern="eol_*.csv" \
  --continue-on-error
```

**Dry run to preview execution:**
```bash
batch-dispatcher \
  --config=dispatcher-config.yaml \
  --executable=bnl_bulk_edit \
  --input-folder=./csv \
  --workers=4 \
  --param group=dba \
  --dry-run
```

---

## 4. Execution Flow

```
1. Parse CLI arguments
2. Load & validate config file (YAML)
3. Validate executable configuration exists
4. Scan input folder for files matching pattern
5. Create folder structure:
   - <input-folder>/processed/
   - <input-folder>/errors/     (if --continue-on-error)
   - <log-folder>/              (default: <input-folder>/logs/)
6. Initialize execution log
7. Determine actual worker count:
   actualWorkers = min(requestedWorkers, numberOfFiles)
8. Create worker pool (N goroutines)
9. Create job queue (buffered channel with file paths)
10. Start workers pulling from queue
11. Workers execute for each file:
    a. Build command (config defaults + runtime params + input file)
    b. Set environment variables from config
    c. Execute process with timeout
    d. Capture stdout/stderr
    e. Write individual log file
    f. On success: move file to processed/
    g. On error: 
       - fail-fast: signal all workers to stop
       - continue-on-error: move file to errors/, continue
    h. Apply delay_between_files (if configured)
12. Wait for all workers to complete
13. Write summary to execution log
14. Print summary to console
15. Exit with appropriate code
```

---

## 5. Folder Structure During Execution

```
<input-folder>/
├── file1.csv              # Pending
├── file2.csv              # Pending
├── file3.csv              # Pending
├── processed/             # Successfully completed files
│   ├── file1.csv
│   └── file2.csv
├── errors/                # Failed files (only with --continue-on-error)
│   └── file3.csv
└── logs/                  # Default log location (or custom --log-folder)
    ├── execution_20251103_143522.log       # Central execution log
    ├── file1_20251103_143525.log           # Individual file logs
    ├── file2_20251103_143532.log
    └── file3_20251103_143540.log
```

**Folder Creation:**
- `processed/` - Always created
- `errors/` - Only created when `--continue-on-error` is specified
- `logs/` - Created if it doesn't exist (or custom location)

**File Operations:**
- Files are **moved** (not copied or deleted) atomically where possible
- Original files never deleted without successful move
- Failed moves are logged but don't stop processing

---

## 6. Logging Details

### A. Execution Log (`execution_YYYYMMDD_HHMMSS.log`)

**Location:** `<log-folder>/execution_YYYYMMDD_HHMMSS.log`

**Content:**
```
=================================================================================
Batch Dispatcher v1.0.0 - Execution Log
=================================================================================
Start Time: 2025-11-03 14:35:22
Configuration File: C:/Path/To/dispatcher-config.yaml
Executable: bnl_bulk_edit
Executable Path: C:/Path/To/BNL_BULK_EDIT.exe

EXECUTION PARAMETERS:
=================================================================================
Input Folder: C:/Data/csv
Log Folder: C:/Data/csv/logs
Processed Folder: C:/Data/csv/processed
Errors Folder: C:/Data/csv/errors (if continue-on-error)
File Pattern: *.csv
Requested Workers: 4
Actual Workers: 4 (min of requested and file count)
Error Mode: fail-fast / continue-on-error
Dry Run: false

ENVIRONMENT VARIABLES:
=================================================================================
JAVA_HOME=C:/Apps/Java/zulu-11-jre
PATH=C:/Apps/Java/zulu-11-jre/bin;<existing PATH>

COMMAND TEMPLATE:
=================================================================================
C:/Path/To/BNL_BULK_EDIT.exe \
  -host=https://nws-int.siemens.cloud/tc \
  -sso=https://nws-int.siemens.cloud/loginservice/sa \
  -appID=TCintAWC \
  -g=dba \
  -r=DBA \
  -inp="<file>"

FILES TO PROCESS (25 files):
=================================================================================
1. eol_2025-12-31.csv
2. eol_2026-12-31.csv
3. eol_2027-12-31.csv
...

=================================================================================
PROCESSING LOG:
=================================================================================
[2025-11-03 14:35:23] [Worker-1] Processing file 1 of 25: eol_2025-12-31.csv
[2025-11-03 14:35:24] [Worker-2] Processing file 2 of 25: eol_2026-12-31.csv
[2025-11-03 14:35:25] [Worker-3] Processing file 3 of 25: eol_2027-12-31.csv
[2025-11-03 14:35:26] [Worker-4] Processing file 4 of 25: eol_2028-12-31.csv
[2025-11-03 14:35:35] [Worker-1] SUCCESS: eol_2025-12-31.csv (12.5s)
[2025-11-03 14:35:36] [Worker-1] Processing file 5 of 25: eol_2029-12-31.csv
[2025-11-03 14:35:38] [Worker-2] FAILED: eol_2026-12-31.csv (8.3s) - Exit code 1
...

=================================================================================
EXECUTION SUMMARY:
=================================================================================
End Time: 2025-11-03 14:42:04
Duration: 6m 42s

STATISTICS:
- Total files found: 25
- Files processed: 25
- Successful: 23
- Failed: 2
- Success Rate: 92.0%

PERFORMANCE:
- Average time per file: 16.08 seconds
- Files per minute: 3.73

FAILED FILES:
- eol_2026-12-31.csv (Exit code: 1)
- eol_2030-12-31.csv (Exit code: 1)

LOG FILES:
- Execution log: C:/Data/csv/logs/execution_20251103_143522.log
- Individual logs location: C:/Data/csv/logs
- Log files created: 25

ENVIRONMENT:
- Hostname: DESKTOP-ABC123
- OS: Windows 11 Pro
- Go Version: go1.21.0
- Dispatcher Version: 1.0.0
- User: john.doe

=================================================================================
```

### B. Individual File Logs (`<filename>_YYYYMMDD_HHMMSS.log`)

**Location:** `<log-folder>/<filename>_YYYYMMDD_HHMMSS.log`

**Content:**
```
=================================================================================
BNL_BULK_EDIT Execution Log - Individual File
=================================================================================
Start Time: 2025-11-03 14:35:23
Worker: Worker-1
Input File: C:/Data/csv/eol_2025-12-31.csv

COMMAND EXECUTED:
=================================================================================
C:/Path/To/BNL_BULK_EDIT.exe \
  -host=https://nws-int.siemens.cloud/tc \
  -sso=https://nws-int.siemens.cloud/loginservice/sa \
  -appID=TCintAWC \
  -g=dba \
  -r=DBA \
  -inp="C:/Data/csv/eol_2025-12-31.csv"

ENVIRONMENT VARIABLES:
=================================================================================
JAVA_HOME=C:/Apps/Java/zulu-11-jre
PATH=C:/Apps/Java/zulu-11-jre/bin;C:/Windows/system32;...

=================================================================================
STANDARD OUTPUT:
=================================================================================
BNL_BULK_EDIT v1.0.0
Connecting to Teamcenter...
Authentication successful
Processing CSV file...
Row 1: Updated ItemRevision ABC123
Row 2: Updated ItemRevision ABC124
...
Processing complete: 150 rows processed, 150 successful, 0 failed

=================================================================================
STANDARD ERROR:
=================================================================================
(empty or error messages)

=================================================================================
EXECUTION RESULT:
=================================================================================
End Time: 2025-11-03 14:35:35
Duration: 12.5s
Exit Code: 0
Status: SUCCESS
File Action: Moved to C:/Data/csv/processed/eol_2025-12-31.csv

=================================================================================
```

### C. Console Output (Real-time)

**Simple text with counters (no progress bars):**

```
Batch Dispatcher v1.0.0
======================

Configuration:
- Config: dispatcher-config.yaml
- Executable: bnl_bulk_edit
- Input Folder: C:/Data/csv
- Workers: 4 (spawning 4 workers for 25 files)
- File Pattern: *.csv
- Error Mode: fail-fast

Files Found: 25
Log Folder: C:/Data/csv/logs
Processed Folder: C:/Data/csv/processed

Starting workers...

[14:35:23] [Worker-1] Processing: eol_2025-12-31.csv (1/25)
[14:35:24] [Worker-2] Processing: eol_2026-12-31.csv (2/25)
[14:35:25] [Worker-3] Processing: eol_2027-12-31.csv (3/25)
[14:35:26] [Worker-4] Processing: eol_2028-12-31.csv (4/25)
[14:35:35] [Worker-1] ✓ SUCCESS: eol_2025-12-31.csv (12.5s)
[14:35:36] [Worker-1] Processing: eol_2029-12-31.csv (5/25)
[14:35:38] [Worker-2] ✗ FAILED: eol_2026-12-31.csv (8.3s) - Exit code: 1
[14:35:39] [Worker-2] Processing: eol_2030-12-31.csv (6/25)
...

======================
EXECUTION COMPLETE
======================
Duration: 6m 42s

Statistics:
- Total Files: 25
- Processed: 25
- Success: 23 (92.0%)
- Failed: 2 (8.0%)

Performance:
- Avg Time/File: 16.08s
- Files/Minute: 3.73

Failed Files:
- eol_2026-12-31.csv
- eol_2030-12-31.csv

Logs:
- Execution: C:/Data/csv/logs/execution_20251103_143522.log
- Individual: C:/Data/csv/logs/*.log

Exit Code: 0
```

---

## 7. Dry Run Output

**When `--dry-run` is specified:**

```
Batch Dispatcher v1.0.0 - DRY RUN MODE
======================================

NO FILES WILL BE PROCESSED - THIS IS A SIMULATION

Configuration:
- Config File: dispatcher-config.yaml
- Executable: bnl_bulk_edit
- Executable Path: C:/Path/To/BNL_BULK_EDIT.exe
- Working Directory: C:/Path/To/
- Workers: 4 (would spawn 4 workers for 25 files)
- Input Folder: C:/Data/csv
- File Pattern: *.csv
- Log Folder: C:/Data/csv/logs
- Error Handling: fail-fast
- Timeout: 30m
- Delay Between Files: 3s

Environment Variables (to be set):
- JAVA_HOME=C:/Apps/Java/zulu-11-jre
- PATH=C:/Apps/Java/zulu-11-jre/bin;C:/Windows/system32;...

Files to Process (25 files):
1. eol_2025-12-31.csv
2. eol_2026-12-31.csv
3. eol_2027-12-31.csv
4. eol_2028-12-31.csv
5. eol_2029-12-31.csv
...
25. eol_2049-12-31.csv

Command Template:
C:/Path/To/BNL_BULK_EDIT.exe \
  -host=https://nws-int.siemens.cloud/tc \
  -sso=https://nws-int.siemens.cloud/loginservice/sa \
  -appID=TCintAWC \
  -g=dba \
  -r=DBA \
  -inp="<file>"

Example Command for First File:
C:/Path/To/BNL_BULK_EDIT.exe \
  -host=https://nws-int.siemens.cloud/tc \
  -sso=https://nws-int.siemens.cloud/loginservice/sa \
  -appID=TCintAWC \
  -g=dba \
  -r=DBA \
  -inp="C:/Data/csv/eol_2025-12-31.csv"

Execution Plan:
- Worker pool size: 4 goroutines
- Job distribution: Queue-based (buffered channel)
- Each worker will pull files from queue until empty
- Files will be moved to: C:/Data/csv/processed/ (on success)
- Logs will be written to: C:/Data/csv/logs/

DRY RUN COMPLETE - No files were processed
To execute for real, run without --dry-run flag
```

---

## 8. Dependencies & Libraries

### Go Modules

**Required External Dependency:**
- `gopkg.in/yaml.v3` - YAML parsing (vendored)

**Standard Library Usage:**
- `flag` - CLI argument parsing
- `os` - File system operations, environment variables
- `os/exec` - Process execution with timeout
- `io` - Stream handling (stdout/stderr capture)
- `path/filepath` - Cross-platform path manipulation
- `time` - Timing, delays, timeouts
- `fmt` - Formatted output
- `strings` - String manipulation
- `context` - Timeout and cancellation
- `sync` - Worker pool synchronization (WaitGroup, Mutex)
- `log` - Basic logging

### Vendoring Strategy

```bash
# Initialize module
go mod init batch-dispatcher

# Add yaml.v3 dependency
go get gopkg.in/yaml.v3

# Vendor dependencies
go mod vendor

# Build with vendored dependencies
go build -mod=vendor -o batch-dispatcher.exe
```

**Vendored directory structure:**
```
vendor/
├── gopkg.in/
│   └── yaml.v3/
│       └── (yaml parser source)
└── modules.txt
```

---

## 9. Error Handling Behavior

### Fail-Fast Mode (Default: `--fail-fast`)

**Behavior:**
1. Worker encounters error executing file
2. Worker logs error to execution log
3. Worker writes detailed error to individual file log
4. Worker signals error to main dispatcher via error channel
5. Main dispatcher receives error signal
6. Main dispatcher sends stop signal to all workers
7. Workers finish **current file only** (graceful shutdown)
8. Workers do not pick up new files from queue
9. Remaining files stay in input folder (not moved)
10. Program exits with error code **1**

**Use Case:** Production environments where any failure requires investigation before continuing

**Exit Codes:**
- `1` - At least one file failed

### Continue-on-Error Mode (`--continue-on-error`)

**Behavior:**
1. Worker encounters error executing file
2. Worker logs error to execution log with timestamp and details
3. Worker writes detailed error to individual file log
4. Worker moves failed file to `<input-folder>/errors/` folder
5. Worker continues to next file in queue (no stop signal)
6. Other workers continue processing unaffected
7. Process continues until all files attempted
8. Summary includes count of successes and failures

**Use Case:** Batch processing where partial completion is acceptable

**Exit Codes:**
- `0` - All files succeeded OR some succeeded, some failed
- `2` - All files failed (100% failure rate)

### Timeout Handling

**Per-file timeout** (from config: `timeout: "30m"`):
1. Each file execution has independent timeout
2. If timeout exceeded:
   - Process is terminated (SIGTERM, then SIGKILL)
   - Treated as execution failure
   - Error logged: "Timeout exceeded: 30m"
   - Same behavior as fail-fast/continue-on-error applies

### Error Details Captured

**In logs:**
- Exit code (non-zero)
- stderr output
- Timeout indication
- File move failures
- Environment setup errors
- Command construction errors

---

## 10. Validation & Safety

### Configuration Validation

**On config load:**
- YAML syntax is valid
- Executable path exists and is accessible
- Executable path points to a file (not directory)
- Timeout format is valid Go duration (e.g., "30m", "1h")
- Delay format is valid Go duration (e.g., "3s", "500ms")
- Environment variable syntax valid (no circular references)
- At least one executable defined
- All required fields present

**Validation errors:**
- Stop execution immediately
- Print clear error message
- Exit with code **3**

### Runtime Validation

**On execution start:**
- Input folder exists and is accessible
- Input folder is a directory (not a file)
- File pattern is valid glob pattern
- Log folder is writable (create if doesn't exist)
- Workers ≥ 1
- Executable name exists in config
- No circular folder references (input != processed != errors)

**Validation errors:**
- Stop before creating any folders or logs
- Print clear error message with hint
- Exit with code **3**

### Worker Pool Safety

**Worker count logic:**
```go
requestedWorkers := flagWorkers  // From CLI
numberOfFiles := len(files)       // Scanned files

actualWorkers := requestedWorkers
if numberOfFiles < requestedWorkers {
    actualWorkers = numberOfFiles
    // Log: "Spawning X workers (Y requested, but only X files)"
}

if actualWorkers < 1 {
    actualWorkers = 1  // Minimum 1 worker
}
```

### File Operation Safety

**Move operations (processed/errors folders):**
1. Check target folder exists (create if needed)
2. Check target file doesn't exist (avoid overwrite)
   - If exists: append timestamp suffix
3. Attempt atomic move/rename where supported by OS
4. If move fails:
   - Log warning (don't stop processing)
   - File remains in original location
   - Mark as "processed but not moved" in logs

**Never:**
- Delete original files without successful move
- Overwrite existing files without confirmation
- Modify input files in place

### Graceful Shutdown

**On Ctrl+C (SIGINT/SIGTERM):**
1. Catch signal
2. Stop accepting new files in queue
3. Wait for current files to complete (with timeout)
4. Write partial summary to execution log
5. Exit with code **130** (interrupted)

### Environment Variable Safety

**Variable expansion:**
- Support `${VAR}` and `${VAR:-default}` syntax
- Detect circular references (A → B → A)
- Warn on undefined variables (use empty string)
- Preserve existing system environment
- Child process inherits: system env + config env + runtime overrides

### Timeout Safety

**Execution timeout enforcement:**
- Use `context.WithTimeout` for each file execution
- Send SIGTERM to process on timeout
- Wait 5 seconds
- Send SIGKILL if still running
- Log timeout event clearly

---

## Exit Codes Summary

| Code | Meaning |
|------|---------|
| `0`  | Success (all files) or partial success (continue-on-error mode) |
| `1`  | Execution failure (fail-fast mode) |
| `2`  | All files failed (continue-on-error mode, 100% failure) |
| `3`  | Configuration or validation error |
| `130`| Interrupted by user (Ctrl+C) |

---

## Future Enhancements (Out of Scope for v1.0)

- Retry mechanism (X retries per file)
- Email notifications on completion/failure
- Metrics export (Prometheus format)
- Web UI for monitoring
- Distributed execution (multiple machines)
- Resume from checkpoint
- Priority queues for files
- Dynamic worker scaling based on system load

---

## References

- PowerShell wrapper: `Utilities/wrapperScript4BulkEdit/invokeBulkEdit.ps1`
- Example tool: `BNL_BULK_EDIT.exe`
- Existing Go utility: `Utilities/groupData4BulkEdit/`
- GitLab CI vendoring: `.gitlab-ci.yml`

---

**End of Specification**
