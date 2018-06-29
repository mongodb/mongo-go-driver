package clientopt

import (
	"context"
	"net"
	"time"

	"reflect"

	"github.com/mongodb/mongo-go-driver/core/connection"
	"github.com/mongodb/mongo-go-driver/core/connstring"
	"github.com/mongodb/mongo-go-driver/core/readconcern"
	"github.com/mongodb/mongo-go-driver/core/readpref"
	"github.com/mongodb/mongo-go-driver/core/topology"
	"github.com/mongodb/mongo-go-driver/core/writeconcern"
)

var clientBundle = new(ClientBundle)

// ContextDialer makes new network connections
type ContextDialer interface {
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
}

// Option represents a client option
type Option interface {
	clientOption()
}

// optionFunc adds the option to the client
type optionFunc func(*Client) error

// SSLOpt holds client SSL options.
//
// Enabled indicates whether SSL should be enabled.
//
// ClientCertificateKeyFile specifies the file containing the client certificate and private key
// used for authentication.
//
// ClientCertificateKeyPassword provides a callback that returns a password used for decrypting the
// private key of a PEM file (if one is provided).
//
// Insecure indicates whether to skip the verification of the server certificate and hostname.
//
// CaFile specifies the file containing the certificate authority used for SSL connections.
type SSLOpt struct {
	Enabled                      bool
	ClientCertificateKeyFile     string
	ClientCertificateKeyPassword func() string
	Insecure                     bool
	CaFile                       string
}

// Credential holds auth options.
//
// AuthMechanism indicates the mechanism to use for authentication.
// Supported values include "SCRAM-SHA-1", "MONGODB-CR", "PLAIN", "GSSAPI", and "MONGODB-X509".
//
// AuthMechanismProperties specifies additional configuration options which may be used by certain
// authentication mechanisms.
//
// AuthSource specifies the database to authenticate against.
//
// Username specifies the username that will be authenticated.
//
// Password specifies the password used for authentication.
type Credential struct {
	AuthMechanism           string
	AuthMechanismProperties map[string]string
	AuthSource              string
	Username                string
	Password                string
}

// Client represents a client
type Client struct {
	TopologyOptions []topology.Option
	ConnString      connstring.ConnString
	ReadPreference  *readpref.ReadPref
	ReadConcern     *readconcern.ReadConcern
	WriteConcern    *writeconcern.WriteConcern
}

// ClientBundle is a bundle of client options
type ClientBundle struct {
	option Option
	next   *ClientBundle
}

// ClientBundle implements Option
func (*ClientBundle) clientOption() {}

// optionFunc implements clientOption
func (optionFunc) clientOption() {}

// BundleClient bundles client options
func BundleClient(opts ...Option) *ClientBundle {
	head := clientBundle

	for _, opt := range opts {
		newBundle := ClientBundle{
			option: opt,
			next:   head,
		}

		head = &newBundle
	}

	return head
}

// AppName specifies the client application name. This value is used by MongoDB when it logs
// connection information and profile information, such as slow queries.
func (cb *ClientBundle) AppName(s string) *ClientBundle {
	return &ClientBundle{
		option: AppName(s),
		next:   cb,
	}
}

// Auth sets the authentication properties.
func (cb *ClientBundle) Auth(auth Credential) *ClientBundle {
	return &ClientBundle{
		option: Auth(auth),
		next:   cb,
	}
}

// ConnectTimeout specifies the timeout for an initial connection to a server.
// If a custom Dialer is used, this method won't be set and the user is
// responsible for setting the ConnectTimeout for connections on the dialer
// themselves.
func (cb *ClientBundle) ConnectTimeout(d time.Duration) *ClientBundle {
	return &ClientBundle{
		option: ConnectTimeout(d),
		next:   cb,
	}
}

// Dialer specifies a custom dialer used to dial new connections to a server.
func (cb *ClientBundle) Dialer(d ContextDialer) *ClientBundle {
	return &ClientBundle{
		option: Dialer(d),
		next:   cb,
	}
}

// HeartbeatInterval specifies the interval to wait between server monitoring checks.
func (cb *ClientBundle) HeartbeatInterval(d time.Duration) *ClientBundle {
	return &ClientBundle{
		option: HeartbeatInterval(d),
		next:   cb,
	}
}

