package main

import (
	"testing"

	"github.com/matryer/is"
)

func TestListFilesRecursively(t *testing.T) {
	is := is.New(t)
	expectedFiles := []string{"testdata/ba/prefeito/2016/11-fulano.zip", "testdata/ba/prefeito/2020/11-fulano.zip"}
	files, err := listFilesRecursively("testdata")
	is.NoErr(err)
	is.Equal(files, expectedFiles)
}
