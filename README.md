# Go File Search

A high-performance file and directory search utility written in Go, leveraging concurrency for fast searches across large file systems.

[![Go Version](https://img.shields.io/badge/Go-1.23.5+-00ADD8.svg)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

## Overview

Go File Search is a command-line utility that allows users to search for files and directories by name or using regular expressions. It utilizes Go's powerful concurrency primitives to perform searches in parallel, making it significantly faster than sequential search tools, especially on large file systems with many directories.

## Features

- **High-performance concurrent searching** - Automatically distributes search across multiple goroutines
- **Configurable concurrency** - Adjust worker count to optimize for your system
- **Multiple search methods**:
  - Exact filename matching
  - Exact directory name matching
  - Regular expression pattern matching
- **Early termination** - Optionally stop searching after first match
- **Comprehensive statistics** - Track number of matches by type
- **Logging support** - Save search results to a log file
- **Automatic resource management** - Prevents excessive goroutine creation

## Installation

### Prerequisites

- Go 1.23 or higher

### Building from source

```bash
# Clone the repository
git clone https://github.com/0xRadioAc7iv/file-search.git
cd file-search

# Build the binary
go build

# Or if you want to customize the name
go build -o {your_preffered_name}.exe

# Install the binary
go install
```

## Usage

```
file-search [options]
```

### Command-line Options

| Flag       | Description                                | Default                 |
| ---------- | ------------------------------------------ | ----------------------- |
| `-file`    | Name of the file to search                 | ""                      |
| `-dir`     | Name of the directory to search            | ""                      |
| `-regex`   | Regular expression pattern to match        | ""                      |
| `-root`    | Root directory to start the search         | "." (current directory) |
| `-r`       | Return early after finding the first match | false                   |
| `-workers` | Maximum number of concurrent workers       | 10                      |
| `-log`     | Log all matches to a text file             | false                   |
| `-logfile` | Path to log file (used with -log)          | "search_results.log"    |

**Note**: At least one search target (`-file`, `-dir`, or `-regex`) must be provided.

### Examples

Search for a specific file:

```bash
file-search -file "config.json" -root /etc
```

Search for directories named "logs":

```bash
file-search -dir "logs" -root /var/
```

Find all Go source files:

```bash
file-search -regex "\.go$" -root ~/projects
```

Search with limited concurrency:

```bash
file-search -regex "\.jpg$" -root /media/photos -workers 4
```

Stop after first match:

```bash
file-search -file "secret.txt" -r
```

Log results to a file:

```bash
file-search -regex "\.pdf$" -root ~/Documents -log
```

Custom log file location:

```bash
file-search -file "important.doc" -root /home -log -logfile results.txt
```

Combined search:

```bash
file-search -file "data.csv" -dir "backup" -regex "temp.*\.txt$" -root /workspace
```

## Performance Optimization

### Worker Count

The optimal number of workers depends on your system's resources and the characteristics of your file system:

- For I/O-bound operations like file searching, using more workers than CPU cores often improves performance
- Recommended starting point: 2-4× the number of CPU cores

To find the optimal value for your system, benchmark with different worker counts:

```bash
# Test with different worker counts
time filesearch -regex "\.txt$" -root /large/directory -workers 12
time filesearch -regex "\.txt$" -root /large/directory -workers 24
time filesearch -regex "\.txt$" -root /large/directory -workers 36
```

## How It Works

Go Concurrent File Search uses several concurrency patterns to achieve high performance:

1. **Worker Pool**: Limits the maximum number of concurrent goroutines
2. **Work Distribution**: Each directory is processed in its own goroutine
3. **Bounded Concurrency**: Uses a semaphore pattern to limit resource usage
4. **Early Termination**: Signals all goroutines to stop when conditions are met
5. **Thread-safe Results Collection**: Uses mutex to protect shared state

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