// Hosts specifies the initial list of addresses from which to discover the rest of the cluster.
func (cb *ClientBundle) Hosts(s []string) *ClientBundle {
	return &ClientBundle{
		option: Hosts(s),
		next:   cb,
	}
}

// LocalThreshold specifies how far to distribute queries, beyond the server with the fastest
// round-trip time. If a server's roundtrip time is more than LocalThreshold slower than the
// the fastest, the driver will not send queries to that server.
func (cb *ClientBundle) LocalThreshold(d time.Duration) *ClientBundle {
	return &ClientBundle{
		option: LocalThreshold(d),
		next:   cb,
	}
}

// MaxConnIdleTime specifies the maximum number of milliseconds that a connection can remain idle
// in a connection pool before being removed and closed.
func (cb *ClientBundle) MaxConnIdleTime(d time.Duration) *ClientBundle {
	return &ClientBundle{
		option: MaxConnIdleTime(d),
		next:   cb,
	}
}

// MaxConnsPerHost specifies the max size of a server's connection pool.
func (cb *ClientBundle) MaxConnsPerHost(u uint16) *ClientBundle {
	return &ClientBundle{
		option: MaxConnsPerHost(u),
		next:   cb,
	}
}

// MaxIdleConnsPerHost specifies the number of connections in a server's connection pool that can
// be idle at any given time.
func (cb *ClientBundle) MaxIdleConnsPerHost(u uint16) *ClientBundle {
	return &ClientBundle{
		option: MaxIdleConnsPerHost(u),
		next:   cb,
	}
}

// ReadConcern specifies the read concern.
func (cb *ClientBundle) ReadConcern(rc *readconcern.ReadConcern) *ClientBundle {
	return &ClientBundle{
		option: ReadConcern(rc),
		next:   cb,
	}
}

// ReadPreference specifies the read preference.
func (cb *ClientBundle) ReadPreference(rp *readpref.ReadPref) *ClientBundle {
	return &ClientBundle{
		option: ReadPreference(rp),
		next:   cb,
	}
}

// ReplicaSet specifies the name of the replica set of the cluster.
func (cb *ClientBundle) ReplicaSet(s string) *ClientBundle {
	return &ClientBundle{
		option: ReplicaSet(s),
		next:   cb,
	}
}

// ServerSelectionTimeout specifies a timeout in milliseconds to block for server selection.
func (cb *ClientBundle) ServerSelectionTimeout(d time.Duration) *ClientBundle {
	return &ClientBundle{
		option: ServerSelectionTimeout(d),
		next:   cb,
	}
}

// Single specifies whether the driver should connect directly to the server instead of
// auto-discovering other servers in the cluster.
func (cb *ClientBundle) Single(b bool) *ClientBundle {
	return &ClientBundle{
		option: Single(b),
		next:   cb,
	}
}

// SocketTimeout specifies the time in milliseconds to attempt to send or receive on a socket
// before the attempt times out.
func (cb *ClientBundle) SocketTimeout(d time.Duration) *ClientBundle {
	return &ClientBundle{
		option: SocketTimeout(d),
		next:   cb,
	}
}

// SSL sets SSL options.
func (cb *ClientBundle) SSL(ssl *SSLOpt) *ClientBundle {
	return &ClientBundle{
		option: SSL(ssl),
		next:   cb,
	}
}

// WriteConcern specifies the write concern.
func (cb *ClientBundle) WriteConcern(wc *writeconcern.WriteConcern) *ClientBundle {
	return &ClientBundle{
		option: WriteConcern(wc),
		next:   cb,
	}
}

// String prints a string representation of the bundle for debug purposes.
func (cb *ClientBundle) String() string {
	if cb == nil {
		return ""
	}

	debugStr := ""
	for head := cb; head != nil && head.option != nil; head = head.next {
		switch opt := head.option.(type) {
		case *ClientBundle:
			debugStr = opt.String() + debugStr
		case optionFunc:
			debugStr = reflect.TypeOf(opt).String() + "\n" + debugStr
		default:
			return debugStr + "(error: ClientOption can only be *ClientBundle or optionFunc)"
		}
	}

	return debugStr
}

// Unbundle transforms a client given a connectionstring.
func (cb *ClientBundle) Unbundle(connString connstring.ConnString) (*Client, error) {
	client := &Client{
		ConnString: connString,
	}
	err := cb.unbundle(client)
	if err != nil {
		return nil, err
	}

	return client, nil
}

