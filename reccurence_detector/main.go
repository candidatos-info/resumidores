package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/candidatos-info/descritor"
)

const (
	pageSize = 100 // page size of a MongoDB query
)

func main() {
	dbName := flag.String("dbName", "", "nome do banco de dados")
	dbURL := flag.String("dbURL", "", "URL de conexão com banco de dados")
	currentElectionYear := flag.Int("currentElectionYear", 0, "ano da eleição atual")
	prevElectionYear := flag.Int("prevElectionYear", 0, "ano da eleição anterior para fazer a comparação")
	state := flag.String("state", "", "estado a ser processado")
	offset := flag.Int("offset", 1, "ponto de início do processamento")
	flag.Parse()
	if *state == "" {
		log.Fatal("informe o estado a ser processado")
	}
	if *currentElectionYear == 0 {
		log.Fatal("informe o ano da eleição atual")
	}
	if *prevElectionYear == 0 {
		log.Fatal("informe o ano da eleição que deseja comparar")
	}
	if *dbName == "" {
		log.Fatal("informe o nome do banco")
	}
	if *dbURL == "" {
		log.Fatal("informe a URL de conexão com o banco")
	}
	dbClient, err := New(*dbURL, *dbName)
	if err != nil {
		log.Fatalf("falha ao se conectar com banco, error %v", err)
	}
	if err := process(*offset, *state, *currentElectionYear, *prevElectionYear, dbClient); err != nil {
		log.Fatalf("falha ao processar verificação de recorrência, error %v", err)
	}
}

func process(offset int, state string, currentElectionYear, prevElectionYear int, dbClient *Client) error {
	queryFilter := make(map[string]interface{})
	queryFilter["state"] = state
	queryFilter["year"] = currentElectionYear
	_, paginationInfo, err := dbClient.FindCandidatesWithParams(queryFilter, pageSize, 1)
	if err != nil {
		return fmt.Errorf("falha ao buscar informaçōes de paginação do banco, erro %v", err)
	}
	page := offset
	for page = 1; page <= int(paginationInfo.TotalPage); page++ {
		candidates, _, err := dbClient.FindCandidatesWithParams(queryFilter, pageSize, page)
		if err != nil {
			return fmt.Errorf("falha ao buscar candidatos do banco na página [%d]. OFFSET: [%d], erro %v", page, offset, err)
		}
		if err := setRecurrentCandidates(candidates, prevElectionYear, dbClient); err != nil {
			return fmt.Errorf("falha ao setar candidatos recorrentes. OFFSET: [%d], erro %v", offset, err)
		}
	}
	return nil
}

func setRecurrentCandidates(candidates []*descritor.CandidateForDB, prevElectionYear int, dbClient *Client) error {
	for _, candidate := range candidates {
		c, err := dbClient.FindCandidateByYearAndLegalCode(prevElectionYear, candidate.LegalCode)
		if err != nil {
			return fmt.Errorf("falha ao buscar candidato por ano [%d] e CPF [%s], erro %v", prevElectionYear, candidate.LegalCode, err)
		}
		if c == nil {
			log.Printf("candidate with legal code [%s] is NOT recurrent\n", candidate.LegalCode)
			return nil
		}
		candidate.Recurrent = true
		if err := dbClient.UpdateCandidate(candidate); err != nil {
			return fmt.Errorf("falha ao atualizar candidato com CPf [%s], erro %v", candidate.LegalCode, err)
		}
		log.Printf("candidate with legal code [%s] is recurrent\n", candidate.LegalCode)
	}
	return nil
}
