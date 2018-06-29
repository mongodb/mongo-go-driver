package session

// Iterator iterates over a Pool.
type Iterator struct {
	node       *Node
	serverSess *Server
}

func newIterator(p *Pool) *Iterator {
	var node *Node
	if p != nil {
		node = p.Head
	}

	return &Iterator{
		node: node,
	}
}

// Next fetches the next server session in the pool, returning whether or not the next session could be fetched. If true
// is returned, call Element to get the server session. If false is returned, there are no more sessions remaining.
func (i *Iterator) Next() bool {
	if i.node == nil {
		return false
	}

	i.serverSess = i.node.Server
	i.node = i.node.next
	return true
}

// Element returns the current session of the Iterator.
func (i *Iterator) Element() *Server {
	return i.serverSess
}
