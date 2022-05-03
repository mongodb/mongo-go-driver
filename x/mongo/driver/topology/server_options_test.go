package topology

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestServerOptions(t *testing.T) {
	t.Run("newServerConfig", func(t *testing.T) {
		t.Parallel()

		t.Run("minPoolSize should not exceed maxPoolSize", func(t *testing.T) {
			t.Parallel()

			sOpts := []ServerOption{
				WithMaxConnections(func(uint64) uint64 {
					return uint64(10)
				}),
				WithMinConnections(func(uint64) uint64 {
					return uint64(100)
				}),
			}

			defer func() {
				if r := recover(); r == nil {
					assert.Fail(t, "newServerConfig is expected to panic if minPoolSize > maxPoolSize")
				}
			}()
			newServerConfig(sOpts...)
		})
		t.Run("minPoolSize should not exceed maxPoolSize", func(t *testing.T) {
			t.Parallel()

			sOpts := []ServerOption{
				WithMaxConnections(func(uint64) uint64 {
					return uint64(0)
				}),
				WithMinConnections(func(uint64) uint64 {
					return uint64(10)
				}),
			}

			sCfg := newServerConfig(sOpts...)
			assert.Equal(t, uint64(0), sCfg.maxConns)
			assert.Equal(t, uint64(10), sCfg.minConns)
		})
	})
}
