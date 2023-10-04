package operation

import (
	"testing"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/internal/assert"
	"go.mongodb.org/mongo-driver/internal/require"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/x/mongo/driver"
)

func TestDistinct_processResponse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		serverResponse bson.D
		want           bson.D
		err            error
	}{
		{
			name:           "singleton",
			serverResponse: bson.D{{"values", bson.A{"foo"}}},
			//want:           bson.A{"foo"},
			err: nil,
		},
	}

	for _, tcase := range tests {
		tcase := tcase

		t.Run(tcase.name, func(t *testing.T) {
			t.Parallel()

			srvRespBytes, err := bson.Marshal(tcase.serverResponse)
			require.NoError(t, err, "error marshaling server response: %v", err)

			info := driver.ResponseInfo{
				ServerResponse: bsoncore.Document(srvRespBytes),
			}

			op := Distinct{}

			err = op.processResponse(info)
			require.NoError(t, err, "error processing response: %v", err)

			wantBytes, err := bson.Marshal(tcase.want)
			require.NoError(t, err, "error marshaling want document: %v", err)

			want := bsoncore.Document(wantBytes)
			assert.Equal(t, want, op.result, "want: %v, got: %v", want, op.result)

		})
	}
}
