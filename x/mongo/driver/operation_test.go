// Copyright (C) MongoDB, Inc. 2022-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package driver

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/internal"
	"go.mongodb.org/mongo-driver/internal/assert"
	"go.mongodb.org/mongo-driver/internal/uuid"
	"go.mongodb.org/mongo-driver/mongo/address"
	"go.mongodb.org/mongo-driver/mongo/description"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
	"go.mongodb.org/mongo-driver/tag"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/x/mongo/driver/session"
	"go.mongodb.org/mongo-driver/x/mongo/driver/wiremessage"
)

func noerr(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		t.FailNow()
	}
}

func compareErrors(err1, err2 error) bool {
	if err1 == nil && err2 == nil {
		return true
	}

	if err1 == nil || err2 == nil {
		return false
	}

	if err1.Error() != err2.Error() {
		return false
	}

	return true
}

func TestOperation(t *testing.T) {
	t.Run("selectServer", func(t *testing.T) {
		t.Run("returns validation error", func(t *testing.T) {
			op := &Operation{}
			_, err := op.selectServer(context.Background())
			if err == nil {
				t.Error("Expected a validation error from selectServer, but got <nil>")
			}
		})
		t.Run("uses specified server selector", func(t *testing.T) {
			want := new(mockServerSelector)
			d := new(mockDeployment)
			op := &Operation{
				CommandFn:  func([]byte, description.SelectedServer) ([]byte, error) { return nil, nil },
				Deployment: d,
				Database:   "testing",
				Selector:   want,
			}
			_, err := op.selectServer(context.Background())
			noerr(t, err)
			got := d.params.selector
			if !cmp.Equal(got, want) {
				t.Errorf("Did not get expected server selector. got %v; want %v", got, want)
			}
		})
		t.Run("uses a default server selector", func(t *testing.T) {
			d := new(mockDeployment)
			op := &Operation{
				CommandFn:  func([]byte, description.SelectedServer) ([]byte, error) { return nil, nil },
				Deployment: d,
				Database:   "testing",
			}
			_, err := op.selectServer(context.Background())
			noerr(t, err)
			if d.params.selector == nil {
				t.Error("The selectServer method should use a default selector when not specified on Operation, but it passed <nil>.")
			}
		})
	})
	t.Run("Validate", func(t *testing.T) {
		cmdFn := func([]byte, description.SelectedServer) ([]byte, error) { return nil, nil }
		d := new(mockDeployment)
		testCases := []struct {
			name string
			op   *Operation
			err  error
		}{
			{"CommandFn", &Operation{}, InvalidOperationError{MissingField: "CommandFn"}},
			{"Deployment", &Operation{CommandFn: cmdFn}, InvalidOperationError{MissingField: "Deployment"}},
			{"Database", &Operation{CommandFn: cmdFn, Deployment: d}, InvalidOperationError{MissingField: "Database"}},
			{"<nil>", &Operation{CommandFn: cmdFn, Deployment: d, Database: "test"}, nil},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				if tc.op == nil {
					t.Fatal("op cannot be <nil>")
				}
				want := tc.err
				got := tc.op.Validate()
				if !cmp.Equal(got, want, cmp.Comparer(compareErrors)) {
					t.Errorf("Did not validate properly. got %v; want %v", got, want)
				}
			})
		}
	})
	t.Run("retryableWrite", func(t *testing.T) {
		sessPool := session.NewPool(nil)
		id, err := uuid.New()
		noerr(t, err)

		sess, err := session.NewClientSession(sessPool, id, session.Explicit)
		noerr(t, err)

		sessStartingTransaction, err := session.NewClientSession(sessPool, id, session.Explicit)
		noerr(t, err)
		err = sessStartingTransaction.StartTransaction(nil)
		noerr(t, err)

		sessInProgressTransaction, err := session.NewClientSession(sessPool, id, session.Explicit)
		noerr(t, err)
		err = sessInProgressTransaction.StartTransaction(nil)
		noerr(t, err)
		err = sessInProgressTransaction.ApplyCommand(description.Server{})
		noerr(t, err)

		wcAck := writeconcern.New(writeconcern.WMajority())
		wcUnack := writeconcern.New(writeconcern.W(0))

		descRetryable := description.Server{WireVersion: &description.VersionRange{Min: 0, Max: 7}, SessionTimeoutMinutes: 1}
		descNotRetryableWireVersion := description.Server{WireVersion: &description.VersionRange{Min: 0, Max: 5}, SessionTimeoutMinutes: 1}
		descNotRetryableStandalone := description.Server{WireVersion: &description.VersionRange{Min: 0, Max: 7}, SessionTimeoutMinutes: 1, Kind: description.Standalone}

		testCases := []struct {
			name string
			op   Operation
			desc description.Server
			want Type
		}{
			{"deployment doesn't support", Operation{}, description.Server{}, Type(0)},
			{"wire version too low", Operation{Client: sess, WriteConcern: wcAck}, descNotRetryableWireVersion, Type(0)},
			{"standalone not supported", Operation{Client: sess, WriteConcern: wcAck}, descNotRetryableStandalone, Type(0)},
			{
				"transaction in progress",
				Operation{Client: sessInProgressTransaction, WriteConcern: wcAck},
				descRetryable, Type(0),
			},
			{
				"transaction starting",
				Operation{Client: sessStartingTransaction, WriteConcern: wcAck},
				descRetryable, Type(0),
			},
			{"unacknowledged write concern", Operation{Client: sess, WriteConcern: wcUnack}, descRetryable, Type(0)},
			{
				"acknowledged write concern",
				Operation{Client: sess, WriteConcern: wcAck, Type: Write},
				descRetryable, Write,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				got := tc.op.retryable(tc.desc)
				if got != (tc.want != Type(0)) {
					t.Errorf("Did not receive expected Type. got %v; want %v", got, tc.want)
				}
			})
		}
	})
	t.Run("roundTrip", func(t *testing.T) {
		testCases := []struct {
			name    string
			conn    *mockConnection
			paramWM []byte // parameter wire message
			wantWM  []byte // wire message that should be returned
			wantErr error  // error that should be returned
		}{
			{
				"returns write error",
				&mockConnection{rWriteErr: errors.New("write error")},
				nil, nil,
				Error{Message: "write error", Labels: []string{TransientTransactionError, NetworkError}},
			},
			{
				"returns read error",
				&mockConnection{rReadErr: errors.New("read error")},
				nil, nil,
				Error{Message: "read error", Labels: []string{TransientTransactionError, NetworkError}},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				gotWM, _, gotErr := Operation{}.roundTrip(context.Background(), tc.conn, tc.paramWM)
				if !bytes.Equal(gotWM, tc.wantWM) {
					t.Errorf("Returned wire messages are not equal. got %v; want %v", gotWM, tc.wantWM)
				}
				if !cmp.Equal(gotErr, tc.wantErr, cmp.Comparer(compareErrors)) {
					t.Errorf("Returned error is not equal to expected error. got %v; want %v", gotErr, tc.wantErr)
				}
			})
		}
	})
	t.Run("addReadConcern", func(t *testing.T) {
		majorityRc := bsoncore.AppendDocumentElement(nil, "readConcern", bsoncore.BuildDocument(nil,
			bsoncore.AppendStringElement(nil, "level", "majority"),
		))

		testCases := []struct {
			name string
			rc   *readconcern.ReadConcern
			want bsoncore.Document
		}{
			{"nil", nil, nil},
			{"empty", readconcern.New(), nil},
			{"non-empty", readconcern.Majority(), majorityRc},
		}

		for _, tc := range testCases {
			got, err := Operation{ReadConcern: tc.rc}.addReadConcern(nil, description.SelectedServer{})
			noerr(t, err)
			if !bytes.Equal(got, tc.want) {
				t.Errorf("ReadConcern elements do not match. got %v; want %v", got, tc.want)
			}
		}
	})
	t.Run("addWriteConcern", func(t *testing.T) {
		want := bsoncore.AppendDocumentElement(nil, "writeConcern", bsoncore.BuildDocumentFromElements(
			nil, bsoncore.AppendStringElement(nil, "w", "majority"),
		))
		got, err := Operation{WriteConcern: writeconcern.New(writeconcern.WMajority())}.addWriteConcern(nil, description.SelectedServer{})
		noerr(t, err)
		if !bytes.Equal(got, want) {
			t.Errorf("WriteConcern elements do not match. got %v; want %v", got, want)
		}
	})
	t.Run("addSession", func(t *testing.T) { t.Skip("These tests should be covered by spec tests.") })
	t.Run("addClusterTime", func(t *testing.T) {
		t.Run("adds max cluster time", func(t *testing.T) {
			want := bsoncore.AppendDocumentElement(nil, "$clusterTime", bsoncore.BuildDocumentFromElements(nil,
				bsoncore.AppendTimestampElement(nil, "clusterTime", 1234, 5678),
			))
			newer := bsoncore.BuildDocumentFromElements(nil, want)
			older := bsoncore.BuildDocumentFromElements(nil,
				bsoncore.AppendDocumentElement(nil, "$clusterTime", bsoncore.BuildDocumentFromElements(nil,
					bsoncore.AppendTimestampElement(nil, "clusterTime", 1234, 5670),
				)),
			)

			clusterClock := new(session.ClusterClock)
			clusterClock.AdvanceClusterTime(newer)
			sessPool := session.NewPool(nil)
			id, err := uuid.New()
			noerr(t, err)

			sess, err := session.NewClientSession(sessPool, id, session.Explicit)
			noerr(t, err)
			err = sess.AdvanceClusterTime(older)
			noerr(t, err)

			got := Operation{Client: sess, Clock: clusterClock}.addClusterTime(nil, description.SelectedServer{
				Server: description.Server{WireVersion: &description.VersionRange{Min: 0, Max: 7}},
			})
			if !bytes.Equal(got, want) {
				t.Errorf("ClusterTimes do not match. got %v; want %v", got, want)
			}
		})
	})
	t.Run("calculateMaxTimeMS", func(t *testing.T) {
		timeout := 5 * time.Second
		maxTime := 2 * time.Second
		negMaxTime := -2 * time.Second
		shortRTT := 50 * time.Millisecond
		longRTT := 10 * time.Second
		timeoutCtx, cancel := internal.MakeTimeoutContext(context.Background(), timeout)
		defer cancel()

		testCases := []struct {
			name  string
			op    Operation
			ctx   context.Context
			rtt90 time.Duration
			want  uint64
			err   error
		}{
			{
				name:  "uses context deadline and rtt90 with timeout",
				op:    Operation{MaxTime: &maxTime},
				ctx:   timeoutCtx,
				rtt90: shortRTT,
				want:  5000,
				err:   nil,
			},
			{
				name:  "uses MaxTime without timeout",
				op:    Operation{MaxTime: &maxTime},
				ctx:   context.Background(),
				rtt90: longRTT,
				want:  2000,
				err:   nil,
			},
			{
				name:  "errors when remaining timeout is less than rtt90",
				op:    Operation{MaxTime: &maxTime},
				ctx:   timeoutCtx,
				rtt90: timeout,
				want:  0,
				err:   ErrDeadlineWouldBeExceeded,
			},
			{
				name:  "errors when MaxTime is negative",
				op:    Operation{MaxTime: &negMaxTime},
				ctx:   context.Background(),
				rtt90: longRTT,
				want:  0,
				err:   ErrNegativeMaxTime,
			},
		}
		for _, tc := range testCases {
			// Capture test-case for parallel sub-test.
			tc := tc
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				got, err := tc.op.calculateMaxTimeMS(tc.ctx, tc.rtt90, "")

				// Assert that the calculated maxTimeMS is less than or equal to the expected value. A few
				// milliseconds will have elapsed toward the context deadline, and (remainingTimeout
				// - rtt90) will be slightly smaller than the expected value.
				if got > tc.want {
					t.Errorf("maxTimeMS value higher than expected. got %v; wanted at most %v", got, tc.want)
				}
				if !errors.Is(err, tc.err) {
					t.Errorf("error values do not match. got %v; want %v", err, tc.err)
				}
			})
		}
	})
	t.Run("updateClusterTimes", func(t *testing.T) {
		clustertime := bsoncore.BuildDocumentFromElements(nil,
			bsoncore.AppendDocumentElement(nil, "$clusterTime", bsoncore.BuildDocumentFromElements(nil,
				bsoncore.AppendTimestampElement(nil, "clusterTime", 1234, 5678),
			)),
		)

		clusterClock := new(session.ClusterClock)
		sessPool := session.NewPool(nil)
		id, err := uuid.New()
		noerr(t, err)

		sess, err := session.NewClientSession(sessPool, id, session.Explicit)
		noerr(t, err)
		Operation{Client: sess, Clock: clusterClock}.updateClusterTimes(clustertime)

		got := sess.ClusterTime
		if !bytes.Equal(got, clustertime) {
			t.Errorf("ClusterTimes do not match. got %v; want %v", got, clustertime)
		}
		got = clusterClock.GetClusterTime()
		if !bytes.Equal(got, clustertime) {
			t.Errorf("ClusterTimes do not match. got %v; want %v", got, clustertime)
		}

		Operation{}.updateClusterTimes(bsoncore.BuildDocumentFromElements(nil)) // should do nothing
	})
	t.Run("updateOperationTime", func(t *testing.T) {
		want := primitive.Timestamp{T: 1234, I: 4567}

		sessPool := session.NewPool(nil)
		id, err := uuid.New()
		noerr(t, err)

		sess, err := session.NewClientSession(sessPool, id, session.Explicit)
		noerr(t, err)
		if sess.OperationTime != nil {
			t.Fatal("OperationTime should not be set on new session.")
		}
		response := bsoncore.BuildDocumentFromElements(nil, bsoncore.AppendTimestampElement(nil, "operationTime", want.T, want.I))
		Operation{Client: sess}.updateOperationTime(response)
		got := sess.OperationTime
		if got.T != want.T || got.I != want.I {
			t.Errorf("OperationTimes do not match. got %v; want %v", got, want)
		}

		response = bsoncore.BuildDocumentFromElements(nil)
		Operation{Client: sess}.updateOperationTime(response)
		got = sess.OperationTime
		if got.T != want.T || got.I != want.I {
			t.Errorf("OperationTimes do not match. got %v; want %v", got, want)
		}

		Operation{}.updateOperationTime(response) // should do nothing
	})
	t.Run("createReadPref", func(t *testing.T) {
		rpWithTags := bsoncore.BuildDocumentFromElements(nil,
			bsoncore.AppendStringElement(nil, "mode", "secondaryPreferred"),
			bsoncore.BuildArrayElement(nil, "tags",
				bsoncore.Value{Type: bsontype.EmbeddedDocument,
					Data: bsoncore.BuildDocumentFromElements(nil,
						bsoncore.AppendStringElement(nil, "disk", "ssd"),
						bsoncore.AppendStringElement(nil, "use", "reporting"),
					),
				},
			),
		)
		rpWithMaxStaleness := bsoncore.BuildDocumentFromElements(nil,
			bsoncore.AppendStringElement(nil, "mode", "secondaryPreferred"),
			bsoncore.AppendInt32Element(nil, "maxStalenessSeconds", 25),
		)
		// Hedged read preference: {mode: "secondaryPreferred", hedge: {enabled: true}}
		rpWithHedge := bsoncore.BuildDocumentFromElements(nil,
			bsoncore.AppendStringElement(nil, "mode", "secondaryPreferred"),
			bsoncore.AppendDocumentElement(nil, "hedge", bsoncore.BuildDocumentFromElements(nil,
				bsoncore.AppendBooleanElement(nil, "enabled", true),
			)),
		)
		rpWithAllOptions := bsoncore.BuildDocumentFromElements(nil,
			bsoncore.AppendStringElement(nil, "mode", "secondaryPreferred"),
			bsoncore.BuildArrayElement(nil, "tags",
				bsoncore.Value{Type: bsontype.EmbeddedDocument,
					Data: bsoncore.BuildDocumentFromElements(nil,
						bsoncore.AppendStringElement(nil, "disk", "ssd"),
						bsoncore.AppendStringElement(nil, "use", "reporting"),
					),
				},
			),
			bsoncore.AppendInt32Element(nil, "maxStalenessSeconds", 25),
			bsoncore.AppendDocumentElement(nil, "hedge", bsoncore.BuildDocumentFromElements(nil,
				bsoncore.AppendBooleanElement(nil, "enabled", false),
			)),
		)

		rpPrimaryPreferred := bsoncore.BuildDocumentFromElements(nil, bsoncore.AppendStringElement(nil, "mode", "primaryPreferred"))
		rpPrimary := bsoncore.BuildDocumentFromElements(nil, bsoncore.AppendStringElement(nil, "mode", "primary"))
		rpSecondaryPreferred := bsoncore.BuildDocumentFromElements(nil, bsoncore.AppendStringElement(nil, "mode", "secondaryPreferred"))
		rpSecondary := bsoncore.BuildDocumentFromElements(nil, bsoncore.AppendStringElement(nil, "mode", "secondary"))
		rpNearest := bsoncore.BuildDocumentFromElements(nil, bsoncore.AppendStringElement(nil, "mode", "nearest"))

		testCases := []struct {
			name       string
			rp         *readpref.ReadPref
			serverKind description.ServerKind
			topoKind   description.TopologyKind
			opQuery    bool
			want       bsoncore.Document
		}{
			{"nil/single/mongos", nil, description.Mongos, description.Single, false, nil},
			{"nil/single/secondary", nil, description.RSSecondary, description.Single, false, rpPrimaryPreferred},
			{"primary/mongos", readpref.Primary(), description.Mongos, description.Sharded, false, nil},
			{"primary/single", readpref.Primary(), description.RSPrimary, description.Single, false, rpPrimaryPreferred},
			{"primary/primary", readpref.Primary(), description.RSPrimary, description.ReplicaSet, false, rpPrimary},
			{"primaryPreferred", readpref.PrimaryPreferred(), description.RSSecondary, description.ReplicaSet, false, rpPrimaryPreferred},
			{"secondaryPreferred/mongos/opquery", readpref.SecondaryPreferred(), description.Mongos, description.Sharded, true, nil},
			{"secondaryPreferred", readpref.SecondaryPreferred(), description.RSSecondary, description.ReplicaSet, false, rpSecondaryPreferred},
			{"secondary", readpref.Secondary(), description.RSSecondary, description.ReplicaSet, false, rpSecondary},
			{"nearest", readpref.Nearest(), description.RSSecondary, description.ReplicaSet, false, rpNearest},
			{
				"secondaryPreferred/withTags",
				readpref.SecondaryPreferred(readpref.WithTags("disk", "ssd", "use", "reporting")),
				description.RSSecondary, description.ReplicaSet, false, rpWithTags,
			},
			// GODRIVER-2205: Ensure empty tag sets are written as an empty document in the read
			// preference document. Empty tag sets match any server and are used as a fallback when
			// no other tag sets match any servers.
			{
				"secondaryPreferred/withTags/emptyTagSet",
				readpref.SecondaryPreferred(readpref.WithTagSets(
					tag.Set{{Name: "disk", Value: "ssd"}},
					tag.Set{})),
				description.RSSecondary,
				description.ReplicaSet,
				false,
				bsoncore.NewDocumentBuilder().
					AppendString("mode", "secondaryPreferred").
					AppendArray("tags", bsoncore.NewArrayBuilder().
						AppendDocument(bsoncore.NewDocumentBuilder().AppendString("disk", "ssd").Build()).
						AppendDocument(bsoncore.NewDocumentBuilder().Build()).
						Build()).
					Build(),
			},
			{
				"secondaryPreferred/withMaxStaleness",
				readpref.SecondaryPreferred(readpref.WithMaxStaleness(25 * time.Second)),
				description.RSSecondary, description.ReplicaSet, false, rpWithMaxStaleness,
			},
			{
				// A read preference document is generated for SecondaryPreferred if the hedge document is non-nil.
				"secondaryPreferred with hedge to mongos using OP_QUERY",
				readpref.SecondaryPreferred(readpref.WithHedgeEnabled(true)),
				description.Mongos,
				description.Sharded,
				true,
				rpWithHedge,
			},
			{
				"secondaryPreferred with all options",
				readpref.SecondaryPreferred(
					readpref.WithTags("disk", "ssd", "use", "reporting"),
					readpref.WithMaxStaleness(25*time.Second),
					readpref.WithHedgeEnabled(false),
				),
				description.RSSecondary,
				description.ReplicaSet,
				false,
				rpWithAllOptions,
			},
		}

		for _, tc := range testCases {
			tc := tc
			t.Run(tc.name, func(t *testing.T) {
				desc := description.SelectedServer{Kind: tc.topoKind, Server: description.Server{Kind: tc.serverKind}}
				got, err := Operation{ReadPreference: tc.rp}.createReadPref(desc, tc.opQuery)
				if err != nil {
					t.Fatalf("error creating read pref: %v", err)
				}
				if !bytes.Equal(got, tc.want) {
					t.Errorf("Returned documents do not match. got %v; want %v", got, tc.want)
				}
			})
		}
	})
	t.Run("secondaryOK", func(t *testing.T) {
		t.Run("description.SelectedServer", func(t *testing.T) {
			want := wiremessage.SecondaryOK
			desc := description.SelectedServer{
				Kind:   description.Single,
				Server: description.Server{Kind: description.RSSecondary},
			}
			got := Operation{}.secondaryOK(desc)
			if got != want {
				t.Errorf("Did not receive expected query flags. got %v; want %v", got, want)
			}
		})
		t.Run("readPreference", func(t *testing.T) {
			want := wiremessage.SecondaryOK
			got := Operation{ReadPreference: readpref.Secondary()}.secondaryOK(description.SelectedServer{})
			if got != want {
				t.Errorf("Did not receive expected query flags. got %v; want %v", got, want)
			}
		})
		t.Run("not secondaryOK", func(t *testing.T) {
			var want wiremessage.QueryFlag
			got := Operation{}.secondaryOK(description.SelectedServer{})
			if got != want {
				t.Errorf("Did not receive expected query flags. got %v; want %v", got, want)
			}
		})
	})
	t.Run("$query to mongos only", func(t *testing.T) {
		testCases := []struct {
			name   string
			server description.ServerKind
			topo   description.TopologyKind
			rp     *readpref.ReadPref
			want   bool
		}{
			{"mongos/primaryPreferred", description.Mongos, description.Sharded, readpref.PrimaryPreferred(), true},
			{"mongos/primary", description.Mongos, description.Sharded, readpref.Primary(), false},
			{"primary/primaryPreferred", description.RSPrimary, description.ReplicaSet, readpref.PrimaryPreferred(), false},
			{"primary/primary", description.RSPrimary, description.ReplicaSet, readpref.Primary(), false},
			{"secondary/primaryPreferred", description.RSSecondary, description.ReplicaSet, readpref.PrimaryPreferred(), false},
			{"secondary/primary", description.RSSecondary, description.ReplicaSet, readpref.Primary(), false},
			{"none/none", description.ServerKind(0), description.TopologyKind(0), nil, false},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				conn := new(mockConnection)

				op := Operation{
					Database:   "foobar",
					Deployment: SingleConnectionDeployment{C: conn},
					CommandFn: func(dst []byte, desc description.SelectedServer) ([]byte, error) {
						dst = bsoncore.AppendInt32Element(dst, "ping", 1)
						return dst, nil
					},
					ReadPreference: tc.rp,
				}
				var wm []byte
				desc := description.SelectedServer{
					Kind: tc.topo,
					Server: description.Server{
						Kind: tc.server,
					},
				}
				wm, _, err := op.createQueryWireMessage(0, wm, desc)
				noerr(t, err)

				// We know where the $query would be within the OP_QUERY, so we'll just index into there.
				// 16 (msg header) + 4 (flags) + 12 (foobar.$cmd) + 4 (number to skip) + 4 (number to return) + 4 (length) + 1 (document type)
				if len(wm) < 45 {
					t.Fatalf("wire message is too short. Need at least 40 bytes, but only have %d", len(wm))
				}
				got := bytes.HasPrefix(wm[45:], []byte{'$', 'q', 'u', 'e', 'r', 'y', 0x00})
				if got != tc.want {
					t.Errorf("Wiremessage did not have the proper setting for $query. got %t; want %t", got, tc.want)
				}
			})
		}
	})
	t.Run("ExecuteExhaust", func(t *testing.T) {
		t.Run("errors if connection is not streaming", func(t *testing.T) {
			conn := &mockConnection{
				rStreaming: false,
			}
			err := Operation{}.ExecuteExhaust(context.TODO(), conn)
			assert.NotNil(t, err, "expected error, got nil")
		})
	})
	t.Run("exhaustAllowed and moreToCome", func(t *testing.T) {
		// Test the interaction between exhaustAllowed and moreToCome on requests/responses when using the Execute
		// and ExecuteExhaust methods.

		// Create a server response wire message that has moreToCome=false.
		serverResponseDoc := bsoncore.BuildDocumentFromElements(nil,
			bsoncore.AppendInt32Element(nil, "ok", 1),
		)
		nonStreamingResponse := createExhaustServerResponse(serverResponseDoc, false)

		// Create a connection that reports that it cannot stream messages.
		conn := &mockConnection{
			rDesc: description.Server{
				WireVersion: &description.VersionRange{
					Max: 6,
				},
			},
			rReadWM:    nonStreamingResponse,
			rCanStream: false,
		}
		op := Operation{
			CommandFn: func(dst []byte, desc description.SelectedServer) ([]byte, error) {
				return bsoncore.AppendInt32Element(dst, internal.LegacyHello, 1), nil
			},
			Database:   "admin",
			Deployment: SingleConnectionDeployment{conn},
		}
		err := op.Execute(context.TODO())
		assert.Nil(t, err, "Execute error: %v", err)

		// The wire message sent to the server should not have exhaustAllowed=true. After execution, the connection
		// should not be in a streaming state.
		assertExhaustAllowedSet(t, conn.pWriteWM, false)
		assert.False(t, conn.CurrentlyStreaming(), "expected CurrentlyStreaming to be false")

		// Modify the connection to report that it can stream and create a new server response with moreToCome=true.
		streamingResponse := createExhaustServerResponse(serverResponseDoc, true)
		conn.rReadWM = streamingResponse
		conn.rCanStream = true
		err = op.Execute(context.TODO())
		assert.Nil(t, err, "Execute error: %v", err)
		assertExhaustAllowedSet(t, conn.pWriteWM, true)
		assert.True(t, conn.CurrentlyStreaming(), "expected CurrentlyStreaming to be true")

		// Reset the server response and go through ExecuteExhaust to mimic streaming the next response. After
		// execution, the connection should still be in a streaming state.
		conn.rReadWM = streamingResponse
		err = op.ExecuteExhaust(context.TODO(), conn)
		assert.Nil(t, err, "ExecuteExhaust error: %v", err)
		assert.True(t, conn.CurrentlyStreaming(), "expected CurrentlyStreaming to be true")
	})
	t.Run("context deadline exceeded not marked as TransientTransactionError", func(t *testing.T) {
		conn := new(mockConnection)
		// Create a context that's already timed out.
		ctx, cancel := context.WithDeadline(context.Background(), time.Unix(893934480, 0))
		defer cancel()

		op := Operation{
			Database:   "foobar",
			Deployment: SingleConnectionDeployment{C: conn},
			CommandFn: func(dst []byte, desc description.SelectedServer) ([]byte, error) {
				dst = bsoncore.AppendInt32Element(dst, "ping", 1)
				return dst, nil
			},
		}

		err := op.Execute(ctx)
		assert.NotNil(t, err, "expected an error from Execute(), got nil")
		// Assert that error is just context deadline exceeded and is therefore not a driver.Error marked
		// with the TransientTransactionError label.
		assert.Equal(t, err, context.DeadlineExceeded, "expected context.DeadlineExceeded error, got %v", err)
	})
	t.Run("canceled context not marked as TransientTransactionError", func(t *testing.T) {
		conn := new(mockConnection)
		// Create a context and cancel it immediately.
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		op := Operation{
			Database:   "foobar",
			Deployment: SingleConnectionDeployment{C: conn},
			CommandFn: func(dst []byte, desc description.SelectedServer) ([]byte, error) {
				dst = bsoncore.AppendInt32Element(dst, "ping", 1)
				return dst, nil
			},
		}

		err := op.Execute(ctx)
		assert.NotNil(t, err, "expected an error from Execute(), got nil")
		// Assert that error is just context canceled and is therefore not a driver.Error marked with
		// the TransientTransactionError label.
		assert.Equal(t, err, context.Canceled, "expected context.Canceled error, got %v", err)
	})
}

