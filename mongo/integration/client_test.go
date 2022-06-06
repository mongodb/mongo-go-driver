// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package integration

import (
	"context"
	"fmt"
	"net"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsoncodec"
	"go.mongodb.org/mongo-driver/bson/bsonrw"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/event"
	"go.mongodb.org/mongo-driver/internal"
	"go.mongodb.org/mongo-driver/internal/testutil"
	"go.mongodb.org/mongo-driver/internal/testutil/assert"
	"go.mongodb.org/mongo-driver/internal/testutil/monitor"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/x/mongo/driver"
	"go.mongodb.org/mongo-driver/x/mongo/driver/wiremessage"
	"golang.org/x/sync/errgroup"
)

var noClientOpts = mtest.NewOptions().CreateClient(false)

type negateCodec struct {
	ID int64 `bson:"_id"`
}

func (e *negateCodec) EncodeValue(ectx bsoncodec.EncodeContext, vw bsonrw.ValueWriter, val reflect.Value) error {
	return vw.WriteInt64(val.Int())
}

// DecodeValue negates the value of ID when reading
func (e *negateCodec) DecodeValue(ectx bsoncodec.DecodeContext, vr bsonrw.ValueReader, val reflect.Value) error {
	i, err := vr.ReadInt64()
	if err != nil {
		return err
	}

	val.SetInt(i * -1)
	return nil
}

var _ options.ContextDialer = &slowConnDialer{}

// A slowConnDialer dials connections that delay network round trips by the given delay duration.
type slowConnDialer struct {
	dialer *net.Dialer
	delay  time.Duration
}

var slowConnDialerDelay = 300 * time.Millisecond
var reducedHeartbeatInterval = 100 * time.Millisecond

func newSlowConnDialer(delay time.Duration) *slowConnDialer {
	return &slowConnDialer{
		dialer: &net.Dialer{},
		delay:  delay,
	}
}

func (scd *slowConnDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	conn, err := scd.dialer.DialContext(ctx, network, address)
	if err != nil {
		return nil, err
	}
	return &slowConn{
		Conn:  conn,
		delay: scd.delay,
	}, nil
}

var _ net.Conn = &slowConn{}

// slowConn is a net.Conn that delays all calls to Read() by given delay durations. All other
// net.Conn functions behave identically to the embedded net.Conn.
type slowConn struct {
	net.Conn
	delay time.Duration
}

func (sc *slowConn) Read(b []byte) (n int, err error) {
	time.Sleep(sc.delay)
	return sc.Conn.Read(b)
}

