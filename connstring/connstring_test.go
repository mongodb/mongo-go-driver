package connstring_test

import (
	"fmt"
	"testing"

	"time"

	"github.com/10gen/mongo-go-driver/connstring"
	"github.com/stretchr/testify/require"
)

func TestAppName(t *testing.T) {
	tests := []struct {
		s        string
		expected string
		err      bool
	}{
		{s: "appName=Funny", expected: "Funny"},
		{s: "appName=awesome", expected: "awesome"},
		{s: "appName=", expected: ""},
	}

	for _, test := range tests {
		s := fmt.Sprintf("mongodb://localhost/?%s", test.s)
		t.Run(s, func(t *testing.T) {
			cs, err := connstring.Parse(s)
			if test.err {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, test.expected, cs.AppName)
			}
		})
	}
}

func TestAuthMechanism(t *testing.T) {
	tests := []struct {
		s        string
		expected string
		err      bool
	}{
		{s: "authMechanism=scram-sha-1", expected: "scram-sha-1"},
		{s: "authMechanism=mongodb-CR", expected: "mongodb-CR"},
		{s: "authMechanism=LDAP", expected: "LDAP"},
	}

	for _, test := range tests {
		s := fmt.Sprintf("mongodb://localhost/?%s", test.s)
		t.Run(s, func(t *testing.T) {
			cs, err := connstring.Parse(s)
			if test.err {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, test.expected, cs.AuthMechanism)
			}
		})
	}
}

func TestConnect(t *testing.T) {
	tests := []struct {
		s        string
		expected connstring.ConnectMode
		err      bool
	}{
		{s: "connect=auto", expected: connstring.AutoConnect},
		{s: "connect=automatic", expected: connstring.AutoConnect},
		{s: "connect=AUTO", expected: connstring.AutoConnect},
		{s: "connect=single", expected: connstring.SingleConnect},
		{s: "connect=direct", expected: connstring.SingleConnect},
		{s: "connect=blah", err: true},
	}

	for _, test := range tests {
		s := fmt.Sprintf("mongodb://localhost/?%s", test.s)
		t.Run(s, func(t *testing.T) {
			cs, err := connstring.Parse(s)
			if test.err {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, test.expected, cs.Connect)
			}
		})
	}
}

func TestHeartbeatInterval(t *testing.T) {
	tests := []struct {
		s        string
		expected time.Duration
		err      bool
	}{
		{s: "heartbeatIntervalMS=10", expected: time.Duration(10) * time.Millisecond},
		{s: "heartbeatIntervalMS=100", expected: time.Duration(100) * time.Millisecond},
		{s: "heartbeatIntervalMS=-2", err: true},
		{s: "heartbeatIntervalMS=gsdge", err: true},
	}

	for _, test := range tests {
		s := fmt.Sprintf("mongodb://localhost/?%s", test.s)
		t.Run(s, func(t *testing.T) {
			cs, err := connstring.Parse(s)
			if test.err {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, test.expected, cs.HeartbeatInterval)
			}
		})
	}
}

func TestMaxConnIdleTime(t *testing.T) {
	tests := []struct {
		s        string
		expected time.Duration
		err      bool
	}{
		{s: "maxIdleTimeMS=10", expected: time.Duration(10) * time.Millisecond},
		{s: "maxIdleTimeMS=100", expected: time.Duration(100) * time.Millisecond},
		{s: "maxIdleTimeMS=-2", err: true},
		{s: "maxIdleTimeMS=gsdge", err: true},
	}

	for _, test := range tests {
		s := fmt.Sprintf("mongodb://localhost/?%s", test.s)
		t.Run(s, func(t *testing.T) {
			cs, err := connstring.Parse(s)
			if test.err {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, test.expected, cs.MaxConnIdleTime)
			}
		})
	}
}

func TestMaxConnLifeTime(t *testing.T) {
	tests := []struct {
		s        string
		expected time.Duration
		err      bool
	}{
		{s: "maxLifeTimeMS=10", expected: time.Duration(10) * time.Millisecond},
		{s: "maxLifeTimeMS=100", expected: time.Duration(100) * time.Millisecond},
		{s: "maxLifeTimeMS=-2", err: true},
		{s: "maxLifeTimeMS=gsdge", err: true},
	}

	for _, test := range tests {
		s := fmt.Sprintf("mongodb://localhost/?%s", test.s)
		t.Run(s, func(t *testing.T) {
			cs, err := connstring.Parse(s)
			if test.err {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, test.expected, cs.MaxConnLifeTime)
			}
		})
	}
}

