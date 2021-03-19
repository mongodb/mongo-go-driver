// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package unified

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/gridfs"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type storeEventsAsEntitiesOption struct {
	ID     string   `bson:"id"`
	Events []string `bson:"events"`
}

// entityOptions represents all options that can be used to configure an entity. Because there are multiple entity
// types, only a subset of the options that this type contains apply to any given entity.
type entityOptions struct {
	// Options that apply to all entity types.
	ID string `bson:"id"`

	// Options for client entities.
	URIOptions            bson.M                        `bson:"uriOptions"`
	UseMultipleMongoses   *bool                         `bson:"useMultipleMongoses"`
	ObserveEvents         []string                      `bson:"observeEvents"`
	IgnoredCommands       []string                      `bson:"ignoreCommandMonitoringEvents"`
	StoreEventsAsEntities []storeEventsAsEntitiesOption `bson:"storeEventsAsEntities"`
	ServerAPIOptions      *serverAPIOptions             `bson:"serverApi"`

	// Options for database entities.
	DatabaseName    string                 `bson:"databaseName"`
	DatabaseOptions *dbOrCollectionOptions `bson:"databaseOptions"`

	// Options for collection entities.
	CollectionName    string                 `bson:"collectionName"`
	CollectionOptions *dbOrCollectionOptions `bson:"collectionOptions"`

	// Options for session entities.
	SessionOptions *sessionOptions `bson:"sessionOptions"`

	// Options for GridFS bucket entities.
	GridFSBucketOptions *gridFSBucketOptions `bson:"bucketOptions"`

	// Options that reference other entities.
	ClientID   string `bson:"client"`
	DatabaseID string `bson:"database"`
}

// EntityMap is used to store entities during tests. This type enforces uniqueness so no two entities can have the same
// ID, even if they are of different types. It also enforces referential integrity so construction of an entity that
// references another (e.g. a database entity references a client) will fail if the referenced entity does not exist.
type EntityMap struct {
	allEntities     map[string]struct{}
	changeStreams   map[string]*mongo.ChangeStream
	clientEntities  map[string]*clientEntity
	dbEntites       map[string]*mongo.Database
	collEntities    map[string]*mongo.Collection
	sessions        map[string]mongo.Session
	gridfsBuckets   map[string]*gridfs.Bucket
	bsonValues      map[string]bson.RawValue
	eventEntities   map[string][]bson.Raw
	errorEntities   map[string][]bson.Raw
	failureEntities map[string][]bson.Raw
	successValues   map[string]*int32
	iterationValues map[string]*int32

	evtLock  sync.Mutex
	errLock  sync.Mutex
	failLock sync.Mutex
	closed   atomic.Value
}

func newEntityMap() *EntityMap {
	em := &EntityMap{
		allEntities:     make(map[string]struct{}),
		gridfsBuckets:   make(map[string]*gridfs.Bucket),
		bsonValues:      make(map[string]bson.RawValue),
		changeStreams:   make(map[string]*mongo.ChangeStream),
		clientEntities:  make(map[string]*clientEntity),
		collEntities:    make(map[string]*mongo.Collection),
		dbEntites:       make(map[string]*mongo.Database),
		sessions:        make(map[string]mongo.Session),
		eventEntities:   make(map[string][]bson.Raw),
		errorEntities:   make(map[string][]bson.Raw),
		failureEntities: make(map[string][]bson.Raw),
		successValues:   make(map[string]*int32),
		iterationValues: make(map[string]*int32),
	}
	em.closed.Store(false)
	return em
}

func (em *EntityMap) addBSONEntity(id string, val bson.RawValue) error {
	if err := em.verifyEntityDoesNotExist(id); err != nil {
		return err
	}

	em.allEntities[id] = struct{}{}
	em.bsonValues[id] = val
	return nil
}

func (em *EntityMap) addChangeStreamEntity(id string, stream *mongo.ChangeStream) error {
	if err := em.verifyEntityDoesNotExist(id); err != nil {
		return err
	}

	em.allEntities[id] = struct{}{}
	em.changeStreams[id] = stream
	return nil
}

func (em *EntityMap) addErrorsEntityIfDoesntExist(id string) error {
	// Error if a non-error entity exists with the same name
	if _, ok := em.allEntities[id]; ok {
		if _, ok := em.errorEntities[id]; !ok {
			return fmt.Errorf("non-errors entity with ID %q already exists", id)
		}
	}

	em.allEntities[id] = struct{}{}
	em.errorEntities[id] = []bson.Raw{}
	return nil
}