func createExhaustServerResponse(response bsoncore.Document, moreToCome bool) []byte {
	idx, wm := wiremessage.AppendHeaderStart(nil, 0, wiremessage.CurrentRequestID()+1, wiremessage.OpMsg)
	var flags wiremessage.MsgFlag
	if moreToCome {
		flags = wiremessage.MoreToCome
	}
	wm = wiremessage.AppendMsgFlags(wm, flags)
	wm = wiremessage.AppendMsgSectionType(wm, wiremessage.SingleDocument)
	wm = bsoncore.AppendDocument(wm, response)
	return bsoncore.UpdateLength(wm, idx, int32(len(wm)))
}

func assertExhaustAllowedSet(t *testing.T, wm []byte, expected bool) {
	t.Helper()
	_, _, _, _, wm, ok := wiremessage.ReadHeader(wm)
	if !ok {
		t.Fatal("could not read wm header")
	}
	flags, wm, ok := wiremessage.ReadMsgFlags(wm)
	if !ok {
		t.Fatal("could not read wm flags")
	}

	actual := flags&wiremessage.ExhaustAllowed > 0
	assert.Equal(t, expected, actual, "expected exhaustAllowed set %v, got %v", expected, actual)
}

type mockDeployment struct {
	params struct {
		selector description.ServerSelector
	}
	returns struct {
		server Server
		err    error
		retry  bool
		kind   description.TopologyKind
	}
}

