package topology

import (
	"context"
	"errors"
	"math"
	"sync"
	"time"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/mongo/private/options"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/addr"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/command"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/connection"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/description"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/result"
)

const minHeartbeatInterval = 500 * time.Millisecond

// ErrServerClosed occurs when an attempt to get a connection is made after
// the server has been closed.
var ErrServerClosed = errors.New("server is closed")

// Server is a single server within a topology.
type Server struct {
	cfg     *serverConfig
	address addr.Addr

	l        sync.Mutex
	closed   bool
	done     chan struct{}
	checkNow chan struct{}
	pool     connection.Pool

	desc description.Server
	dmtx sync.RWMutex

	averageRTTSet bool
	averageRTT    time.Duration

	subLock             sync.Mutex
	subscribers         map[uint64]chan description.Server
	currentSubscriberID uint64
	subscriptionsClosed bool
}

// NewServer creates a new server. The mongodb server at the address will be monitored
// on an internal monitoring goroutine.
func NewServer(address addr.Addr, opts ...ServerOption) (*Server, error) {
	cfg, err := newServerConfig(opts...)
	if err != nil {
		return nil, err
	}

	s := &Server{
		cfg:     cfg,
		address: address,

		done:     make(chan struct{}),
		checkNow: make(chan struct{}, 1),

		desc: description.Server{Addr: address},

		subscribers: make(map[uint64]chan description.Server),
	}

	var maxConns uint64
	if cfg.maxConns == 0 {
		maxConns = math.MaxInt64
	} else {
		maxConns = uint64(cfg.maxConns)
	}

	// TODO(skriptble): Add a configurer here that will take any newly dialed connections for this pool
	// and put their server descriptions through the fsm.
	s.pool, err = connection.NewPool(address, uint64(cfg.maxIdleConns), maxConns, cfg.connectionOpts...)
	if err != nil {
		return nil, err
	}

	go s.update()

	return s, nil
}

// Close closes the server.
func (s *Server) Close() error {
	s.l.Lock()
	defer s.l.Unlock()

	if s.closed {
		return nil
	}
	s.closed = true

	close(s.done)
	err := s.pool.Close()
	if err != nil {
		return err
	}

	return nil
}

// Connection gets a connection to the server.
func (s *Server) Connection(ctx context.Context) (connection.Connection, error) {
	conn, err := s.pool.Get(ctx)
	if err != nil {
		return nil, err
	}
	return &sconn{Connection: conn, s: s}, nil
}

// Description returns a description of the server as of the last heartbeat.
func (s *Server) Description() description.Server {
	s.dmtx.RLock()
	defer s.dmtx.RUnlock()
	return s.desc
}

// SelectedDescription returns a description.SelectedServer with a Kind of
// Single. This can be used when performing tasks like monitoring a batch
// of servers and you want to run one off commands against those servers.
func (s *Server) SelectedDescription() description.SelectedServer {
	sdesc := s.Description()
	return description.SelectedServer{
		Server: sdesc,
		Kind:   description.Single,
	}
}

// Subscribe returns a ServerSubscription which has a channel on which all
// updated server descriptions will be sent. The channel will have a buffer
// size of one, and will be pre-populated with the current description.
func (s *Server) Subscribe() (*ServerSubscription, error) {
	ch := make(chan description.Server, 1)
	s.dmtx.Lock()
	defer s.dmtx.Unlock()
	ch <- s.desc

	s.subLock.Lock()
	defer s.subLock.Unlock()
	if s.subscriptionsClosed {
		return nil, ErrSubscribeAfterClosed
	}
	id := s.currentSubscriberID
	s.subscribers[id] = ch
	s.currentSubscriberID++

	ss := &ServerSubscription{
		C:  ch,
		s:  s,
		id: id,
	}

	return ss, nil
}

// RequestImmediateCheck will cause the server to send a heartbeat immediately
// instead of waiting for the heartbeat timeout.
func (s *Server) RequestImmediateCheck() {
	select {
	case s.checkNow <- struct{}{}:
	default:
	}
}

