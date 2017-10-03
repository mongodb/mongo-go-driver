// In order to test that the SCRAM-SHA client key is cached on successful authentication, we need
// to be able to access the clientKey field on ScramSHA1Authenticator. In order to forgo any
// user-facing API, the IsClientKeyNil() method is defined in a *_test.go file so that it only
// builds with `go test` (and not `go build`).

package auth

func (a *ScramSHA1Authenticator) IsClientKeyNil() bool {
	return a.clientKey == nil
}
