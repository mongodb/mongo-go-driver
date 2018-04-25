package topology

import (
	"testing"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/stretchr/testify/assert"
)

func TestCursorNextDoesNotPanicIfContextisNil(t *testing.T) {
	// all collection/cursor iterators should take contexts, but
	// permit passing nils for contexts, which should not
	// panic.
	//
	// While more through testing might be ideal this check
	// prevents a regression of GODRIVER-298

	c := cursor{batch: bson.NewArray(bson.VC.String("a"), bson.VC.String("b"))}

	var iterNext bool
	assert.NotPanics(t, func() {
		iterNext = c.Next(nil)
	})
	assert.True(t, iterNext)
}