// Helper that recursively unwraps bundle.
func (cb *ClientBundle) unbundle(client *Client) error {
	if cb == nil {
		return nil
	}

	for head := cb; head != nil && head.option != nil; head = head.next {
		var err error
		switch opt := head.option.(type) {
		case *ClientBundle:
			err = opt.unbundle(client) // add all bundle's options to client
		case optionFunc:
			err = opt(client) // add option to client
		default:
			return nil
		}
		if err != nil {
			return err
		}
	}

	return nil

}

// AppName specifies the client application name. This value is used by MongoDB when it logs
// connection information and profile information, such as slow queries.
func AppName(s string) Option {
	return optionFunc(
		func(c *Client) error {
			if c.ConnString.AppName == "" {
				c.ConnString.AppName = s
			}
			return nil
		})
}

// Auth sets the authentication options.
func Auth(auth Credential) Option {
	return optionFunc(
		func(c *Client) error {
			if len(c.ConnString.AuthMechanism) == 0 {
				c.ConnString.AuthMechanism = auth.AuthMechanism
			}
			if c.ConnString.AuthMechanismProperties == nil {
				c.ConnString.AuthMechanismProperties = auth.AuthMechanismProperties
			}
			if len(c.ConnString.AuthSource) == 0 {
				c.ConnString.AuthSource = auth.AuthSource
			}
			if len(c.ConnString.Username) == 0 {
				c.ConnString.Username = auth.Username
			}
			if len(c.ConnString.Password) == 0 {
				c.ConnString.Password = auth.Password
			}
			return nil
		})
}

// ConnectTimeout specifies the timeout for an initial connection to a server.
// If a custom Dialer is used, this method won't be set and the user is
// responsible for setting the ConnectTimeout for connections on the dialer
// themselves.
func ConnectTimeout(d time.Duration) Option {
	return optionFunc(
		func(c *Client) error {
			if !c.ConnString.ConnectTimeoutSet {
				c.ConnString.ConnectTimeout = d
				c.ConnString.ConnectTimeoutSet = true
			}
			return nil
		})
}

// Dialer specifies a custom dialer used to dial new connections to a server.
func Dialer(d ContextDialer) Option {
	return optionFunc(
		func(c *Client) error {
			c.TopologyOptions = append(
				c.TopologyOptions,
				topology.WithServerOptions(func(opts ...topology.ServerOption) []topology.ServerOption {
					return append(
						opts,
						topology.WithConnectionOptions(func(opts ...connection.Option) []connection.Option {
							return append(
								opts,
								connection.WithDialer(func(connection.Dialer) connection.Dialer {
									return d
								}),
							)
						}),
					)
				}),
			)
			return nil
		})
}

// HeartbeatInterval specifies the interval to wait between server monitoring checks.
func HeartbeatInterval(d time.Duration) Option {
	return optionFunc(
		func(c *Client) error {
			if !c.ConnString.HeartbeatIntervalSet {
				c.ConnString.HeartbeatInterval = d
				c.ConnString.HeartbeatIntervalSet = true
			}
			return nil
		})
}

// Hosts specifies the initial list of addresses from which to discover the rest of the cluster.
func Hosts(s []string) Option {
	return optionFunc(
		func(c *Client) error {
			if c.ConnString.Hosts == nil {
				c.ConnString.Hosts = s
			}
			return nil
		})
}

// LocalThreshold specifies how far to distribute queries, beyond the server with the fastest
// round-trip time. If a server's roundtrip time is more than LocalThreshold slower than the
// the fastest, the driver will not send queries to that server.
func LocalThreshold(d time.Duration) Option {
	return optionFunc(
		func(c *Client) error {
			if !c.ConnString.LocalThresholdSet {
				c.ConnString.LocalThreshold = d
				c.ConnString.LocalThresholdSet = true
			}
			return nil
		})
}

// MaxConnIdleTime specifies the maximum number of milliseconds that a connection can remain idle
// in a connection pool before being removed and closed.
func MaxConnIdleTime(d time.Duration) Option {
	return optionFunc(
		func(c *Client) error {
			if !c.ConnString.MaxConnIdleTimeSet {
				c.ConnString.MaxConnIdleTime = d
				c.ConnString.MaxConnIdleTimeSet = true
			}
			return nil
		})
}

