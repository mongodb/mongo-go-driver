package auth_test

import (
	"testing"

	. "github.com/10gen/mongo-go-driver/auth"
	"github.com/stretchr/testify/require"
)

func TestCreateAuthenticator(t *testing.T) {

	tests := []struct {
		name   string
		auther Authenticator
	}{
		{name: "", auther: &DefaultAuthenticator{}},
		{name: "SCRAM-SHA-1", auther: &ScramSHA1Authenticator{}},
		{name: "MONGODB-CR", auther: &MongoDBCRAuthenticator{}},
		{name: "PLAIN", auther: &PlainAuthenticator{}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			a, err := CreateAuthenticator(test.name, "blah", "user", "pencil", nil)
			require.NoError(t, err)
			require.IsType(t, test.auther, a)
		})
	}
}
