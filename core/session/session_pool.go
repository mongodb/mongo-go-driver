package session

import (
	"sync"

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
	DescChannel    <-chan description.Topology // channel to read topology descriptions to update the session timeout
	Head           *Node
	Tail           *Node
	SessionTimeout uint32
	Mutex          sync.Mutex // mutex to protect list and SessionTimeout
}

// NewPool creates a new server session pool
func NewPool(descChan <-chan description.Topology) *Pool {
	p := &Pool{
		DescChannel: descChan,
	}

	go p.update()
	return p
}

// GetSession retrieves an unexpired session from the pool.
func (p *Pool) GetSession() (*Server, error) {
	p.Mutex.Lock() // prevent changing SessionTimeout while seeing if sessions have expired
	defer p.Mutex.Unlock()

	// empty pool
	if p.Head == nil && p.Tail == nil {
		return newServerSession()
	}

	for p.Head != nil {
		// pull session from head of queue and return if it is valid for at least 1 more minute
		if p.Head.expired(p.SessionTimeout) {
			p.Head.endSession()
			p.Head = p.Head.next
			continue
		}

		// found unexpired session
		session := p.Head.Server
		if p.Head.next != nil {
			p.Head.next.prev = nil
		}
		p.Head = p.Head.next
		return session, nil
	}

	// no valid session found
	p.Tail = nil // empty list
	return newServerSession()
}

// ReturnSession returns a session to the pool if it has not expired.
func (p *Pool) ReturnSession(ss *Server) {
	p.Mutex.Lock()
	defer p.Mutex.Unlock()

	// check sessions at end of queue for expired
	// stop checking after hitting the first valid session
	for p.Tail != nil && p.Tail.expired(p.SessionTimeout) {
		p.Tail.endSession()
		if p.Tail.prev != nil {
			p.Tail.prev.next = nil
		}
		p.Tail = p.Tail.prev
	}

	// session expired
	if ss.expired(p.SessionTimeout) {
		ss.endSession()
		return
	}

	newNode := &Node{
		Server: ss,
		next:   nil,
		prev:   nil,
	}

	// empty list
	if p.Tail == nil {
		p.Head = newNode
		p.Tail = newNode
		return
	}

	// at least 1 valid session in list
	newNode.next = p.Head
	p.Head.prev = newNode
	p.Head = newNode
}

// Iterator creates an iterator for this session.
func (p *Pool) Iterator() *Iterator {
	return newIterator(p)
}

func (p *Pool) update() {
	for {
		select {
		case desc := <-p.DescChannel:
			p.Mutex.Lock()
			p.SessionTimeout = desc.SessionTimeoutMinutes
			p.Mutex.Unlock()
		default:
			// no new description waiting --> no update
		}
	}
}

func (p *Pool) String() string {
	s := ""

	for head := p.Head; head != nil; head = head.next {
		s += head.SessionID.String() + "\n"
	}

	return s
}
