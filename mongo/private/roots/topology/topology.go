// Package topology contains types that handles the discovery, monitoring, and selection
// of servers. This package is designed to expose enough inner workings of service discovery
// and monitoring to allow low level applications to have fine grained control, while hiding
// most of the detailed implementation of the algorithms.
package topology

import (
	"context"
	"errors"
	"math/rand"
	"sync"
	"time"

	"github.com/mongodb/mongo-go-driver/mongo/private/roots/addr"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/description"
	"github.com/mongodb/mongo-go-driver/mongo/readpref"
)

// ErrSubscribeAfterClosed is returned when a user attempts to subscribe to a
// closed Server or Topology.
var ErrSubscribeAfterClosed = errors.New("cannot subscribe after close")

// ErrTopologyClosed is returned when a user attempts to call a method on a
// closed Topology.
var ErrTopologyClosed = errors.New("topology is closed")

// ErrServerSelectionTimeout is returned from server selection when the server
// selection process took longer than allowed by the timeout.
var ErrServerSelectionTimeout = errors.New("server selection timeout")

type MonitorMode uint8

const (
	AutomaticMode MonitorMode = iota
	SingleMode
)

// Topology respresents a MongoDB deployment.
type Topology struct {
	// There are too many closed booleans, but we can fix that later
	// with a refactor. For now, just making a definitive one to guard
	// the Close method.
	closed      bool
	initialized bool
	l           sync.Mutex

	cfg *config

	desc description.Topology
	dmtx sync.Mutex

	fsm     *fsm
	changes chan description.Server

	subscribers         map[uint64]chan description.Topology
	currentSubscriberID uint64
	subscriptionsClosed bool
	subLock             sync.Mutex

	serversLock   sync.Mutex
	serversClosed bool
	servers       map[addr.Addr]*Server

	wg sync.WaitGroup
}

// New creates a new topology.
func New(opts ...Option) (*Topology, error) {
	cfg, err := newConfig(opts...)
	if err != nil {
		return nil, err
	}

	t := &Topology{
		cfg:         cfg,
		fsm:         newFSM(),
		changes:     make(chan description.Server),
		subscribers: make(map[uint64]chan description.Topology),
		servers:     make(map[addr.Addr]*Server),
	}

	if cfg.replicaSetName != "" {
		t.fsm.SetName = cfg.replicaSetName
		t.fsm.Kind = description.ReplicaSetNoPrimary
	}

	if cfg.mode == SingleMode {
		t.fsm.Kind = description.Single
	}

	return t, nil
}

func (t *Topology) Init() {
	t.l.Lock()
	defer t.l.Unlock()
	if t.initialized {
		return
	}
	t.initialized = true

	t.serversLock.Lock()
	for _, a := range t.cfg.seedList {
		address := addr.Addr(a).Canonicalize()
		t.fsm.Servers = append(t.fsm.Servers, description.Server{Addr: address})
		t.addServer(address)
	}
	t.serversLock.Unlock()

	go t.update()
}

// Close closes the topology. It stops the monitoring thread and
// closes all open subscriptions.
func (t *Topology) Close() error {
	t.l.Lock()
	defer t.l.Unlock()
	if t.closed {
		return nil
	}
	t.closed = true

	t.serversLock.Lock()
	t.serversClosed = true
	for address, server := range t.servers {
		t.removeServer(address, server)
	}
	t.serversLock.Unlock()

	t.wg.Wait()

	t.dmtx.Lock()
	t.desc = description.Topology{}
	t.dmtx.Unlock()

	return nil
}

// Description returns a description of the topology.
func (t *Topology) Description() description.Topology {
	t.dmtx.Lock()
	defer t.dmtx.Unlock()
	return t.desc
}

// Subscribe returns a Subscription on which all updated description.Topologys
// will be sent. The channel of the subscription will have a buffer size of one,
// and will be pre-populated with the current description.Topology.
func (t *Topology) Subscribe() (*Subscription, error) {
	ch := make(chan description.Topology, 1)
	t.dmtx.Lock()
	ch <- t.desc
	t.dmtx.Unlock()

	t.subLock.Lock()
	defer t.subLock.Unlock()
	if t.subscriptionsClosed {
		return nil, ErrSubscribeAfterClosed
	}
	id := t.currentSubscriberID
	t.subscribers[id] = ch
	t.currentSubscriberID++

	return &Subscription{
		C:  ch,
		t:  t,
		id: id,
	}, nil
}

// RequestImmediateCheck will send heartbeats to all the servers in the
// topology right away, instead of waiting for the heartbeat timeout.
func (t *Topology) RequestImmediateCheck() {
	t.serversLock.Lock()
	for _, server := range t.servers {
		server.RequestImmediateCheck()
	}
	t.serversLock.Unlock()
}

