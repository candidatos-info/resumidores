package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"strings"

	"github.com/candidatos-info/descritor"
)

func main() {
	fakeCandidatesFilePath := flag.String("fakeCandidatesFilePath", "", "caminho para o arquivo com candidatos falsos")
	dbURL := flag.String("dbURL", "", "URL de conexão com banco MongoDB")
	dbName := flag.String("dbName", "", "nome do banco de dados")
	emailToRemove := flag.String("emailToRemove", "", "email para ser removido")
	year := flag.Int("year", 0, "ano para da eleição para remover candidato")
	flag.Parse()
	if *dbURL == "" {
		log.Fatal("informe a URL de conexão com banco MongoDB")
	}
	if *dbName == "" {
		log.Fatal("informe o nome do banco de dados")
	}
	c, err := New(*dbURL, *dbName)
	if err != nil {
		log.Fatalf("failed to connect with data base: %v\n", err)
	}
	if *emailToRemove == "" {
		if *fakeCandidatesFilePath == "" {
			log.Fatal("informe o caminho para o arquivo contendo candidatos falsos")
		}
		if err := addFakeCandidate(c, *fakeCandidatesFilePath); err != nil {
			log.Fatalf("falha ao adicionar candidato falso, erro %v", err)
		}
	} else {
		if *year == 0 {
			log.Fatal("informe o ano da eleição para remover o candidato falso")
		}
		if err := removeFakeCandidate(c, *emailToRemove, *year); err != nil {
			log.Fatalf("falha ao remover candidato falso, erro %v", err)
		}
	}
}

func removeFakeCandidate(c *Client, emailToRemove string, year int) error {
	return c.RemoveCandidate(strings.ToUpper(emailToRemove), year)
}

func addFakeCandidate(c *Client, fakeCandidatesFilePath string) error {
	var fakeCandidates []*descritor.CandidateForDB
	b, err := ioutil.ReadFile(fakeCandidatesFilePath)
	if err != nil {
		return fmt.Errorf("falha ao abrir arquivo JSON com candidatos falsos [%s], erro %v", fakeCandidatesFilePath, err)
	}
	if err := json.Unmarshal(b, &fakeCandidates); err != nil {
		return fmt.Errorf("falha ao deserializar arquivo JSON de candidaturas falsas [%s], erro %v", fakeCandidatesFilePath, err)
	}
	for _, candidate := range fakeCandidates {
		counter := 0.0
		if candidate.Biography != "" {
			counter++
		}
		if candidate.Proposals != nil && len(candidate.Proposals) > 0 {
			counter++
		}
		if candidate.Contacts != nil && len(candidate.Contacts) > 0 {
			counter++
		}
		candidate.Transparency = counter / 3.0
		candidate.SequencialCandidate = fmt.Sprintf("@%s", candidate.SequencialCandidate) // All fake candidate sequencial number should start with '@'.
		if _, err := c.SaveCandidate(candidate); err != nil {
			return fmt.Errorf("falha ao salvar canidato falso no banco, erro %v", err)
		}
		log.Printf("saved fake canidate [%s]\n", candidate.Name)
	}
	return nil
}