func (m *mockDeployment) SelectServer(ctx context.Context, desc description.ServerSelector) (Server, error) {
	m.params.selector = desc
	return m.returns.server, m.returns.err
}

func (m *mockDeployment) Kind() description.TopologyKind { return m.returns.kind }

type mockServerSelector struct{}

func (m *mockServerSelector) SelectServer(description.Topology, []description.Server) ([]description.Server, error) {
	panic("not implemented")
}

type mockConnection struct {
	// parameters
	pWriteWM []byte
	pReadDst []byte

	// returns
	rWriteErr     error
	rReadWM       []byte
	rReadErr      error
	rDesc         description.Server
	rCloseErr     error
	rID           string
	rServerConnID *int32
	rAddr         address.Address
	rCanStream    bool
	rStreaming    bool
}

func (m *mockConnection) Description() description.Server { return m.rDesc }
func (m *mockConnection) Close() error                    { return m.rCloseErr }
func (m *mockConnection) ID() string                      { return m.rID }
func (m *mockConnection) ServerConnectionID() *int32      { return m.rServerConnID }
func (m *mockConnection) Address() address.Address        { return m.rAddr }
func (m *mockConnection) SupportsStreaming() bool         { return m.rCanStream }
func (m *mockConnection) CurrentlyStreaming() bool        { return m.rStreaming }
func (m *mockConnection) SetStreaming(streaming bool)     { m.rStreaming = streaming }
func (m *mockConnection) Stale() bool                     { return false }

