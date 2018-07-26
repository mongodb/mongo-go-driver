package gridfs

import (
	"testing"

	"reflect"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/core/readconcern"
	"github.com/mongodb/mongo-go-driver/core/readpref"
	"github.com/mongodb/mongo-go-driver/core/writeconcern"
	"github.com/mongodb/mongo-go-driver/internal/testutil/helpers"
)

var wc1 = writeconcern.New(writeconcern.WMajority())
var wc2 = writeconcern.New(writeconcern.W(5))

var rpPrimary = readpref.Primary()
var rpSecondary = readpref.Secondary()

var rcMajority = readconcern.Majority()
var rcLocal = readconcern.Local()

var doc1 = bson.NewDocument(
	bson.EC.Int32("x", 1),
)

var doc2 = bson.NewDocument(
	bson.EC.Int32("y", 2),
)

var doc3 = bson.NewDocument(
	bson.EC.Int32("z", 3),
)

func createNestedBucketBundle1(t *testing.T) *BucketBundle {
	nested := BundleBucket(ReadPreference(rpSecondary), ReadConcern(rcLocal))
	testhelpers.RequireNotNil(t, nested, "nested bundle was nil")

	outer := BundleBucket(BucketName("bucket2"), BucketName("bucket1"), ReadPreference(rpPrimary), nested)
	testhelpers.RequireNotNil(t, nested, "nested bundle was nil")

	return outer
}

func createNestedBucketBundle2(t *testing.T) *BucketBundle {
	b1 := BundleBucket(WriteConcern(wc1))
	testhelpers.RequireNotNil(t, b1, "b1 was nil")

	b2 := BundleBucket(b1, ReadConcern(rcMajority))
	testhelpers.RequireNotNil(t, b2, "b2 was nil")

	outer := BundleBucket(BucketName("bucket1"), BucketName("bucket2"), b2, WriteConcern(wc2))
	testhelpers.RequireNotNil(t, outer, "outer was nil")

	return outer
}

func createNestedUploadBundle1(t *testing.T) *UploadBundle {
	nested := BundleUpload(ChunkSizeBytes(100), Metadata(doc1))
	testhelpers.RequireNotNil(t, nested, "nested bundle was nil")

	outer := BundleUpload(ChunkSizeBytes(200), ChunkSizeBytes(300), Metadata(doc2), nested)
	testhelpers.RequireNotNil(t, nested, "nested bundle was nil")

	return outer
}

func createNestedUploadBundle2(t *testing.T) *UploadBundle {
	b1 := BundleUpload(ChunkSizeBytes(100))
	testhelpers.RequireNotNil(t, b1, "b1 was nil")

	b2 := BundleUpload(b1, Metadata(doc1))
	testhelpers.RequireNotNil(t, b2, "b2 was nil")

	outer := BundleUpload(Metadata(doc2), ChunkSizeBytes(500), b2, Metadata(doc3))
	testhelpers.RequireNotNil(t, outer, "outer was nil")

	return outer
}

func createNestedFindBundle1(t *testing.T) *FindBundle {
	nested := BundleFind(BatchSize(100), Sort(doc1))
	testhelpers.RequireNotNil(t, nested, "nested bundle was nil")

	outer := BundleFind(BatchSize(200), BatchSize(300), Sort(doc2), nested)
	testhelpers.RequireNotNil(t, nested, "nested bundle was nil")

	return outer
}

func createNestedFindBundle2(t *testing.T) *FindBundle {
	b1 := BundleFind(NoCursorTimeout(true))
	testhelpers.RequireNotNil(t, b1, "b1 was nil")

	b2 := BundleFind(b1, Skip(5))
	testhelpers.RequireNotNil(t, b2, "b2 was nil")

	return BundleFind(Skip(10), NoCursorTimeout(false), b2, Skip(15))
}

