package session

import (
	"testing"

	"github.com/mongodb/mongo-go-driver/core/description"
	"github.com/mongodb/mongo-go-driver/internal/testutil/helpers"
)

func TestSessionPool(t *testing.T) {
	t.Run("TestLifo", func(t *testing.T) {
		descChan := make(chan description.Topology)
		p := NewPool(descChan)
		p.timeout = 30 // Set to some arbitrarily high number greater than 1 minute.

		first, err := p.GetSession()
		testhelpers.RequireNil(t, err, "error getting session", err)
		firstID := first.SessionID

		second, err := p.GetSession()
		testhelpers.RequireNil(t, err, "error getting session", err)
		secondID := second.SessionID

		p.ReturnSession(first)
		p.ReturnSession(second)

		sess, err := p.GetSession()
		testhelpers.RequireNil(t, err, "error getting session", err)
		nextSess, err := p.GetSession()
		testhelpers.RequireNil(t, err, "error getting session", err)

		if sess.SessionID != secondID {
			t.Errorf("first sesssion ID mismatch. got %s expected %s", sess.SessionID, secondID)
		}

		if nextSess.SessionID != firstID {
			t.Errorf("second sesssion ID mismatch. got %s expected %s", nextSess.SessionID, firstID)
		}
	})

	t.Run("TestExpiredRemoved", func(t *testing.T) {
		descChan := make(chan description.Topology)
		p := NewPool(descChan)
		// New sessions will always become stale when returned
		p.timeout = 0

		first, err := p.GetSession()
		testhelpers.RequireNil(t, err, "error getting session", err)
		firstID := first.SessionID

		second, err := p.GetSession()
		testhelpers.RequireNil(t, err, "error getting session", err)
		secondID := second.SessionID

		p.ReturnSession(first)
		p.ReturnSession(second)

		sess, err := p.GetSession()
		testhelpers.RequireNil(t, err, "error getting session", err)

		if sess.SessionID == firstID || sess.SessionID == secondID {
			t.Errorf("Expired sessions not removed!")
		}
	})
}
