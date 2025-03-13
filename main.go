package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"time"
)

// SearchResult holds information about a found item
type SearchResult struct {
	Path    string
	IsDir   bool
	Matched string // What matched (file, dir, or regex)
}

// SearchStats tracks various statistics about the search
type SearchStats struct {
	RegexMatches int
	FilesFound   int
	DirsFound    int
}

func searchConcurrent(rootDir, fileName, dirName, regexPattern string, returnEarly, suppresErrors bool, maxWorkers int, logFile *os.File) (fileFound, dirFound bool, stats SearchStats, err error) {
	var re *regexp.Regexp
	if regexPattern != "" {
		re, err = regexp.Compile(regexPattern)
		if err != nil {
			return false, false, stats, fmt.Errorf("invalid regex pattern: %v", err)
		}
	}

	// Create a channel to receive search results
	resultChan := make(chan SearchResult)

	// Channel to signal early termination to all workers
	doneChan := make(chan struct{})

	// Use WaitGroup to track when all goroutines are done
	var wg sync.WaitGroup

	// Create a semaphore channel to limit concurrent goroutines
	// This prevents spawning too many goroutines at once
	semaphore := make(chan struct{}, maxWorkers)

	// Create a mutex to protect shared variables
	var mu sync.Mutex

	// Function to process a directory
	var processDir func(path string, depth int)
	processDir = func(path string, depth int) {
		defer wg.Done()

		// Read directory entries
		entries, err := os.ReadDir(path)
		if !suppresErrors && err != nil {
			log.Printf("Error reading directory %s: %v", path, err)
			return
		}

		// First, queue subdirectories to be processed concurrently
		for _, entry := range entries {
			// Check if early termination was signaled
			select {
			case <-doneChan:
				return
			default:
				// Continue processing
			}

			entryPath := filepath.Join(path, entry.Name())

			// Check for matches
			matched := false
			matchType := ""

			// Regex match
			if re != nil && re.MatchString(entry.Name()) {
				matched = true
				matchType = "regex"
			}

			// Directory match
			if dirName != "" && entry.IsDir() && entry.Name() == dirName {
				matched = true
				matchType = "dir"
			}

			// File match
			if fileName != "" && !entry.IsDir() && entry.Name() == fileName {
				matched = true
				matchType = "file"
			}

			// If there's a match, send the result
			if matched {
				result := SearchResult{
					Path:    entryPath,
					IsDir:   entry.IsDir(),
					Matched: matchType,
				}

				// Send result to channel
				select {
				case resultChan <- result:
					// Successfully sent result
				case <-doneChan:
					return
				}
			}

			// If it's a directory, process it concurrently
			if entry.IsDir() {
				wg.Add(1)

				// Try to acquire a slot from the semaphore
				// This blocks if we already have maxWorkers goroutines running
				select {
				case semaphore <- struct{}{}:
					// We acquired a slot, process in a new goroutine
					go func(dirPath string, d int) {
						defer func() { <-semaphore }() // Release the semaphore slot when done
						processDir(dirPath, d+1)
					}(entryPath, depth+1)
				case <-doneChan:
					wg.Done() // We're not going to run this task, so decrement WaitGroup
					return
				default:
					// We've hit our concurrency limit, process synchronously instead
					processDir(entryPath, depth+1)
				}
			}
		}
	}

	// Create a mutex for file logging to prevent interleaved writes
	var logMutex sync.Mutex

	// Start a goroutine to collect results
	go func() {
		for result := range resultChan {
			mu.Lock()

			var outputMsg string
			switch result.Matched {
			case "file":
				outputMsg = fmt.Sprintf("File found at path: %s", result.Path)
				fmt.Println(outputMsg)
				fileFound = true
				stats.FilesFound++
			case "dir":
				outputMsg = fmt.Sprintf("Directory found at path: %s", result.Path)
				fmt.Println(outputMsg)
				dirFound = true
				stats.DirsFound++
			case "regex":
				outputMsg = fmt.Sprintf("Match found at path: %s", result.Path)
				fmt.Println(outputMsg)
				stats.RegexMatches++
				if result.IsDir {
					dirFound = true
				} else {
					fileFound = true
				}
			}

			// Write to log file if provided
			if logFile != nil {
				logMutex.Lock()
				fmt.Fprintf(logFile, "%s\n", outputMsg)
				logMutex.Unlock()
			}

			// If returnEarly flag is set and we found what we're looking for
			shouldTerminate := returnEarly
			if returnEarly {
				if fileName != "" && dirName != "" {
					shouldTerminate = fileFound && dirFound
				} else if fileName != "" {
					shouldTerminate = fileFound
				} else if dirName != "" {
					shouldTerminate = dirFound
				}
			}

			if shouldTerminate {
				close(doneChan) // Signal all goroutines to terminate
			}
			mu.Unlock()
		}
	}()

	// Start the initial search from the root directory
	wg.Add(1)
	go processDir(rootDir, 0)

	// Wait for all goroutines to finish
	wg.Wait()

	// Close the result channel to terminate the collector goroutine
	close(resultChan)

	return fileFound, dirFound, stats, nil
}

