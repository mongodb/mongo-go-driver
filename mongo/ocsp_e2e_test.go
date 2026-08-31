// Copyright (C) MongoDB, Inc. 2026-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"math/big"
	"net"
	"sync"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/internal/driverutil"
	"go.mongodb.org/mongo-driver/v2/internal/require"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
	"go.mongodb.org/mongo-driver/v2/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/wiremessage"
	"golang.org/x/crypto/ocsp"
)

// TestOCSPStapleSigners pins the behavior of the driver's built-in OCSP verification for each RFC 6960 4.2.2.2 signer
// shape, connecting a default client over a real TLS handshake to a server stapling each one.
//
// It is the baseline that TestOCSPDisableCertificateRevocationCheck below is measured against: the same four fixtures
// have the opposite expectation once revocation checking is switched off.
func TestOCSPStapleSigners(t *testing.T) {
	t.Parallel()

	pki, err := getOCSPPKI()
	require.NoError(t, err, "getOCSPPKI error: %v", err)

	// Expected outcome per fixture against a default client, where the empty string means the connection is expected
	// to succeed. embed_ca is rejected even though RFC 6960 4.2.2.2 permits it: x/crypto/ocsp assumes any certificate
	// embedded in the response is a delegated responder signed by the issuer, so it rejects a response the issuing CA
	// signed with its own certificate embedded. See golang/go#59641 and GODRIVER-4101. forged must stay rejected --
	// it is signed by a key unrelated to the issuer.
	wantErr := map[ocspFixture]string{
		fixtureEmbedCA:   "bad OCSP signature",
		fixtureNoCerts:   "",
		fixtureDelegated: "",
		fixtureForged:    "bad OCSP signature",
	}

	for _, fixture := range ocspFixtures {
		t.Run(string(fixture), func(t *testing.T) {
			t.Parallel()

			srv, err := pki.StartServer(fixture)
			require.NoError(t, err, "StartServer error: %v", err)
			defer srv.Close()

			clientOpts := options.Client().
				ApplyURI(srv.URI()).
				SetTLSConfig(&tls.Config{RootCAs: pki.RootPool(), MinVersion: tls.VersionTLS12}).
				// Only the rejected fixtures wait out this timeout; the others select a server in a few milliseconds.
				SetServerSelectionTimeout(time.Second)

			client, err := Connect(clientOpts)
			require.NoError(t, err, "Connect error: %v", err)
			defer func() { _ = client.Disconnect(bgCtx) }()

			err = client.Ping(bgCtx, readpref.Primary())

			if want := wantErr[fixture]; want != "" {
				// Log the error so the failure mode is visible when this runs in CI.
				t.Logf("got expected Ping error for %s: %v", fixture, err)
				require.Error(t, err, "expected Ping to fail for %s (%s)", string(fixture), fixture)
				require.ErrorContains(t, err, want,
					"expected the %s rejection to name the OCSP failure", string(fixture))

				return
			}

			require.NoError(t, err, "expected Ping to succeed for %s (%s), got: %v", string(fixture), fixture, err)
		})
	}
}

// TestOCSPDisableCertificateRevocationCheck verifies that tlsDisableCertificateRevocationCheck suppresses OCSP
// verification.
func TestOCSPDisableCertificateRevocationCheck(t *testing.T) {
	t.Parallel()

	pki, err := getOCSPPKI()
	require.NoError(t, err, "GetPKI error: %v", err)

	for _, fixture := range ocspFixtures {
		t.Run(string(fixture), func(t *testing.T) {
			t.Parallel()

			srv, err := pki.StartServer(fixture)
			require.NoError(t, err, "StartServer error: %v", err)
			defer srv.Close()

			clientOpts := options.Client().
				ApplyURI(srv.URI() + "&tlsDisableCertificateRevocationCheck=true").
				SetTLSConfig(&tls.Config{RootCAs: pki.RootPool(), MinVersion: tls.VersionTLS12}).
				SetServerSelectionTimeout(time.Second)

			client, err := Connect(clientOpts)
			require.NoError(t, err, "Connect error: %v", err)
			defer func() { _ = client.Disconnect(bgCtx) }()

			err = client.Ping(bgCtx, readpref.Primary())
			require.NoError(t, err, "expected Ping to succeed for %s (%s) with revocation checking disabled, got: %v",
				fixture, fixture.String(), err)
		})
	}
}