func (m *mockConnection) WriteWireMessage(_ context.Context, wm []byte) error {
	m.pWriteWM = wm
	return m.rWriteErr
}

func (m *mockConnection) ReadWireMessage(_ context.Context, dst []byte) ([]byte, error) {
	m.pReadDst = dst
	return m.rReadWM, m.rReadErr
}

type retryableError struct {
	error
}

func (retryableError) Retryable() bool { return true }

var _ RetryablePoolError = retryableError{}

// mockRetryServer is used to test retry of connection checkout. Returns a retryable error from
// Connection().
type mockRetryServer struct {
	numCallsToConnection int
}

// Connection records the number of calls and returns retryable errors until the provided context
// times out or is cancelled, then returns the context error.
func (ms *mockRetryServer) Connection(ctx context.Context) (Connection, error) {
	ms.numCallsToConnection++

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	time.Sleep(1 * time.Millisecond)
	return nil, retryableError{error: errors.New("test error")}
}

func (ms *mockRetryServer) RTTMonitor() RTTMonitor {
	return &internal.ZeroRTTMonitor{}
}

func TestRetry(t *testing.T) {
	t.Run("retries multiple times with RetryContext", func(t *testing.T) {
		d := new(mockDeployment)
		ms := new(mockRetryServer)
		d.returns.server = ms

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		retry := RetryContext
		err := Operation{
			CommandFn:  func([]byte, description.SelectedServer) ([]byte, error) { return nil, nil },
			Deployment: d,
			Database:   "testing",
			RetryMode:  &retry,
			Type:       Read,
		}.Execute(ctx)
		assert.NotNil(t, err, "expected an error from Execute()")

		// Expect Connection() to be called at least 3 times. The first call is the initial attempt
		// to run the operation and the second is the retry. The third indicates that we retried
		// more than once, which is the behavior we want to assert.
		assert.True(t,
			ms.numCallsToConnection >= 3,
			"expected Connection() to be called at least 3 times")

		deadline, _ := ctx.Deadline()
		assert.True(t,
			time.Now().After(deadline),
			"expected operation to complete only after the context deadline is exceeded")
	})
}
