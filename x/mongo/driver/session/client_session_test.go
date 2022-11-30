// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package session

import (
	"bytes"
	"testing"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/internal/assert"
	"go.mongodb.org/mongo-driver/internal/require"
	"go.mongodb.org/mongo-driver/internal/uuid"
	"go.mongodb.org/mongo-driver/mongo/description"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
)

var consistent = true
var sessionOpts = &ClientOptions{
	CausalConsistency: &consistent,
}

func compareOperationTimes(t *testing.T, expected *primitive.Timestamp, actual *primitive.Timestamp) {
	if expected.T != actual.T {
		t.Fatalf("T value mismatch; expected %d got %d", expected.T, actual.T)
	}

	if expected.I != actual.I {
		t.Fatalf("I value mismatch; expected %d got %d", expected.I, actual.I)
	}
}

func TestClientSession(t *testing.T) {
	var clusterTime1 = bsoncore.BuildDocument(nil, bsoncore.AppendDocumentElement(nil, "$clusterTime", bsoncore.BuildDocument(nil, bsoncore.AppendTimestampElement(nil, "clusterTime", 10, 5))))
	var clusterTime2 = bsoncore.BuildDocument(nil, bsoncore.AppendDocumentElement(nil, "$clusterTime", bsoncore.BuildDocument(nil, bsoncore.AppendTimestampElement(nil, "clusterTime", 5, 5))))
	var clusterTime3 = bsoncore.BuildDocument(nil, bsoncore.AppendDocumentElement(nil, "$clusterTime", bsoncore.BuildDocument(nil, bsoncore.AppendTimestampElement(nil, "clusterTime", 5, 0))))

	t.Run("TestMaxClusterTime", func(t *testing.T) {
		maxTime := MaxClusterTime(clusterTime1, clusterTime2)
		if !bytes.Equal(maxTime, clusterTime1) {
			t.Errorf("Wrong max time")
		}

		maxTime = MaxClusterTime(clusterTime3, clusterTime2)
		if !bytes.Equal(maxTime, clusterTime2) {
			t.Errorf("Wrong max time")
		}
	})

	t.Run("TestAdvanceClusterTime", func(t *testing.T) {
		id, _ := uuid.New()
		sess, err := NewClientSession(&Pool{}, id, Explicit, sessionOpts)
		require.Nil(t, err, "Unexpected error")
		err = sess.AdvanceClusterTime(clusterTime2)
		require.Nil(t, err, "Unexpected error")
		if !bytes.Equal(sess.ClusterTime, clusterTime2) {
			t.Errorf("Session cluster time incorrect, expected %v, received %v", clusterTime2, sess.ClusterTime)
		}
		err = sess.AdvanceClusterTime(clusterTime3)
		require.Nil(t, err, "Unexpected error")
		if !bytes.Equal(sess.ClusterTime, clusterTime2) {
			t.Errorf("Session cluster time incorrect, expected %v, received %v", clusterTime2, sess.ClusterTime)
		}
		err = sess.AdvanceClusterTime(clusterTime1)
		require.Nil(t, err, "Unexpected error")
		if !bytes.Equal(sess.ClusterTime, clusterTime1) {
			t.Errorf("Session cluster time incorrect, expected %v, received %v", clusterTime1, sess.ClusterTime)
		}
		sess.EndSession()
	})

	t.Run("TestEndSession", func(t *testing.T) {
		id, _ := uuid.New()
		sess, err := NewClientSession(&Pool{}, id, Explicit, sessionOpts)
		require.Nil(t, err, "Unexpected error")
		sess.EndSession()
		err = sess.UpdateUseTime()
		require.NotNil(t, err, "Expected error, received nil")
	})

	t.Run("TestAdvanceOperationTime", func(t *testing.T) {
		id, _ := uuid.New()
		sess, err := NewClientSession(&Pool{}, id, Explicit, sessionOpts)
		require.Nil(t, err, "Unexpected error")

		optime1 := &primitive.Timestamp{
			T: 1,
			I: 0,
		}
		err = sess.AdvanceOperationTime(optime1)
		assert.Nil(t, err, "error updating first operation time: %s", err)
		compareOperationTimes(t, optime1, sess.OperationTime)

		optime2 := &primitive.Timestamp{
			T: 2,
			I: 0,
		}
		err = sess.AdvanceOperationTime(optime2)
		assert.Nil(t, err, "error updating second operation time: %s", err)
		compareOperationTimes(t, optime2, sess.OperationTime)

		optime3 := &primitive.Timestamp{
			T: 2,
			I: 1,
		}
		err = sess.AdvanceOperationTime(optime3)
		assert.Nil(t, err, "error updating third operation time: %s", err)
		compareOperationTimes(t, optime3, sess.OperationTime)

		err = sess.AdvanceOperationTime(&primitive.Timestamp{
			T: 1,
			I: 10,
		})
		assert.Nil(t, err, "error updating fourth operation time: %s", err)
		compareOperationTimes(t, optime3, sess.OperationTime)
		sess.EndSession()
	})

	t.Run("TestTransactionState", func(t *testing.T) {
		id, _ := uuid.New()
		sess, err := NewClientSession(&Pool{}, id, Explicit, nil)
		require.Nil(t, err, "Unexpected error")

		err = sess.CommitTransaction()
		if err != ErrNoTransactStarted {
			t.Errorf("expected error, got %v", err)
		}

		err = sess.AbortTransaction()
		if err != ErrNoTransactStarted {
			t.Errorf("expected error, got %v", err)
		}

		if sess.TransactionState != None {
			t.Errorf("incorrect session state, expected None, received %v", sess.TransactionState)
		}

		err = sess.StartTransaction(nil)
		require.Nil(t, err, "error starting transaction: %s", err)
		if sess.TransactionState != Starting {
			t.Errorf("incorrect session state, expected Starting, received %v", sess.TransactionState)
		}

		err = sess.StartTransaction(nil)
		if err != ErrTransactInProgress {
			t.Errorf("expected error, got %v", err)
		}

		err = sess.ApplyCommand(description.Server{Kind: description.Standalone})
		assert.Nil(t, err, "ApplyCommand error: %v", err)
		if sess.TransactionState != InProgress {
			t.Errorf("incorrect session state, expected InProgress, received %v", sess.TransactionState)
		}

		err = sess.StartTransaction(nil)
		if err != ErrTransactInProgress {
			t.Errorf("expected error, got %v", err)
		}

		err = sess.CommitTransaction()
		require.Nil(t, err, "error committing transaction: %s", err)
		if sess.TransactionState != Committed {
			t.Errorf("incorrect session state, expected Committed, received %v", sess.TransactionState)
		}

		err = sess.AbortTransaction()
		if err != ErrAbortAfterCommit {
			t.Errorf("expected error, got %v", err)
		}

		err = sess.StartTransaction(nil)
		require.Nil(t, err, "error starting transaction: %s", err)
		if sess.TransactionState != Starting {
			t.Errorf("incorrect session state, expected Starting, received %v", sess.TransactionState)
		}

		err = sess.AbortTransaction()
		require.Nil(t, err, "error aborting transaction: %s", err)
		if sess.TransactionState != Aborted {
			t.Errorf("incorrect session state, expected Aborted, received %v", sess.TransactionState)
		}

		err = sess.AbortTransaction()
		if err != ErrAbortTwice {
			t.Errorf("expected error, got %v", err)
		}

		err = sess.CommitTransaction()
		if err != ErrCommitAfterAbort {
			t.Errorf("expected error, got %v", err)
		}
	})

	t.Run("causal consistency and snapshot", func(t *testing.T) {
		falseVal := false
		trueVal := true

		// A test for Consistent and Snapshot both being true and causing an error can be found
		// in TestSessionsProse.
		testCases := []struct {
			description        string
			consistent         *bool
			snapshot           *bool
			expectedConsistent bool
			expectedSnapshot   bool
		}{
			{
				"both unset",
				nil,
				nil,
				true,
				false,
			},
			{
				"both false",
				&falseVal,
				&falseVal,
				false,
				false,
			},
			{
				"cc unset snapshot true",
				nil,
				&trueVal,
				false,
				true,
			},
			{
				"cc unset snapshot false",
				nil,
				&falseVal,
				true,
				false,
			},
			{
				"cc true snapshot unset",
				&trueVal,
				nil,
				true,
				false,
			},
			{
				"cc false snapshot unset",
				&falseVal,
				nil,
				false,
				false,
			},
			{
				"cc false snapshot true",
				&falseVal,
				&trueVal,
				false,
				true,
			},
			{
				"cc true snapshot false",
				&trueVal,
				&falseVal,
				true,
				false,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.description, func(t *testing.T) {
				sessOpts := &ClientOptions{
					CausalConsistency: tc.consistent,
					Snapshot:          tc.snapshot,
				}

				id, _ := uuid.New()
				sess, err := NewClientSession(&Pool{}, id, Explicit, sessOpts)
				require.Nil(t, err, "unexpected NewClientSession error %v", err)

				require.Equal(t, tc.expectedConsistent, sess.Consistent,
					"expected Consistent to be %v, got %v", tc.expectedConsistent, sess.Consistent)
				require.Equal(t, tc.expectedSnapshot, sess.Snapshot,
					"expected Snapshot to be %v, got %v", tc.expectedSnapshot, sess.Snapshot)
			})
		}
	})
}