// The remainder of this file is a self-contained OCSP reproduction harness: a certificate
// hierarchy, OCSP responses covering each RFC 6960 4.2.2.2 signer shape, and a fake
// TLS-enabled MongoDB server that staples a chosen response.

// ocspFixture names the shape of the OCSP response a server staples.
type ocspFixture string

const (
	// fixtureEmbedCA is the customer's shape: the issuing CA signs the response and embeds
	// its own certificate in the optional certs field.
	fixtureEmbedCA ocspFixture = "embed_ca"

	// fixtureNoCerts is the issuing CA signing the response with nothing embedded.
	fixtureNoCerts ocspFixture = "no_certs"

	// fixtureDelegated is a delegated responder carrying id-kp-OCSPSigning, embedded.
	fixtureDelegated ocspFixture = "delegated"

	// fixtureForged is signed by a key unrelated to the issuer, with nothing embedded. It
	// must always be rejected.
	fixtureForged ocspFixture = "forged"
)

// ocspFixtures lists every fixture, in the order the reproduction reports them.
var ocspFixtures = []ocspFixture{fixtureEmbedCA, fixtureNoCerts, fixtureDelegated, fixtureForged}

// String returns a human-readable description of the fixture's signer shape.
func (f ocspFixture) String() string {
	switch f {
	case fixtureEmbedCA:
		return "issuing CA signs, CA's own certificate embedded"
	case fixtureNoCerts:
		return "issuing CA signs, nothing embedded"
	case fixtureDelegated:
		return "delegate with id-kp-OCSPSigning signs, delegate embedded"
	case fixtureForged:
		return "unrelated key signs, nothing embedded"
	}
	return string(f)
}

// ocspPKI is a certificate hierarchy covering the signer shapes above.
//
//	root --> intermediate --> leaf (CN=localhost, the server certificate)
//	                      |-> delegate (id-kp-OCSPSigning)
//	foreignRoot (unrelated, used to forge responses)
type ocspPKI struct {
	RootCert  *x509.Certificate
	InterCert *x509.Certificate
	interKey  *rsa.PrivateKey
	leafCert  *x509.Certificate
	leafKey   *rsa.PrivateKey

	delegateCert *x509.Certificate
	delegateKey  *rsa.PrivateKey

	foreignRootCert *x509.Certificate
	foreignRootKey  *rsa.PrivateKey
}

var (
	pkiOnce       sync.Once
	pki           *ocspPKI
	pkiErr        error
	serialCounter int64
)

// getOCSPPKI builds the certificate hierarchy on first call and returns the same one
// thereafter. Generating RSA keys dominates the runtime of the reproduction, so it is done
// only once per process.
func getOCSPPKI() (*ocspPKI, error) {
	pkiOnce.Do(func() {
		pki, pkiErr = buildPKI()
	})
	return pki, pkiErr
}

// RootPool returns a certificate pool containing the test root, for use as the client's
// RootCAs.
func (p *ocspPKI) RootPool() *x509.CertPool {
	pool := x509.NewCertPool()
	pool.AddCert(p.RootCert)
	return pool
}

func buildPKI() (*ocspPKI, error) {
	p := &ocspPKI{}
	var err error

	caTemplate := func(cn string) *x509.Certificate {
		return &x509.Certificate{
			Subject:               pkix.Name{CommonName: cn},
			IsCA:                  true,
			BasicConstraintsValid: true,
			KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		}
	}

	var rootKey *rsa.PrivateKey
	if p.RootCert, rootKey, err = issueCert(caTemplate("Test Root CA"), nil, nil); err != nil {
		return nil, err
	}
	if p.InterCert, p.interKey, err = issueCert(caTemplate("Test Intermediate CA"), p.RootCert, rootKey); err != nil {
		return nil, err
	}

	// The server certificate. It must carry localhost as a SAN so the driver's hostname
	// verification passes and a verified chain is available for OCSP.
	leafTmpl := &x509.Certificate{
		Subject:     pkix.Name{CommonName: "localhost"},
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:    []string{"localhost"},
		IPAddresses: []net.IP{net.ParseIP("127.0.0.1")},
	}
	if p.leafCert, p.leafKey, err = issueCert(leafTmpl, p.InterCert, p.interKey); err != nil {
		return nil, err
	}

	delegateTmpl := &x509.Certificate{
		Subject:     pkix.Name{CommonName: "Test Delegate Responder"},
		KeyUsage:    x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageOCSPSigning},
	}
	if p.delegateCert, p.delegateKey, err = issueCert(delegateTmpl, p.InterCert, p.interKey); err != nil {
		return nil, err
	}

	// An unrelated CA, used only to forge responses that must never be accepted.
	if p.foreignRootCert, p.foreignRootKey, err = issueCert(caTemplate("Foreign Root CA"), nil, nil); err != nil {
		return nil, err
	}

	return p, nil
}

