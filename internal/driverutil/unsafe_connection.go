package driverutil

import (
	"context"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"go.mongodb.org/mongo-driver/mongo/address"
	"go.mongodb.org/mongo-driver/mongo/description"
	"go.mongodb.org/mongo-driver/x/mongo/driver/wiremessage"
)

var globalConnectionID uint64 = 1

// Connection state constants.
const (
	ConnDisconnected int64 = iota
	ConnConnected
	ConnInitialized
)

type Pool interface {
	GetState() int
	Stale(*UnsafeConnection) bool
}

// UnsafeConnection is holds concurrent-unsafe logic for making a connection to
// a server.
//
// This type is seperated from mnet.Connection for non-concurrent use to avoid
// lock contention.
type UnsafeConnection struct {
	nc         net.Conn // When nil, the connection is closed.
	compressor wiremessage.CompressorID
	zliblevel  int
	zstdLevel  int
	//config               *UnsafeConnectionConfig
	connectContextMade  chan struct{}
	canStream           bool
	currentlyStreaming  bool
	connectContextMutex sync.Mutex
	//cancellationListener cancellationListener

	// state must be accessed using the atomic package and should be at the
	// beginning of the struct.
	//
	// - atomic bug: https://pkg.go.dev/sync/atomic#pkg-note-BUG
	// - suggested layout: https://go101.org/article/memory-layout.html
	State int64

	CancelConnectContext context.CancelFunc
	ConnectDone          chan struct{}
	ConnectTimeout       time.Duration
	Desc                 description.Server
	DriverConnectionID   int64
	Generation           uint64
	HelloRTT             time.Duration
	ID                   string
	IdleTimeout          time.Duration
	IdleDeadline         atomic.Value
	Pool                 Pool
	ReadTimeout          time.Duration
	ServerConnectionID   *int64 // the server's ID for this client's connection
	WriteTimeout         time.Duration
}

func nextConnectionID() uint64 {
	return atomic.AddUint64(&globalConnectionID, 1)
}

func NewUnsafeConnection(addr address.Address, opts *UnsafeConnectionOptions) *UnsafeConnection {
	id := fmt.Sprintf("%s[-%d]", addr, nextConnectionID())

	c := &UnsafeConnection{
		ID: id,
		//	IdleTimeout: opts.IdleTimeout,
		//ReadTimeout:  opts.ReadTimeout,
		//WriteTimeout: opts.WriteTimeout,
		//ConnectDone:  make(chan struct{}),
	}
	// Connections to non-load balanced deployments should eagerly set the generation numbers so errors encountered
	// at any point during connection establishment can be processed without the connection being considered stale.
	if !opts.LoadBalanced {
		c.SetGenerationNumber()
	}

	atomic.StoreInt64(&c.State, ConnInitialized)

	return c
}

// setGenerationNumber sets the connection's generation number if a callback has
// been provided to do so in connection configuration.
func (uconn *UnsafeConnection) SetGenerationNumber() {}

// HasGenerationNumber returns true if the connection has set its generation
// number. If so, this indicates that the generationNumberFn provided via the
// connection options has been called exactly once.
func (uconn *UnsafeConnection) HasGenerationNumber() bool {
	// TODO

	return false
}

// Connect handles the I/O for a connection. It will dial, configure TLS, and
// perform initialization handshakes. All errors returned by connect are
// considered "before the handshake completes" and must be handled by calling
// the appropriate SDAM handshake error handler.
func (uconn *UnsafeConnection) Connect(ctx context.Context) (err error) {
	// TODO

	return nil
}

func (uconn *UnsafeConnection) Wait() {
	// TODO
}

func (uconn *UnsafeConnection) CloseConnectContext() {
	// TODO
}

func (uconn *UnsafeConnection) cancellationListenerCallback() {
	// TODO
}

func (uconn *UnsafeConnection) WriteWireMessage(uconntx context.Context, wm []byte) error { return nil }

func (uconn *UnsafeConnection) write(uconntx context.Context, wm []byte) (err error) {
	// TODO

	return nil
}

// readWireMessage reads a wiremessage from the connection. The dst parameter
// will be overwritten.
func (uconn *UnsafeConnection) readWireMessage(uconntx context.Context) ([]byte, error) {
	// TODO

	return nil, nil
}

func (uconn *UnsafeConnection) read(uconntx context.Context) (bytesRead []byte, errMsg string, err error) {
	// TODO

	return nil, "", nil
}

func (uconn *UnsafeConnection) Close() error {
	// TODO

	return nil
}

func (uconn *UnsafeConnection) Closed() bool {
	// TODO

	return false
}

func (uconn *UnsafeConnection) IdleTimeoutExpired() bool {
	// TODO

	return false
}

func (uconn *UnsafeConnection) BumpIdleDeadline() {
	// TODO
}

func (uconn *UnsafeConnection) Streamable(bool) {
	// TODO
}

func (uconn *UnsafeConnection) setStreaming(bool) {
	// TODO
}

func (uconn *UnsafeConnection) IsStreaming() bool {
	// TODO, renamed from getCurrentlyStreaming

	return false
}

func (uconn *UnsafeConnection) SetSocketTimeout(timeout time.Duration) {
	// TODO
}

//func (uconn *UnsafeConnection) ID() string {
//	// TODO
//
//	return ""
//}
