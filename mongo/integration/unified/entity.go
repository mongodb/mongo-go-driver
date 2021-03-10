// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package unified

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/gridfs"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// entityOptions represents all options that can be used to configure an entity. Because there are multiple entity
// types, only a subset of the options that this type contains apply to any given entity.
type entityOptions struct {
	// Options that apply to all entity types.
	ID string `bson:"id"`

	// Options for client entities.
	URIOptions          bson.M            `bson:"uriOptions"`
	UseMultipleMongoses *bool             `bson:"useMultipleMongoses"`
	ObserveEvents       []string          `bson:"observeEvents"`
	IgnoredCommands     []string          `bson:"ignoreCommandMonitoringEvents"`
	serverAPIOptions    *serverAPIOptions `bson:"serverApi"`

	// Options for database entities.
	DatabaseName    string                 `bson:"databaseName"`
	DatabaseOptions *dbOrCollectionOptions `bson:"databaseOptions"`

	// Options for collection entities.
	CollectionName    string                 `bson:"collectionName"`
	CollectionOptions *dbOrCollectionOptions `bson:"collectionOptions"`

	// Options for session entities.
	sessionOptions *sessionOptions `bson:"sessionOptions"`

	// Options for GridFS bucket entities.
	gridFSBucketOptions *gridFSBucketOptions `bson:"bucketOptions"`

	// Options that reference other entities.
	ClientID   string `bson:"client"`
	DatabaseID string `bson:"database"`
}

// entityMap is used to store entities during tests. This type enforces uniqueness so no two entities can have the same
// ID, even if they are of different types. It also enforces referential integrity so construction of an entity that
// references another (e.g. a database entity references a client) will fail if the referenced entity does not exist.
type entityMap struct {
	allEntities    map[string]struct{}
	changeStreams  map[string]*mongo.ChangeStream
	clientEntities map[string]*clientEntity
	dbEntites      map[string]*mongo.Database
	collEntities   map[string]*mongo.Collection
	sessions       map[string]mongo.Session
	gridfsBuckets  map[string]*gridfs.Bucket
	bsonValues     map[string]bson.RawValue
}

func newEntityMap() *entityMap {
	return &entityMap{
		allEntities:    make(map[string]struct{}),
		gridfsBuckets:  make(map[string]*gridfs.Bucket),
		bsonValues:     make(map[string]bson.RawValue),
		changeStreams:  make(map[string]*mongo.ChangeStream),
		clientEntities: make(map[string]*clientEntity),
		collEntities:   make(map[string]*mongo.Collection),
		dbEntites:      make(map[string]*mongo.Database),
		sessions:       make(map[string]mongo.Session),
	}
}

func (em *entityMap) addBSONEntity(id string, val bson.RawValue) error {
	if err := em.verifyEntityDoesNotExist(id); err != nil {
		return err
	}

	em.allEntities[id] = struct{}{}
	em.bsonValues[id] = val
	return nil
}

func (em *entityMap) addChangeStreamEntity(id string, stream *mongo.ChangeStream) error {
	if err := em.verifyEntityDoesNotExist(id); err != nil {
		return err
	}

	em.allEntities[id] = struct{}{}
	em.changeStreams[id] = stream
	return nil
}

func (em *entityMap) addEntity(ctx context.Context, entityType string, entityOptions *entityOptions) error {
	if err := em.verifyEntityDoesNotExist(entityOptions.ID); err != nil {
		return err
	}

	var err error
	switch entityType {
	case "client":
		err = em.addClientEntity(ctx, entityOptions)
	case "database":
		err = em.addDatabaseEntity(entityOptions)
	case "collection":
		err = em.addCollectionEntity(entityOptions)
	case "session":
		err = em.addSessionEntity(entityOptions)
	case "bucket":
		err = em.addGridFSBucketEntity(entityOptions)
	default:
		return fmt.Errorf("unrecognized entity type %q", entityType)
	}

	if err != nil {
		return fmt.Errorf("error constructing entity of type %q: %v", entityType, err)
	}
	em.allEntities[entityOptions.ID] = struct{}{}
	return nil
}

func (em *entityMap) gridFSBucket(id string) (*gridfs.Bucket, error) {
	bucket, ok := em.gridfsBuckets[id]
	if !ok {
		return nil, newEntityNotFoundError("gridfs bucket", id)
	}
	return bucket, nil
}

func (em *entityMap) bsonValue(id string) (bson.RawValue, error) {
	val, ok := em.bsonValues[id]
	if !ok {
		return emptyRawValue, newEntityNotFoundError("BSON", id)
	}
	return val, nil
}

