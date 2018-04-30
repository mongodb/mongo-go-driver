// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

// NOTE: Any time this file is modified, a WEBSITE ticket should be opened to sync the changes with
// the "What is MongoDB" webpage, which the example was originally added to as part of WEBSITE-5148.

package documentation_examples_test

import (
	"context"
	"testing"

	"github.com/mongodb/mongo-go-driver/examples/documentation_examples"
	"github.com/mongodb/mongo-go-driver/mongo"
	"github.com/stretchr/testify/require"
)

func TestDocumentationExamples(t *testing.T) {
	client, err := mongo.Connect(context.Background(), "mongodb://localhost:27017", nil)
	require.NoError(t, err)

	db := client.Database("documentation_examples")

	documentation_examples.InsertExamples(t, db)
	documentation_examples.QueryToplevelFieldsExamples(t, db)
	documentation_examples.QueryEmbeddedDocumentsExamples(t, db)
	documentation_examples.QueryArraysExamples(t, db)
	documentation_examples.QueryArrayEmbeddedDocumentsExamples(t, db)
	documentation_examples.QueryNullMissingFieldsExamples(t, db)
	documentation_examples.ProjectionExamples(t, db)
	documentation_examples.UpdateExamples(t, db)
	documentation_examples.DeleteExamples(t, db)
}