func issueCert(tmpl, parent *x509.Certificate, parentKey *rsa.PrivateKey) (*x509.Certificate, *rsa.PrivateKey, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}

	serialCounter++
	tmpl.SerialNumber = big.NewInt(serialCounter)
	tmpl.NotBefore = time.Now().Add(-time.Hour)
	tmpl.NotAfter = time.Now().Add(24 * time.Hour)

	signer, signerKey := parent, parentKey
	if signer == nil {
		signer, signerKey = tmpl, key
	}

	der, err := x509.CreateCertificate(rand.Reader, tmpl, signer, &key.PublicKey, signerKey)
	if err != nil {
		return nil, nil, err
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, nil, err
	}
	return cert, key, nil
}

// Staple builds the OCSP response for the given fixture, reporting the leaf certificate as
// good. The certs field of the response is what distinguishes the fixtures, and is what
// x/crypto/ocsp gets wrong.
func (p *ocspPKI) Staple(f ocspFixture) ([]byte, error) {
	tmpl := ocsp.Response{
		Status:       ocsp.Good,
		SerialNumber: p.leafCert.SerialNumber,
		ThisUpdate:   time.Now().Add(-time.Hour),
		NextUpdate:   time.Now().Add(time.Hour),
	}

	responderCert, signerKey := p.InterCert, p.interKey
	switch f {
	case fixtureEmbedCA:
		tmpl.Certificate = p.InterCert
	case fixtureNoCerts:
	case fixtureDelegated:
		responderCert, signerKey = p.delegateCert, p.delegateKey
		tmpl.Certificate = p.delegateCert
	case fixtureForged:
		responderCert, signerKey = p.foreignRootCert, p.foreignRootKey
	default:
		return nil, fmt.Errorf("unknown fixture %q", f)
	}

	return ocsp.CreateResponse(p.InterCert, responderCert, tmpl, signerKey)
}

// ocspServer is a fake TLS-enabled MongoDB server that staples a fixed OCSP response.
type ocspServer struct {
	listener net.Listener
	wg       sync.WaitGroup
}

// StartServer starts a fake mongod on localhost that presents the leaf certificate and
// staples the response for the given fixture. Close must be called when the caller is done.
func (p *ocspPKI) StartServer(f ocspFixture) (*ocspServer, error) {
	staple, err := p.Staple(f)
	if err != nil {
		return nil, err
	}

	tlsCert := tls.Certificate{
		Certificate: [][]byte{p.leafCert.Raw, p.InterCert.Raw},
		PrivateKey:  p.leafKey,
		OCSPStaple:  staple,
	}

	// Listen on 127.0.0.1 but expect clients to dial "localhost" so the leaf's SAN matches.
	listener, err := tls.Listen("tcp", "127.0.0.1:0", &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
		MinVersion:   tls.VersionTLS12,
	})
	if err != nil {
		return nil, err
	}

	srv := &ocspServer{listener: listener}
	srv.wg.Add(1)
	go func() {
		defer srv.wg.Done()

		for {
			conn, err := srv.listener.Accept()
			if err != nil {
				return
			}

			srv.wg.Add(1)
			go func() {
				defer srv.wg.Done()
				defer conn.Close()
				serveConn(conn)
			}()
		}
	}()
	return srv, nil
}

// URI returns a connection string for the server with TLS enabled.
func (s *ocspServer) URI() string {
	_, port, _ := net.SplitHostPort(s.listener.Addr().String())
	return "mongodb://" + net.JoinHostPort("localhost", port) + "/?tls=true"
}

