// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import (
	"context"
	"encoding/json"
	"math"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/internal/testutil"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
	"go.mongodb.org/mongo-driver/x/bsonx"
	"go.mongodb.org/mongo-driver/x/mongo/driver/connstring"
	"go.mongodb.org/mongo-driver/x/mongo/driver/session"
	"go.mongodb.org/mongo-driver/x/mongo/driver/uuid"
)

// temporary file to hold test helpers as we are porting tests to the new integration testing framework.

var wcMajority = writeconcern.New(writeconcern.WMajority())

var doc1 = bsonx.Doc{
	{"x", bsonx.Int32(1)},
}

func createTestClient(t *testing.T) *Client {
	id, _ := uuid.New()
	return &Client{
		id:             id,
		deployment:     testutil.Topology(t),
		connString:     testutil.ConnString(t),
		readPreference: readpref.Primary(),
		clock:          &session.ClusterClock{},
		registry:       bson.DefaultRegistry,
		retryWrites:    true,
		sessionPool:    testutil.SessionPool(),
	}
}

func createTestClientWithConnstring(t *testing.T, cs connstring.ConnString) *Client {
	id, _ := uuid.New()
	return &Client{
		id:             id,
		deployment:     testutil.TopologyWithConnString(t, cs),
		connString:     cs,
		readPreference: readpref.Primary(),
		clock:          &session.ClusterClock{},
		registry:       bson.DefaultRegistry,
	}
}

func createTestDatabase(t *testing.T, name *string, opts ...*options.DatabaseOptions) *Database {
	if name == nil {
		db := testutil.DBName(t)
		name = &db
	}

	client := createTestClient(t)

	dbOpts := []*options.DatabaseOptions{options.Database().SetWriteConcern(writeconcern.New(writeconcern.WMajority()))}
	dbOpts = append(dbOpts, opts...)
	return client.Database(*name, dbOpts...)
}

func createTestCollection(t *testing.T, dbName *string, collName *string, opts ...*options.CollectionOptions) *Collection {
	if collName == nil {
		coll := testutil.ColName(t)
		collName = &coll
	}

	db := createTestDatabase(t, dbName)
	db.RunCommand(
		context.Background(),
		bsonx.Doc{{"create", bsonx.String(*collName)}},
	)

	collOpts := []*options.CollectionOptions{options.Collection().SetWriteConcern(writeconcern.New(writeconcern.WMajority()))}
	collOpts = append(collOpts, opts...)
	return db.Collection(*collName, collOpts...)
}

func skipIfBelow34(t *testing.T, db *Database) {
	versionStr, err := getServerVersion(db)
	if err != nil {
		t.Fatalf("error getting server version: %s", err)
	}
	if compareVersions(t, versionStr, "3.4") < 0 {
		t.Skip("skipping collation test for server version < 3.4")
	}
}

func initCollection(t *testing.T, coll *Collection) {
	docs := []interface{}{}
	var i int32
	for i = 1; i <= 5; i++ {
		docs = append(docs, bsonx.Doc{{"x", bsonx.Int32(i)}})
	}

	_, err := coll.InsertMany(ctx, docs)
	require.Nil(t, err)
}

func skipIfBelow36(t *testing.T) {
	serverVersion, err := getServerVersion(createTestDatabase(t, nil))
	require.NoError(t, err, "unable to get server version of database")

	if compareVersions(t, serverVersion, "3.6") < 0 {
		t.Skip()
	}
}

func skipIfBelow32(t *testing.T) {
	serverVersion, err := getServerVersion(createTestDatabase(t, nil))
	require.NoError(t, err)

	if compareVersions(t, serverVersion, "3.2") < 0 {
		t.Skip()
	}
}

func noerr(t *testing.T, err error) {
	if err != nil {
		t.Helper()
		t.Errorf("Unexpected error: (%T)%v", err, err)
		t.FailNow()
	}
}

type runOn struct {
	MinServerVersion string   `json:"minServerVersion" bson:"minServerVersion"`
	MaxServerVersion string   `json:"maxServerVersion" bson:"maxServerVersion"`
	Topology         []string `json:"topology" bson:"topology"`
}

type failPoint struct {
	ConfigureFailPoint string          `json:"configureFailPoint"`
	Mode               json.RawMessage `json:"mode"`
	Data               *failPointData  `json:"data"`
}

type failPointData struct {
	FailCommands                  []string `json:"failCommands"`
	CloseConnection               bool     `json:"closeConnection"`
	ErrorCode                     int32    `json:"errorCode"`
	FailBeforeCommitExceptionCode int32    `json:"failBeforeCommitExceptionCode"`
	WriteConcernError             *struct {
		Code   int32  `json:"code"`
		Name   string `json:"codeName"`
		Errmsg string `json:"errmsg"`
	} `json:"writeConcernError"`
}

type expectation struct {
	CommandStartedEvent struct {
		CommandName  string          `json:"command_name"`
		DatabaseName string          `json:"database_name"`
		Command      json.RawMessage `json:"command"`
	} `json:"command_started_event"`
}

func readPrefFromString(s string) *readpref.ReadPref {
	switch strings.ToLower(s) {
	case "primary":
		return readpref.Primary()
	case "primarypreferred":
		return readpref.PrimaryPreferred()
	case "secondary":
		return readpref.Secondary()
	case "secondarypreferred":
		return readpref.SecondaryPreferred()
	case "nearest":
		return readpref.Nearest()
	}
	return readpref.Primary()
}

