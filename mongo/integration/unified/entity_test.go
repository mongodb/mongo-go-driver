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

// EntityOptions represents all options that can be used to configure an entity. Because there are multiple entity
// types, only a subset of the options that this type contains apply to any given entity.
type EntityOptions struct {
	// Options that apply to all entity types.
	ID string `bson:"id"`

	// Options for client entities.
	URIOptions          bson.M   `bson:"uriOptions"`
	UseMultipleMongoses *bool    `bson:"useMultipleMongoses"`
	ObserveEvents       []string `bson:"observeEvents"`
	IgnoredCommands     []string `bson:"ignoreCommandMonitoringEvents"`

	// Options for database entities.
	DatabaseName    string                 `bson:"databaseName"`
	DatabaseOptions *DBOrCollectionOptions `bson:"databaseOptions"`

	// Options for collection entities.
	CollectionName    string                 `bson:"collectionName"`
	CollectionOptions *DBOrCollectionOptions `bson:"collectionOptions"`

	// Options for session entities.
	SessionOptions *SessionOptions `bson:"sessionOptions"`

	// Options for GridFS bucket entities.
	GridFSBucketOptions *GridFSBucketOptions `bson:"bucketOptions"`

	// Options that reference other entities.
	ClientID   string `bson:"client"`
	DatabaseID string `bson:"database"`
}

// EntityMap is used to store entities during tests. This type enforces uniqueness so no two entities can have the same
// ID, even if they are of different types. It also enforces referential integrity so construction of an entity that
// references another (e.g. a database entity references a client) will fail if the referenced entity does not exist.
type EntityMap struct {
	allEntities   map[string]struct{}
	changeStreams map[string]*mongo.ChangeStream
	clients       map[string]*ClientEntity
	dbs           map[string]*mongo.Database
	collections   map[string]*mongo.Collection
	sessions      map[string]mongo.Session
	gridfsBuckets map[string]*gridfs.Bucket
	bsonValues    map[string]bson.RawValue
}

func NewEntityMap() *EntityMap {
	return &EntityMap{
		allEntities:   make(map[string]struct{}),
		gridfsBuckets: make(map[string]*gridfs.Bucket),
		bsonValues:    make(map[string]bson.RawValue),
		changeStreams: make(map[string]*mongo.ChangeStream),
		clients:       make(map[string]*ClientEntity),
		collections:   make(map[string]*mongo.Collection),
		dbs:           make(map[string]*mongo.Database),
		sessions:      make(map[string]mongo.Session),
	}
}

func (em *EntityMap) AddBSONEntity(id string, val bson.RawValue) error {
	if err := em.verifyEntityDoesNotExist(id); err != nil {
		return err
	}

	em.allEntities[id] = struct{}{}
	em.bsonValues[id] = val
	return nil
}

func (em *EntityMap) AddChangeStreamEntity(id string, stream *mongo.ChangeStream) error {
	if err := em.verifyEntityDoesNotExist(id); err != nil {
		return err
	}

	em.allEntities[id] = struct{}{}
	em.changeStreams[id] = stream
	return nil
}

