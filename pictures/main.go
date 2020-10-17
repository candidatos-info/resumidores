package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"strconv"

	"github.com/gocarina/gocsv"
	"golang.org/x/text/encoding/charmap"
)

type referencesSchema struct {
	PictureURI   string `csv:"picture_uri"`
	SequentialID string `csv:"tse_sequencial_id"`
}

func main() {
	dbURL := flag.String("dbURL", "", "URL de conexão com banco MongoDB")
	dbName := flag.String("dbName", "", "nome do banco de dados")
	year := flag.Int("year", 0, "ano para da eleição para remover candidato")
	picturesReferences := flag.String("picturesReferences", "", "caminho do arquivo de referência das fotos dos candidatos")
	offset := flag.Int("offset", 0, "ponto de início de processamento")
	flag.Parse()
	if *dbURL == "" {
		log.Fatal("informe a URL do banco")
	}
	if *dbName == "" {
		log.Fatal("informe o nome do banco")
	}
	if *year == 0 {
		log.Fatal("informe o ano a ser processado")
	}
	if *picturesReferences == "" {
		log.Fatal("informe o path do arquivo de referências das fotos")
	}
	c, err := New(*dbURL, *dbName)
	if err != nil {
		log.Fatalf("failed to connect with data base: %v\n", err)
	}
	if err := process(c, *picturesReferences, *year, *offset); err != nil {
		log.Fatalf("falha ao processar, erro %v", err)
	}
}

func process(c *Client, picturesReferences string, year, offset int) error {
	nextOffset := offset
	file, err := os.Open(picturesReferences)
	if err != nil {
		return fmt.Errorf("falha ao abrir arquivo .csv descomprimido %s. OFFSET: [%d], erro %q", picturesReferences, nextOffset, err)
	}
	defer file.Close()
	gocsv.SetCSVReader(func(in io.Reader) gocsv.CSVReader {
		// Enforcing reading the TSE zip file as ISO 8859-1 (latin 1)
		r := csv.NewReader(charmap.ISO8859_1.NewDecoder().Reader(in))
		r.LazyQuotes = true
		r.Comma = ','
		return r
	})
	var refs []*referencesSchema
	if err := gocsv.UnmarshalFile(file, &refs); err != nil {
		return fmt.Errorf("falha ao inflar slice de referências de fotos usando arquivo csv [%s]. OFFSET: [%d], erro %v", picturesReferences, nextOffset, err)
	}
	sort.Slice(refs, func(i, j int) bool { // sorting list using sequencial ID gotten from local path
		prevIndex, err := strconv.Atoi(refs[i].SequentialID)
		if err != nil {
			log.Fatalf("falha ao converter o sequencial ID [%s] para inteiro, erro %v", refs[i].SequentialID, err)
		}
		nextIndex, err := strconv.Atoi(refs[j].SequentialID)
		if err != nil {
			log.Fatalf("falha ao converter o sequencial ID [%s] para inteiro, erro %v", refs[j].SequentialID, err)
		}
		return prevIndex < nextIndex
	})
	for _, pictureReference := range refs[offset:] {
		if err := c.UpdateCandidate(year, pictureReference.SequentialID, pictureReference.PictureURI); err != nil {
			return fmt.Errorf("falha ao salvar foto do candidato [%s]. OFFSET: [%d], erro %v", pictureReference.SequentialID, nextOffset, err)
		}
		log.Printf("changed picture of candidate [%s]\n", pictureReference.SequentialID)
		nextOffset++
	}
	return nil
}
