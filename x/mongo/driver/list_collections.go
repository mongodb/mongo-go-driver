// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package driver

import (
	"context"

	"errors"
	"github.com/mongodb/mongo-go-driver/mongo/options"
	"github.com/mongodb/mongo-go-driver/x/bsonx"
	"github.com/mongodb/mongo-go-driver/x/mongo/driver/session"
	"github.com/mongodb/mongo-go-driver/x/mongo/driver/topology"
	"github.com/mongodb/mongo-go-driver/x/mongo/driver/uuid"
	"github.com/mongodb/mongo-go-driver/x/network/command"
	"github.com/mongodb/mongo-go-driver/x/network/connection"
	"github.com/mongodb/mongo-go-driver/x/network/description"
	"github.com/mongodb/mongo-go-driver/x/network/wiremessage"
)

// ErrFilterType is thrown when a non-string filter is specified.
var ErrFilterType = errors.New("filter must be a string")

// ListCollections handles the full cycle dispatch and execution of a listCollections command against the provided
// topology.
func ListCollections(
	ctx context.Context,
	cmd command.ListCollections,
	topo *topology.Topology,
	selector description.ServerSelector,
	clientID uuid.UUID,
	pool *session.Pool,
	opts ...*options.ListCollectionsOptions,
) (command.Cursor, error) {

	ss, err := topo.SelectServer(ctx, selector)
	if err != nil {
		return nil, err
	}

	conn, err := ss.Connection(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	if ss.Description().WireVersion.Max < 3 {
		return legacyListCollections(ctx, cmd, ss, conn)
	}

	rp, err := getReadPrefBasedOnTransaction(cmd.ReadPref, cmd.Session)
	if err != nil {
		return nil, err
	}
	cmd.ReadPref = rp

	// If no explicit session and deployment supports sessions, start implicit session.
	if cmd.Session == nil && topo.SupportsSessions() {
		cmd.Session, err = session.NewClientSession(pool, clientID, session.Implicit)
		if err != nil {
			return nil, err
		}
	}

	lc := options.MergeListCollectionsOptions(opts...)
	if lc.NameOnly != nil {
		cmd.Opts = append(cmd.Opts, bsonx.Elem{"nameOnly", bsonx.Boolean(*lc.NameOnly)})
	}

	c, err := cmd.RoundTrip(ctx, ss.Description(), ss, conn)
	if err != nil {
		closeImplicitSession(cmd.Session)
	}

	return c, err
}

func legacyListCollections(
	ctx context.Context,
	cmd command.ListCollections,
	ss *topology.SelectedServer,
	conn connection.Connection,
) (command.Cursor, error) {
	query := wiremessage.Query{
		FullCollectionName: cmd.DB + ".system.namespaces",
	}

	// querying a secondary requires slaveOK to be set
	if ss.Server.Description().Kind == description.RSSecondary {
		query.Flags |= wiremessage.SlaveOK
	}

	queryDoc, err := createQueryDocument(cmd.Filter, cmd.DB)
	if err != nil {
		return nil, err
	}

	queryRaw, err := queryDoc.MarshalBSON()
	if err != nil {
		return nil, err
	}
	query.Query = queryRaw

	reply, err := roundTripQuery(ctx, query, conn)
	if err != nil {
		return nil, err
	}

	return ss.BuildListCollCursor(command.Namespace{}, 0, reply.Documents)
}

// modify the user-supplied filter to prefix the "name" field with the database name.
// returns the original filter if the name field is not present or a copy with the modified name field if it is
func transformFilter(filter bsonx.Doc, dbName string) (bsonx.Doc, error) {
	if filter == nil {
		return filter, nil
	}

	if nameVal, err := filter.LookupErr("name"); err == nil {
		name, ok := nameVal.StringValueOK()
		if !ok {
			return nil, ErrFilterType
		}

		filterCopy := filter.Copy()
		filterCopy.Set("name", bsonx.String(dbName+"."+name))
		return filterCopy, nil
	}
	return filter, nil
}

func createQueryDocument(filter bsonx.Doc, dbName string) (bsonx.Doc, error) {
	transformedFilter, err := transformFilter(filter, dbName)
	if err != nil {
		return nil, err
	}

	// filter out all collection names containing $
	regexDoc := bsonx.Doc{
		{"name", bsonx.Regex("^[^$]*$", "")},
	}
	var queryDoc bsonx.Doc
	if transformedFilter == nil || len(transformedFilter) == 0 {
		queryDoc = regexDoc
	} else {
		// create query document to use both the user-supplied filter and the regex filter
		arr := bsonx.Arr{
			bsonx.Document(regexDoc),
			bsonx.Document(transformedFilter),
		}

		queryDoc = bsonx.Doc{
			{"$and", bsonx.Array(arr)},
		}
	}

	return queryDoc, nil
}
