// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package unified

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"os"
	"sync"
	"sync/atomic"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/gridfs"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
)

// ErrEntityMapOpen is returned when a slice entity is accessed while the EntityMap is open
var ErrEntityMapOpen = errors.New("slices cannot be accessed while EntityMap is open")

var (
	tlsCAFile                   = os.Getenv("CSFLE_TLS_CA_FILE")
	tlsClientCertificateKeyFile = os.Getenv("CSFLE_TLS_CERTIFICATE_KEY_FILE")
)

type storeEventsAsEntitiesConfig struct {
	EventListID string   `bson:"id"`
	Events      []string `bson:"events"`
}

// entityOptions represents all options that can be used to configure an entity. Because there are multiple entity
// types, only a subset of the options that this type contains apply to any given entity.
type entityOptions struct {
	// Options that apply to all entity types.
	ID string `bson:"id"`

	// Options for client entities.
	URIOptions               bson.M                        `bson:"uriOptions"`
	UseMultipleMongoses      *bool                         `bson:"useMultipleMongoses"`
	ObserveEvents            []string                      `bson:"observeEvents"`
	IgnoredCommands          []string                      `bson:"ignoreCommandMonitoringEvents"`
	ObserveSensitiveCommands *bool                         `bson:"observeSensitiveCommands"`
	StoreEventsAsEntities    []storeEventsAsEntitiesConfig `bson:"storeEventsAsEntities"`
	ServerAPIOptions         *serverAPIOptions             `bson:"serverApi"`

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

	ClientEncryptionOpts *clientEncryptionOpts `bson:"clientEncryptionOpts"`
}
type clientEncryptionOpts struct {
	KeyVaultClient    string              `bson:"keyVaultClient"`
	KeyVaultNamespace string              `bson:"keyVaultNamespace"`
	KmsProviders      map[string]bson.Raw `bson:"kmsProviders"`
}

// EntityMap is used to store entities during tests. This type enforces uniqueness so no two entities can have the same
// ID, even if they are of different types. It also enforces referential integrity so construction of an entity that
// references another (e.g. a database entity references a client) will fail if the referenced entity does not exist.
// Accessors are available for the BSON entities.
type EntityMap struct {
	allEntities              map[string]struct{}
	cursorEntities           map[string]cursor
	clientEntities           map[string]*clientEntity
	dbEntites                map[string]*mongo.Database
	collEntities             map[string]*mongo.Collection
	sessions                 map[string]mongo.Session
	gridfsBuckets            map[string]*gridfs.Bucket
	bsonValues               map[string]bson.RawValue
	eventListEntities        map[string][]bson.Raw
	bsonArrayEntities        map[string][]bson.Raw // for storing errors and failures from a loop operation
	successValues            map[string]int32
	iterationValues          map[string]int32
	clientEncryptionEntities map[string]*mongo.ClientEncryption
	evtLock                  sync.Mutex
	closed                   atomic.Value
	// keyVaultClientIDs tracks IDs of clients used as a keyVaultClient in ClientEncryption objects.
	// ClientEncryption.Close() calls Disconnect on the keyVaultClient.
	// EntityMap.close() must skip calling Disconnect on any client entity referenced in keyVaultClientIDs.
	keyVaultClientIDs map[string]bool
}

func (em *EntityMap) isClosed() bool {
	return em.closed.Load().(bool)
}

func (em *EntityMap) setClosed(val bool) {
	em.closed.Store(val)
}