// Close stops the server and waits for its goroutines to exit.
func (s *ocspServer) Close() {
	_ = s.listener.Close()
	s.wg.Wait()
}

var (
	helloDoc = bsoncore.NewDocumentBuilder().
			AppendInt32("ok", 1).
			AppendBoolean("isWritablePrimary", true).
			AppendBoolean("helloOk", true).
			AppendInt32("minWireVersion", driverutil.MinWireVersion).
			AppendInt32("maxWireVersion", driverutil.MaxWireVersion).
			AppendInt32("maxBsonObjectSize", 16*1024*1024).
			AppendInt32("maxMessageSizeBytes", 48*1000*1000).
			AppendInt32("maxWriteBatchSize", 100000).
			AppendDateTime("localTime", time.Now().UnixMilli()).
			AppendInt32("logicalSessionTimeoutMinutes", 30).
			Build()

	okDoc = bsoncore.NewDocumentBuilder().AppendInt32("ok", 1).Build()
)

func serveConn(conn net.Conn) {
	for {
		wm, err := readWireMessage(conn)
		if err != nil {
			return
		}

		_, reqID, _, opcode, rem, ok := wiremessage.ReadHeader(wm)
		if !ok {
			return
		}

		var reply []byte
		switch opcode {
		case wiremessage.OpQuery:
			reply = makeReply(reqID, helloDoc)
		case wiremessage.OpMsg:
			reply = makeMsg(reqID, commandReply(rem))
		default:
			return
		}

		if _, err := conn.Write(reply); err != nil {
			return
		}
	}
}

func commandReply(rem []byte) bsoncore.Document {
	_, rem, ok := wiremessage.ReadMsgFlags(rem)
	if !ok {
		return okDoc
	}
	stype, rem, ok := wiremessage.ReadMsgSectionType(rem)
	if !ok || stype != wiremessage.SingleDocument {
		return okDoc
	}
	doc, _, ok := wiremessage.ReadMsgSectionSingleDocument(rem)
	if !ok {
		return okDoc
	}

	elems, err := doc.Elements()
	if err != nil || len(elems) == 0 {
		return okDoc
	}

	switch elems[0].Key() {
	case "hello", "isMaster", "ismaster":
		return helloDoc
	}
	return okDoc
}

func readWireMessage(conn net.Conn) ([]byte, error) {
	header := make([]byte, 4)
	if err := readFull(conn, header); err != nil {
		return nil, err
	}

	length := int32(header[0]) | int32(header[1])<<8 | int32(header[2])<<16 | int32(header[3])<<24
	if length < 4 || length > 48*1000*1000 {
		return nil, fmt.Errorf("invalid wire message length %d", length)
	}

	wm := make([]byte, length)
	copy(wm, header)
	if err := readFull(conn, wm[4:]); err != nil {
		return nil, err
	}
	return wm, nil
}

func readFull(conn net.Conn, buf []byte) error {
	var read int
	for read < len(buf) {
		n, err := conn.Read(buf[read:])
		read += n
		if err != nil {
			return err
		}
	}
	return nil
}

func makeReply(respTo int32, doc bsoncore.Document) []byte {
	var dst []byte
	idx, dst := wiremessage.AppendHeaderStart(dst, wiremessage.NextRequestID(), respTo, wiremessage.OpReply)
	dst = wiremessage.AppendReplyFlags(dst, wiremessage.AwaitCapable)
	dst = wiremessage.AppendReplyCursorID(dst, 0)
	dst = wiremessage.AppendReplyStartingFrom(dst, 0)
	dst = wiremessage.AppendReplyNumberReturned(dst, 1)
	dst = append(dst, doc...)
	return bsoncore.UpdateLength(dst, idx, int32(len(dst[idx:])))
}

func makeMsg(respTo int32, doc bsoncore.Document) []byte {
	var dst []byte
	idx, dst := wiremessage.AppendHeaderStart(dst, wiremessage.NextRequestID(), respTo, wiremessage.OpMsg)
	dst = wiremessage.AppendMsgFlags(dst, 0)
	dst = wiremessage.AppendMsgSectionType(dst, wiremessage.SingleDocument)
	dst = append(dst, doc...)
	return bsoncore.UpdateLength(dst, idx, int32(len(dst[idx:])))
}
