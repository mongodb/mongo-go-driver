package topology

import "github.com/mongodb/mongo-tools/common/connstring"

type Option func(*config) error

func WithConnString(func(connstring.ConnString) connstring.ConnString) Option { return nil }
func WithMode(func(MonitorMode) MonitorMode) Option                           { return nil }
func WithReplicaSetname(func(string) string) Option                           { return nil }
func WithSeedList(func(...string) []string) Option                            { return nil }
func WithServerOptions(func(...ServerOption) []ServerOption) Option           { return nil }

type config struct{}
