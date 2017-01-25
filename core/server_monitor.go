package core

import (
	"fmt"

	"github.com/craiggwilson/mongo-go-driver/core/util"

	"sync"
	"time"
)

// StartServerMonitor returns a new ServerMonitor containing a channel
// that will send a ServerDesc everytime it is updated.
func StartServerMonitor(opts ServerOptions) (*ServerMonitor, error) {
	if err := opts.validate(); err != nil {
		return nil, err
	}

	opts.fillDefaults()

	// TODO: should really be using a ring buffer. We want
	// to throw away the oldest data, not the newest.
	c := make(chan *ServerDesc, 1)
	done := make(chan struct{}, 1)
	m := &ServerMonitor{
		C:              c,
		done:           done,
		connectionOpts: opts.ConnectionOptions,
	}

	go func() {
		timer := time.NewTimer(0)
		for {
			select {
			case <-timer.C:
				// weird syntax for a non-blocking send...
				desc := m.heartbeat()
				m.descLock.Lock()
				m.desc = desc
				m.descLock.Unlock()
				select {
				case c <- desc:
				default:
					// TODO: drain the channel to make the next
					// write visible
				}
				timer.Stop()
				timer.Reset(opts.HeartbeatInterval)
			case <-done:
				timer.Stop()
				close(c)
				return
			}
		}
	}()

	return m, nil
}

// ServerMonitor holds a channel that delivers updates to a server.
type ServerMonitor struct {
	C <-chan *ServerDesc // The channel on which the updates are delivered.

	conn           ConnectionCloser
	connectionOpts ConnectionOptions
	desc           *ServerDesc
	descLock       sync.Mutex
	done           chan<- struct{}
}

// Desc gets the current ServerDesc.
func (m *ServerMonitor) Desc() *ServerDesc {
	m.descLock.Lock()
	desc := m.desc
	m.descLock.Unlock()
	return desc
}

// Stop turns off the monitor.
func (m *ServerMonitor) Stop() {
	close(m.done)
}

func (m *ServerMonitor) heartbeat() *ServerDesc {
	const maxRetryCount = 2
	var savedErr error
	var desc *ServerDesc
	for i := 1; i <= maxRetryCount; i++ {
		if m.conn == nil {
			conn, err := DialConnection(m.connectionOpts)
			if err != nil {
				savedErr = err
				if conn != nil {
					conn.Close()
				}
				m.conn = nil
				continue
			}
			m.conn = conn
		}

		now := time.Now()
		isMasterResult, buildInfoResult, err := describeServer(m.conn)
		if err != nil {
			savedErr = err
			m.conn.Close()
			m.conn = nil
			continue
		}
		delay := time.Since(now)

		desc = buildServerDesc(m.connectionOpts.Endpoint, isMasterResult, buildInfoResult)
		desc.averageRTT = delay // TODO: actually calculate this properly
	}

	if desc == nil {
		desc = &ServerDesc{
			endpoint:  m.connectionOpts.Endpoint,
			lastError: savedErr,
		}
	}

	return desc
}

func buildServerDesc(endpoint Endpoint, isMasterResult *isMasterResult, buildInfoResult *buildInfoResult) *ServerDesc {
	desc := &ServerDesc{
		endpoint: endpoint,

		canonicalEndpoint:  Endpoint(isMasterResult.Me),
		electionID:         isMasterResult.ElectionID,
		lastWriteTimestamp: isMasterResult.LastWriteTimestamp,
		maxBatchCount:      isMasterResult.MaxWriteBatchSize,
		maxDocumentSize:    isMasterResult.MaxBSONObjectSize,
		maxMessageSize:     isMasterResult.MaxMessageSizeBytes,
		members:            isMasterResult.Members(),
		serverType:         isMasterResult.ServerType(),
		setName:            isMasterResult.SetName,
		setVersion:         isMasterResult.SetVersion,
		tags:               isMasterResult.Tags,
		wireVersion: util.Range{
			Min: isMasterResult.MinWireVersion,
			Max: isMasterResult.MaxWireVersion,
		},
		version: util.Version{
			Desc:  buildInfoResult.Version,
			Parts: buildInfoResult.VersionArray,
		},
	}

	if !isMasterResult.OK {
		desc.lastError = fmt.Errorf("not ok")
	}

	return desc
}
