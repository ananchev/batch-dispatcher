# Batch Dispatcher

A dispatcher tool that runs an arbitrary executable multiple times in parallel, supplying input files from a common directory.
- Dynamic work distribution: Workers pull files from a shared queue - the first available worker processes the next file
- Worker-specific output: Processed or erroring files are moved to separate folders per worker
- Extensive logging: Central dispatcher log plus detailed per-file execution logs
- Flexible configuration: Define command-line parameters, environment variables incl. expansion, timeouts, and working directory

```bash
batch-dispatcher.exe --config=config.yaml
```


## Configuration

Configuration is done through YAML config file as follows:

```yaml
executable:
  # Example execution: processor.exe --input "C:\data\input\file1.csv" --flag1 --parameter1 value1 -flag2=value2
  path: "C:\\tools\\processor.exe"               # Path to executable or script
  default_args:                                # Arguments template (use {input} placeholder for file path)
    - "--input"
    - "{input}"                                # IMPORTANT: "{input}" is a hardcoded placeholder - do not change this literal string
    - "--flag1"                                # Optional: Add any additional flags your executable needs
    - "--parameter1"                           # Optional: Add parameters with values
    - "value1"                                 # Optional: The value for --parameter1
    - "-flag2=value2"                          # Optional: Another example flag using -flag=value notation
  timeout: "5m"                                # Per input file timeout (e.g., "5m", "30m", "1h", "0" for infinite)
  environment:                                 # Optional environment variables
    VAR1: "value1"
  working_directory: ""                        # Optional working directory

input:
  source_directory: "C:\\data\\input"            # Where to find input files files
  file_pattern: "*.csv"                        # Glob matching pattern (e.g. "*.csv", "data_*.txt") to filter input files
  max_files: 0                                 # Number of files to process (0 = all): sort alphabetically, then first N are processed
output:
  processed_directory: "C:\\data\\processed"     # Base name for successful files (creates: C:\data\processed_worker01, processed_worker02, etc.)
  errors_directory: "C:\\data\\errors"           # Base name for failed files (creates: C:\data\errors_worker01, errors_worker02, etc.)

workers:
  count: 4                                     # Number of parallel workers (minimum 1)

logging:
  log_file: "C:\\data\\logs\\dispatcher.log"      # Central log file path, timestamp added automatically, e.g., dispatcher_20251104-123045.log)
  per_file_log_directory: "C:\\data\\logs\\executions"  # Per-file logs directory (omit or leave empty to disable per-file logs)

advanced:
  show_progress: true                          # Display progress counter
  continue_on_error: false                     # If true: process all files despite failures; if false: stop on first failure (fail-fast)
  dry_run: false                               # Preview execution without running (shows config, files, command template)
```



## Folder Structure

### Input
```
C:\data\input\
├── file1.csv
├── file2.csv
├── file3.csv
└── file4.csv
```

### Output (After Processing)
- Processed or erroring input files are moved into worker-specific output folders.
- Each worker gets its own folder by appending `_worker01`, `_worker02`, etc. to the base directory name.
- Format: `<base_directory>_worker01`, `<base_directory>_worker02`, etc.

```
C:\data\
├── processed_worker01\
│   ├── file1.csv
│   └── file3.csv
├── processed_worker02\
│   └── file2.csv
├── errors_worker03\
│   └── file4.csv
└── logs\
    ├── dispatcher.log
    └── executions\
        ├── success\
        │   ├── file1_20251103_143525.log
        │   ├── file2_20251103_143532.log
        │   └── file3_20251103_143540.log
        ├── failed\
        │   └── file4_20251103_143548.log
        └── timeout\
```


## Logging

### Central Log
```
2025/11/03 14:35:22 INFO: Batch Dispatcher starting
2025/11/03 14:35:22 INFO: Found 25 files to process
2025/11/03 14:35:22 INFO: Starting dispatcher with 4 workers
2025/11/03 14:35:23 INFO: Worker 01: Starting job: file1.csv
2025/11/03 14:35:35 INFO: Worker 01: Job completed: file1.csv (Success, 12.5s)
...
2025/11/03 14:42:04 INFO: Processing complete: 25 total, 23 successful, 2 failed
```

### Per-File Logs
```
=================================================================================
Execution Log - file1.csv
=================================================================================
Worker: 01
Start Time: 2025-11-03 14:35:23
File: C:\data\input\file1.csv

COMMAND:
C:\tools\processor.exe --input "C:\data\input\file1.csv"

ENVIRONMENT VARIABLES:
VAR1=value1
VAR2=value2

STANDARD OUTPUT:
Processing file1.csv...
Row 1: Success
...

STANDARD ERROR:
(empty or error messages)

RESULT:
Status: SUCCESS
Exit Code: 0
Duration: 12.5s
```

