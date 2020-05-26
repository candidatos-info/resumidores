package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
)

func getFilesFromFolder(folder string) (files []string, err error) {
	err = filepath.Walk(folder,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() == false {
				files = append(files, path)
			}
			return nil
		})
	return files, err
}

func main() {
	files, err := getFilesFromFolder(os.Args[1])
	if err != nil {
		log.Println(err)
	} else {
		fmt.Println(files)
	}
}