func (em *entityMap) changeStream(id string) (*mongo.ChangeStream, error) {
	client, ok := em.changeStreams[id]
	if !ok {
		return nil, newEntityNotFoundError("change stream", id)
	}
	return client, nil
}

func (em *entityMap) client(id string) (*clientEntity, error) {
	client, ok := em.clientEntities[id]
	if !ok {
		return nil, newEntityNotFoundError("client", id)
	}
	return client, nil
}

func (em *entityMap) clients() map[string]*clientEntity {
	return em.clientEntities
}

func (em *entityMap) collections() map[string]*mongo.Collection {
	return em.collEntities
}

func (em *entityMap) collection(id string) (*mongo.Collection, error) {
	coll, ok := em.collEntities[id]
	if !ok {
		return nil, newEntityNotFoundError("collection", id)
	}
	return coll, nil
}

func (em *entityMap) database(id string) (*mongo.Database, error) {
	db, ok := em.dbEntites[id]
	if !ok {
		return nil, newEntityNotFoundError("database", id)
	}
	return db, nil
}

func (em *entityMap) session(id string) (mongo.Session, error) {
	sess, ok := em.sessions[id]
	if !ok {
		return nil, newEntityNotFoundError("session", id)
	}
	return sess, nil
}

// close disposes of the session and client entities associated with this map.
func (em *entityMap) close(ctx context.Context) []error {
	for _, sess := range em.sessions {
		sess.EndSession(ctx)
	}

	var errs []error
	for id, client := range em.clientEntities {
		if err := client.Disconnect(ctx); err != nil {
			errs = append(errs, fmt.Errorf("error closing client with ID %q: %v", id, err))
		}
	}
	return errs
}

func (em *entityMap) addClientEntity(ctx context.Context, entityOptions *entityOptions) error {
	var client *clientEntity
	client, err := newClientEntity(ctx, entityOptions)
	if err != nil {
		return fmt.Errorf("error creating client entity: %v", err)
	}

	em.clientEntities[entityOptions.ID] = client
	return nil
}

func (em *entityMap) addDatabaseEntity(entityOptions *entityOptions) error {
	client, ok := em.clientEntities[entityOptions.ClientID]
	if !ok {
		return newEntityNotFoundError("client", entityOptions.ClientID)
	}

	dbOpts := options.Database()
	if entityOptions.DatabaseOptions != nil {
		dbOpts = entityOptions.DatabaseOptions.DBOptions
	}

	em.dbEntites[entityOptions.ID] = client.Database(entityOptions.DatabaseName, dbOpts)
	return nil
}

func (em *entityMap) addCollectionEntity(entityOptions *entityOptions) error {
	db, ok := em.dbEntites[entityOptions.DatabaseID]
	if !ok {
		return newEntityNotFoundError("database", entityOptions.DatabaseID)
	}

	collOpts := options.Collection()
	if entityOptions.CollectionOptions != nil {
		collOpts = entityOptions.CollectionOptions.CollectionOptions
	}

	em.collEntities[entityOptions.ID] = db.Collection(entityOptions.CollectionName, collOpts)
	return nil
}

func (em *entityMap) addSessionEntity(entityOptions *entityOptions) error {
	client, ok := em.clientEntities[entityOptions.ClientID]
	if !ok {
		return newEntityNotFoundError("client", entityOptions.ClientID)
	}

	sessionOpts := options.Session()
	if entityOptions.sessionOptions != nil {
		sessionOpts = entityOptions.sessionOptions.SessionOptions
	}

	sess, err := client.StartSession(sessionOpts)
	if err != nil {
		return fmt.Errorf("error starting session: %v", err)
	}

	em.sessions[entityOptions.ID] = sess
	return nil
}

func (em *entityMap) addGridFSBucketEntity(entityOptions *entityOptions) error {
	db, ok := em.dbEntites[entityOptions.DatabaseID]
	if !ok {
		return newEntityNotFoundError("database", entityOptions.DatabaseID)
	}

	bucketOpts := options.GridFSBucket()
	if entityOptions.gridFSBucketOptions != nil {
		bucketOpts = entityOptions.gridFSBucketOptions.BucketOptions
	}

	bucket, err := gridfs.NewBucket(db, bucketOpts)
	if err != nil {
		return fmt.Errorf("error creating GridFS bucket: %v", err)
	}

	em.gridfsBuckets[entityOptions.ID] = bucket
	return nil
}

func (em *entityMap) verifyEntityDoesNotExist(id string) error {
	if _, ok := em.allEntities[id]; ok {
		return fmt.Errorf("entity with ID %q already exists", id)
	}
	return nil
}

func newEntityNotFoundError(entityType, entityID string) error {
	return fmt.Errorf("no %s entity found with ID %q", entityType, entityID)
}