func TestClient(t *testing.T) {
	mt := mtest.New(t, noClientOpts)
	defer mt.Close()

	registryOpts := options.Client().
		SetRegistry(bson.NewRegistryBuilder().RegisterCodec(reflect.TypeOf(int64(0)), &negateCodec{}).Build())
	mt.RunOpts("registry passed to cursors", mtest.NewOptions().ClientOptions(registryOpts), func(mt *mtest.T) {
		_, err := mt.Coll.InsertOne(context.Background(), negateCodec{ID: 10})
		assert.Nil(mt, err, "InsertOne error: %v", err)
		var got negateCodec
		err = mt.Coll.FindOne(context.Background(), bson.D{}).Decode(&got)
		assert.Nil(mt, err, "Find error: %v", err)

		assert.Equal(mt, int64(-10), got.ID, "expected ID -10, got %v", got.ID)
	})
	mt.RunOpts("tls connection", mtest.NewOptions().MinServerVersion("3.0").SSL(true), func(mt *mtest.T) {
		var result bson.Raw
		err := mt.Coll.Database().RunCommand(context.Background(), bson.D{
			{"serverStatus", 1},
		}).Decode(&result)
		assert.Nil(mt, err, "serverStatus error: %v", err)

		security := result.Lookup("security")
		assert.Equal(mt, bson.TypeEmbeddedDocument, security.Type,
			"expected security field to be type %v, got %v", bson.TypeMaxKey, security.Type)
		_, found := security.Document().LookupErr("SSLServerSubjectName")
		assert.Nil(mt, found, "SSLServerSubjectName not found in result")
		_, found = security.Document().LookupErr("SSLServerHasCertificateAuthority")
		assert.Nil(mt, found, "SSLServerHasCertificateAuthority not found in result")
	})
	mt.RunOpts("x509", mtest.NewOptions().Auth(true).SSL(true), func(mt *mtest.T) {
		testCases := []struct {
			certificate string
			password    string
		}{
			{
				"MONGO_GO_DRIVER_KEY_FILE",
				"",
			},
			{
				"MONGO_GO_DRIVER_PKCS8_ENCRYPTED_KEY_FILE",
				"&sslClientCertificateKeyPassword=password",
			},
			{
				"MONGO_GO_DRIVER_PKCS8_UNENCRYPTED_KEY_FILE",
				"",
			},
		}
		for _, tc := range testCases {
			mt.Run(tc.certificate, func(mt *mtest.T) {
				const user = "C=US,ST=New York,L=New York City,O=MDB,OU=Drivers,CN=client"
				db := mt.Client.Database("$external")

				// We don't care if the user doesn't already exist.
				_ = db.RunCommand(
					context.Background(),
					bson.D{{"dropUser", user}},
				)
				err := db.RunCommand(
					context.Background(),
					bson.D{
						{"createUser", user},
						{"roles", bson.A{
							bson.D{{"role", "readWrite"}, {"db", "test"}},
						}},
					},
				).Err()
				assert.Nil(mt, err, "createUser error: %v", err)

				baseConnString := mtest.ClusterURI()
				// remove username/password from base conn string
				revisedConnString := "mongodb://"
				split := strings.Split(baseConnString, "@")
				assert.Equal(t, 2, len(split), "expected 2 parts after split, got %v (connstring %v)", split, baseConnString)
				revisedConnString += split[1]

				cs := fmt.Sprintf(
					"%s&sslClientCertificateKeyFile=%s&authMechanism=MONGODB-X509&authSource=$external%s",
					revisedConnString,
					os.Getenv(tc.certificate),
					tc.password,
				)
				authClientOpts := options.Client().ApplyURI(cs)
				testutil.AddTestServerAPIVersion(authClientOpts)
				authClient, err := mongo.Connect(context.Background(), authClientOpts)
				assert.Nil(mt, err, "authClient Connect error: %v", err)
				defer func() { _ = authClient.Disconnect(context.Background()) }()

				rdr, err := authClient.Database("test").RunCommand(context.Background(), bson.D{
					{"connectionStatus", 1},
				}).DecodeBytes()
				assert.Nil(mt, err, "connectionStatus error: %v", err)
				users, err := rdr.LookupErr("authInfo", "authenticatedUsers")
				assert.Nil(mt, err, "authenticatedUsers not found in response")
				elems, err := users.Array().Elements()
				assert.Nil(mt, err, "error getting users elements: %v", err)

				for _, userElem := range elems {
					rdr := userElem.Value().Document()
					var u struct {
						User string
						DB   string
					}

					if err := bson.Unmarshal(rdr, &u); err != nil {
						continue
					}
					if u.User == user && u.DB == "$external" {
						return
					}
				}
				mt.Fatal("unable to find authenticated user")
			})
		}
	})
	mt.RunOpts("list databases", noClientOpts, func(mt *mtest.T) {
		mt.RunOpts("filter", noClientOpts, func(mt *mtest.T) {
			testCases := []struct {
				name             string
				filter           bson.D
				hasTestDb        bool
				minServerVersion string
			}{
				{"empty", bson.D{}, true, ""},
				{"non-empty", bson.D{{"name", "foobar"}}, false, "3.6"},
			}

			for _, tc := range testCases {
				opts := mtest.NewOptions()
				if tc.minServerVersion != "" {
					opts.MinServerVersion(tc.minServerVersion)
				}

				mt.RunOpts(tc.name, opts, func(mt *mtest.T) {
					res, err := mt.Client.ListDatabases(context.Background(), tc.filter)
					assert.Nil(mt, err, "ListDatabases error: %v", err)

					var found bool
					for _, db := range res.Databases {
						if db.Name == mtest.TestDb {
							found = true
							break
						}
					}
					assert.Equal(mt, tc.hasTestDb, found, "expected to find test db: %v, found: %v", tc.hasTestDb, found)
				})
			}
		})
		mt.Run("options", func(mt *mtest.T) {
			allOpts := options.ListDatabases().SetNameOnly(true).SetAuthorizedDatabases(true)
			mt.ClearEvents()

			_, err := mt.Client.ListDatabases(context.Background(), bson.D{}, allOpts)
			assert.Nil(mt, err, "ListDatabases error: %v", err)

			evt := mt.GetStartedEvent()
			assert.Equal(mt, "listDatabases", evt.CommandName, "expected ")

			expectedDoc := bsoncore.BuildDocumentFromElements(nil,
				bsoncore.AppendBooleanElement(nil, "nameOnly", true),
				bsoncore.AppendBooleanElement(nil, "authorizedDatabases", true),
			)
			err = compareDocs(mt, expectedDoc, evt.Command)
			assert.Nil(mt, err, "compareDocs error: %v", err)
		})
	})
	mt.RunOpts("list database names", noClientOpts, func(mt *mtest.T) {
		mt.RunOpts("filter", noClientOpts, func(mt *mtest.T) {
			testCases := []struct {
				name             string
				filter           bson.D
				hasTestDb        bool
				minServerVersion string
			}{
				{"no filter", bson.D{}, true, ""},
				{"filter", bson.D{{"name", "foobar"}}, false, "3.6"},
			}

			for _, tc := range testCases {
				opts := mtest.NewOptions()
				if tc.minServerVersion != "" {
					opts.MinServerVersion(tc.minServerVersion)
				}

				mt.RunOpts(tc.name, opts, func(mt *mtest.T) {
					dbs, err := mt.Client.ListDatabaseNames(context.Background(), tc.filter)
					assert.Nil(mt, err, "ListDatabaseNames error: %v", err)

					var found bool
					for _, db := range dbs {
						if db == mtest.TestDb {
							found = true
							break
						}
					}
					assert.Equal(mt, tc.hasTestDb, found, "expected to find test db: %v, found: %v", tc.hasTestDb, found)
				})
			}
		})
		mt.Run("options", func(mt *mtest.T) {
			allOpts := options.ListDatabases().SetNameOnly(true).SetAuthorizedDatabases(true)
			mt.ClearEvents()

			_, err := mt.Client.ListDatabaseNames(context.Background(), bson.D{}, allOpts)
			assert.Nil(mt, err, "ListDatabaseNames error: %v", err)

			evt := mt.GetStartedEvent()
			assert.Equal(mt, "listDatabases", evt.CommandName, "expected ")

			expectedDoc := bsoncore.BuildDocumentFromElements(nil,
				bsoncore.AppendBooleanElement(nil, "nameOnly", true),
				bsoncore.AppendBooleanElement(nil, "authorizedDatabases", true),
			)
			err = compareDocs(mt, expectedDoc, evt.Command)
			assert.Nil(mt, err, "compareDocs error: %v", err)
		})
	})
	mt.RunOpts("ping", noClientOpts, func(mt *mtest.T) {
		mt.Run("default read preference", func(mt *mtest.T) {
			err := mt.Client.Ping(context.Background(), nil)
			assert.Nil(mt, err, "Ping error: %v", err)
		})
		mt.Run("invalid host", func(mt *mtest.T) {
			// manually create client rather than using RunOpts with ClientOptions because the testing lib will
			// apply the correct URI.
			invalidClientOpts := options.Client().
				SetServerSelectionTimeout(100 * time.Millisecond).SetHosts([]string{"invalid:123"}).
				SetConnectTimeout(500 * time.Millisecond).SetSocketTimeout(500 * time.Millisecond)
			testutil.AddTestServerAPIVersion(invalidClientOpts)
			client, err := mongo.Connect(context.Background(), invalidClientOpts)
			assert.Nil(mt, err, "Connect error: %v", err)
			err = client.Ping(context.Background(), readpref.Primary())
			assert.NotNil(mt, err, "expected error for pinging invalid host, got nil")
			_ = client.Disconnect(context.Background())
		})
	})
	mt.RunOpts("disconnect", noClientOpts, func(mt *mtest.T) {
		mt.Run("nil context", func(mt *mtest.T) {
			err := mt.Client.Disconnect(nil)
			assert.Nil(mt, err, "Disconnect error: %v", err)
		})
	})
	mt.RunOpts("watch", noClientOpts, func(mt *mtest.T) {
		mt.Run("disconnected", func(mt *mtest.T) {
			c, err := mongo.NewClient(options.Client().ApplyURI(mtest.ClusterURI()))
			assert.Nil(mt, err, "NewClient error: %v", err)
			_, err = c.Watch(context.Background(), mongo.Pipeline{})
			assert.Equal(mt, mongo.ErrClientDisconnected, err, "expected error %v, got %v", mongo.ErrClientDisconnected, err)
		})
	})
	mt.RunOpts("end sessions", mtest.NewOptions().MinServerVersion("3.6"), func(mt *mtest.T) {
		_, err := mt.Client.ListDatabases(context.Background(), bson.D{})
		assert.Nil(mt, err, "ListDatabases error: %v", err)

		mt.ClearEvents()
		err = mt.Client.Disconnect(context.Background())
		assert.Nil(mt, err, "Disconnect error: %v", err)

		started := mt.GetStartedEvent()
		assert.Equal(mt, "endSessions", started.CommandName, "expected cmd name endSessions, got %v", started.CommandName)
	})
	mt.RunOpts("hello lastWriteDate", mtest.NewOptions().Topologies(mtest.ReplicaSet), func(mt *mtest.T) {
		_, err := mt.Coll.InsertOne(context.Background(), bson.D{{"x", 1}})
		assert.Nil(mt, err, "InsertOne error: %v", err)
	})
	sessionOpts := mtest.NewOptions().MinServerVersion("3.6.0").CreateClient(false)
	mt.RunOpts("causal consistency", sessionOpts, func(mt *mtest.T) {
		testCases := []struct {
			name       string
			opts       *options.SessionOptions
			consistent bool
		}{
			{"default", options.Session(), true},
			{"true", options.Session().SetCausalConsistency(true), true},
			{"false", options.Session().SetCausalConsistency(false), false},
		}

		for _, tc := range testCases {
			mt.Run(tc.name, func(mt *mtest.T) {
				sess, err := mt.Client.StartSession(tc.opts)
				assert.Nil(mt, err, "StartSession error: %v", err)
				defer sess.EndSession(context.Background())
				xs := sess.(mongo.XSession)
				consistent := xs.ClientSession().Consistent
				assert.Equal(mt, tc.consistent, consistent, "expected consistent to be %v, got %v", tc.consistent, consistent)
			})
		}
	})
	retryOpts := mtest.NewOptions().MinServerVersion("3.6.0").ClientType(mtest.Mock)
	mt.RunOpts("retry writes error 20 wrapped", retryOpts, func(mt *mtest.T) {
		writeErrorCode20 := mtest.CreateWriteErrorsResponse(mtest.WriteError{
			Message: "Transaction numbers",
			Code:    20,
		})
		writeErrorCode19 := mtest.CreateWriteErrorsResponse(mtest.WriteError{
			Message: "Transaction numbers",
			Code:    19,
		})
		writeErrorCode20WrongMsg := mtest.CreateWriteErrorsResponse(mtest.WriteError{
			Message: "Not transaction numbers",
			Code:    20,
		})
		cmdErrCode20 := mtest.CreateCommandErrorResponse(mtest.CommandError{
			Message: "Transaction numbers",
			Code:    20,
		})
		cmdErrCode19 := mtest.CreateCommandErrorResponse(mtest.CommandError{
			Message: "Transaction numbers",
			Code:    19,
		})
		cmdErrCode20WrongMsg := mtest.CreateCommandErrorResponse(mtest.CommandError{
			Message: "Not transaction numbers",
			Code:    20,
		})

		testCases := []struct {
			name                 string
			errResponse          bson.D
			expectUnsupportedMsg bool
		}{
			{"write error code 20", writeErrorCode20, true},
			{"write error code 20 wrong msg", writeErrorCode20WrongMsg, false},
			{"write error code 19 right msg", writeErrorCode19, false},
			{"command error code 20", cmdErrCode20, true},
			{"command error code 20 wrong msg", cmdErrCode20WrongMsg, false},
			{"command error code 19 right msg", cmdErrCode19, false},
		}
		for _, tc := range testCases {
			mt.Run(tc.name, func(mt *mtest.T) {
				mt.ClearMockResponses()
				mt.AddMockResponses(tc.errResponse)

				sess, err := mt.Client.StartSession()
				assert.Nil(mt, err, "StartSession error: %v", err)
				defer sess.EndSession(context.Background())

				_, err = mt.Coll.InsertOne(context.Background(), bson.D{{"x", 1}})
				assert.NotNil(mt, err, "expected err but got nil")
				if tc.expectUnsupportedMsg {
					assert.Equal(mt, driver.ErrUnsupportedStorageEngine.Error(), err.Error(),
						"expected error %v, got %v", driver.ErrUnsupportedStorageEngine, err)
					return
				}
				assert.NotEqual(mt, driver.ErrUnsupportedStorageEngine.Error(), err.Error(),
					"got ErrUnsupportedStorageEngine but wanted different error")
			})
		}
	})

	testAppName := "foo"
	appNameClientOpts := options.Client().
		SetAppName(testAppName)
	appNameMtOpts := mtest.NewOptions().
		ClientType(mtest.Proxy).
		ClientOptions(appNameClientOpts).
		Topologies(mtest.Single)
	mt.RunOpts("app name is always sent", appNameMtOpts, func(mt *mtest.T) {
		err := mt.Client.Ping(context.Background(), mtest.PrimaryRp)
		assert.Nil(mt, err, "Ping error: %v", err)

		msgPairs := mt.GetProxiedMessages()
		assert.True(mt, len(msgPairs) >= 2, "expected at least 2 events sent, got %v", len(msgPairs))

		// First two messages should be connection handshakes: one for the heartbeat connection and the other for the
		// application connection.
		for idx, pair := range msgPairs[:2] {
			helloCommand := internal.LegacyHello
			//  Expect "hello" command name with API version.
			if os.Getenv("REQUIRE_API_VERSION") == "true" {
				helloCommand = "hello"
			}
			assert.Equal(mt, pair.CommandName, helloCommand, "expected command name %s at index %d, got %s", helloCommand, idx,
				pair.CommandName)

			sent := pair.Sent
			appNameVal, err := sent.Command.LookupErr("client", "application", "name")
			assert.Nil(mt, err, "expected command %s at index %d to contain app name", sent.Command, idx)
			appName := appNameVal.StringValue()
			assert.Equal(mt, testAppName, appName, "expected app name %v at index %d, got %v", testAppName, idx,
				appName)
		}
	})

	// Test that direct connections work as expected.
	firstServerAddr := mtest.GlobalTopology().Description().Servers[0].Addr
	directConnectionOpts := options.Client().
		ApplyURI(fmt.Sprintf("mongodb://%s", firstServerAddr)).
		SetReadPreference(readpref.Primary()).
		SetDirect(true)
	mtOpts := mtest.NewOptions().
		ClientOptions(directConnectionOpts).
		CreateCollection(false).
		MinServerVersion("3.6").     // Minimum server version 3.6 to force OP_MSG.
		Topologies(mtest.ReplicaSet) // Read preference isn't sent to standalones so we can test on replica sets.
	mt.RunOpts("direct connection made", mtOpts, func(mt *mtest.T) {
		_, err := mt.Coll.Find(context.Background(), bson.D{})
		assert.Nil(mt, err, "Find error: %v", err)

		// When connected directly, the primary read preference should be overwritten to primaryPreferred.
		evt := mt.GetStartedEvent()
		assert.Equal(mt, "find", evt.CommandName, "expected 'find' event, got '%s'", evt.CommandName)
		modeVal, err := evt.Command.LookupErr("$readPreference", "mode")
		assert.Nil(mt, err, "expected command %s to include $readPreference", evt.Command)

		mode := modeVal.StringValue()
		assert.Equal(mt, mode, "primaryPreferred", "expected read preference mode primaryPreferred, got %v", mode)
	})

	// Test that using a client with minPoolSize set doesn't cause a data race.
	mtOpts = mtest.NewOptions().ClientOptions(options.Client().SetMinPoolSize(5))
	mt.RunOpts("minPoolSize", mtOpts, func(mt *mtest.T) {
		err := mt.Client.Ping(context.Background(), readpref.Primary())
		assert.Nil(t, err, "unexpected error calling Ping: %v", err)
	})

	mt.Run("minimum RTT is monitored", func(mt *mtest.T) {
		// Reset the client with a dialer that delays all network round trips by 300ms and set the
		// heartbeat interval to 100ms to reduce the time it takes to collect RTT samples.
		mt.ResetClient(options.Client().
			SetDialer(newSlowConnDialer(slowConnDialerDelay)).
			SetHeartbeatInterval(reducedHeartbeatInterval))

		// Assert that the minimum RTT is eventually >250ms.
		topo := getTopologyFromClient(mt.Client)
		assert.Soon(mt, func(ctx context.Context) {
			for {
				// Stop loop if callback has been canceled.
				select {
				case <-ctx.Done():
					return
				default:
				}

				time.Sleep(100 * time.Millisecond)

				// Wait for all of the server's minimum RTTs to be >250ms.
				done := true
				for _, desc := range topo.Description().Servers {
					server, err := topo.FindServer(desc)
					assert.Nil(mt, err, "FindServer error: %v", err)
					if server.MinRTT() <= 250*time.Millisecond {
						done = false
					}
				}
				if done {
					return
				}
			}
		}, 10*time.Second)
	})

	// Test that if the minimum RTT is greater than the remaining timeout for an operation, the
	// operation is not sent to the server and no connections are closed.
	mt.Run("minimum RTT used to prevent sending requests", func(mt *mtest.T) {
		// Assert that we can call Ping with a 250ms timeout.
		ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
		defer cancel()
		err := mt.Client.Ping(ctx, nil)
		assert.Nil(mt, err, "Ping error: %v", err)

		// Reset the client with a dialer that delays all network round trips by 300ms and set the
		// heartbeat interval to 100ms to reduce the time it takes to collect RTT samples.
		tpm := monitor.NewTestPoolMonitor()
		mt.ResetClient(options.Client().
			SetPoolMonitor(tpm.PoolMonitor).
			SetDialer(newSlowConnDialer(slowConnDialerDelay)).
			SetHeartbeatInterval(reducedHeartbeatInterval))

		// Assert that the minimum RTT is eventually >250ms.
		topo := getTopologyFromClient(mt.Client)
		assert.Soon(mt, func(ctx context.Context) {
			for {
				// Stop loop if callback has been canceled.
				select {
				case <-ctx.Done():
					return
				default:
				}

				time.Sleep(100 * time.Millisecond)

				// Wait for all of the server's minimum RTTs to be >250ms.
				done := true
				for _, desc := range topo.Description().Servers {
					server, err := topo.FindServer(desc)
					assert.Nil(mt, err, "FindServer error: %v", err)
					if server.MinRTT() <= 250*time.Millisecond {
						done = false
					}
				}
				if done {
					return
				}
			}
		}, 10*time.Second)

		// Once we've waited for the minimum RTT for the single server to be >250ms, run a bunch of
		// Ping operations with a timeout of 250ms and expect that they return errors.
		for i := 0; i < 10; i++ {
			ctx, cancel = context.WithTimeout(context.Background(), 250*time.Millisecond)
			err := mt.Client.Ping(ctx, nil)
			cancel()
			assert.NotNil(mt, err, "expected Ping to return an error")
		}

		// Assert that the Ping timeouts result in no connections being closed.
		closed := len(tpm.Events(func(e *event.PoolEvent) bool { return e.Type == event.ConnectionClosed }))
		assert.Equal(t, 0, closed, "expected no connections to be closed")
	})

	mt.Run("RTT90 is monitored", func(mt *mtest.T) {
		if testing.Short() {
			t.Skip("skipping integration test in short mode")
		}

		// Reset the client with a dialer that delays all network round trips by 300ms and set the
		// heartbeat interval to 100ms to reduce the time it takes to collect RTT samples.
		mt.ResetClient(options.Client().
			SetDialer(newSlowConnDialer(slowConnDialerDelay)).
			SetHeartbeatInterval(reducedHeartbeatInterval))

		// Assert that RTT90s are eventually >300ms.
		topo := getTopologyFromClient(mt.Client)
		assert.Soon(mt, func(ctx context.Context) {
			for {
				// Stop loop if callback has been canceled.
				select {
				case <-ctx.Done():
					return
				default:
				}

				time.Sleep(100 * time.Millisecond)

				// Wait for all of the server's RTT90s to be >300ms.
				done := true
				for _, desc := range topo.Description().Servers {
					server, err := topo.FindServer(desc)
					assert.Nil(mt, err, "FindServer error: %v", err)
					if server.RTT90() <= 300*time.Millisecond {
						done = false
					}
				}
				if done {
					return
				}
			}
		}, 10*time.Second)
	})

	// Test that if Timeout is set and the RTT90 is greater than the remaining timeout for an operation, the
	// operation is not sent to the server, fails with a timeout error, and no connections are closed.
	mt.Run("RTT90 used to prevent sending requests", func(mt *mtest.T) {
		if testing.Short() {
			t.Skip("skipping integration test in short mode")
		}

		// Assert that we can call Ping with a 250ms timeout.
		ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
		defer cancel()
		err := mt.Client.Ping(ctx, nil)
		assert.Nil(mt, err, "Ping error: %v", err)

		// Reset the client with a dialer that delays all network round trips by 300ms, set the
		// heartbeat interval to 100ms to reduce the time it takes to collect RTT samples, and
		// set a Timeout of 0 (infinite) on the Client to ensure that RTT90 is used as a sending
		// threshold.
		tpm := monitor.NewTestPoolMonitor()
		mt.ResetClient(options.Client().
			SetPoolMonitor(tpm.PoolMonitor).
			SetDialer(newSlowConnDialer(slowConnDialerDelay)).
			SetHeartbeatInterval(reducedHeartbeatInterval).
			SetTimeout(0))

		// Assert that RTT90s are eventually >300ms.
		topo := getTopologyFromClient(mt.Client)
		assert.Soon(mt, func(ctx context.Context) {
			for {
				// Stop loop if callback has been canceled.
				select {
				case <-ctx.Done():
					return
				default:
				}

				time.Sleep(100 * time.Millisecond)

				// Wait for all of the server's RTT90s to be >300ms.
				done := true
				for _, desc := range topo.Description().Servers {
					server, err := topo.FindServer(desc)
					assert.Nil(mt, err, "FindServer error: %v", err)
					if server.RTT90() <= 300*time.Millisecond {
						done = false
					}
				}
				if done {
					return
				}
			}
		}, 10*time.Second)

		// Once we've waited for the RTT90 for the servers to be >300ms, run 10 Ping operations
		// with a timeout of 300ms and expect that they return timeout errors.
		for i := 0; i < 10; i++ {
			ctx, cancel = context.WithTimeout(context.Background(), 300*time.Millisecond)
			err := mt.Client.Ping(ctx, nil)
			cancel()
			assert.NotNil(mt, err, "expected Ping to return an error")
			assert.True(mt, mongo.IsTimeout(err), "expected a timeout error: got %v", err)
		}

		// Assert that the Ping timeouts result in no connections being closed.
		closed := len(tpm.Events(func(e *event.PoolEvent) bool { return e.Type == event.ConnectionClosed }))
		assert.Equal(t, 0, closed, "expected no connections to be closed")
	})

	// Test that OP_MSG is used for authentication-related commands on 3.6+ (WV 6+). Do not test when API version is
	// set, as handshakes will always use OP_MSG.
	opMsgOpts := mtest.NewOptions().ClientType(mtest.Proxy).MinServerVersion("3.6").Auth(true).RequireAPIVersion(false)
	mt.RunOpts("OP_MSG used for authentication on 3.6+", opMsgOpts, func(mt *mtest.T) {
		err := mt.Client.Ping(context.Background(), mtest.PrimaryRp)
		assert.Nil(mt, err, "Ping error: %v", err)

		msgPairs := mt.GetProxiedMessages()
		assert.True(mt, len(msgPairs) >= 3, "expected at least 3 events, got %v", len(msgPairs))

		// First message should a be connection handshake. This handshake should use OP_QUERY as the OpCode, as wire
		// version is not yet known.
		pair := msgPairs[0]
		assert.Equal(mt, internal.LegacyHello, pair.CommandName, "expected command name %s at index 0, got %s",
			internal.LegacyHello, pair.CommandName)
		assert.Equal(mt, wiremessage.OpQuery, pair.Sent.OpCode,
			"expected 'OP_QUERY' OpCode in wire message, got %q", pair.Sent.OpCode.String())

		// Look for a saslContinue in the remaining proxied messages and assert that it uses the OP_MSG OpCode, as wire
		// version is now known to be >= 6.
		var saslContinueFound bool
		for _, pair := range msgPairs[1:] {
			if pair.CommandName == "saslContinue" {
				saslContinueFound = true
				assert.Equal(mt, wiremessage.OpMsg, pair.Sent.OpCode,
					"expected 'OP_MSG' OpCode in wire message, got %s", pair.Sent.OpCode.String())
				break
			}
		}
		assert.True(mt, saslContinueFound, "did not find 'saslContinue' command in proxied messages")
	})

	// Test that OP_MSG is used for handshakes when API version is declared.
	opMsgSAPIOpts := mtest.NewOptions().ClientType(mtest.Proxy).MinServerVersion("5.0").RequireAPIVersion(true)
	mt.RunOpts("OP_MSG used for handshakes when API version declared", opMsgSAPIOpts, func(mt *mtest.T) {
		err := mt.Client.Ping(context.Background(), mtest.PrimaryRp)
		assert.Nil(mt, err, "Ping error: %v", err)

		msgPairs := mt.GetProxiedMessages()
		assert.True(mt, len(msgPairs) >= 3, "expected at least 3 events, got %v", len(msgPairs))

		// First three messages should be connection handshakes: one for the heartbeat connection, another for the
		// application connection, and a final one for the RTT monitor connection.
		for idx, pair := range msgPairs[:3] {
			assert.Equal(mt, "hello", pair.CommandName, "expected command name 'hello' at index %d, got %s", idx,
				pair.CommandName)

			// Assert that appended OpCode is OP_MSG when API version is set.
			assert.Equal(mt, wiremessage.OpMsg, pair.Sent.OpCode,
				"expected 'OP_MSG' OpCode in wire message, got %q", pair.Sent.OpCode.String())
		}
	})

	// Test that OP_MSG is used for handshakes when loadBalanced is true.
	opMsgLBOpts := mtest.NewOptions().ClientType(mtest.Proxy).MinServerVersion("5.0").Topologies(mtest.LoadBalanced)
	mt.RunOpts("OP_MSG used for handshakes when loadBalanced is true", opMsgLBOpts, func(mt *mtest.T) {
		err := mt.Client.Ping(context.Background(), mtest.PrimaryRp)
		assert.Nil(mt, err, "Ping error: %v", err)

		msgPairs := mt.GetProxiedMessages()
		assert.True(mt, len(msgPairs) >= 3, "expected at least 3 events, got %v", len(msgPairs))

		// First three messages should be connection handshakes: one for the heartbeat connection, another for the
		// application connection, and a final one for the RTT monitor connection.
		for idx, pair := range msgPairs[:3] {
			assert.Equal(mt, "hello", pair.CommandName, "expected command name 'hello' at index %d, got %s", idx,
				pair.CommandName)

			// Assert that appended OpCode is OP_MSG when loadBalanced is true.
			assert.Equal(mt, wiremessage.OpMsg, pair.Sent.OpCode,
				"expected 'OP_MSG' OpCode in wire message, got %q", pair.Sent.OpCode.String())
		}
	})
}

