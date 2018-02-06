package mongo

import (
	"testing"

	"github.com/10gen/mongo-go-driver/bson"
	"github.com/google/go-cmp/cmp"
)

func TestTransformDocument(t *testing.T) {
	testCases := []struct {
		name     string
		document interface{}
		want     *bson.Document
		err      error
	}{
		{
			"bson.Marshaler",
			bMarsh{bson.NewDocument(bson.C.String("foo", "bar"))},
			bson.NewDocument(bson.C.String("foo", "bar")),
			nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := TransformDocument(tc.document)
			if !cmp.Equal(err, tc.err) {
				t.Errorf("Error does not match expected error. got %v; want %v", err, tc.err)
			}

			if diff := cmp.Diff(got, tc.want, cmp.AllowUnexported(bson.Document{}, bson.Element{}, bson.Value{})); diff != "" {
				t.Errorf("Returned documents differ: (-got +want)\n%s", diff)
			}
		})
	}
}

type bMarsh struct {
	*bson.Document
}

func (b bMarsh) MarshalBSON() ([]byte, error) {
	return b.Document.MarshalBSON()
}