// update handles performing heartbeats and updating any subscribers of the
// newest description.Server retrieved.
func (s *Server) update() {
	defer func() {
		// TODO(skriptble): What should we do here?
		_ = recover()
	}()
	heartbeatTicker := time.NewTicker(s.cfg.heartbeatInterval)
	rateLimiter := time.NewTicker(minHeartbeatInterval)
	checkNow := s.checkNow
	done := s.done

	var conn connection.Connection
	var desc description.Server

	desc, conn = s.heartbeat(nil)
	s.updateDescription(desc, true)

	for {
		select {
		case <-heartbeatTicker.C:
		case <-checkNow:
		case <-done:
			s.subLock.Lock()
			for id, c := range s.subscribers {
				close(c)
				delete(s.subscribers, id)
			}
			s.subscriptionsClosed = true
			s.subLock.Unlock()
			conn.Close()
			return
		}

		<-rateLimiter.C

		desc, conn = s.heartbeat(conn)
		s.updateDescription(desc, false)
	}
}

// updateDescription handles updating the description on the Server, notifying
// subscribers, and potentially draining the connection pool. The initial
// parameter is used to determine if this is the first description from the
// server.
func (s *Server) updateDescription(desc description.Server, initial bool) {
	s.dmtx.Lock()
	s.desc = desc
	s.dmtx.Unlock()

	s.subLock.Lock()
	for _, c := range s.subscribers {
		select {
		// drain the channel if it isn't empty
		case <-c:
		default:
		}
		c <- desc
	}
	s.subLock.Unlock()

	if initial {
		// We don't clear the pool on the first update on the description.
		return
	}

	switch desc.Kind {
	case description.Unknown:
		_ = s.pool.Drain()
	}
}

// heartbeat sends a heartbeat to the server using the given connection. The connection can be nil.
func (s *Server) heartbeat(conn connection.Connection) (description.Server, connection.Connection) {
	const maxRetry = 2
	var saved error
	var desc description.Server
	var set bool
	var err error
	ctx := context.Background()
	for i := 1; i <= maxRetry; i++ {
		if conn != nil && conn.Expired() {
			conn.Close()
			conn = nil
		}

		if conn == nil {
			opts := []connection.Option{
				connection.WithConnectTimeout(func(time.Duration) time.Duration { return s.cfg.heartbeatTimeout }),
				connection.WithReadTimeout(func(time.Duration) time.Duration { return s.cfg.heartbeatTimeout }),
			}
			opts = append(opts, s.cfg.connectionOpts...)
			conn, err = connection.New(ctx, s.address, opts...)
			if err != nil {
				saved = err
				if conn != nil {
					conn.Close()
				}
				conn = nil
				continue
			}
		}

		now := time.Now()
		isMaster, err := (&command.IsMaster{}).RoundTrip(ctx, conn)
		if err != nil {
			saved = err
			conn.Close()
			conn = nil
			continue
		}
		delay := time.Since(now)

		desc = description.NewServer(s.address, isMaster, result.BuildInfo{}).SetAverageRTT(s.updateAverageRTT(delay))
		desc.HeartbeatInterval = s.cfg.heartbeatInterval
		set = true

		break
	}

	if !set {
		desc = description.Server{
			Addr:      s.address,
			LastError: saved,
		}
	}

	return desc, conn
}

func (s *Server) updateAverageRTT(delay time.Duration) time.Duration {
	if !s.averageRTTSet {
		s.averageRTT = delay
	} else {
		alpha := 0.2
		s.averageRTT = time.Duration(alpha*float64(delay) + (1-alpha)*float64(s.averageRTT))
	}
	return s.averageRTT
}

// Drain will drain the connection pool of this server. This is mainly here so the
// pool for the server doesn't need to be directly exposed and so that when an error
// is returned from reading or writing, a client can drain the pool for this server.
// This is exposed here so we don't have to wrap the Connection type and sniff responses
// for errors that would cause the pool to be drained, which can in turn centralize the
// logic for handling errors in the Client type.
func (s *Server) Drain() error { return s.pool.Drain() }

// BuildCursor implements the command.CursorBuilder interface for the Server type.
func (s *Server) BuildCursor(result bson.Reader, opts ...options.CursorOptioner) (command.Cursor, error) {
	return newCursor(result, s, opts...)
}

// ServerSubscription represents a subscription to the description.Server updates for
// a specific server.
type ServerSubscription struct {
	C  <-chan description.Server
	s  *Server
	id uint64
}

// Unsubscribe unsubscribes this ServerSubscription from updates and closes the
// subscription channel.
func (ss *ServerSubscription) Unsubscribe() error {
	ss.s.subLock.Lock()
	defer ss.s.subLock.Unlock()
	if ss.s.subscriptionsClosed {
		return nil
	}

	ch, ok := ss.s.subscribers[ss.id]
	if !ok {
		return nil
	}

	close(ch)
	delete(ss.s.subscribers, ss.id)

	return nil
}
