package connection

import (
	"context"
	"testing"

	"github.com/mongodb/mongo-go-driver/mongo/private/roots/addr"
)

func TestConnection(t *testing.T) {
	t.Run("New", func(t *testing.T) {
		address := addr.Addr("localhost:27017")
		_, err := New(context.Background(), address)
		if err != nil {
			t.Errorf("Whoops: %s", err)
		}
	})
}
