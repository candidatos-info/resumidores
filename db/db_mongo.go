package main

import (
	"context"
	"fmt"
	"time"

	"github.com/candidatos-info/descritor"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
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

// SaveCandidate saves a new user
func (db *Client) SaveCandidate(c *descritor.CandidateForDB) (*descritor.CandidateForDB, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout*time.Second)
	defer cancel()
	if _, err := db.client.Database(db.dbName).Collection(descritor.CandidaturesCollection).InsertOne(ctx, c); err != nil {
		return nil, fmt.Errorf("failed to persist candidature data into db, error %v", err)
	}
	return c, nil
}

// SaveLocation saves a new location
func (db *Client) SaveLocation(l *descritor.Location) (*descritor.Location, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout*time.Second)
	defer cancel()
	if _, err := db.client.Database(db.dbName).Collection(descritor.LocationsCollection).InsertOne(ctx, l); err != nil {
		return nil, fmt.Errorf("failed to persist location data into db, error %v", err)
	}
	return l, nil
}