func TestBucketOpt(t *testing.T) {
	nilBundle := BundleBucket()
	var nilOpts []BucketOptioner

	var bundle1 *BucketBundle
	bundle1 = bundle1.BucketName("bucket1").BucketName("bucket2").WriteConcern(wc1)
	testhelpers.RequireNotNil(t, bundle1, "created bundle was nil")
	bundle1Opts := []BucketOptioner{
		BucketName("bucket1"),
		BucketName("bucket2"),
		WriteConcern(wc1),
	}

	bundle1DedupOpts := []BucketOptioner{
		BucketName("bucket2"),
		WriteConcern(wc1),
	}

	bundle2 := BundleBucket(BucketName("bucket1"), ReadPreference(rpPrimary), ReadConcern(rcMajority))
	bundle2Opts := []BucketOptioner{
		BucketName("bucket1"),
		ReadPreference(rpPrimary),
		ReadConcern(rcMajority),
	}

	nested1 := createNestedBucketBundle1(t)
	nested1Opts := []BucketOptioner{
		BucketName("bucket2"),
		BucketName("bucket1"),
		ReadPreference(rpPrimary),
		ReadPreference(rpSecondary),
		ReadConcern(rcLocal),
	}
	nested1DedupOpts := []BucketOptioner{
		BucketName("bucket1"),
		ReadPreference(rpSecondary),
		ReadConcern(rcLocal),
	}

	nested2 := createNestedBucketBundle2(t)
	nested2Opts := []BucketOptioner{
		BucketName("bucket1"),
		BucketName("bucket2"),
		WriteConcern(wc1),
		ReadConcern(rcMajority),
		WriteConcern(wc2),
	}
	nested2DedupOpts := []BucketOptioner{
		BucketName("bucket2"),
		ReadConcern(rcMajority),
		WriteConcern(wc2),
	}

	t.Run("TestAll", func(t *testing.T) {
		opts := []BucketOptioner{
			BucketName("hello"),
			ChunkSizeBytes(500),
			ReadConcern(rcLocal),
			ReadPreference(rpPrimary),
			WriteConcern(wc1),
		}

		bundle := BundleBucket(opts...)
		bucketOpts, err := bundle.Unbundle(true)
		testhelpers.RequireNil(t, err, "error unbundling: %s", err)

		if len(bucketOpts) != len(opts) {
			t.Fatalf("len mismatch. expected %d got %d", len(opts), len(bucketOpts))
		}

		for i, opt := range opts {
			if !reflect.DeepEqual(opt, bucketOpts[i]) {
				t.Fatalf("opt mismatch. expected %#v got %#v", opt, bucketOpts[i])
			}
		}
	})

	t.Run("Unbundle", func(t *testing.T) {
		var cases = []struct {
			name         string
			bundle       *BucketBundle
			expectedOpts []BucketOptioner
			dedup        bool
		}{
			{"NilBundle", nilBundle, nilOpts, false},
			{"NilBundleDedup", nilBundle, nilOpts, true},
			{"Bundle1", bundle1, bundle1Opts, false},
			{"Bundle1Dedup", bundle1, bundle1DedupOpts, true},
			{"Bundle2", bundle2, bundle2Opts, false},
			{"Bundle2Dedup", bundle2, bundle2Opts, true},
			{"Nested1", nested1, nested1Opts, false},
			{"Nested1Dedup", nested1, nested1DedupOpts, true},
			{"Nested2", nested2, nested2Opts, false},
			{"Nested2Dedup", nested2, nested2DedupOpts, true},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				opts, err := tc.bundle.Unbundle(tc.dedup)
				testhelpers.RequireNil(t, err, "error unbundling: %s", err)

				if len(opts) != len(tc.expectedOpts) {
					t.Fatalf("options length mismatch; expected %d got %d", len(tc.expectedOpts), len(opts))
				}

				for i, opt := range opts {
					if !reflect.DeepEqual(opt, tc.expectedOpts[i]) {
						t.Fatalf("expected opt %s got %s", tc.expectedOpts[i], opt)
					}
				}
			})
		}
	})
}

