package driverlegacy

import (
	"testing"
)

func TestBatchCursor(t *testing.T) {
	t.Run("Does not panic if context is nil", func(t *testing.T) {
		// all collection/cursor iterators should take contexts, but
		// permit passing nils for contexts, which should not
		// panic.
		//
		// While more through testing might be ideal this check
		// prevents a regression of GODRIVER-298

		c := &BatchCursor{}

		defer func() {
			if err := recover(); err != nil {
				t.Errorf("Expected cursor to not panic with nil context, but got error: %v", err)
			}
		}()
		if c.Next(nil) {
			t.Errorf("Expect next to return false, but returned true")
		}
	})
}
