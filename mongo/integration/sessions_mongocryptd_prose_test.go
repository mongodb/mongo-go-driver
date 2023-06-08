//go:build cse
// +build cse

package integration

import (
	"context"
	"fmt"
	"testing"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// TestSessionsMongocryptdProse tests session prose tests that should use a
// mongocryptd server as the test server (available with server versions 4.2+).
func TestSessionsMongocryptdProse(t *testing.T) {
	t.Parallel()

	mtOpts := mtest.
		NewOptions().
		MinServerVersion("4.2").
		Enterprise(true).
		CreateClient(false)

	mt := mtest.New(t, mtOpts)
	defer mt.Close()

	kmsProviders := map[string]map[string]interface{}{
		"local": {
			"key": localMasterKey,
		},
	}

	mt.Run("18 implicit session is ignored if connection does not support sessions", func(mt *mtest.T) {
		fmt.Println("where the test actual starts --------------------")
		ctx := context.Background()

		aeo := options.AutoEncryption().SetKmsProviders(kmsProviders)

		cse := setup(mt, aeo, nil, nil)
		fmt.Println(cse)

		cse.cseColl.FindOne(ctx, bson.D{{"x", 1}})

		for _, event := range cse.cseStarted {
			fmt.Printf("event: %+v\n", event)
		}

		fmt.Println("where the test ends -----------------------------")
	})
}