func main() {
	fileName := flag.String("file", "", "name of the file to search")
	dirName := flag.String("dir", "", "specify if it's a directory")
	rootDir := flag.String("root", ".", "Root directory to start the search")
	returnEarly := flag.Bool("r", false, "Return early after finding the first match")
	regexPattern := flag.String("regex", "", "Regex pattern to match file/directory names")
	workers := flag.Int("workers", 10, "Maximum number of concurrent workers")
	enableLog := flag.Bool("log", false, "Log all matches to a text file")
	logFilePath := flag.String("logfile", "search_results.log", "Path to log file (used with -log)")
	suppresErrors := flag.Bool("noerrors", false, "whether to log errors or not")
	flag.Parse()

	if *fileName == "" && *dirName == "" && *regexPattern == "" {
		fmt.Println("Please provide at least one search target (-file, -dir, or -regex)")
		return
	}

	// Validate if the root directory exists
	if _, err := os.Stat(*rootDir); os.IsNotExist(err) {
		log.Fatalf("Error: Specified root directory '%s' does not exist.\n", *rootDir)
	}

	// Open log file if logging is enabled
	var logFile *os.File
	if *enableLog {
		var err error
		logFile, err = os.Create(*logFilePath)
		if err != nil {
			log.Fatalf("Error creating log file: %v", err)
		}
		defer logFile.Close()

		// Write header to log file
		timestamp := time.Now().Format("2024-03-05 15:04:05")
		fmt.Fprintf(logFile, "Search results - %s\n", timestamp)
		fmt.Fprintf(logFile, "Root directory: %s\n", *rootDir)
		if *fileName != "" {
			fmt.Fprintf(logFile, "File name: %s\n", *fileName)
		}
		if *dirName != "" {
			fmt.Fprintf(logFile, "Directory name: %s\n", *dirName)
		}
		if *regexPattern != "" {
			fmt.Fprintf(logFile, "Regex pattern: %s\n", *regexPattern)
		}
		fmt.Fprintf(logFile, "-------------------------------------------\n")
	}

	start := time.Now()
	fileFound, dirFound, stats, err := searchConcurrent(*rootDir, *fileName, *dirName, *regexPattern, *returnEarly, *suppresErrors, *workers, logFile)

	if err != nil {
		log.Fatal("Error during search: ", err)
	}

	if *fileName != "" && !fileFound {
		fmt.Println("File not found")
	}
	if *dirName != "" && !dirFound {
		fmt.Println("Directory not found")
	}

	fmt.Println("\nSearch Statistics:")
	if *regexPattern != "" {
		fmt.Printf("- Regex matches found: %d\n", stats.RegexMatches)
	}
	if *fileName != "" {
		fmt.Printf("- Named files found: %d\n", stats.FilesFound)
	}
	if *dirName != "" {
		fmt.Printf("- Named directories found: %d\n", stats.DirsFound)
	}

	// Write statistics to log file if enabled
	if *enableLog {
		fmt.Fprintf(logFile, "\nSearch Statistics:\n")
		if *regexPattern != "" {
			fmt.Fprintf(logFile, "- Regex matches found: %d\n", stats.RegexMatches)
		}
		if *fileName != "" {
			fmt.Fprintf(logFile, "- Named files found: %d\n", stats.FilesFound)
		}
		if *dirName != "" {
			fmt.Fprintf(logFile, "- Named directories found: %d\n", stats.DirsFound)
		}
		fmt.Fprintf(logFile, "Search completed in %v\n", time.Since(start))

		fmt.Printf("Results written to log file: %s\n", *logFilePath)
	}

	fmt.Printf("Search completed in %v\n", time.Since(start))
}
