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

// Client represents a client
type Client struct {
	topologyOptions []topology.Option
	topology        *topology.Topology
	connString      connstring.ConnString
	localThreshold  time.Duration
	readPreference  *readpref.ReadPref
	readConcern     *readconcern.ReadConcern
	writeConcern    *writeconcern.WriteConcern
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

// AuthMechanism indicates the mechanism to use for authentication.
//
// Supported values include "SCRAM-SHA-1", "MONGODB-CR", "PLAIN", "GSSAPI", and "MONGODB-X509".
func (cb *ClientBundle) AuthMechanism(s string) *ClientBundle {
	return &ClientBundle{
		option: AuthMechanism(s),
		next:   cb,
	}
}

// AuthMechanismProperties specifies additional configuration options which may be used by certain
// authentication mechanisms.
func (cb *ClientBundle) AuthMechanismProperties(m map[string]string) *ClientBundle {
	return &ClientBundle{
		option: AuthMechanismProperties(m),
		next:   cb,
	}
}

// AuthSource specifies the database to authenticate against.
func (cb *ClientBundle) AuthSource(s string) *ClientBundle {
	return &ClientBundle{
		option: AuthSource(s),
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

// Journal specifies the "j" field of the default write concern to set on the Client.
func (cb *ClientBundle) Journal(b bool) *ClientBundle {
	return &ClientBundle{
		option: Journal(b),
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

// Password specifies the password used for authentication.
func (cb *ClientBundle) Password(s string) *ClientBundle {
	return &ClientBundle{
		option: Password(s),
		next:   cb,
	}
}

// ReadConcernLevel specifies the read concern level of the default read concern to set on the
// client.
func (cb *ClientBundle) ReadConcernLevel(rc *readconcern.ReadConcern) *ClientBundle {
	return &ClientBundle{
		option: ReadConcernLevel(rc),
		next:   cb,
	}
}

// ReadPreference specifies the read preference mode of the default read preference to set on the
// client.
func (cb *ClientBundle) ReadPreference(s string) *ClientBundle {
	return &ClientBundle{
		option: ReadPreference(s),
		next:   cb,
	}
}

// ReadPreferenceTagSets specifies the read preference tagsets of the default read preference to
// set on the client.
func (cb *ClientBundle) ReadPreferenceTagSets(m []map[string]string) *ClientBundle {
	return &ClientBundle{
		option: ReadPreferenceTagSets(m),
		next:   cb,
	}
}

// MaxStaleness sets the "maxStaleness" field of the read pref to set on the client.
func (cb *ClientBundle) MaxStaleness(d time.Duration) *ClientBundle {
	return &ClientBundle{
		option: MaxStaleness(d),
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

// SSL indicates whether SSL should be enabled.
func (cb *ClientBundle) SSL(b bool) *ClientBundle {
	return &ClientBundle{
		option: SSL(b),
		next:   cb,
	}
}

// SSLClientCertificateKeyFile specifies the file containing the client certificate and private key
// used for authentication.
func (cb *ClientBundle) SSLClientCertificateKeyFile(s string) *ClientBundle {
	return &ClientBundle{
		option: SSLClientCertificateKeyFile(s),
		next:   cb,
	}
}

// SSLClientCertificateKeyPassword provides a callback that returns a password used for decrypting the
// private key of a PEM file (if one is provided).
func (cb *ClientBundle) SSLClientCertificateKeyPassword(s func() string) *ClientBundle {
	return &ClientBundle{
		option: SSLClientCertificateKeyPassword(s),
		next:   cb,
	}
}

// SSLInsecure indicates whether to skip the verification of the server certificate and hostname.
func (cb *ClientBundle) SSLInsecure(b bool) *ClientBundle {
	return &ClientBundle{
		option: SSLInsecure(b),
		next:   cb,
	}
}

// SSLCaFile specifies the file containing the certificate authority used for SSL connections.
func (cb *ClientBundle) SSLCaFile(s string) *ClientBundle {
	return &ClientBundle{
		option: SSLCaFile(s),
		next:   cb,
	}
}

// WString sets the "w" field of the default write concern to set on the client.
func (cb *ClientBundle) WString(s string) *ClientBundle {
	return &ClientBundle{
		option: WString(s),
		next:   cb,
	}
}

// WNumber sets the "w" field of the default write concern to set on the client.
func (cb *ClientBundle) WNumber(i int) *ClientBundle {
	return &ClientBundle{
		option: WNumber(i),
		next:   cb,
	}
}

// Username specifies the username that will be authenticated.
func (cb *ClientBundle) Username(s string) *ClientBundle {
	return &ClientBundle{
		option: Username(s),
		next:   cb,
	}
}

// WTimeout sets the "wtimeout" field of the default write concern to set on the client.
func (cb *ClientBundle) WTimeout(d time.Duration) *ClientBundle {
	return &ClientBundle{
		option: WTimeout(d),
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
			debugStr += opt.String() + "\n"
		case optionFunc:
			debugStr += reflect.TypeOf(opt).String() + "\n"
		default:
			return debugStr + "(error: ClientOption can only be *ClientBundle or optionFunc)"
		}
	}

	return debugStr
}

// Unbundle transforms a bundle into a slice of options, optionally deduplicating
func (cb *ClientBundle) Unbundle() (*Client, error) {
	client := &Client{}
	err := cb.unbundle(client)
	if err != nil {
		return nil, err
	}

	return client, nil
}

// Helper that recursively unwraps bundle into slice of options
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
			if c.connString.AppName == "" {
				c.connString.AppName = s
			}
			return nil
		})
}

// AuthMechanism indicates the mechanism to use for authentication.
//
// Supported values include "SCRAM-SHA-1", "MONGODB-CR", "PLAIN", "GSSAPI", and "MONGODB-X509".
func AuthMechanism(s string) Option {
	return optionFunc(
		func(c *Client) error {
			if len(c.connString.AuthMechanism) == 0 {
				c.connString.AuthMechanism = s
			}
			return nil
		})
}

// AuthMechanismProperties specifies additional configuration options which may be used by certain
// authentication mechanisms.
func AuthMechanismProperties(m map[string]string) Option {
	return optionFunc(
		func(c *Client) error {
			if c.connString.AuthMechanismProperties == nil {
				c.connString.AuthMechanismProperties = m
			}
			return nil
		})
}

// AuthSource specifies the database to authenticate against.
func AuthSource(s string) Option {
	return optionFunc(
		func(c *Client) error {
			if len(c.connString.AuthSource) == 0 {
				c.connString.AuthSource = s
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
			if !c.connString.ConnectTimeoutSet {
				c.connString.ConnectTimeout = d
				c.connString.ConnectTimeoutSet = true
			}
			return nil
		})
}

// Dialer specifies a custom dialer used to dial new connections to a server.
func Dialer(d ContextDialer) Option {
	return optionFunc(
		func(c *Client) error {
			c.topologyOptions = append(
				c.topologyOptions,
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
			if !c.connString.HeartbeatIntervalSet {
				c.connString.HeartbeatInterval = d
				c.connString.HeartbeatIntervalSet = true
			}
			return nil
		})
}

// Hosts specifies the initial list of addresses from which to discover the rest of the cluster.
func Hosts(s []string) Option {
	return optionFunc(
		func(c *Client) error {
			if c.connString.Hosts == nil {
				c.connString.Hosts = s
			}
			return nil
		})
}

// Journal specifies the "j" field of the default write concern to set on the Client.
func Journal(b bool) Option {
	return optionFunc(func(c *Client) error {
		if !c.connString.JSet {
			c.connString.J = b
			c.connString.JSet = true
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
			if !c.connString.LocalThresholdSet {
				c.connString.LocalThreshold = d
				c.connString.LocalThresholdSet = true
			}
			return nil
		})
}

// MaxConnIdleTime specifies the maximum number of milliseconds that a connection can remain idle
// in a connection pool before being removed and closed.
func MaxConnIdleTime(d time.Duration) Option {
	return optionFunc(
		func(c *Client) error {
			if !c.connString.MaxConnIdleTimeSet {
				c.connString.MaxConnIdleTime = d
				c.connString.MaxConnIdleTimeSet = true
			}
			return nil
		})
}

// MaxConnsPerHost specifies the max size of a server's connection pool.
func MaxConnsPerHost(u uint16) Option {
	return optionFunc(
		func(c *Client) error {
			if !c.connString.MaxConnsPerHostSet {
				c.connString.MaxConnsPerHost = u
				c.connString.MaxConnsPerHostSet = true
			}
			return nil
		})
}

// MaxIdleConnsPerHost specifies the number of connections in a server's connection pool that can
// be idle at any given time.
func MaxIdleConnsPerHost(u uint16) Option {
	return optionFunc(
		func(c *Client) error {
			if !c.connString.MaxIdleConnsPerHostSet {
				c.connString.MaxIdleConnsPerHost = u
				c.connString.MaxIdleConnsPerHostSet = true
			}
			return nil
		})
}

// Password specifies the password used for authentication.
func Password(s string) Option {
	return optionFunc(
		func(c *Client) error {
			if !c.connString.PasswordSet {
				c.connString.Password = s
				c.connString.PasswordSet = true
			}
			return nil
		})
}

// ReadConcernLevel specifies the read concern level of the default read concern to set on the
// client.
func ReadConcernLevel(rc *readconcern.ReadConcern) Option {
	return optionFunc(func(c *Client) error {
		if c.connString.ReadConcernLevel == "" {
			c.connString.ReadConcernLevel = rc
		}
		return nil
	})
}

// ReadPreference specifies the read preference mode of the default read preference to set on the
// client.
func ReadPreference(s string) Option {
	return optionFunc(
		func(c *Client) error {
			if c.connString.ReadPreference == "" {
				c.connString.ReadPreference = s
			}
			return nil
		})
}

// ReadPreferenceTagSets specifies the read preference tagsets of the default read preference to
// set on the client.
func ReadPreferenceTagSets(m []map[string]string) Option {
	return optionFunc(
		func(c *Client) error {
			if c.connString.ReadPreferenceTagSets == nil {
				c.connString.ReadPreferenceTagSets = m
			}
			return nil
		})
}

//MaxStaleness sets the "maxStaleness" field of the read pref to set on the client.
func MaxStaleness(d time.Duration) Option {
	return optionFunc(
		func(c *Client) error {
			if !c.connString.MaxStalenessSet {
				c.connString.MaxStaleness = d
				c.connString.MaxStalenessSet = true
			}
			return nil
		})
}

// ReplicaSet specifies the name of the replica set of the cluster.
func ReplicaSet(s string) Option {
	return optionFunc(
		func(c *Client) error {
			if c.connString.ReplicaSet == "" {
				c.connString.ReplicaSet = s
			}
			return nil
		})
}

// ServerSelectionTimeout specifies a timeout in milliseconds to block for server selection.
func ServerSelectionTimeout(d time.Duration) Option {
	return optionFunc(
		func(c *Client) error {
			if !c.connString.ServerSelectionTimeoutSet {
				c.connString.ServerSelectionTimeout = d
				c.connString.ServerSelectionTimeoutSet = true
			}
			return nil
		})
}

// Single specifies whether the driver should connect directly to the server instead of
// auto-discovering other servers in the cluster.
func Single(b bool) Option {
	return optionFunc(
		func(c *Client) error {
			if !c.connString.ConnectSet {
				if b {
					c.connString.Connect = connstring.SingleConnect
				} else {
					c.connString.Connect = connstring.AutoConnect
				}

				c.connString.ConnectSet = true
			}
			return nil
		})
}

// SocketTimeout specifies the time in milliseconds to attempt to send or receive on a socket
// before the attempt times out.
func SocketTimeout(d time.Duration) Option {
	return optionFunc(
		func(c *Client) error {
			if !c.connString.SocketTimeoutSet {
				c.connString.SocketTimeout = d
				c.connString.SocketTimeoutSet = true
			}
			return nil
		})
}

// SSL indicates whether SSL should be enabled.
func SSL(b bool) Option {
	return optionFunc(
		func(c *Client) error {
			if !c.connString.SSLSet {
				c.connString.SSL = b
				c.connString.SSLSet = true
			}
			return nil
		})
}

// SSLClientCertificateKeyFile specifies the file containing the client certificate and private key
// used for authentication.
func SSLClientCertificateKeyFile(s string) Option {
	return optionFunc(
		func(c *Client) error {
			if !c.connString.SSLClientCertificateKeyFileSet {
				c.connString.SSLClientCertificateKeyFile = s
				c.connString.SSLClientCertificateKeyFileSet = true
			}
			return nil
		})
}

// SSLClientCertificateKeyPassword provides a callback that returns a password used for decrypting the
// private key of a PEM file (if one is provided).
func SSLClientCertificateKeyPassword(s func() string) Option {
	return optionFunc(
		func(c *Client) error {
			if !c.connString.SSLClientCertificateKeyPasswordSet {
				c.connString.SSLClientCertificateKeyPassword = s
				c.connString.SSLClientCertificateKeyPasswordSet = true
			}
			return nil
		})
}

// SSLInsecure indicates whether to skip the verification of the server certificate and hostname.
func SSLInsecure(b bool) Option {
	return optionFunc(
		func(c *Client) error {
			if !c.connString.SSLInsecureSet {
				c.connString.SSLInsecure = b
				c.connString.SSLInsecureSet = true
			}
			return nil
		})
}

// SSLCaFile specifies the file containing the certificate authority used for SSL connections.
func SSLCaFile(s string) Option {
	return optionFunc(
		func(c *Client) error {
			if !c.connString.SSLCaFileSet {
				c.connString.SSLCaFile = s
				c.connString.SSLCaFileSet = true
			}
			return nil
		})
}

// WString sets the "w" field of the default write concern to set on the client.
func WString(s string) Option {
	return optionFunc(
		func(c *Client) error {
			if !c.connString.WNumberSet && c.connString.WString == "" {
				c.connString.WString = s
				c.connString.WNumberSet = true
			}
			return nil
		})
}

// WNumber sets the "w" field of the default write concern to set on the client.
func WNumber(i int) Option {
	return optionFunc(
		func(c *Client) error {
			if !c.connString.WNumberSet && c.connString.WString == "" {
				c.connString.WNumber = i
				c.connString.WNumberSet = true
			}
			return nil
		})
}

// Username specifies the username that will be authenticated.
func Username(s string) Option {
	return optionFunc(
		func(c *Client) error {
			if c.connString.Username == "" {
				c.connString.Username = s
			}
			return nil
		})
}

// WTimeout sets the "wtimeout" field of the default write concern to set on the client.
func WTimeout(d time.Duration) Option {
	return optionFunc(
		func(c *Client) error {
			if !c.connString.WTimeoutSet && !c.connString.WTimeoutSetFromOption {
				c.connString.WTimeout = d
				c.connString.WTimeoutSetFromOption = true
			}
			return nil
		})
}