func (em *EntityMap) addFailuresEntityIfDoesntExist(id string) error {
	// Error if a non-error entity exists with the same name
	if _, ok := em.allEntities[id]; ok {
		if _, ok := em.failureEntities[id]; !ok {
			return fmt.Errorf("non-failures entity with ID %q already exists", id)
		}
	}

	em.allEntities[id] = struct{}{}
	em.failureEntities[id] = []bson.Raw{}
	return nil
}

func (em *EntityMap) addSuccessesEntity(id string) error {
	if err := em.verifyEntityDoesNotExist(id); err != nil {
		return err
	}

	em.allEntities[id] = struct{}{}
	em.successValues[id] = new(int32)
	return nil
}

func (em *EntityMap) addIterationsEntity(id string) error {
	if err := em.verifyEntityDoesNotExist(id); err != nil {
		return err
	}

	em.allEntities[id] = struct{}{}
	em.iterationValues[id] = new(int32)
	return nil
}

func (em *EntityMap) addEventsEntity(id string) error {
	if err := em.verifyEntityDoesNotExist(id); err != nil {
		return err
	}
	em.allEntities[id] = struct{}{}
	em.eventEntities[id] = []bson.Raw{}
	return nil
}

func (em *EntityMap) incrementSuccesses(id string) {
	if _, ok := em.successValues[id]; ok {
		atomic.AddInt32(em.successValues[id], 1)
	}
}

func (em *EntityMap) incrementIterations(id string) {
	if _, ok := em.iterationValues[id]; ok {
		atomic.AddInt32(em.iterationValues[id], 1)
	}
}

func (em *EntityMap) appendEventsEntity(id string, doc bson.Raw) {
	em.evtLock.Lock()
	defer em.evtLock.Unlock()
	if _, ok := em.eventEntities[id]; ok {
		em.eventEntities[id] = append(em.eventEntities[id], doc)
	}
}

func (em *EntityMap) appendErrorsEntity(id string, doc bson.Raw) {
	em.errLock.Lock()
	defer em.errLock.Unlock()
	if _, ok := em.errorEntities[id]; ok {
		em.errorEntities[id] = append(em.errorEntities[id], doc)
	}
}

func (em *EntityMap) appendFailuresEntity(id string, doc bson.Raw) {
	em.failLock.Lock()
	defer em.failLock.Unlock()
	if _, ok := em.failureEntities[id]; ok {
		em.failureEntities[id] = append(em.failureEntities[id], doc)
	}
}

