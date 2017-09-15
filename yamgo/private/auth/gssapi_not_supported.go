//+build gssapi,!windows,!linux,!darwin

package auth

import (
	"fmt"
	"runtime"
)

// GSSAPI is the mechanism name for GSSAPI.
const GSSAPI = "GSSAPI"

func newGSSAPIAuthenticator(cred *Cred) (Authenticator, error) {
	return nil, fmt.Errorf("GSSAPI is not supported on %s", runtime.GOOS)
}
