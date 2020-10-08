package main

import (
	"context"
	"fmt"
	"time"

	"github.com/candidatos-info/descritor"
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

// UpdateCandidate updates candidate's data
func (db *Client) UpdateCandidate(year int, sequencialID, pictureURL string) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout*time.Second)
	defer cancel()
	filter := bson.M{
		"sequencial_candidate": sequencialID,
		"year":                 year,
	}
	update := bson.M{
		"$set": bson.M{
			"photo_url": pictureURL,
		},
	}
	if _, err := db.client.Database(db.dbName).Collection(descritor.CandidaturesCollection).UpdateOne(ctx, filter, update); err != nil {
		return fmt.Errorf("falha ao salvar foto de candidato de sequencial ID [%s], erro %v", sequencialID, err)
	}
	return nil
}
