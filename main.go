package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
)

func search(rootDir, fileName, dirName, regexPattern string, returnEarly bool) (fileFound, dirFound bool, err error) {
	var re *regexp.Regexp
	if regexPattern != "" {
		re, err = regexp.Compile(regexPattern)
		if err != nil {
			return false, false, fmt.Errorf("invalid regex pattern: %v", err)
		}
	}

	err = filepath.WalkDir(rootDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		name := d.Name()

		if re != nil && re.MatchString(name) {
			fmt.Printf("Match found at path: %s\n", path)
			if d.IsDir() {
				dirFound = true
			} else {
				fileFound = true
			}
			if returnEarly {
				return filepath.SkipAll // Stop search if -r flag is enabled
			}
		}

		// Searches for directories
		if dirName != "" && d.IsDir() && name == dirName {
			fmt.Println("Directory found at path:", path)
			dirFound = true
			if returnEarly {
				return filepath.SkipAll
			}
		}

		// Searches for files
		if fileName != "" && !d.IsDir() && name == fileName {
			fmt.Println("File found at path:", path)
			fileFound = true
			if returnEarly {
				return filepath.SkipAll
			}
		}

		// If both are found and -r flag is enabled, stop searching
		if returnEarly && fileFound && dirFound {
			return filepath.SkipAll // Stops further search
		}

		return nil
	})

	return fileFound, dirFound, err
}

func main() {
	fileName := flag.String("file", "", "name of the file to search")
	dirName := flag.String("dir", "", "specify if it's a directory")
	rootDir := flag.String("root", ".", "Root directory to start the search")
	returnEarly := flag.Bool("r", false, "Return early after finding the first match")
	regexPattern := flag.String("regex", "", "Regex pattern to match file/directory names")
	flag.Parse()

	if *fileName == "" && *dirName == "" && *regexPattern == "" {
		fmt.Println("Please provide at least one search target (-file, -dir, or -regex)")
		return
	}

	fileFound, dirFound, err := search(*rootDir, *fileName, *dirName, *regexPattern, *returnEarly)

	// Validate if the root directory exists
	if _, err := os.Stat(*rootDir); os.IsNotExist(err) {
		log.Fatalf("Error: Specified root directory '%s' does not exist.\n", *rootDir)
	}

	if err != nil {
		log.Fatal("Error during search: ", err)
	}

	if *fileName != "" && !fileFound {
		fmt.Println("File not found")
	}
	if *dirName != "" && !dirFound {
		fmt.Println("Directory not found")
	}
}