// MaxConnsPerHost specifies the max size of a server's connection pool.
func MaxConnsPerHost(u uint16) Option {
	return optionFunc(
		func(c *Client) error {
			if !c.ConnString.MaxConnsPerHostSet {
				c.ConnString.MaxConnsPerHost = u
				c.ConnString.MaxConnsPerHostSet = true
			}
			return nil
		})
}

// MaxIdleConnsPerHost specifies the number of connections in a server's connection pool that can
// be idle at any given time.
func MaxIdleConnsPerHost(u uint16) Option {
	return optionFunc(
		func(c *Client) error {
			if !c.ConnString.MaxIdleConnsPerHostSet {
				c.ConnString.MaxIdleConnsPerHost = u
				c.ConnString.MaxIdleConnsPerHostSet = true
			}
			return nil
		})
}

// ReadConcern specifies the read concern.
func ReadConcern(rc *readconcern.ReadConcern) Option {
	return optionFunc(func(c *Client) error {
		if c.ReadConcern == nil {
			c.ReadConcern = rc
		}
		return nil
	})
}

// ReadPreference specifies the read preference
func ReadPreference(rp *readpref.ReadPref) Option {
	return optionFunc(
		func(c *Client) error {
			if c.ReadPreference == nil {
				c.ReadPreference = rp
			}
			return nil
		})
}

// ReplicaSet specifies the name of the replica set of the cluster.
func ReplicaSet(s string) Option {
	return optionFunc(
		func(c *Client) error {
			if c.ConnString.ReplicaSet == "" {
				c.ConnString.ReplicaSet = s
			}
			return nil
		})
}

// ServerSelectionTimeout specifies a timeout in milliseconds to block for server selection.
func ServerSelectionTimeout(d time.Duration) Option {
	return optionFunc(
		func(c *Client) error {
			if !c.ConnString.ServerSelectionTimeoutSet {
				c.ConnString.ServerSelectionTimeout = d
				c.ConnString.ServerSelectionTimeoutSet = true
			}
			return nil
		})
}

// Single specifies whether the driver should connect directly to the server instead of
// auto-discovering other servers in the cluster.
func Single(b bool) Option {
	return optionFunc(
		func(c *Client) error {
			if !c.ConnString.ConnectSet {
				if b {
					c.ConnString.Connect = connstring.SingleConnect
				} else {
					c.ConnString.Connect = connstring.AutoConnect
				}

				c.ConnString.ConnectSet = true
			}
			return nil
		})
}

// SocketTimeout specifies the time in milliseconds to attempt to send or receive on a socket
// before the attempt times out.
func SocketTimeout(d time.Duration) Option {
	return optionFunc(
		func(c *Client) error {
			if !c.ConnString.SocketTimeoutSet {
				c.ConnString.SocketTimeout = d
				c.ConnString.SocketTimeoutSet = true
			}
			return nil
		})
}

// SSL sets SSL options.
func SSL(ssl *SSLOpt) Option {
	return optionFunc(
		func(c *Client) error {
			if !c.ConnString.SSLSet {
				c.ConnString.SSL = ssl.Enabled
				c.ConnString.SSLSet = true
			}
			if !c.ConnString.SSLClientCertificateKeyFileSet {
				c.ConnString.SSLClientCertificateKeyFile = ssl.ClientCertificateKeyFile
				c.ConnString.SSLClientCertificateKeyFileSet = true
			}
			if !c.ConnString.SSLClientCertificateKeyPasswordSet {
				c.ConnString.SSLClientCertificateKeyPassword = ssl.ClientCertificateKeyPassword
				c.ConnString.SSLClientCertificateKeyPasswordSet = true
			}
			if !c.ConnString.SSLInsecureSet {
				c.ConnString.SSLInsecure = ssl.Insecure
				c.ConnString.SSLInsecureSet = true
			}
			if !c.ConnString.SSLCaFileSet {
				c.ConnString.SSLCaFile = ssl.CaFile
				c.ConnString.SSLCaFileSet = true
			}
			return nil
		})
}

// WriteConcern sets the write concern.
func WriteConcern(wc *writeconcern.WriteConcern) Option {
	return optionFunc(
		func(c *Client) error {
			if c.WriteConcern == nil {
				c.WriteConcern = wc
			}
			return nil
		})
}