// compareVersions compares two version number strings (i.e. positive integers separated by
// periods). Comparisons are done to the lesser precision of the two versions. For example, 3.2 is
// considered equal to 3.2.11, whereas 3.2.0 is considered less than 3.2.11.
//
// Returns a positive int if version1 is greater than version2, a negative int if version1 is less
// than version2, and 0 if version1 is equal to version2.
func compareVersions(t *testing.T, v1 string, v2 string) int {
	n1 := strings.Split(v1, ".")
	n2 := strings.Split(v2, ".")

	for i := 0; i < int(math.Min(float64(len(n1)), float64(len(n2)))); i++ {
		i1, err := strconv.Atoi(n1[i])
		if err != nil {
			return 1
		}

		i2, err := strconv.Atoi(n2[i])
		if err != nil {
			return -1
		}

		difference := i1 - i2
		if difference != 0 {
			return difference
		}
	}

	return 0
}

func getServerVersion(db *Database) (string, error) {
	var serverStatus bsonx.Doc
	err := db.RunCommand(
		context.Background(),
		bsonx.Doc{{"serverStatus", bsonx.Int32(1)}},
	).Decode(&serverStatus)
	if err != nil {
		return "", err
	}

	version, err := serverStatus.LookupErr("version")
	if err != nil {
		return "", err
	}

	return version.StringValue(), nil
}

func getReadConcern(opt interface{}) *readconcern.ReadConcern {
	return readconcern.New(readconcern.Level(opt.(map[string]interface{})["level"].(string)))
}

func getWriteConcern(opt interface{}) *writeconcern.WriteConcern {
	if w, ok := opt.(map[string]interface{}); ok {
		var newTimeout time.Duration
		if conv, ok := w["wtimeout"].(float64); ok {
			newTimeout = time.Duration(int(conv)) * time.Millisecond
		}
		var newJ bool
		if conv, ok := w["j"].(bool); ok {
			newJ = conv
		}
		if conv, ok := w["w"].(string); ok && conv == "majority" {
			return writeconcern.New(writeconcern.WMajority(), writeconcern.J(newJ), writeconcern.WTimeout(newTimeout))
		} else if conv, ok := w["w"].(float64); ok {
			return writeconcern.New(writeconcern.W(int(conv)), writeconcern.J(newJ), writeconcern.WTimeout(newTimeout))
		}
	}
	return nil
}

func createFailPointDoc(t *testing.T, failPoint *failPoint) bsonx.Doc {
	failDoc := bsonx.Doc{{"configureFailPoint", bsonx.String(failPoint.ConfigureFailPoint)}}

	modeBytes, err := failPoint.Mode.MarshalJSON()
	require.NoError(t, err)

	var modeStruct struct {
		Times int32 `json:"times"`
		Skip  int32 `json:"skip"`
	}
	err = json.Unmarshal(modeBytes, &modeStruct)
	if err != nil {
		failDoc = append(failDoc, bsonx.Elem{"mode", bsonx.String("alwaysOn")})
	} else {
		modeDoc := bsonx.Doc{}
		if modeStruct.Times != 0 {
			modeDoc = append(modeDoc, bsonx.Elem{"times", bsonx.Int32(modeStruct.Times)})
		}
		if modeStruct.Skip != 0 {
			modeDoc = append(modeDoc, bsonx.Elem{"skip", bsonx.Int32(modeStruct.Skip)})
		}
		failDoc = append(failDoc, bsonx.Elem{"mode", bsonx.Document(modeDoc)})
	}

	if failPoint.Data != nil {
		dataDoc := bsonx.Doc{}

		if failPoint.Data.FailCommands != nil {
			failCommandElems := make(bsonx.Arr, len(failPoint.Data.FailCommands))
			for i, str := range failPoint.Data.FailCommands {
				failCommandElems[i] = bsonx.String(str)
			}
			dataDoc = append(dataDoc, bsonx.Elem{"failCommands", bsonx.Array(failCommandElems)})
		}

		if failPoint.Data.CloseConnection {
			dataDoc = append(dataDoc, bsonx.Elem{"closeConnection", bsonx.Boolean(failPoint.Data.CloseConnection)})
		}

		if failPoint.Data.ErrorCode != 0 {
			dataDoc = append(dataDoc, bsonx.Elem{"errorCode", bsonx.Int32(failPoint.Data.ErrorCode)})
		}

		if failPoint.Data.WriteConcernError != nil {
			dataDoc = append(dataDoc,
				bsonx.Elem{"writeConcernError", bsonx.Document(bsonx.Doc{
					{"code", bsonx.Int32(failPoint.Data.WriteConcernError.Code)},
					{"codeName", bsonx.String(failPoint.Data.WriteConcernError.Name)},
					{"errmsg", bsonx.String(failPoint.Data.WriteConcernError.Errmsg)},
				})},
			)
		}

		if failPoint.Data.FailBeforeCommitExceptionCode != 0 {
			dataDoc = append(dataDoc, bsonx.Elem{"failBeforeCommitExceptionCode", bsonx.Int32(failPoint.Data.FailBeforeCommitExceptionCode)})
		}

		failDoc = append(failDoc, bsonx.Elem{"data", bsonx.Document(dataDoc)})
	}

	return failDoc
}
