package topology

import (
	"time"

	"github.com/mongodb/mongo-go-driver/mongo/connstring"
	"github.com/mongodb/mongo-go-driver/mongo/private/auth"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/connection"
)

type Option func(*config) error

type config struct {
	mode                   MonitorMode
	replicaSetName         string
	seedList               []string
	serverOpts             []ServerOption
	cs                     connstring.ConnString
	serverSelectionTimeout time.Duration
}

func newConfig(opts ...Option) (*config, error) {
	cfg := &config{
		seedList:               []string{"localhost:27017"},
		serverSelectionTimeout: 30 * time.Second,
	}

	for _, opt := range opts {
		err := opt(cfg)
		if err != nil {
			return nil, err
		}
	}

	return cfg, nil
}

// WithConnString configures the topology using the connection string.
func WithConnString(fn func(connstring.ConnString) connstring.ConnString) Option {
	return func(c *config) error {
		cs := fn(c.cs)
		c.cs = cs
		var connOpts []connection.Option

		if cs.AppName != "" {
			connOpts = append(connOpts, connection.WithAppName(func(string) string { return cs.AppName }))
		}

		switch cs.Connect {
		case connstring.SingleConnect:
			c.mode = SingleMode
		}

		c.seedList = cs.Hosts

		if cs.ConnectTimeout > 0 {
			connOpts = append(connOpts, connection.WithConnectTimeout(func(time.Duration) time.Duration { return cs.ConnectTimeout }))
		}

		if cs.HeartbeatInterval > 0 {
			c.serverOpts = append(c.serverOpts, WithHeartbeatInterval(func(time.Duration) time.Duration { return cs.HeartbeatInterval }))
		}

		if cs.MaxConnIdleTime > 0 {
			connOpts = append(connOpts, connection.WithIdleTimeout(func(time.Duration) time.Duration { return cs.MaxConnIdleTime }))
		}

		if cs.MaxConnLifeTime > 0 {
			connOpts = append(connOpts, connection.WithIdleTimeout(func(time.Duration) time.Duration { return cs.MaxConnLifeTime }))
		}

		if cs.MaxConnsPerHostSet {
			c.serverOpts = append(c.serverOpts, WithMaxConnections(func(uint16) uint16 { return cs.MaxConnsPerHost }))
		}

		if cs.MaxIdleConnsPerHostSet {
			c.serverOpts = append(c.serverOpts, WithMaxIdleConnections(func(uint16) uint16 { return cs.MaxIdleConnsPerHost }))
		}

		if cs.ReplicaSet != "" {
			c.replicaSetName = cs.ReplicaSet
		}

		if cs.Username != "" || cs.AuthMechanism == auth.GSSAPI {
			cred := &auth.Cred{
				Source:      "admin",
				Username:    cs.Username,
				Password:    cs.Password,
				PasswordSet: cs.PasswordSet,
				Props:       cs.AuthMechanismProperties,
			}

			if cs.AuthSource != "" {
				cred.Source = cs.AuthSource
			} else {
				switch cs.AuthMechanism {
				case auth.GSSAPI, auth.PLAIN:
					cred.Source = "$external"
				default:
					cred.Source = cs.Database
				}
			}

			authenticator, err := auth.CreateAuthenticator(cs.AuthMechanism, cred)
			if err != nil {
				return err
			}

			connOpts = append(connOpts, connection.WithConfigurer(func(cf connection.Configurer) connection.Configurer {
				return auth.Configurer(cf, authenticator)
			}))
		}

		if cs.SSL {
			tls := connection.NewTLSConfig()

			if cs.SSLCaFileSet {
				err := tls.AddCACertFromFile(cs.SSLCaFile)
				if err != nil {
					return err
				}
			}

			if cs.SSLInsecure {
				tls.SetInsecure(true)
			}

			connOpts = append(connOpts, connection.WithTLSConfig(func(*connection.TLSConfig) *connection.TLSConfig { return tls }))
		}

		if len(connOpts) > 0 {
			c.serverOpts = append(c.serverOpts, WithConnectionOptions(func(opts ...connection.Option) []connection.Option {
				return append(opts, connOpts...)
			}))
		}

		return nil
	}
}

// WithMode configures the topology's monitor mode.
func WithMode(fn func(MonitorMode) MonitorMode) Option {
	return func(cfg *config) error {
		cfg.mode = fn(cfg.mode)
		return nil
	}
}

// WithReplicaSetName configures the topology's default replica set name.
func WithReplicaSetName(fn func(string) string) Option {
	return func(cfg *config) error {
		cfg.replicaSetName = fn(cfg.replicaSetName)
		return nil
	}
}

// WithSeedList configures a topology's seed list.
func WithSeedList(fn func(...string) []string) Option {
	return func(cfg *config) error {
		cfg.seedList = fn(cfg.seedList...)
		return nil
	}
}

// WithServerOptions configures a topology's server options for when a new server
// needs to be created.
func WithServerOptions(fn func(...ServerOption) []ServerOption) Option {
	return func(cfg *config) error {
		cfg.serverOpts = fn(cfg.serverOpts...)
		return nil
	}
}

// WithServerSelectionTimeout configures a topology's server selection timeout.
// A server selection timeout of 0 means there is no timeout for server selection.
func WithServerSelectionTimeout(fn func(time.Duration) time.Duration) Option {
	return func(cfg *config) error {
		cfg.serverSelectionTimeout = fn(cfg.serverSelectionTimeout)
		return nil
	}
}
