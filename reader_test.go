package main

import (
	"testing"

	"github.com/matryer/is"
)

func TestGetAllFilesFromFolder(t *testing.T) {
	is := is.New(t)
	expectedFiles := []string{"fixtures/storage/bahia/prefeito/2016/11-fulano.zip", "fixtures/storage/bahia/prefeito/2020/11-fulano.zip"}
	files, err := getFilesFromFolder("fixtures")
	is.NoErr(err)
	is.Equal(files, expectedFiles)
}
