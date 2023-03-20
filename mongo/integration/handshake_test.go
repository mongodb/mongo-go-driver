package integration

import (
	"context"
	"fmt"
	"testing"

	"go.mongodb.org/mongo-driver/internal/assert"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
)

func TestHandshakeProse(t *testing.T) {
	t.Parallel()

	mt := mtest.New(t)
	defer mt.Close()

	opts := mtest.NewOptions().
		CreateCollection(false).
		ClientType(mtest.Proxy)

	for _, tcase := range []struct {
		name string
	}{
		{
			name: "valid AWS",
		},
	} {
		tcase := tcase

		mt.RunOpts(tcase.name, opts, func(mt *mtest.T) {
			// Ping the server to ensure the handshake has completed.
			err := mt.Client.Ping(context.Background(), nil)
			assert.Nil(mt, err, "Ping error: %v", err)

			messages := mt.GetProxiedMessages()
			for _, message := range messages {
				fmt.Printf("rec: %+v\n", message.Received)
			}

		})
	}
}
