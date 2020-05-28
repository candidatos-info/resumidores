package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func listFilesRecursively(folder string) ([]string, error) {
	var files []string
	err := filepath.Walk(folder,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return fmt.Errorf("Erro iterando no arquivo \"%s\". Diretório base \"%s\". Erro: %q", path, folder, err)
			}

			if !info.IsDir() {
				files = append(files, path)
			}
			return nil
		})
	return files, err
}

func main() {
	files, err := listFilesRecursively(os.Args[1])
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(files)
	}
}
