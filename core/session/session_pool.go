package session

import (
	"sync"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/core/description"
)

// Node represents a server session in a linked list
type Node struct {
	*Server
	next *Node
	prev *Node
}

// Pool is a pool of server sessions that can be reused.
type Pool struct {
	descChannel    <-chan description.Topology // channel to read topology descriptions to update the session timeout
	head           *Node
	tail           *Node
	sessionTimeout uint32
	mutex          sync.Mutex // mutex to protect list and sessionTimeout
}

// NewPool creates a new server session pool
func NewPool(descChan <-chan description.Topology) *Pool {
	p := &Pool{
		descChannel: descChan,
	}

	// Make sure we guarantee the latest timeout before passing this into goroutine
	p.updateSessionTimeout()
	go p.update()
	return p
}

// GetSession retrieves an unexpired session from the pool.
func (p *Pool) GetSession() (*Server, error) {
	p.mutex.Lock() // prevent changing the linked list while seeing if sessions have expired
	defer p.mutex.Unlock()

	// empty pool
	if p.head == nil && p.tail == nil {
		return newServerSession()
	}

	for p.head != nil {
		// pull session from head of queue and return if it is valid for at least 1 more minute
		if p.head.expired(p.sessionTimeout) {
			p.head = p.head.next
			continue
		}

		// found unexpired session
		session := p.head.Server
		if p.head.next != nil {
			p.head.next.prev = nil
		}
		if p.tail == p.head {
			p.tail = nil
			p.head = nil
		} else {
			p.head = p.head.next
		}
		return session, nil
	}

	// no valid session found
	p.tail = nil // empty list
	return newServerSession()
}

// ReturnSession returns a session to the pool if it has not expired.
func (p *Pool) ReturnSession(ss *Server) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	// check sessions at end of queue for expired
	// stop checking after hitting the first valid session
	for p.tail != nil && p.tail.expired(p.sessionTimeout) {
		if p.tail.prev != nil {
			p.tail.prev.next = nil
		}
		p.tail = p.tail.prev
	}

	// session expired
	if ss.expired(p.sessionTimeout) {
		return
	}

	newNode := &Node{
		Server: ss,
		next:   nil,
		prev:   nil,
	}

	// empty list
	if p.tail == nil {
		p.head = newNode
		p.tail = newNode
		return
	}

	// at least 1 valid session in list
	newNode.next = p.head
	p.head.prev = newNode
	p.head = newNode
}

// IDSlice returns a slice of session IDs for each session in the pool
func (p *Pool) IDSlice() []*bson.Document {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	ids := []*bson.Document{}
	for node := p.head; node != nil; node = node.next {
		ids = append(ids, node.SessionID)
	}

	return ids
}

func (p *Pool) update() {
	for {
		p.updateSessionTimeout()
	}
}

func (p *Pool) updateSessionTimeout() {
	select {
	case desc := <-p.descChannel:
		p.mutex.Lock()
		p.sessionTimeout = desc.SessionTimeoutMinutes
		p.mutex.Unlock()
	default:
		// no new description waiting --> no update
	}
}

// String implements the Stringer interface
func (p *Pool) String() string {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	s := ""
	for head := p.head; head != nil; head = head.next {
		s += head.SessionID.String() + "\n"
	}

	return s
}
