package options

import (
	"context"
	"crypto/tls"
	"net"
	"reflect"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/bson/bsoncodec"
	"github.com/mongodb/mongo-go-driver/event"
	"github.com/mongodb/mongo-go-driver/mongo/readconcern"
	"github.com/mongodb/mongo-go-driver/mongo/readpref"
	"github.com/mongodb/mongo-go-driver/mongo/writeconcern"
)

var tClientOptions = reflect.TypeOf(&ClientOptions{})

func TestClientOptions(t *testing.T) {
	t.Run("Set", func(t *testing.T) {
		testCases := []struct {
			name        string
			fn          interface{} // method to be run
			arg         interface{} // argument for method
			field       string      // field to be set
			dereference bool        // Should we compare a pointer or the field
		}{
			{"AppName", (*ClientOptions).SetAppName, "example-application", "AppName", true},
			{"Auth", (*ClientOptions).SetAuth, Credential{Username: "foo", Password: "bar"}, "Auth", true},
			{"Compressors", (*ClientOptions).SetCompressors, []string{"zstd", "snappy", "zlib"}, "Compressors", true},
			{"ConnectTimeout", (*ClientOptions).SetConnectTimeout, 5 * time.Second, "ConnectTimeout", true},
			{"Database", (*ClientOptions).SetDatabase, "example-database", "Database", true},
			{"Dialer", (*ClientOptions).SetDialer, testDialer{Num: 12345}, "Dialer", true},
			{"HeartbeatInterval", (*ClientOptions).SetHeartbeatInterval, 5 * time.Second, "HeartbeatInterval", true},
			{"Hosts", (*ClientOptions).SetHosts, []string{"localhost:27017", "localhost:27018", "localhost:27019"}, "Hosts", true},
			{"LocalThreshold", (*ClientOptions).SetLocalThreshold, 5 * time.Second, "LocalThreshold", true},
			{"MaxConnIdleTime", (*ClientOptions).SetMaxConnIdleTime, 5 * time.Second, "MaxConnIdleTime", true},
			{"MaxPoolSize", (*ClientOptions).SetMaxPoolSize, uint16(250), "MaxPoolSize", true},
			{"Monitor", (*ClientOptions).SetMonitor, &event.CommandMonitor{}, "Monitor", false},
			{"Password", (*ClientOptions).SetPassword, "example-password", "Password", true},
			{"ReadConcern", (*ClientOptions).SetReadConcern, readconcern.Majority(), "ReadConcern", false},
			{"ReadPreference", (*ClientOptions).SetReadPreference, readpref.SecondaryPreferred(), "ReadPreference", false},
			{"Registry", (*ClientOptions).SetRegistry, bson.NewRegistryBuilder().Build(), "Registry", false},
			{"ReplicaSet", (*ClientOptions).SetReplicaSet, "example-replicaset", "ReplicaSet", true},
			{"RetryWrites", (*ClientOptions).SetRetryWrites, true, "RetryWrites", true},
			{"ServerSelectionTimeout", (*ClientOptions).SetServerSelectionTimeout, 5 * time.Second, "ServerSelectionTimeout", true},
			{"Single", (*ClientOptions).SetSingle, true, "Single", true},
			{"SocketTimeout", (*ClientOptions).SetSocketTimeout, 5 * time.Second, "SocketTimeout", true},
			{"TLSConfig", (*ClientOptions).SetTLSConfig, &tls.Config{}, "TLSConfig", false},
			{"Username", (*ClientOptions).SetUsername, "example-username", "Username", true},
			{"WriteConcern", (*ClientOptions).SetWriteConcern, writeconcern.New(writeconcern.WMajority()), "WriteConcern", false},
			{"ZlibLevel", (*ClientOptions).SetZlibLevel, 6, "ZlibLevel", true},
		}

		opt1, opt2, optResult := Client(), Client(), Client()
		for idx, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				fn := reflect.ValueOf(tc.fn)
				if fn.Kind() != reflect.Func {
					t.Fatal("fn argument must be a function")
				}
				if fn.Type().NumIn() < 2 || fn.Type().In(0) != tClientOptions {
					t.Fatal("fn argument must have a *ClientOptions as the first argument and one other argument")
				}
				if _, exists := tClientOptions.Elem().FieldByName(tc.field); !exists {
					t.Fatalf("field (%s) does not exist in ClientOptions", tc.field)
				}
				args := make([]reflect.Value, 2)
				client := reflect.New(tClientOptions.Elem())
				args[0] = client
				want := reflect.ValueOf(tc.arg)
				args[1] = want

				if !want.IsValid() || !want.CanInterface() {
					t.Fatal("arg property of test case must be valid")
				}

				_ = fn.Call(args)

				// To avoid duplication we're piggybacking on the Set* tests to make the
				// MergeClientOptions test simpler and more thorough.
				// To do this we set the odd numbered test cases to the first opt, the even and
				// divisible by three test cases to the second, and the result of merging the two to
				// the result option. This gives us coverage of options set by the first option, by
				// the second, and by both.
				if idx%2 != 0 {
					args[0] = reflect.ValueOf(opt1)
					_ = fn.Call(args)
				}
				if idx%2 == 0 || idx%3 == 0 {
					args[0] = reflect.ValueOf(opt2)
					_ = fn.Call(args)
				}
				args[0] = reflect.ValueOf(optResult)
				_ = fn.Call(args)

				got := client.Elem().FieldByName(tc.field)
				if !got.IsValid() || !got.CanInterface() {
					t.Fatal("cannot create concrete instance from retrieved field")
				}

				if got.Kind() == reflect.Ptr && tc.dereference {
					got = got.Elem()
				}

				if !cmp.Equal(
					got.Interface(), want.Interface(),
					cmp.AllowUnexported(readconcern.ReadConcern{}, writeconcern.WriteConcern{}, readpref.ReadPref{}),
					cmp.Comparer(func(r1, r2 *bsoncodec.Registry) bool { return r1 == r2 }),
					cmp.Comparer(func(cfg1, cfg2 *tls.Config) bool { return cfg1 == cfg2 }),
				) {
					t.Errorf("Field not set properly. got %v; want %v", got.Interface(), want.Interface())
				}
			})
		}
		t.Run("MergeClientOptions/all set", func(t *testing.T) {
			want := optResult
			got := MergeClientOptions(nil, opt1, opt2)
			if diff := cmp.Diff(
				got, want,
				cmp.AllowUnexported(readconcern.ReadConcern{}, writeconcern.WriteConcern{}, readpref.ReadPref{}),
				cmp.Comparer(func(r1, r2 *bsoncodec.Registry) bool { return r1 == r2 }),
				cmp.Comparer(func(cfg1, cfg2 *tls.Config) bool { return cfg1 == cfg2 }),
			); diff != "" {
				t.Errorf("diff:\n%s", diff)
				t.Errorf("Merged client options do not match. got %v; want %v", got, want)
			}
		})
	})
}

type testDialer struct {
	Num int
}

func (testDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	return nil, nil
}
