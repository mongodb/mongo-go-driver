package logger

import (
	"strconv"
	"time"
)

const (
	ConnectionMessagePoolCreatedDefault = "Connection pool created"
)

// ConnectionMessage contains data that all connection log messages MUST contain.
type ConnectionMessage struct {
	// MessageLiteral is the literal message to be logged defining the
	// underlying event.
	MessageLiteral string

	// ServerHost is the hostname, IP address, or Unix domain socket path
	// for the endpoint the pool is for.
	ServerHost string

	// Port is the port for the endpoint the pool is for. If the user does
	// not specify a port and the default (27017) is used, the driver SHOULD
	// include it here.
	ServerPort string
}

func (*ConnectionMessage) Component() Component {
	return ComponentConnection
}

func (msg *ConnectionMessage) Message() string {
	return msg.MessageLiteral
}

func serialiseConnection(msg ConnectionMessage) ([]interface{}, error) {
	keysAndValues := []interface{}{
		"message", msg.MessageLiteral,
		"serverHost", msg.ServerHost,
	}

	// Convert the ServerPort into an integer.
	port, err := strconv.ParseInt(msg.ServerPort, 0, 32)
	if err != nil {
		return nil, err
	}

	keysAndValues = append(keysAndValues, "serverPort", int(port))

	return keysAndValues, nil
}

/*
message	String	"Connection pool created"
maxIdleTimeMS	Int	The maxIdleTimeMS value for this pool. Optional; only required to include if the user specified a value.
minPoolSize	Int	The minPoolSize value for this pool. Optional; only required to include if the user specified a value.
maxPoolSize	Int	The maxPoolSize value for this pool. Optional; only required to include if the user specified a value.
maxConnecting	Int	The maxConnecting value for this pool. Optional; only required to include if the driver supports this option and the user specified a value.
waitQueueTimeoutMS	Int	The waitQueueTimeoutMS value for this pool. Optional; only required to include if the driver supports this option and the user specified a value.
waitQueueSize	Int	The waitQueueSize value for this pool. Optional; only required to include if the driver supports this option and the user specified a value.
waitQueueMultiple	Int	The waitQueueMultiple value for this pool. Optional; only required to include if the driver supports this option and the user specified a value.
*/

// PoolCreatedMessage occurs when a connection pool is created.
type PoolCreatedMessage struct {
	ConnectionMessage

	// MaxIdleTime is the maxIdleTimeMS value for this pool. This field is
	// only required if the user specified a value for it.
	MaxIdleTime time.Duration

	// MinPoolSize is the minPoolSize value for this pool. This field is
	// only required to include if the user specified a value.
	MinPoolSize uint64

	// MaxPoolSize is the maxPoolSize value for this pool. This field is
	// only required to include if the user specified a value. The default
	// value is defined by "defaultMaxPoolSize" in the "mongo" package.
	MaxPoolSize uint64

	// MaxConnecting is the maxConnecting value for this pool. This field
	// is only required to include if the user specified a value.
	MaxConnecting uint64
}

func (msg *PoolCreatedMessage) Serialize(_ uint) ([]interface{}, error) {
	keysAndValues, err := serialiseConnection(msg.ConnectionMessage)
	if err != nil {
		return nil, err
	}

	if msg.MaxIdleTime > 0 {
		keysAndValues = append(keysAndValues, "maxIdleTimeMS", int(msg.MaxIdleTime/time.Millisecond))
	}

	if msg.MinPoolSize > 0 {
		keysAndValues = append(keysAndValues, "minPoolSize", int(msg.MinPoolSize))
	}

	if msg.MaxPoolSize > 0 {
		keysAndValues = append(keysAndValues, "maxPoolSize", int(msg.MaxPoolSize))
	}

	if msg.MaxConnecting > 0 {
		keysAndValues = append(keysAndValues, "maxConnecting", int(msg.MaxConnecting))
	}

	return keysAndValues, nil
}