### Console Output
```
2025/11/03 14:35:22 INFO: Batch Dispatcher starting
2025/11/03 14:35:22 INFO: Found 25 files to process

Progress: [===========>            ] 12/25 (48%) | Success: 11 | Failed: 1

2025/11/03 14:42:04 INFO: Processing complete: 25 total, 23 successful, 2 failed
```



## Placeholder Replacement

The `{input}` placeholder in `default_args` is replaced with the absolute path of each file being processed.

YAML supports two equivalent formats for `default_args`:
- Multi-line: `default_args:` followed by `- "arg1"` on separate lines (better for readability and inline comments)
- Inline: `default_args: ["arg1", "arg2", "arg3"]` (more compact)

Both formats are functionally identical and can be used interchangeably.

### Example
Configuration:
```yaml
default_args:
  - "--input-file"
  - "{input}"
  - "--output"
  - "result.xml"
```

Executed Command:
```bash
processor.exe --input-file "C:\data\input\file1.csv" --output "result.xml"
```



## Environment Variables

Environment variables can be defined in the configuration and support `${VAR}` expansion syntax. The executable inherits system environment variables, with config values overriding system values.

Windows example:
```yaml
executable:
  environment:
    JAVA_HOME: "C:\\Apps\\Java\\jdk-17"
    PATH: "${JAVA_HOME}\\bin;${PATH}"          # Windows uses semicolon (;) as path separator
    USERPROFILE: "${USERPROFILE}"              # Pass through user profile directory
    APPDATA: "${APPDATA}"                      # Pass through application data directory
    LOCALAPPDATA: "${LOCALAPPDATA}"            # Pass through local application data directory
    CUSTOM_VAR: "value"
```

Linux/Mac example:
```yaml
executable:
  environment:
    JAVA_HOME: "/usr/lib/jvm/java-17"
    PATH: "${JAVA_HOME}/bin:${PATH}"           # Linux/Mac uses colon (:) as path separator
    HOME: "${HOME}"                            # Pass through user home directory
    CUSTOM_VAR: "value"
```

Both will execute with the environment variables expanded and merged with system environment. If system PATH is `C:\Windows\system32` (Windows) or `/usr/bin:/bin` (Linux), the executable will run with:
- Windows: `PATH=C:\Apps\Java\jdk-17\bin;C:\Windows\system32`
- Linux/Mac: `PATH=/usr/lib/jvm/java-17/bin:/usr/bin:/bin`



## Other Examples

### Basic Batch Processing
```yaml
# config.yaml
executable:
  path: "C:\\tools\\processor.exe"
  default_args: ["--input", "{input}"]
  timeout: "5m"

input:
  source_directory: "C:\\data\\input"
  file_pattern: "*.csv"

output:
  processed_directory: "C:\\data\\processed"
  errors_directory: "C:\\data\\errors"

workers:
  count: 4

logging:
  log_file: "C:\\data\\logs\\dispatcher.log"
  per_file_log_directory: "C:\\data\\logs\\executions"

advanced:
  show_progress: true
```

### With Environment Variables

An example how to run a Java application with custom environment variables. This will execute as:
```bash
java -jar C:\tools\app.jar --input "C:\data\input\file1.csv"
java -jar C:\tools\app.jar --input "C:\data\input\file2.csv"
...
```

This is a partial configuration showing only the `executable` section. A complete configuration requires `input`, `output`, `workers`, and `logging` sections as shown in the basic example above.

```yaml
# config.yaml (partial - executable section only)
executable:
  path: "java"                                 # Java executable (will use JAVA_HOME from environment below)
  environment:
    JAVA_HOME: "C:\\Apps\\Java\\jdk-17"
    PATH: "${JAVA_HOME}\\bin;${PATH}"           # Add Java to PATH using variable expansion
    USERPROFILE: "${USERPROFILE}"              # Pass through user profile directory
    APPDATA: "${APPDATA}"                      # Pass through application data directory
    LOCALAPPDATA: "${LOCALAPPDATA}"            # Pass through local application data directory
  default_args: ["-jar", "C:\\tools\\app.jar", "--input", "{input}"]  # Run JAR with input file
  timeout: "10m"
```




## Project Structure

```
batch-dispatcher\
├── main.go                     # Entry point
├── config\
│   ├── config.go               # YAML loading & validation
│   └── config_test.go
├── dispatcher\
│   └── dispatcher.go           # Worker pool orchestration
├── executor\
│   ├── executor.go             # Command execution
│   └── executor_test.go
├── filesystem\
│   ├── scanner.go              # File scanning
│   ├── scanner_test.go
│   ├── handler.go              # File moving
│   └── handler_test.go
├── logger\
│   ├── logger.go               # Logging
│   ├── counter.go              # Progress counter
│   └── logger_test.go
├── models\
│   └── types.go                # Data structures
├── config.example.yaml         # Configuration template
└── README.md                   
```