func TestMaxConnsPerHost(t *testing.T) {
	tests := []struct {
		s        string
		expected uint16
		err      bool
	}{
		{s: "maxConnsPerHost=10", expected: 10},
		{s: "maxConnsPerHost=100", expected: 100},
		{s: "maxConnsPerHost=-2", err: true},
		{s: "maxConnsPerHost=gsdge", err: true},
	}

	for _, test := range tests {
		s := fmt.Sprintf("mongodb://localhost/?%s", test.s)
		t.Run(s, func(t *testing.T) {
			cs, err := connstring.Parse(s)
			if test.err {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.True(t, cs.MaxConnsPerHostSet)
				require.Equal(t, test.expected, cs.MaxConnsPerHost)
			}
		})
	}
}

func TestMaxIdleConnsPerHost(t *testing.T) {
	tests := []struct {
		s        string
		expected uint16
		err      bool
	}{
		{s: "maxIdleConnsPerHost=10", expected: 10},
		{s: "maxIdleConnsPerHost=100", expected: 100},
		{s: "maxIdleConnsPerHost=-2", err: true},
		{s: "maxIdleConnsPerHost=gsdge", err: true},
	}

	for _, test := range tests {
		s := fmt.Sprintf("mongodb://localhost/?%s", test.s)
		t.Run(s, func(t *testing.T) {
			cs, err := connstring.Parse(s)
			if test.err {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.True(t, cs.MaxIdleConnsPerHostSet)
				require.Equal(t, test.expected, cs.MaxIdleConnsPerHost)
			}
		})
	}
}

func TestMaxPoolSize(t *testing.T) {
	tests := []struct {
		s        string
		expected uint16
		err      bool
	}{
		{s: "maxPoolSize=10", expected: 10},
		{s: "maxPoolSize=100", expected: 100},
		{s: "maxPoolSize=-2", err: true},
		{s: "maxPoolSize=gsdge", err: true},
	}

	for _, test := range tests {
		s := fmt.Sprintf("mongodb://localhost/?%s", test.s)
		t.Run(s, func(t *testing.T) {
			cs, err := connstring.Parse(s)
			if test.err {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.True(t, cs.MaxConnsPerHostSet)
				require.Equal(t, test.expected, cs.MaxConnsPerHost)
				require.True(t, cs.MaxIdleConnsPerHostSet)
				require.Equal(t, test.expected, cs.MaxIdleConnsPerHost)
			}
		})
	}
}

func TestReplicaSet(t *testing.T) {
	tests := []struct {
		s        string
		expected string
		err      bool
	}{
		{s: "replicaSet=auto", expected: "auto"},
		{s: "replicaSet=rs0", expected: "rs0"},
	}

	for _, test := range tests {
		s := fmt.Sprintf("mongodb://localhost/?%s", test.s)
		t.Run(s, func(t *testing.T) {
			cs, err := connstring.Parse(s)
			if test.err {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, test.expected, cs.ReplicaSet)
			}
		})
	}
}

func TestServerSelectionTimeout(t *testing.T) {
	tests := []struct {
		s        string
		expected time.Duration
		err      bool
	}{
		{s: "serverSelectionTimeoutMS=10", expected: time.Duration(10) * time.Millisecond},
		{s: "serverSelectionTimeoutMS=100", expected: time.Duration(100) * time.Millisecond},
		{s: "serverSelectionTimeoutMS=-2", err: true},
		{s: "serverSelectionTimeoutMS=gsdge", err: true},
	}

	for _, test := range tests {
		s := fmt.Sprintf("mongodb://localhost/?%s", test.s)
		t.Run(s, func(t *testing.T) {
			cs, err := connstring.Parse(s)
			if test.err {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, test.expected, cs.ServerSelectionTimeout)
			}
		})
	}
}

func TestWTimeout(t *testing.T) {
	tests := []struct {
		s        string
		expected time.Duration
		err      bool
	}{
		{s: "wtimeoutMS=10", expected: time.Duration(10) * time.Millisecond},
		{s: "wtimeoutMS=100", expected: time.Duration(100) * time.Millisecond},
		{s: "wtimeoutMS=-2", err: true},
		{s: "wtimeoutMS=gsdge", err: true},
	}

	for _, test := range tests {
		s := fmt.Sprintf("mongodb://localhost/?%s", test.s)
		t.Run(s, func(t *testing.T) {
			cs, err := connstring.Parse(s)
			if test.err {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, test.expected, cs.WTimeout)
			}
		})
	}
}
