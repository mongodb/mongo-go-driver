package changestreamopt

import (
	"testing"

	"reflect"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/core/option"
	"github.com/mongodb/mongo-go-driver/internal/testutil/helpers"
	"github.com/mongodb/mongo-go-driver/mongo/mongoopt"
)

var resumeAfter1 = bson.NewDocument()
var resumeAfter2 = bson.NewDocument()

var c1 = &mongoopt.Collation{
	Locale: "string locale 1",
}

var c2 = &mongoopt.Collation{
	Locale: "string locale 1",
}

func createNestedCsBundle(t *testing.T) *ChangeStreamBundle {
	nestedBundle := BundleChangeStream(ResumeAfter(resumeAfter1), FullDocument(mongoopt.Default))
	testhelpers.RequireNotNil(t, nestedBundle, "nested bundle was nil")

	outerBundle := BundleChangeStream(ResumeAfter(resumeAfter2), MaxAwaitTime(500), nestedBundle, BatchSize(1000))
	testhelpers.RequireNotNil(t, outerBundle, "outer bundle was nil")

	return outerBundle
}

// Test doubly nested bundle
func createNestedCsBundle2(t *testing.T) *ChangeStreamBundle {
	b1 := BundleChangeStream(ResumeAfter(resumeAfter1), FullDocument(mongoopt.Default))
	testhelpers.RequireNotNil(t, b1, "nested bundle was nil")

	b2 := BundleChangeStream(MaxAwaitTime(100), b1, FullDocument(mongoopt.UpdateLookup))
	testhelpers.RequireNotNil(t, b2, "nested bundle was nil")

	outerBundle := BundleChangeStream(ResumeAfter(resumeAfter2), MaxAwaitTime(500), b2, BatchSize(1000))
	testhelpers.RequireNotNil(t, outerBundle, "outer bundle was nil")

	return outerBundle
}

// Test two top level nested bundles
func createNestedCsBundle3(t *testing.T) *ChangeStreamBundle {
	b1 := BundleChangeStream(ResumeAfter(resumeAfter1), FullDocument(mongoopt.Default))
	testhelpers.RequireNotNil(t, b1, "nested bundle was nil")

	b2 := BundleChangeStream(MaxAwaitTime(100), b1, FullDocument(mongoopt.UpdateLookup))
	testhelpers.RequireNotNil(t, b2, "nested bundle was nil")

	b3 := BundleChangeStream(ResumeAfter(resumeAfter2), FullDocument(mongoopt.Default))
	testhelpers.RequireNotNil(t, b3, "nested bundle was nil")

	b4 := BundleChangeStream(MaxAwaitTime(100), b3, FullDocument(mongoopt.UpdateLookup))
	testhelpers.RequireNotNil(t, b4, "nested bundle was nil")

	outerBundle := BundleChangeStream(b4, MaxAwaitTime(500), b2, BatchSize(1000))
	testhelpers.RequireNotNil(t, outerBundle, "outer bundle was nil")

	return outerBundle
}