func (em *EntityMap) AddEntity(ctx context.Context, entityType string, entityOptions *EntityOptions) error {
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

func (em *EntityMap) GridFSBucket(id string) (*gridfs.Bucket, error) {
	bucket, ok := em.gridfsBuckets[id]
	if !ok {
		return nil, newEntityNotFoundError("gridfs bucket", id)
	}
	return bucket, nil
}

func (em *EntityMap) BSONValue(id string) (bson.RawValue, error) {
	val, ok := em.bsonValues[id]
	if !ok {
		return emptyRawValue, newEntityNotFoundError("BSON", id)
	}
	return val, nil
}

func (em *EntityMap) ChangeStream(id string) (*mongo.ChangeStream, error) {
	client, ok := em.changeStreams[id]
	if !ok {
		return nil, newEntityNotFoundError("change stream", id)
	}
	return client, nil
}

func (em *EntityMap) Client(id string) (*ClientEntity, error) {
	client, ok := em.clients[id]
	if !ok {
		return nil, newEntityNotFoundError("client", id)
	}
	return client, nil
}

func (em *EntityMap) Clients() map[string]*ClientEntity {
	return em.clients
}

func (em *EntityMap) Collections() map[string]*mongo.Collection {
	return em.collections
}

func (em *EntityMap) Collection(id string) (*mongo.Collection, error) {
	coll, ok := em.collections[id]
	if !ok {
		return nil, newEntityNotFoundError("collection", id)
	}
	return coll, nil
}

func (em *EntityMap) Database(id string) (*mongo.Database, error) {
	db, ok := em.dbs[id]
	if !ok {
		return nil, newEntityNotFoundError("database", id)
	}
	return db, nil
}

func (em *EntityMap) Session(id string) (mongo.Session, error) {
	sess, ok := em.sessions[id]
	if !ok {
		return nil, newEntityNotFoundError("session", id)
	}
	return sess, nil
}

// Close disposes of the session and client entities associated with this map.
func (em *EntityMap) Close(ctx context.Context) []error {
	for _, sess := range em.sessions {
		sess.EndSession(ctx)
	}

	var errs []error
	for id, client := range em.clients {
		if err := client.Disconnect(ctx); err != nil {
			errs = append(errs, fmt.Errorf("error closing client with ID %q: %v", id, err))
		}
	}
	return errs
}

func (em *EntityMap) addClientEntity(ctx context.Context, EntityOptions *EntityOptions) error {
	var client *ClientEntity
	client, err := NewClientEntity(ctx, EntityOptions)
	if err != nil {
		return fmt.Errorf("error creating client entity: %v", err)
	}

	em.clients[EntityOptions.ID] = client
	return nil
}

func (em *EntityMap) addDatabaseEntity(EntityOptions *EntityOptions) error {
	client, ok := em.clients[EntityOptions.ClientID]
	if !ok {
		return newEntityNotFoundError("client", EntityOptions.ClientID)
	}

	dbOpts := options.Database()
	if EntityOptions.DatabaseOptions != nil {
		dbOpts = EntityOptions.DatabaseOptions.DBOptions
	}

	em.dbs[EntityOptions.ID] = client.Database(EntityOptions.DatabaseName, dbOpts)
	return nil
}

func (em *EntityMap) addCollectionEntity(EntityOptions *EntityOptions) error {
	db, ok := em.dbs[EntityOptions.DatabaseID]
	if !ok {
		return newEntityNotFoundError("database", EntityOptions.DatabaseID)
	}

	collOpts := options.Collection()
	if EntityOptions.CollectionOptions != nil {
		collOpts = EntityOptions.CollectionOptions.CollectionOptions
	}

	em.collections[EntityOptions.ID] = db.Collection(EntityOptions.CollectionName, collOpts)
	return nil
}

func (em *EntityMap) addSessionEntity(EntityOptions *EntityOptions) error {
	client, ok := em.clients[EntityOptions.ClientID]
	if !ok {
		return newEntityNotFoundError("client", EntityOptions.ClientID)
	}

	sessionOpts := options.Session()
	if EntityOptions.SessionOptions != nil {
		sessionOpts = EntityOptions.SessionOptions.SessionOptions
	}

	sess, err := client.StartSession(sessionOpts)
	if err != nil {
		return fmt.Errorf("error starting session: %v", err)
	}

	em.sessions[EntityOptions.ID] = sess
	return nil
}

func (em *EntityMap) addGridFSBucketEntity(EntityOptions *EntityOptions) error {
	db, ok := em.dbs[EntityOptions.DatabaseID]
	if !ok {
		return newEntityNotFoundError("database", EntityOptions.DatabaseID)
	}

	bucketOpts := options.GridFSBucket()
	if EntityOptions.GridFSBucketOptions != nil {
		bucketOpts = EntityOptions.GridFSBucketOptions.BucketOptions
	}

	bucket, err := gridfs.NewBucket(db, bucketOpts)
	if err != nil {
		return fmt.Errorf("error creating GridFS bucket: %v", err)
	}

	em.gridfsBuckets[EntityOptions.ID] = bucket
	return nil
}

func (em *EntityMap) verifyEntityDoesNotExist(id string) error {
	if _, ok := em.allEntities[id]; ok {
		return fmt.Errorf("entity with ID %q already exists", id)
	}
	return nil
}

func newEntityNotFoundError(entityType, entityID string) error {
	return fmt.Errorf("no %s entity found with ID %q", entityType, entityID)
}