func TestUploadOpt(t *testing.T) {
	nilBundle := BundleUpload()
	var nilOpts []UploadOptioner

	var bundle1 *UploadBundle
	bundle1 = bundle1.ChunkSizeBytes(100).Metadata(doc1).Metadata(doc2).ChunkSizeBytes(200)
	testhelpers.RequireNotNil(t, bundle1, "created bundle was nil")
	bundle1Opts := []UploadOptioner{
		ChunkSizeBytes(100),
		Metadata(doc1),
		Metadata(doc2),
		ChunkSizeBytes(200),
	}

	bundle1DedupOpts := []UploadOptioner{
		Metadata(doc2),
		ChunkSizeBytes(200),
	}

	bundle2 := BundleUpload(Metadata(doc1), ChunkSizeBytes(100), Metadata(doc2), ChunkSizeBytes(200))
	bundle2Opts := []UploadOptioner{
		Metadata(doc1),
		ChunkSizeBytes(100),
		Metadata(doc2),
		ChunkSizeBytes(200),
	}
	bundle2DedupOpts := []UploadOptioner{
		Metadata(doc2),
		ChunkSizeBytes(200),
	}

	nested1 := createNestedUploadBundle1(t)
	nested1Opts := []UploadOptioner{
		ChunkSizeBytes(200),
		ChunkSizeBytes(300),
		Metadata(doc2),
		ChunkSizeBytes(100),
		Metadata(doc1),
	}
	nested1DedupOpts := []UploadOptioner{
		ChunkSizeBytes(100),
		Metadata(doc1),
	}

	nested2 := createNestedUploadBundle2(t)
	nested2Opts := []UploadOptioner{
		Metadata(doc2),
		ChunkSizeBytes(500),
		ChunkSizeBytes(100),
		Metadata(doc1),
		Metadata(doc3),
	}
	nested2DedupOpts := []UploadOptioner{
		ChunkSizeBytes(100),
		Metadata(doc3),
	}

	t.Run("TestAll", func(t *testing.T) {
		opts := []UploadOptioner{
			Metadata(doc1),
			ChunkSizeBytes(100),
		}

		bundle := BundleUpload(opts...)
		bucketOpts, err := bundle.Unbundle(true)
		testhelpers.RequireNil(t, err, "error unbundling: %s", err)

		if len(bucketOpts) != len(opts) {
			t.Fatalf("len mismatch. expected %d got %d", len(opts), len(bucketOpts))
		}

		for i, opt := range opts {
			if !reflect.DeepEqual(opt, bucketOpts[i]) {
				t.Fatalf("opt mismatch. expected %#v got %#v", opt, bucketOpts[i])
			}
		}
	})

	t.Run("Unbundle", func(t *testing.T) {
		var cases = []struct {
			name         string
			bundle       *UploadBundle
			expectedOpts []UploadOptioner
			dedup        bool
		}{
			{"NilBundle", nilBundle, nilOpts, false},
			{"NilBundle", nilBundle, nilOpts, true},
			{"Bundle1", bundle1, bundle1Opts, false},
			{"Bundle1Dedup", bundle1, bundle1DedupOpts, true},
			{"Bundle2", bundle2, bundle2Opts, false},
			{"Bundle2Dedup", bundle2, bundle2DedupOpts, true},
			{"Nested1", nested1, nested1Opts, false},
			{"Nested1Dedup", nested1, nested1DedupOpts, true},
			{"Nested2", nested2, nested2Opts, false},
			{"Nested2Dedup", nested2, nested2DedupOpts, true},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				opts, err := tc.bundle.Unbundle(tc.dedup)
				testhelpers.RequireNil(t, err, "error unbundling: %s", err)

				if len(opts) != len(tc.expectedOpts) {
					t.Fatalf("options length mismatch; expected %d got %d", len(tc.expectedOpts), len(opts))
				}

				for i, opt := range opts {
					if !reflect.DeepEqual(opt, tc.expectedOpts[i]) {
						t.Fatalf("expected opt %s got %s", tc.expectedOpts[i], opt)
					}
				}
			})
		}
	})
}

