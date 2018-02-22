package connection

import "crypto/tls"

type TLSConfig struct{ tls.Config }

func NewTLSConfig() *TLSConfig                         { return nil }
func (*TLSConfig) SetInsecure(allow bool)              { return }
func (*TLSConfig) AddCACertFromFile(file string) error { return nil }
