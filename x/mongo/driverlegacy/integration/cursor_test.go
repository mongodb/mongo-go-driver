package integration

import (
	"testing"
)

func TestBatchCursor(t *testing.T) {
	// t.Run("Next", func(t *testing.T) {
	// 	t.Run("Returns false on cancelled context", func(t *testing.T) {
	// 		// Next should return false if an error occurs
	// 		// here the error is the Context being cancelled
	//
	// 		s := createDefaultConnectedServer(t, false)
	// 		c := cursor{
	// 			id:     1,
	// 			batch:  []bson.RawValue{},
	// 			server: s,
	// 		}
	//
	// 		ctx, cancel := context.WithCancel(context.Background())
	//
	// 		cancel()
	//
	// 		assert.False(t, c.Next(ctx))
	// 	})
	// 	t.Run("Returns false if error occurred", func(t *testing.T) {
	// 		// Next should return false if an error occurs
	// 		// here the error is an invalid namespace (""."")
	//
	// 		s := createDefaultConnectedServer(t, true)
	// 		c := cursor{
	// 			id:     1,
	// 			batch:  []bson.RawValue{},
	// 			server: s,
	// 		}
	// 		assert.False(t, c.Next(nil))
	// 	})
	// 	t.Run("Returns false if cursor ID is zero", func(t *testing.T) {
	// 		// Next should return false if the cursor id is 0 and there are no documents in the next batch
	//
	// 		c := cursor{id: 0, batch: []bson.RawValue{}}
	// 		assert.False(t, c.Next(nil))
	// 	})
	// })
}
