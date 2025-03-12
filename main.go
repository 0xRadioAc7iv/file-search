package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

func search(rootDir, fileName, dirName string, returnEarly bool) (fileFound, dirFound bool, err error) {
	err = filepath.WalkDir(rootDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Searches for directories
		if dirName != "" && d.IsDir() && d.Name() == dirName {
			fmt.Println("Directory found at path:", path)
			dirFound = true
		}

		// Searches for files
		if fileName != "" && !d.IsDir() && d.Name() == fileName {
			fmt.Println("File found at path:", path)
			fileFound = true
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
	rootDir := flag.String("root", ".", "Root directory to start the search (default: current directory)")
	returnEarly := flag.Bool("r", false, "Return early after finding the first match")
	flag.Parse()

	if *fileName == "" && *dirName == "" {
		fmt.Println("Please provide a file or directory name using -file or -dir flag respectively")
		return
	}

	fileFound, dirFound, err := search(*rootDir, *fileName, *dirName, *returnEarly)

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