func newEntityMap() *EntityMap {
	em := &EntityMap{
		allEntities:              make(map[string]struct{}),
		gridfsBuckets:            make(map[string]*gridfs.Bucket),
		bsonValues:               make(map[string]bson.RawValue),
		cursorEntities:           make(map[string]cursor),
		clientEntities:           make(map[string]*clientEntity),
		collEntities:             make(map[string]*mongo.Collection),
		dbEntites:                make(map[string]*mongo.Database),
		sessions:                 make(map[string]mongo.Session),
		eventListEntities:        make(map[string][]bson.Raw),
		bsonArrayEntities:        make(map[string][]bson.Raw),
		successValues:            make(map[string]int32),
		iterationValues:          make(map[string]int32),
		clientEncryptionEntities: make(map[string]*mongo.ClientEncryption),
		keyVaultClientIDs:        make(map[string]bool),
	}
	em.setClosed(false)
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

func (em *EntityMap) addCursorEntity(id string, cursor cursor) error {
	if err := em.verifyEntityDoesNotExist(id); err != nil {
		return err
	}

	em.allEntities[id] = struct{}{}
	em.cursorEntities[id] = cursor
	return nil
}

func (em *EntityMap) addBSONArrayEntity(id string) error {
	// Error if a non-BSON array entity exists with the same name
	if _, ok := em.allEntities[id]; ok {
		if _, ok := em.bsonArrayEntities[id]; !ok {
			return fmt.Errorf("non-BSON array entity with ID %q already exists", id)
		}
		return nil
	}

	em.allEntities[id] = struct{}{}
	em.bsonArrayEntities[id] = []bson.Raw{}
	return nil
}

func (em *EntityMap) addSuccessesEntity(id string) error {
	if err := em.verifyEntityDoesNotExist(id); err != nil {
		return err
	}

	em.allEntities[id] = struct{}{}
	em.successValues[id] = 0
	return nil
}

func (em *EntityMap) addIterationsEntity(id string) error {
	if err := em.verifyEntityDoesNotExist(id); err != nil {
		return err
	}

	em.allEntities[id] = struct{}{}
	em.iterationValues[id] = 0
	return nil
}

func (em *EntityMap) addEventsEntity(id string) error {
	if err := em.verifyEntityDoesNotExist(id); err != nil {
		return err
	}
	em.allEntities[id] = struct{}{}
	em.eventListEntities[id] = []bson.Raw{}
	return nil
}

func (em *EntityMap) incrementSuccesses(id string) error {
	if _, ok := em.successValues[id]; !ok {
		return newEntityNotFoundError("successes", id)
	}
	em.successValues[id]++
	return nil
}

func (em *EntityMap) incrementIterations(id string) error {
	if _, ok := em.iterationValues[id]; !ok {
		return newEntityNotFoundError("iterations", id)
	}
	em.iterationValues[id]++
	return nil
}

func (em *EntityMap) appendEventsEntity(id string, doc bson.Raw) {
	em.evtLock.Lock()
	defer em.evtLock.Unlock()
	if _, ok := em.eventListEntities[id]; ok {
		em.eventListEntities[id] = append(em.eventListEntities[id], doc)
	}
}

func (em *EntityMap) appendBSONArrayEntity(id string, doc bson.Raw) error {
	if _, ok := em.bsonArrayEntities[id]; !ok {
		return newEntityNotFoundError("BSON array", id)
	}
	em.bsonArrayEntities[id] = append(em.bsonArrayEntities[id], doc)
	return nil
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
	case "clientEncryption":
		err = em.addClientEncryptionEntity(entityOptions)
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

func (em *EntityMap) cursor(id string) (cursor, error) {
	cursor, ok := em.cursorEntities[id]
	if !ok {
		return nil, newEntityNotFoundError("cursor", id)
	}
	return cursor, nil
}

func (em *EntityMap) client(id string) (*clientEntity, error) {
	client, ok := em.clientEntities[id]
	if !ok {
		return nil, newEntityNotFoundError("client", id)
	}
	return client, nil
}

func (em *EntityMap) clientEncryption(id string) (*mongo.ClientEncryption, error) {
	cee, ok := em.clientEncryptionEntities[id]
	if !ok {
		return nil, newEntityNotFoundError("client", id)
	}
	return cee, nil
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

// BSONValue returns the bson.RawValue associated with id
func (em *EntityMap) BSONValue(id string) (bson.RawValue, error) {
	val, ok := em.bsonValues[id]
	if !ok {
		return emptyRawValue, newEntityNotFoundError("BSON", id)
	}
	return val, nil
}

// EventList returns the array of event documents associated with id. This should only be accessed
// after the test is finished running
func (em *EntityMap) EventList(id string) ([]bson.Raw, error) {
	if !em.isClosed() {
		return nil, ErrEntityMapOpen
	}
	val, ok := em.eventListEntities[id]
	if !ok {
		return nil, newEntityNotFoundError("event list", id)
	}
	return val, nil
}

// BSONArray returns the BSON document array associated with id. This should only be accessed
// after the test is finished running
func (em *EntityMap) BSONArray(id string) ([]bson.Raw, error) {
	if !em.isClosed() {
		return nil, ErrEntityMapOpen
	}
	val, ok := em.bsonArrayEntities[id]
	if !ok {
		return nil, newEntityNotFoundError("BSON array", id)
	}
	return val, nil
}

// Successes returns the number of successes associated with id
func (em *EntityMap) Successes(id string) (int32, error) {
	val, ok := em.successValues[id]
	if !ok {
		return 0, newEntityNotFoundError("successes", id)
	}
	return val, nil
}

// Iterations returns the number of iterations associated with id
func (em *EntityMap) Iterations(id string) (int32, error) {
	val, ok := em.iterationValues[id]
	if !ok {
		return 0, newEntityNotFoundError("iterations", id)
	}
	return val, nil
}

// close disposes of the session and client entities associated with this map.
func (em *EntityMap) close(ctx context.Context) []error {
	for _, sess := range em.sessions {
		sess.EndSession(ctx)
	}

	var errs []error
	for id, cursor := range em.cursorEntities {
		if err := cursor.Close(ctx); err != nil {
			errs = append(errs, fmt.Errorf("error closing cursor with ID %q: %v", id, err))
		}
	}

	for id, client := range em.clientEntities {
		if ok := em.keyVaultClientIDs[id]; ok {
			// Client will be closed in clientEncryption.Close()
			continue
		}
		if err := client.Disconnect(ctx); err != nil {
			errs = append(errs, fmt.Errorf("error closing client with ID %q: %v", id, err))
		}
	}

	for id, clientEncryption := range em.clientEncryptionEntities {
		if err := clientEncryption.Close(ctx); err != nil {
			errs = append(errs, fmt.Errorf("error closing clientEncryption with ID: %q: %v", id, err))
		}
	}

	em.setClosed(true)
	return errs
}

func (em *EntityMap) addClientEntity(ctx context.Context, entityOptions *entityOptions) error {
	var client *clientEntity

	for _, eventsAsEntity := range entityOptions.StoreEventsAsEntities {
		if entityOptions.ID == eventsAsEntity.EventListID {
			return fmt.Errorf("entity with ID %q already exists", entityOptions.ID)
		}
		if err := em.addEventsEntity(eventsAsEntity.EventListID); err != nil {
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

// getKmsCredential processes a value of an input KMS provider credential.
// An empty document returns from the environment.
// A string is returned as-is.
func getKmsCredential(kmsDocument bson.Raw, credentialName string, envVar string, defaultValue string) (string, error) {
	credentialVal, err := kmsDocument.LookupErr(credentialName)
	if err == bsoncore.ErrElementNotFound {
		return "", nil
	}
	if err != nil {
		return "", err
	}

	if str, ok := credentialVal.StringValueOK(); ok {
		return str, nil
	}

	var ok bool
	var doc bson.Raw
	if doc, ok = credentialVal.DocumentOK(); !ok {
		return "", fmt.Errorf("expected String or Document for %v, got: %v", credentialName, credentialVal)
	}

	placeholderDoc := bsoncore.NewDocumentBuilder().AppendInt32("$$placeholder", 1).Build()

	// Check if document is a placeholder.
	if !bytes.Equal(doc, placeholderDoc) {
		return "", fmt.Errorf("unexpected non-empty document for %v: %v", credentialName, doc)
	}
	if envVar == "" {
		return defaultValue, nil
	}
	if os.Getenv(envVar) == "" {
		if defaultValue != "" {
			return defaultValue, nil
		}
		return "", fmt.Errorf("unable to get environment value for %v. Please set the CSFLE environment variable: %v", credentialName, envVar)
	}
	return os.Getenv(envVar), nil

}

func (em *EntityMap) addClientEncryptionEntity(entityOptions *entityOptions) error {
	// Construct KMS providers.
	kmsProviders := make(map[string]map[string]interface{})
	ceo := entityOptions.ClientEncryptionOpts
	tlsconf := make(map[string]*tls.Config)
	if aws, ok := ceo.KmsProviders["aws"]; ok {
		kmsProviders["aws"] = make(map[string]interface{})

		awsSessionToken, err := getKmsCredential(aws, "sessionToken", "CSFLE_AWS_TEMP_SESSION_TOKEN", "")
		if err != nil {
			return err
		}
		if awsSessionToken != "" {
			// Get temporary AWS credentials.
			kmsProviders["aws"]["sessionToken"] = awsSessionToken
			awsAccessKeyID, err := getKmsCredential(aws, "accessKeyId", "CSFLE_AWS_TEMP_ACCESS_KEY_ID", "")
			if err != nil {
				return err
			}
			if awsAccessKeyID != "" {
				kmsProviders["aws"]["accessKeyId"] = awsAccessKeyID
			}

			awsSecretAccessKey, err := getKmsCredential(aws, "secretAccessKey", "CSFLE_AWS_TEMP_SECRET_ACCESS_KEY", "")
			if err != nil {
				return err
			}
			if awsSecretAccessKey != "" {
				kmsProviders["aws"]["secretAccessKey"] = awsSecretAccessKey
			}
		} else {
			awsAccessKeyID, err := getKmsCredential(aws, "accessKeyId", "AWS_ACCESS_KEY_ID", "")
			if err != nil {
				return err
			}
			if awsAccessKeyID != "" {
				kmsProviders["aws"]["accessKeyId"] = awsAccessKeyID
			}

			awsSecretAccessKey, err := getKmsCredential(aws, "secretAccessKey", "AWS_SECRET_ACCESS_KEY", "")
			if err != nil {
				return err
			}
			if awsSecretAccessKey != "" {
				kmsProviders["aws"]["secretAccessKey"] = awsSecretAccessKey
			}
		}

	}

	if azure, ok := ceo.KmsProviders["azure"]; ok {
		kmsProviders["azure"] = make(map[string]interface{})

		azureTenantID, err := getKmsCredential(azure, "tenantId", "AZURE_TENANT_ID", "")
		if err != nil {
			return err
		}
		if azureTenantID != "" {
			kmsProviders["azure"]["tenantId"] = azureTenantID
		}

		azureClientID, err := getKmsCredential(azure, "clientId", "AZURE_CLIENT_ID", "")
		if err != nil {
			return err
		}
		if azureClientID != "" {
			kmsProviders["azure"]["clientId"] = azureClientID
		}

		azureClientSecret, err := getKmsCredential(azure, "clientSecret", "AZURE_CLIENT_SECRET", "")
		if err != nil {
			return err
		}
		if azureClientSecret != "" {
			kmsProviders["azure"]["clientSecret"] = azureClientSecret
		}
	}

	if gcp, ok := ceo.KmsProviders["gcp"]; ok {
		kmsProviders["gcp"] = make(map[string]interface{})

		gcpEmail, err := getKmsCredential(gcp, "email", "GCP_EMAIL", "")
		if err != nil {
			return err
		}
		if gcpEmail != "" {
			kmsProviders["gcp"]["email"] = gcpEmail
		}

		gcpPrivateKey, err := getKmsCredential(gcp, "privateKey", "GCP_PRIVATE_KEY", "")
		if err != nil {
			return err
		}
		if gcpPrivateKey != "" {
			kmsProviders["gcp"]["privateKey"] = gcpPrivateKey
		}
	}

	if kmip, ok := ceo.KmsProviders["kmip"]; ok {
		kmsProviders["kmip"] = make(map[string]interface{})

		kmipEndpoint, err := getKmsCredential(kmip, "endpoint", "", "localhost:5698")
		if err != nil {
			return err
		}

		if tlsClientCertificateKeyFile != "" && tlsCAFile != "" {
			cfg, err := options.BuildTLSConfig(map[string]interface{}{
				"tlsCertificateKeyFile": tlsClientCertificateKeyFile,
				"tlsCAFile":             tlsCAFile,
			})
			if err != nil {
				return fmt.Errorf("error constructing tls config: %v", err)
			}
			tlsconf["kmip"] = cfg
		}

		if kmipEndpoint != "" {
			kmsProviders["kmip"]["endpoint"] = kmipEndpoint
		}
	}

	if local, ok := ceo.KmsProviders["local"]; ok {
		kmsProviders["local"] = make(map[string]interface{})

		defaultLocalKeyBase64 := "Mng0NCt4ZHVUYUJCa1kxNkVyNUR1QURhZ2h2UzR2d2RrZzh0cFBwM3R6NmdWMDFBMUN3YkQ5aXRRMkhGRGdQV09wOGVNYUMxT2k3NjZKelhaQmRCZGJkTXVyZG9uSjFk"
		localKey, err := getKmsCredential(local, "key", "", defaultLocalKeyBase64)
		if err != nil {
			return err
		}
		if localKey != "" {
			kmsProviders["local"]["key"] = localKey
		}
	}

	em.keyVaultClientIDs[ceo.KeyVaultClient] = true
	keyVaultClient, ok := em.clientEntities[ceo.KeyVaultClient]
	if !ok {
		return newEntityNotFoundError("client", ceo.KeyVaultClient)
	}

	ce, err := mongo.NewClientEncryption(
		keyVaultClient.Client,
		options.ClientEncryption().
			SetKeyVaultNamespace(ceo.KeyVaultNamespace).
			SetTLSConfig(tlsconf).
			SetKmsProviders(kmsProviders))
	if err != nil {
		return err
	}

	em.clientEncryptionEntities[entityOptions.ID] = ce

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