func TestFindOpt(t *testing.T) {
	nilBundle := BundleFind()
	var nilOpts []FindOptioner

	var bundle1 *FindBundle
	bundle1 = bundle1.Skip(10).BatchSize(5).BatchSize(15).Skip(50)
	testhelpers.RequireNotNil(t, bundle1, "created bundle was nil")
	bundle1Opts := []FindOptioner{
		Skip(10),
		BatchSize(5),
		BatchSize(15),
		Skip(50),
	}

	bundle1DedupOpts := []FindOptioner{
		BatchSize(15),
		Skip(50),
	}

	bundle2 := BundleFind(NoCursorTimeout(true), Sort(doc1), NoCursorTimeout(false), Sort(doc2))
	bundle2Opts := []FindOptioner{
		NoCursorTimeout(true),
		Sort(doc1),
		NoCursorTimeout(false),
		Sort(doc2),
	}
	bundle2DedupOpts := []FindOptioner{
		NoCursorTimeout(false),
		Sort(doc2),
	}

	nested1 := createNestedFindBundle1(t)
	nested1Opts := []FindOptioner{
		BatchSize(200),
		BatchSize(300),
		Sort(doc2),
		BatchSize(100),
		Sort(doc1),
	}
	nested1DedupOpts := []FindOptioner{
		BatchSize(100),
		Sort(doc1),
	}

	nested2 := createNestedFindBundle2(t)
	nested2Opts := []FindOptioner{
		Skip(10),
		NoCursorTimeout(false),
		NoCursorTimeout(true),
		Skip(5),
		Skip(15),
	}
	nested2DedupOpts := []FindOptioner{
		NoCursorTimeout(true),
		Skip(15),
	}

	t.Run("TestAll", func(t *testing.T) {
		opts := []FindOptioner{
			BatchSize(10),
			Limit(5),
			MaxTime(100),
			NoCursorTimeout(true),
			Skip(550),
			Sort(doc1),
		}

		bundle := BundleFind(opts...)
		findOpts, err := bundle.Unbundle(true)
		testhelpers.RequireNil(t, err, "error unbundling: %s", err)

		if len(findOpts) != len(opts) {
			t.Fatalf("len mismatch. expected %d got %d", len(opts), len(findOpts))
		}

		for i, opt := range opts {
			if !reflect.DeepEqual(opt, findOpts[i]) {
				t.Fatalf("opt mismatch. expected %#v got %#v", opt, findOpts[i])
			}
		}
	})

	t.Run("Unbundle", func(t *testing.T) {
		var cases = []struct {
			name         string
			bundle       *FindBundle
			expectedOpts []FindOptioner
			dedup        bool
		}{
			{"NilBundle", nilBundle, nilOpts, false},
			{"NilBundle", nilBundle, nilOpts, true},
			{"Bundle1", bundle1, bundle1Opts, false},
			{"Bundle1Dedup", bundle1, bundle1DedupOpts, true},
			{"Bundle2", bundle2, bundle2Opts, false},
			{"Bundle2Dedup", bundle2, bundle2DedupOpts, true},
			{"Nested1", nested1, nested1Opts, false},
			{"Nested1Dedup", nested1, nested1DedupOpts, true},
			{"Nested2", nested2, nested2Opts, false},
			{"Nested2Dedup", nested2, nested2DedupOpts, true},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				opts, err := tc.bundle.Unbundle(tc.dedup)
				testhelpers.RequireNil(t, err, "error unbundling: %s", err)

				if len(opts) != len(tc.expectedOpts) {
					t.Fatalf("options length mismatch; expected %d got %d", len(tc.expectedOpts), len(opts))
				}

				for i, opt := range opts {
					if !reflect.DeepEqual(opt, tc.expectedOpts[i]) {
						t.Fatalf("expected opt %s got %s", tc.expectedOpts[i], opt)
					}
				}
			})
		}
	})
}

func TestNameOpt(t *testing.T) {
	nilBundle := BundleName()
	var nilOpts []NameOptioner

	var bundle1 *NameBundle
	bundle1 = bundle1.Revision(1).Revision(2)
	bundle1Opts := []NameOptioner{
		Revision(1),
		Revision(2),
	}

	bundle1DedupOpts := []NameOptioner{
		Revision(2),
	}

	nested2 := BundleName(Revision(3))
	nested := BundleName(nested2, Revision(1))
	outer := BundleName(Revision(2), nested)

	nestedOpts := []NameOptioner{
		Revision(2),
		Revision(3),
		Revision(1),
	}
	nestedDedupOpts := []NameOptioner{
		Revision(1),
	}

	t.Run("TestAll", func(t *testing.T) {
		opts := []NameOptioner{
			Revision(1),
		}

		bundle := BundleName(opts...)
		findOpts, err := bundle.Unbundle(true)
		testhelpers.RequireNil(t, err, "error unbundling: %s", err)

		if len(findOpts) != len(opts) {
			t.Fatalf("len mismatch. expected %d got %d", len(opts), len(findOpts))
		}

		for i, opt := range opts {
			if !reflect.DeepEqual(opt, findOpts[i]) {
				t.Fatalf("opt mismatch. expected %#v got %#v", opt, findOpts[i])
			}
		}
	})

	t.Run("Unbundle", func(t *testing.T) {
		var cases = []struct {
			name         string
			bundle       *NameBundle
			expectedOpts []NameOptioner
			dedup        bool
		}{
			{"NilBundle", nilBundle, nilOpts, false},
			{"NilBundle", nilBundle, nilOpts, true},
			{"Bundle1", bundle1, bundle1Opts, false},
			{"Bundle1Dedup", bundle1, bundle1DedupOpts, true},
			{"Nested", outer, nestedOpts, false},
			{"NestedDedup", outer, nestedDedupOpts, true},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				opts, err := tc.bundle.Unbundle(tc.dedup)
				testhelpers.RequireNil(t, err, "error unbundling: %s", err)

				if len(opts) != len(tc.expectedOpts) {
					t.Fatalf("options length mismatch; expected %d got %d", len(tc.expectedOpts), len(opts))
				}

				for i, opt := range opts {
					if !reflect.DeepEqual(opt, tc.expectedOpts[i]) {
						t.Fatalf("expected opt %s got %s", tc.expectedOpts[i], opt)
					}
				}
			})
		}
	})
}
