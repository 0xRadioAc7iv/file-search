package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

func lookForFile(file_name string) (bool, error) {
	found := false

	err := filepath.WalkDir(".", func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.Name() == file_name {
			found = true
		}

		return nil
	})

	return found, err
}

func main() {
	file_name := flag.String("file", "", "name of the file to search")
	flag.Parse()

	found, err := lookForFile(*file_name)

	if err != nil {
		log.Fatal(err)
	}

	if found {
		fmt.Println("File was found")
	} else {
		fmt.Println("File was not found")
	}
}