func (em *EntityMap) addEntity(ctx context.Context, entityType string, entityOptions *entityOptions) error {
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

func (em *EntityMap) gridFSBucket(id string) (*gridfs.Bucket, error) {
	bucket, ok := em.gridfsBuckets[id]
	if !ok {
		return nil, newEntityNotFoundError("gridfs bucket", id)
	}
	return bucket, nil
}

func (em *EntityMap) bsonValue(id string) (bson.RawValue, error) {
	val, ok := em.bsonValues[id]
	if !ok {
		return emptyRawValue, newEntityNotFoundError("BSON", id)
	}
	return val, nil
}

func (em *EntityMap) changeStream(id string) (*mongo.ChangeStream, error) {
	client, ok := em.changeStreams[id]
	if !ok {
		return nil, newEntityNotFoundError("change stream", id)
	}
	return client, nil
}

func (em *EntityMap) client(id string) (*clientEntity, error) {
	client, ok := em.clientEntities[id]
	if !ok {
		return nil, newEntityNotFoundError("client", id)
	}
	return client, nil
}

func (em *EntityMap) clients() map[string]*clientEntity {
	return em.clientEntities
}

func (em *EntityMap) collections() map[string]*mongo.Collection {
	return em.collEntities
}

func (em *EntityMap) collection(id string) (*mongo.Collection, error) {
	coll, ok := em.collEntities[id]
	if !ok {
		return nil, newEntityNotFoundError("collection", id)
	}
	return coll, nil
}

func (em *EntityMap) database(id string) (*mongo.Database, error) {
	db, ok := em.dbEntites[id]
	if !ok {
		return nil, newEntityNotFoundError("database", id)
	}
	return db, nil
}

func (em *EntityMap) session(id string) (mongo.Session, error) {
	sess, ok := em.sessions[id]
	if !ok {
		return nil, newEntityNotFoundError("session", id)
	}
	return sess, nil
}

// GetEvents returns the array of event documents associated with id. This should only be accessed
// after the test is finished running
func (em *EntityMap) GetEvents(id string) ([]bson.Raw, bool) {
	if !em.closed.Load().(bool) {
		return nil, false
	}
	val, ok := em.eventEntities[id]
	return val, ok
}

// GetErrors returns the array of error documents associated with id. This should only be accessed
// after the test is finished running
func (em *EntityMap) GetErrors(id string) ([]bson.Raw, bool) {
	if !em.closed.Load().(bool) {
		return nil, false
	}
	val, ok := em.errorEntities[id]
	return val, ok
}

// GetFailures returns the array of failure documents associated with id. This should only be accessed
// after the test is finished running
func (em *EntityMap) GetFailures(id string) ([]bson.Raw, bool) {
	if !em.closed.Load().(bool) {
		return nil, false
	}
	val, ok := em.failureEntities[id]
	return val, ok
}

// GetBSON returns the bson.RawValue associated with id
func (em *EntityMap) GetBSON(id string) (bson.RawValue, bool) {
	val, ok := em.bsonValues[id]
	return val, ok
}

// GetSuccesses returns the array of event documents associated with id
func (em *EntityMap) GetSuccesses(id string) (int32, bool) {
	val, ok := em.successValues[id]
	if !ok {
		return 0, ok
	}
	return atomic.LoadInt32(val), ok
}

// GetIterations returns the array of event documents associated with id
func (em *EntityMap) GetIterations(id string) (int32, bool) {
	val, ok := em.iterationValues[id]
	if !ok {
		return 0, ok
	}
	return atomic.LoadInt32(val), ok
}

// close disposes of the session and client entities associated with this map.
func (em *EntityMap) close(ctx context.Context) []error {
	for _, sess := range em.sessions {
		sess.EndSession(ctx)
	}

	var errs []error
	for id, client := range em.clientEntities {
		if err := client.Disconnect(ctx); err != nil {
			errs = append(errs, fmt.Errorf("error closing client with ID %q: %v", id, err))
		}
	}
	em.closed.Store(true)
	return errs
}

func (em *EntityMap) addClientEntity(ctx context.Context, entityOptions *entityOptions) error {
	var client *clientEntity

	for _, eventsAsEntity := range entityOptions.StoreEventsAsEntities {
		if err := em.addEventsEntity(eventsAsEntity.ID); err != nil {
			return err
		}
	}

	client, err := newClientEntity(ctx, em, entityOptions)
	if err != nil {
		return fmt.Errorf("error creating client entity: %v", err)
	}

	em.clientEntities[entityOptions.ID] = client
	return nil
}

func (em *EntityMap) addDatabaseEntity(entityOptions *entityOptions) error {
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

func (em *EntityMap) addCollectionEntity(entityOptions *entityOptions) error {
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

func (em *EntityMap) addSessionEntity(entityOptions *entityOptions) error {
	client, ok := em.clientEntities[entityOptions.ClientID]
	if !ok {
		return newEntityNotFoundError("client", entityOptions.ClientID)
	}

	sessionOpts := options.Session()
	if entityOptions.SessionOptions != nil {
		sessionOpts = entityOptions.SessionOptions.SessionOptions
	}

	sess, err := client.StartSession(sessionOpts)
	if err != nil {
		return fmt.Errorf("error starting session: %v", err)
	}

	em.sessions[entityOptions.ID] = sess
	return nil
}

func (em *EntityMap) addGridFSBucketEntity(entityOptions *entityOptions) error {
	db, ok := em.dbEntites[entityOptions.DatabaseID]
	if !ok {
		return newEntityNotFoundError("database", entityOptions.DatabaseID)
	}

	bucketOpts := options.GridFSBucket()
	if entityOptions.GridFSBucketOptions != nil {
		bucketOpts = entityOptions.GridFSBucketOptions.BucketOptions
	}

	bucket, err := gridfs.NewBucket(db, bucketOpts)
	if err != nil {
		return fmt.Errorf("error creating GridFS bucket: %v", err)
	}

	em.gridfsBuckets[entityOptions.ID] = bucket
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
