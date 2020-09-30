package main

import (
	"context"
	"fmt"
	"time"

	"github.com/candidatos-info/descritor"
	pagination "github.com/gobeam/mongo-go-pagination"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gopkg.in/mgo.v2/bson"
)

const (
	timeout = 10 // in seconds
)

//Client manages all iteractions with mongodb
type Client struct {
	client *mongo.Client
	dbName string
}

//New returns an db connection instance that can be used for CRUD opetations
func New(dbURL, dbName string) (*Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(dbURL))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB at link [%s], error %v", dbURL, err)
	}
	return &Client{
		client: client,
		dbName: dbName,
	}, nil
}

// UpdateCandidate sets candidate's recurrent flag
func (c *Client) UpdateCandidate(candidate *descritor.CandidateForDB) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout*time.Second)
	defer cancel()
	filter := bson.M{"legal_code": candidate.LegalCode, "year": candidate.Year}
	update := bson.M{"$set": bson.M{"recurrent": candidate.Recurrent}}
	if _, err := c.client.Database(c.dbName).Collection(descritor.CandidaturesCollection).UpdateOne(ctx, filter, update); err != nil {
		return fmt.Errorf("falha ao setar flag de recorrência no candidato [%s], error %v", candidate.LegalCode, err)
	}
	return nil
}

// FindCandidateByYearAndLegalCode searches for a candidate using year and legal code
func (c *Client) FindCandidateByYearAndLegalCode(year int, legalCode string) (*descritor.CandidateForDB, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout*time.Second)
	defer cancel()
	var candidate descritor.CandidateForDB
	filter := bson.M{"legal_code": legalCode, "year": year}
	if err := c.client.Database(c.dbName).Collection(descritor.CandidaturesCollection).FindOne(ctx, filter).Decode(&candidate); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("Falha ao buscar candidato pelo ano [%d] e pelo legal code [%s] no banco na collection [%s], erro %v", year, legalCode, descritor.CandidaturesCollection, err)
	}
	return &candidate, nil
}

// FindCandidatesWithParams searches for a list of candidates with given params
func (c *Client) FindCandidatesWithParams(queryMap map[string]interface{}, pageSize, page int) ([]*descritor.CandidateForDB, *pagination.PaginationData, error) {
	var candidatures []*descritor.CandidateForDB
	paginatedData, err := pagination.New(c.client.Database(c.dbName).Collection(descritor.CandidaturesCollection)).Limit(int64(pageSize)).Page(int64(page)).Sort("transparency", -1).Filter(resolveQuery(queryMap)).Find()
	if err != nil {
		return nil, nil, fmt.Errorf("Falha ao buscar por lista candidatos, erro %v", err)
	}
	for _, raw := range paginatedData.Data {
		var candidature *descritor.CandidateForDB
		if err := bson.Unmarshal(raw, &candidature); err != nil {
			return nil, nil, fmt.Errorf("Falha ao deserializar struct de candidatura a partir da resposta do banco, erro %v", err)
		}
		candidatures = append(candidatures, candidature)
	}
	return candidatures, &paginatedData.Pagination, nil
}

func resolveQuery(query map[string]interface{}) bson.M {
	result := make(bson.M, len(query))
	for k, v := range query {
		result[k] = v
	}
	return result
}
