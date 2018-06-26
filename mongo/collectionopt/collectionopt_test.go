package collectionopt

import (
	"testing"

	"github.com/mongodb/mongo-go-driver/core/readconcern"
	"github.com/mongodb/mongo-go-driver/core/readpref"
	"github.com/mongodb/mongo-go-driver/core/writeconcern"
	"github.com/mongodb/mongo-go-driver/internal/testutil/helpers"
)

var rcLocal = readconcern.Local()
var rcMajority = readconcern.Majority()

var wc1 = writeconcern.New(writeconcern.W(10))
var wc2 = writeconcern.New(writeconcern.W(20))

var rpPrimary = readpref.Primary()
var rpSeconadary = readpref.Secondary()

func requireCollectionEqual(t *testing.T, expected *Collection, actual *Collection) {
	switch {
	case expected.ReadConcern != actual.ReadConcern:
		t.Errorf("read concerns don't match")
	case expected.WriteConcern != actual.WriteConcern:
		t.Errorf("write concerns don't match")
	case expected.ReadPreference != actual.ReadPreference:
		t.Errorf("read preferences don't match")
	}
}

func createNestedBundle1(t *testing.T) *CollectionBundle {
	nested := BundleCollection(ReadConcern(rcMajority))
	testhelpers.RequireNotNil(t, nested, "nested bundle was nil")

	outer := BundleCollection(ReadConcern(rcMajority), WriteConcern(wc1), nested)
	testhelpers.RequireNotNil(t, nested, "nested bundle was nil")

	return outer
}

func createdNestedBundle2(t *testing.T) *CollectionBundle {
	b1 := BundleCollection(WriteConcern(wc2))
	testhelpers.RequireNotNil(t, b1, "b1 was nil")

	b2 := BundleCollection(ReadPreference(rpPrimary), b1)
	testhelpers.RequireNotNil(t, b2, "b2 was nil")

	outer := BundleCollection(WriteConcern(wc1), ReadPreference(rpSeconadary), b2)
	testhelpers.RequireNotNil(t, outer, "outer was nil")

	return outer
}

func createNestedBundle3(t *testing.T) *CollectionBundle {
	b1 := BundleCollection(ReadConcern(rcMajority))
	testhelpers.RequireNotNil(t, b1, "b1 was nil")

	b2 := BundleCollection(WriteConcern(wc1), b1)
	testhelpers.RequireNotNil(t, b2, "b1 was nil")

	b3 := BundleCollection(ReadPreference(rpPrimary))
	testhelpers.RequireNotNil(t, b3, "b1 was nil")

	b4 := BundleCollection(WriteConcern(wc2), b3)
	testhelpers.RequireNotNil(t, b4, "b1 was nil")

	outer := BundleCollection(b4, WriteConcern(wc1), b2)
	testhelpers.RequireNotNil(t, outer, "b1 was nil")

	return outer
}

func TestCollectionOpt(t *testing.T) {
	nilBundle := BundleCollection()
	var nilDb = &Collection{}

	var bundle1 *CollectionBundle
	bundle1 = bundle1.ReadConcern(rcMajority).ReadConcern(rcLocal)
	testhelpers.RequireNotNil(t, bundle1, "created bundle was nil")
	bundle1Db := &Collection{
		ReadConcern: rcLocal,
	}

	bundle2 := BundleCollection(ReadConcern(rcLocal))
	bundle2Db := &Collection{
		ReadConcern: rcLocal,
	}

	nested1 := createNestedBundle1(t)
	nested1Db := &Collection{
		ReadConcern:  rcMajority,
		WriteConcern: wc1,
	}

	nested2 := createdNestedBundle2(t)
	nested2Db := &Collection{
		ReadPreference: rpPrimary,
		WriteConcern:   wc2,
	}

	nested3 := createNestedBundle3(t)
	nested3Db := &Collection{
		ReadConcern:    rcMajority,
		ReadPreference: rpPrimary,
		WriteConcern:   wc1,
	}

	t.Run("TestAll", func(t *testing.T) {
		opts := []Option{
			ReadConcern(rcLocal),
			WriteConcern(wc1),
			ReadPreference(rpPrimary),
		}

		db, err := BundleCollection(opts...).Unbundle()
		testhelpers.RequireNil(t, err, "got non-nil error from unbundle: %s", err)
		requireCollectionEqual(t, db, &Collection{
			ReadConcern:    rcLocal,
			WriteConcern:   wc1,
			ReadPreference: rpPrimary,
		})
	})

	t.Run("Unbundle", func(t *testing.T) {
		var cases = []struct {
			name       string
			bundle     *CollectionBundle
			collection *Collection
		}{
			{"NilBundle", nilBundle, nilDb},
			{"Bundle1", bundle1, bundle1Db},
			{"Bundle2", bundle2, bundle2Db},
			{"Nested1", nested1, nested1Db},
			{"Nested2", nested2, nested2Db},
			{"Nested3", nested3, nested3Db},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				collection, err := tc.bundle.Unbundle()
				testhelpers.RequireNil(t, err, "err unbundling db: %s", err)

				switch {
				case collection.ReadConcern != tc.collection.ReadConcern:
					t.Errorf("read concerns don't match")
				case collection.WriteConcern != tc.collection.WriteConcern:
					t.Errorf("write concerns don't match")
				case collection.ReadPreference != tc.collection.ReadPreference:
					t.Errorf("read preferences don't match")
				}
			})
		}
	})
}
