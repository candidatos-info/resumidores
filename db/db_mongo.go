package main

import (
	"github.com/candidatos-info/descritor"
	"gopkg.in/mgo.v2"
)

//Client manages all iteractions with mongodb
type Client struct {
	client *mgo.Database
	dbName string
}

//New returns an db connection instance that can be used for CRUD opetations
func New(url, database string) (*Client, error) {
	session, err := mgo.Dial(url)
	if err != nil {
		return nil, err
	}
	return &Client{
		client: session.DB(database),
		dbName: database,
	}, nil
}

// SaveCandidate saves a new user
func (db *Client) SaveCandidate(c *descritor.CandidateForDB) (*descritor.CandidateForDB, error) {
	return c, db.client.C(descritor.CandidaturesCollection).Insert(c)
}
