package uuid

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	m := make(map[UUID]bool)
	for i := 1; i < 100; i++ {
		uuid, err := New()
		assert.NoError(t, err, "New error")
		assert.False(t, m[uuid], "New returned a duplicate UUID %v", uuid)
		m[uuid] = true
	}
}
