package session

import (
	"testing"

	"github.com/mongodb/mongo-go-driver/core/description"
)

func TestSessionPool(t *testing.T) {
	t.Run("TestLifo", func(t *testing.T) {
		descChan := make(chan description.Topology)
		p := NewPool(descChan)

		first := p.GetSession()
		firstID := first.SessionID

		second := p.GetSession()
		secondID := second.SessionID

		p.ReturnSession(first)
		p.ReturnSession(second)

		sess := p.GetSession()
		nextSess := p.GetSession()

		if sess.SessionID != secondID {
			t.Errorf("first sesssion ID mismatch. got %s expected %s", sess.SessionID, secondID)
		}

		if nextSess.SessionID != firstID {
			t.Errorf("second sesssion ID mismatch. got %s expected %s", nextSess.SessionID, firstID)
		}
	})
}
