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

func searchConcurrent(rootDir, fileName, dirName, regexPattern string, returnEarly bool, maxWorkers int) (fileFound, dirFound bool, stats SearchStats, err error) {
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
		if err != nil {
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

	// Start a goroutine to collect results
	go func() {
		for result := range resultChan {
			mu.Lock()
			switch result.Matched {
			case "file":
				fmt.Println("File found at path:", result.Path)
				fileFound = true
				stats.FilesFound++
			case "dir":
				fmt.Println("Directory found at path:", result.Path)
				dirFound = true
				stats.DirsFound++
			case "regex":
				fmt.Printf("Match found at path: %s\n", result.Path)
				stats.RegexMatches++
				if result.IsDir {
					dirFound = true
				} else {
					fileFound = true
				}
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
	flag.Parse()

	if *fileName == "" && *dirName == "" && *regexPattern == "" {
		fmt.Println("Please provide at least one search target (-file, -dir, or -regex)")
		return
	}

	// Validate if the root directory exists
	if _, err := os.Stat(*rootDir); os.IsNotExist(err) {
		log.Fatalf("Error: Specified root directory '%s' does not exist.\n", *rootDir)
	}

	start := time.Now()
	fileFound, dirFound, stats, err := searchConcurrent(*rootDir, *fileName, *dirName, *regexPattern, *returnEarly, *workers)
	fmt.Printf("\nSearch completed in %v\n", time.Since(start))

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
		fmt.Printf("- Files found: %d\n", stats.FilesFound)
	}
	if *dirName != "" {
		fmt.Printf("- Directories found: %d\n", stats.DirsFound)
	}
}
