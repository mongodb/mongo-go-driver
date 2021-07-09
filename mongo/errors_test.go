package mongo

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
)

func TestErrorContains(t *testing.T) {
	details, err := bson.Marshal(bson.D{{"details", bson.D{{"operatorName", "$jsonSchema"}}}})
	require.Nil(t, err, "unexpected error marshaling BSON")

	cases := []struct {
		desc     string
		err      error
		contains []string
	}{
		{
			desc: "WriteException error message should contain the WriteError Message and Details",
			err: WriteException{
				WriteErrors: WriteErrors{
					{
						Message: "test message",
						Details: details,
					},
				},
			},
			contains: []string{"test message", `{"details": {"operatorName": "$jsonSchema"}}`},
		},
		{
			desc: "BulkWriteException error message should contain the WriteError Message and Details",
			err: BulkWriteException{
				WriteErrors: []BulkWriteError{
					{
						WriteError: WriteError{
							Message: "test message",
							Details: details,
						},
					},
				},
			},
			contains: []string{"test message", `{"details": {"operatorName": "$jsonSchema"}}`},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()

			for _, str := range tc.contains {
				assert.Contains(t, tc.err.Error(), str)
			}
		})
	}
}