func TestChangeStreamOpt(t *testing.T) {
	var bundle1 *ChangeStreamBundle
	bundle1 = bundle1.Collation(c1).FullDocument(mongoopt.Default).Collation(c2)
	testhelpers.RequireNotNil(t, bundle1, "created bundle was nil")
	bundle1Opts := []option.ChangeStreamOptioner{
		Collation(c1).ConvertChangeStreamOption(),
		FullDocument(mongoopt.Default).ConvertChangeStreamOption(),
		Collation(c2).ConvertChangeStreamOption(),
	}
	bundle1DedupOpts := []option.ChangeStreamOptioner{
		FullDocument(mongoopt.Default).ConvertChangeStreamOption(),
		Collation(c2).ConvertChangeStreamOption(),
	}

	bundle2 := BundleChangeStream(BatchSize(1))
	bundle2Opts := []option.ChangeStreamOptioner{
		OptBatchSize(1).ConvertChangeStreamOption(),
	}

	bundle3 := BundleChangeStream().
		BatchSize(1).
		FullDocument(mongoopt.Default).
		BatchSize(2).
		Collation(c1).
		Collation(c2).
		FullDocument(mongoopt.UpdateLookup)

	bundle3Opts := []option.ChangeStreamOptioner{
		OptBatchSize(1).ConvertChangeStreamOption(),
		OptFullDocument(mongoopt.Default).ConvertChangeStreamOption(),
		OptBatchSize(2).ConvertChangeStreamOption(),
		OptCollation{Collation: c1.Convert()}.ConvertChangeStreamOption(),
		OptCollation{Collation: c2.Convert()}.ConvertChangeStreamOption(),
		OptFullDocument(mongoopt.UpdateLookup).ConvertChangeStreamOption(),
	}

	bundle3DedupOpts := []option.ChangeStreamOptioner{
		OptBatchSize(2).ConvertChangeStreamOption(),
		OptCollation{Collation: c2.Convert()}.ConvertChangeStreamOption(),
		OptFullDocument(mongoopt.UpdateLookup).ConvertChangeStreamOption(),
	}

	nilBundle := BundleChangeStream()
	var nilBundleOpts []option.ChangeStreamOptioner

	nestedBundle1 := createNestedCsBundle(t)
	nestedBundleOpts1 := []option.ChangeStreamOptioner{
		OptResumeAfter{ResumeAfter: resumeAfter2}.ConvertChangeStreamOption(),
		OptMaxAwaitTime(500).ConvertChangeStreamOption(),
		OptResumeAfter{ResumeAfter: resumeAfter1}.ConvertChangeStreamOption(),
		OptFullDocument(mongoopt.Default).ConvertChangeStreamOption(),
		OptBatchSize(1000).ConvertChangeStreamOption(),
	}
	nestedBundleDedupOpts1 := []option.ChangeStreamOptioner{
		OptMaxAwaitTime(500).ConvertChangeStreamOption(),
		OptResumeAfter{ResumeAfter: resumeAfter1}.ConvertChangeStreamOption(),
		OptFullDocument(mongoopt.Default).ConvertChangeStreamOption(),
		OptBatchSize(1000).ConvertChangeStreamOption(),
	}

	nestedBundle2 := createNestedCsBundle2(t)
	nestedBundleOpts2 := []option.ChangeStreamOptioner{
		OptResumeAfter{ResumeAfter: resumeAfter2}.ConvertChangeStreamOption(),
		OptMaxAwaitTime(500).ConvertChangeStreamOption(),
		OptMaxAwaitTime(100).ConvertChangeStreamOption(),
		OptResumeAfter{ResumeAfter: resumeAfter1}.ConvertChangeStreamOption(),
		OptFullDocument(mongoopt.Default).ConvertChangeStreamOption(),
		OptFullDocument(mongoopt.UpdateLookup).ConvertChangeStreamOption(),
		OptBatchSize(1000).ConvertChangeStreamOption(),
	}
	nestedBundleDedupOpts2 := []option.ChangeStreamOptioner{
		OptMaxAwaitTime(100).ConvertChangeStreamOption(),
		OptResumeAfter{ResumeAfter: resumeAfter1}.ConvertChangeStreamOption(),
		OptFullDocument(mongoopt.UpdateLookup).ConvertChangeStreamOption(),
		OptBatchSize(1000).ConvertChangeStreamOption(),
	}

	nestedBundle3 := createNestedCsBundle3(t)
	nestedBundleOpts3 := []option.ChangeStreamOptioner{
		OptMaxAwaitTime(100).ConvertChangeStreamOption(),
		OptResumeAfter{ResumeAfter: resumeAfter2}.ConvertChangeStreamOption(),
		OptFullDocument(mongoopt.Default).ConvertChangeStreamOption(),
		OptFullDocument(mongoopt.UpdateLookup).ConvertChangeStreamOption(),
		OptMaxAwaitTime(500).ConvertChangeStreamOption(),
		OptMaxAwaitTime(100).ConvertChangeStreamOption(),
		OptResumeAfter{ResumeAfter: resumeAfter1}.ConvertChangeStreamOption(),
		OptFullDocument(mongoopt.Default).ConvertChangeStreamOption(),
		OptFullDocument(mongoopt.UpdateLookup).ConvertChangeStreamOption(),
		OptBatchSize(1000).ConvertChangeStreamOption(),
	}
	nestedBundleDedupOpts3 := []option.ChangeStreamOptioner{
		OptMaxAwaitTime(100).ConvertChangeStreamOption(),
		OptResumeAfter{ResumeAfter: resumeAfter1}.ConvertChangeStreamOption(),
		OptFullDocument(mongoopt.UpdateLookup).ConvertChangeStreamOption(),
		OptBatchSize(1000).ConvertChangeStreamOption(),
	}

	t.Run("TestAll", func(t *testing.T) {
		c := &mongoopt.Collation{
			Locale: "string locale",
		}

		opts := []ChangeStream{
			BatchSize(5),
			Collation(c),
			FullDocument(mongoopt.UpdateLookup),
			MaxAwaitTime(5000),
			ResumeAfter(resumeAfter2),
		}
		bundle := BundleChangeStream(opts...)

		csOpts, err := bundle.Unbundle(true)
		testhelpers.RequireNil(t, err, "got non-nill error from unbundle: %s", err)

		if len(csOpts) != len(opts) {
			t.Errorf("expected unbundled opts len %d. got %d", len(opts), len(csOpts))
		}

		for i, opt := range opts {
			if !reflect.DeepEqual(opt.ConvertChangeStreamOption(), csOpts[i]) {
				t.Errorf("opt mismatch. expected %#v, got %#v", opt, csOpts[i])
			}
		}
	})

	t.Run("MakeOptions", func(t *testing.T) {
		head := bundle1

		bundleLen := 0
		for head != nil && head.option != nil {
			bundleLen++
			head = head.next
		}

		if bundleLen != len(bundle1Opts) {
			t.Errorf("expected bundle length %d. got: %d", len(bundle1Opts), bundleLen)
		}
	})

	t.Run("Unbundle", func(t *testing.T) {
		var cases = []struct {
			name         string
			dedup        bool
			bundle       *ChangeStreamBundle
			expectedOpts []option.ChangeStreamOptioner
		}{
			{"NilBundle", false, nilBundle, nilBundleOpts},
			{"Bundle1", false, bundle1, bundle1Opts},
			{"Bundle1Dedup", true, bundle1, bundle1DedupOpts},
			{"Bundle2", false, bundle2, bundle2Opts},
			{"Bundle2Dedup", true, bundle2, bundle2Opts},
			{"Bundle3", false, bundle3, bundle3Opts},
			{"Bundle3Dedup", true, bundle3, bundle3DedupOpts},
			{"NestedBundle1_DedupFalse", false, nestedBundle1, nestedBundleOpts1},
			{"NestedBundle1_DedupTrue", true, nestedBundle1, nestedBundleDedupOpts1},
			{"NestedBundle2_DedupFalse", false, nestedBundle2, nestedBundleOpts2},
			{"NestedBundle2_DedupTrue", true, nestedBundle2, nestedBundleDedupOpts2},
			{"NestedBundle3_DedupFalse", false, nestedBundle3, nestedBundleOpts3},
			{"NestedBundle3_DedupTrue", true, nestedBundle3, nestedBundleDedupOpts3},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				options, err := tc.bundle.Unbundle(tc.dedup)
				testhelpers.RequireNil(t, err, "got non-nill error from unbundle: %s", err)

				if len(options) != len(tc.expectedOpts) {
					t.Errorf("options length does not match expected length. got %d expected %d", len(options),
						len(tc.expectedOpts))
				} else {
					for i, opt := range options {
						if !reflect.DeepEqual(opt, tc.expectedOpts[i]) {
							t.Errorf("expected: %s\nreceived: %s", opt, tc.expectedOpts[i])
						}
					}
				}
			})
		}
	})
}