// SelectServer selects a server given a selector.SelectServer complies with the
// server selection spec, and will time out after severSelectionTimeout or when the
// parent context is done.
func (t *Topology) SelectServer(ctx context.Context, ss ServerSelector, rp *readpref.ReadPref) (*SelectedServer, error) {
	var ssTimeoutCh <-chan time.Time

	if t.cfg.serverSelectionTimeout > 0 {
		ssTimeout := time.NewTimer(t.cfg.serverSelectionTimeout)
		ssTimeoutCh = ssTimeout.C
	}

	sub, err := t.Subscribe()
	if err != nil {
		return nil, err
	}
	defer sub.Unsubscribe()

	for {
		suitable, err := t.selectServer(ctx, sub.C, ss, ssTimeoutCh)
		if err != nil {
			return nil, err
		}

		selected := suitable[rand.Intn(len(suitable))]
		selectedS, err := t.findServer(selected, rp)
		switch {
		case err != nil:
			return nil, err
		case selectedS != nil:
			return selectedS, nil
		default:
			// We don't have an actual server for the provided description.
			// This could happen for a number of reasons, including that the
			// server has since stopped being a part of this topology, or that
			// the server selector returned no suitable servers.
		}
	}
}

// findServer will attempt to find a server that fits the given server description.
// This method will return nil, nil if a matching server could not be found.
func (t *Topology) findServer(selected description.Server, rp *readpref.ReadPref) (*SelectedServer, error) {
	t.l.Lock()
	defer t.l.Unlock()
	if t.closed {
		return nil, ErrTopologyClosed
	}
	server, ok := t.servers[selected.Addr]
	if !ok {
		return nil, nil
	}

	return &SelectedServer{
		Server:   server,
		ReadPref: rp,
	}, nil
}

// selectServer is the core piece of server selection. It handles getting
// topology descriptions and running sever selection on those descriptions.
func (t *Topology) selectServer(ctx context.Context, subscriptionCh <-chan description.Topology, ss ServerSelector, timeoutCh <-chan time.Time) ([]description.Server, error) {
	var current description.Topology
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timeoutCh:
			return nil, ErrServerSelectionTimeout
		case current = <-subscriptionCh:
		}

		var allowed []description.Server
		for _, s := range current.Servers {
			if s.Kind != description.Unknown {
				allowed = append(allowed, s)
			}
		}

		suitable, err := ss.SelectServer(current, allowed)
		if err != nil {
			return nil, err
		}

		if len(suitable) > 0 {
			return suitable, nil
		}

		t.RequestImmediateCheck()
	}
}

func (t *Topology) update() {
	defer func() {
		//  ¯\_(ツ)_/¯
		recover()
	}()

	for change := range t.changes {
		current, err := t.apply(change)
		if err != nil {
			continue
		}

		t.dmtx.Lock()
		t.desc = current
		t.dmtx.Unlock()

		t.subLock.Lock()
		for _, ch := range t.subscribers {
			// We drain the description if there's one in the channel
			select {
			case <-ch:
			default:
			}
			ch <- current
		}
		t.subLock.Unlock()
	}
	t.subLock.Lock()
	for id, ch := range t.subscribers {
		close(ch)
		delete(t.subscribers, id)
	}
	t.subscriptionsClosed = true
	t.subLock.Unlock()
}

func (t *Topology) apply(desc description.Server) (description.Topology, error) {
	var err error
	prev := t.fsm.Topology

	current, err := t.fsm.apply(desc)
	if err != nil {
		return description.Topology{}, err
	}

	diff := description.DiffTopology(prev, current)
	t.serversLock.Lock()
	if t.serversClosed {
		t.serversLock.Unlock()
		return description.Topology{}, nil
	}

	for _, removed := range diff.Removed {
		if s, ok := t.servers[removed.Addr]; ok {
			t.removeServer(removed.Addr, s)
		}
	}

	for _, added := range diff.Added {
		t.addServer(added.Addr)
	}
	t.serversLock.Unlock()
	return current, nil
}

func (t *Topology) addServer(address addr.Addr) {
	if _, ok := t.servers[address]; ok {
		return
	}

	svr, err := NewServer(address, t.cfg.serverOpts...)
	if err != nil {
		//  ¯\_(ツ)_/¯
		return
	}

	t.servers[address] = svr
	var sub *ServerSubscription
	sub, err = svr.Subscribe()
	if err != nil {
		// ¯\_(ツ)_/¯
		return
	}

	t.wg.Add(1)
	go func() {
		for c := range sub.C {
			t.changes <- c
		}
		t.wg.Done()
	}()
}

func (t *Topology) removeServer(address addr.Addr, server *Server) {
	server.Close()
	delete(t.servers, address)
}

// Subscription is a subscription to updates to the description of the Topology that created this
// Subscription.
type Subscription struct {
	C  <-chan description.Topology
	t  *Topology
	id uint64
}

// Unsubscribe unsubscribes this Subscription from updates and closes the
// subscription channel.
func (s *Subscription) Unsubscribe() error {
	s.t.subLock.Lock()
	defer s.t.subLock.Unlock()
	if s.t.subscriptionsClosed {
		return nil
	}

	ch, ok := s.t.subscribers[s.id]
	if !ok {
		return nil
	}

	close(ch)
	delete(s.t.subscribers, s.id)

	return nil
}
