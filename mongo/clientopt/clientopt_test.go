package clientopt

import (
	"testing"

	"github.com/mongodb/mongo-go-driver/core/connstring"
	"github.com/mongodb/mongo-go-driver/internal/testutil/helpers"
)

func createNestedBundle1(t *testing.T) *ClientBundle {
	nested := BundleClient(ReplicaSet("hello"))
	testhelpers.RequireNotNil(t, nested, "nested bundle was nil")

	outer := BundleClient(ReplicaSet("test"), MaxConnsPerHost(10), nested)
	testhelpers.RequireNotNil(t, nested, "nested bundle was nil")

	return outer
}

func createdNestedBundle2(t *testing.T) *ClientBundle {
	b1 := BundleClient(MaxConnsPerHost(20))
	testhelpers.RequireNotNil(t, b1, "b1 was nil")

	b2 := BundleClient(b1, LocalThreshold(10))
	testhelpers.RequireNotNil(t, b2, "b2 was nil")

	outer := BundleClient(MaxConnsPerHost(10), LocalThreshold(20), b2, ReplicaSet("test"))
	testhelpers.RequireNotNil(t, outer, "outer was nil")

	return outer
}

func createNestedBundle3(t *testing.T) *ClientBundle {
	b1 := BundleClient(ReplicaSet("test"))
	testhelpers.RequireNotNil(t, b1, "b1 was nil")

	b2 := BundleClient(MaxConnsPerHost(20), b1)
	testhelpers.RequireNotNil(t, b2, "b1 was nil")

	b3 := BundleClient(LocalThreshold(10))
	testhelpers.RequireNotNil(t, b3, "b1 was nil")

	b4 := BundleClient(MaxConnsPerHost(20), b3)
	testhelpers.RequireNotNil(t, b4, "b1 was nil")

	outer := BundleClient(b4, MaxConnsPerHost(10), b2)
	testhelpers.RequireNotNil(t, outer, "b1 was nil")

	return outer
}

func TestClientOpt(t *testing.T) {
	nilBundle := BundleClient()
	var nilClient = &Client{}

	var bundle1 *ClientBundle
	bundle1 = bundle1.ReplicaSet("hello").ReplicaSet("bye")
	testhelpers.RequireNotNil(t, bundle1, "created bundle was nil")
	bundle1Client := &Client{
		ConnString: connstring.ConnString{ReplicaSet: "bye"},
	}

	bundle2 := BundleClient(ReplicaSet("test"))
	bundle2Client := &Client{
		ConnString: connstring.ConnString{ReplicaSet: "test"},
	}

	nested1 := createNestedBundle1(t)
	nested1Client := &Client{
		ConnString: connstring.ConnString{
			ReplicaSet:      "hello",
			MaxConnsPerHost: 10,
		},
	}

	nested2 := createdNestedBundle2(t)
	nested2Client := &Client{
		ConnString: connstring.ConnString{
			ReplicaSet:      "test",
			MaxConnsPerHost: 20,
			LocalThreshold:  10,
		},
	}

	nested3 := createNestedBundle3(t)
	nested3Client := &Client{
		ConnString: connstring.ConnString{
			ReplicaSet:      "test",
			MaxConnsPerHost: 20,
			LocalThreshold:  10,
		},
	}

	t.Run("Unbundle", func(t *testing.T) {
		var cases = []struct {
			name   string
			bundle *ClientBundle
			client *Client
		}{
			{"NilBundle", nilBundle, nilClient},
			{"Bundle1", bundle1, bundle1Client},
			{"Bundle2", bundle2, bundle2Client},
			{"Nested1", nested1, nested1Client},
			{"Nested2", nested2, nested2Client},
			{"Nested3", nested3, nested3Client},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				client, err := tc.bundle.Unbundle(connstring.ConnString{})
				testhelpers.RequireNil(t, err, "err unbundling client: %s", err)

				switch {
				case client.ConnString.ReplicaSet != tc.client.ConnString.ReplicaSet:
					t.Errorf("replica sets don't match")
				case client.ConnString.MaxConnsPerHost != tc.client.ConnString.MaxConnsPerHost:
					t.Errorf("max conns per host don't match")
				case client.ConnString.LocalThreshold != tc.client.ConnString.LocalThreshold:
					t.Errorf("local thresholds don't match")
				}
			})
		}
	})
}
