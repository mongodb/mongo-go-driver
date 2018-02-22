package connection

import "crypto/tls"

// TLSConfig contains options for configuring a TLS connection to the server.
type TLSConfig struct{ tls.Config }

// NewTLSConfig creates a new TLSConfig.
func NewTLSConfig() *TLSConfig { return nil }

// SetInsecure sets whether the client should verify the server's certificate
// chain and hostnames.
func (*TLSConfig) SetInsecure(allow bool) { return }

// AddCACertFromFile adds a root CA certificate to the configuration given a path
// to the containing file.
func (*TLSConfig) AddCACertFromFile(file string) error { return nil }