func TestClientStress(t *testing.T) {
	mtOpts := mtest.NewOptions().CreateClient(false)
	mt := mtest.New(t, mtOpts)
	defer mt.Close()

	// Test that a Client can recover from a massive traffic spike after the traffic spike is over.
	mt.Run("Client recovers from traffic spike", func(mt *mtest.T) {
		oid := primitive.NewObjectID()
		doc := bson.D{{Key: "_id", Value: oid}, {Key: "key", Value: "value"}}
		_, err := mt.Coll.InsertOne(context.Background(), doc)
		assert.Nil(mt, err, "InsertOne error: %v", err)

		// findOne calls FindOne("_id": oid) on the given collection and with the given timeout. It
		// returns any errors.
		findOne := func(coll *mongo.Collection, timeout time.Duration) error {
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()
			var res map[string]interface{}
			return coll.FindOne(ctx, bson.D{{Key: "_id", Value: oid}}).Decode(&res)
		}

		// findOneFor calls FindOne on the given collection and with the given timeout in a loop for
		// the given duration and returns any errors returned by FindOne.
		findOneFor := func(coll *mongo.Collection, timeout time.Duration, d time.Duration) []error {
			errs := make([]error, 0)
			start := time.Now()
			for time.Since(start) <= d {
				err := findOne(coll, timeout)
				if err != nil {
					errs = append(errs, err)
				}
				time.Sleep(10 * time.Microsecond)
			}
			return errs
		}

		// Calculate the maximum observed round-trip time by measuring the duration of some FindOne
		// operations and picking the max.
		var maxRTT time.Duration
		for i := 0; i < 50; i++ {
			start := time.Now()
			err := findOne(mt.Coll, 10*time.Second)
			assert.Nil(t, err, "FindOne error: %v", err)
			duration := time.Since(start)
			if duration > maxRTT {
				maxRTT = duration
			}
		}
		assert.True(mt, maxRTT > 0, "RTT must be greater than 0")

		// Run tests with various "maxPoolSize" values, including 1-connection pools and the default
		// size of 100, to test how the client handles traffic spikes using different connection
		// pool configurations.
		maxPoolSizes := []uint64{1, 10, 100}
		for _, maxPoolSize := range maxPoolSizes {
			tpm := monitor.NewTestPoolMonitor()
			maxPoolSizeOpt := mtest.NewOptions().ClientOptions(
				options.Client().
					SetPoolMonitor(tpm.PoolMonitor).
					SetMaxPoolSize(maxPoolSize))
			mt.RunOpts(fmt.Sprintf("maxPoolSize %d", maxPoolSize), maxPoolSizeOpt, func(mt *mtest.T) {
				// Print the count of connection created, connection closed, and pool clear events
				// collected during the test to help with debugging.
				defer func() {
					created := len(tpm.Events(func(e *event.PoolEvent) bool { return e.Type == event.ConnectionCreated }))
					closed := len(tpm.Events(func(e *event.PoolEvent) bool { return e.Type == event.ConnectionClosed }))
					poolCleared := len(tpm.Events(func(e *event.PoolEvent) bool { return e.Type == event.PoolCleared }))
					mt.Logf("Connections created: %d, connections closed: %d, pool clears: %d", created, closed, poolCleared)
				}()

				doc := bson.D{{Key: "_id", Value: oid}, {Key: "key", Value: "value"}}
				_, err := mt.Coll.InsertOne(context.Background(), doc)
				assert.Nil(mt, err, "InsertOne error: %v", err)

				// Set the timeout to be 100x the maximum observed RTT. Use a minimum 100ms timeout to
				// prevent spurious test failures due to extremely low timeouts.
				timeout := maxRTT * 100
				minTimeout := 100 * time.Millisecond
				if timeout < minTimeout {
					timeout = minTimeout
				}
				mt.Logf("Max RTT %v; using a timeout of %v", maxRTT, timeout)

				// Warm up the client for 1 second to allow connections to be established. Ignore
				// any errors.
				_ = findOneFor(mt.Coll, timeout, 1*time.Second)

				// Simulate normal traffic by running one FindOne loop for 1 second and assert that there
				// are no errors.
				errs := findOneFor(mt.Coll, timeout, 1*time.Second)
				assert.True(mt, len(errs) == 0, "expected no errors, but got %d (%v)", len(errs), errs)

				// Simulate an extreme traffic spike by running 1,000 FindOne loops in parallel for 10
				// seconds and expect at least some errors to occur.
				g := new(errgroup.Group)
				for i := 0; i < 1000; i++ {
					g.Go(func() error {
						errs := findOneFor(mt.Coll, timeout, 10*time.Second)
						if len(errs) == 0 {
							return nil
						}
						return errs[len(errs)-1]
					})
				}
				err = g.Wait()
				mt.Logf("Error from extreme traffic spike (errors are expected): %v", err)

				// Simulate normal traffic again for 5 second. Ignore any errors to allow any outstanding
				// connection errors to stop.
				_ = findOneFor(mt.Coll, timeout, 5*time.Second)

				// Simulate normal traffic again for 1 second and assert that there are no errors.
				errs = findOneFor(mt.Coll, timeout, 1*time.Second)
				assert.True(mt, len(errs) == 0, "expected no errors, but got %d (%v)", len(errs), errs)
			})
		}
	})
}
