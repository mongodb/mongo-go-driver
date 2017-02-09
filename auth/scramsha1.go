package auth

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"fmt"
	"io"
	"math/rand"
	"strconv"

	"golang.org/x/crypto/pbkdf2"

	"strings"

	"encoding/base64"

	"github.com/10gen/mongo-go-driver/core"
)

const scramSHA1 = "SCRAM-SHA-1"
const scramSHA1NonceLen = 24

var usernameSanitizer = strings.NewReplacer("=", "=3D", ",", "=2D")

// ScramSHA1Authenticator uses the SCRAM-SHA-1 algorithm over SASL to authenticate a connection.
type ScramSHA1Authenticator struct {
	DB       string
	Username string
	Password string

	NonceGenerator func([]byte) error
}

// Name returns SCRAM-SHA-1.
func (a *ScramSHA1Authenticator) Name() string {
	return scramSHA1
}

// Auth authenticates the connection.
func (a *ScramSHA1Authenticator) Auth(c core.Connection) error {
	return conductSaslConversation(c, a.DB, &scramSaslClient{
		username:       a.Username,
		password:       a.Password,
		nonceGenerator: a.NonceGenerator,
	})
}

type scramSaslClient struct {
	username       string
	password       string
	nonceGenerator func([]byte) error

	step                   uint8
	clientNonce            []byte
	clientFirstMessageBare string
	serverSignature        []byte
}

func (c *scramSaslClient) Start() (string, []byte, error) {
	if err := c.generateClientNonce(scramSHA1NonceLen); err != nil {
		return scramSHA1, nil, err
	}

	c.clientFirstMessageBare = "n=" + usernameSanitizer.Replace(c.username) + ",r=" + string(c.clientNonce)

	return scramSHA1, []byte("n,," + c.clientFirstMessageBare), nil
}

func (c *scramSaslClient) Next(challenge []byte) ([]byte, error) {
	c.step++
	switch c.step {
	case 1:
		return c.step1(challenge)
	case 2:
		return c.step2(challenge)
	default:
		return nil, fmt.Errorf("unexpected server challenge")
	}
}

func (c *scramSaslClient) step1(challenge []byte) ([]byte, error) {
	fields := bytes.Split(challenge, []byte{','})
	if len(fields) != 3 {
		return nil, fmt.Errorf("expected 3 fields in first server challenge")
	}

	if !bytes.HasPrefix(fields[0], []byte("r=")) || len(fields[0]) < 2 {
		return nil, fmt.Errorf("invalid nonce")
	}
	r := fields[0][2:]
	if !bytes.HasPrefix(r, c.clientNonce) {
		return nil, fmt.Errorf("invalid server nonce")
	}

	if !bytes.HasPrefix(fields[1], []byte("s=")) || len(fields[1]) < 6 {
		return nil, fmt.Errorf("invalid salt")
	}
	s := make([]byte, base64.StdEncoding.DecodedLen(len(fields[1][2:])))
	n, err := base64.StdEncoding.Decode(s, fields[1][2:])
	if err != nil {
		return nil, fmt.Errorf("invalid salt")
	}
	s = s[:n]

	if !bytes.HasPrefix(fields[2], []byte("i=")) || len(fields[2]) < 6 {
		return nil, fmt.Errorf("invalid iteration count")
	}
	i, err := strconv.Atoi(string(fields[2][2:]))
	if err != nil {
		return nil, fmt.Errorf("invalid iteration count")
	}

	clientFinalMessageWithoutProof := "c=biws,r=" + string(r)
	authMessage := c.clientFirstMessageBare + "," + string(challenge) + "," + clientFinalMessageWithoutProof
	saltedPassword := pbkdf2.Key([]byte(mongoPasswordDigest(c.username, c.password)), s, i, 20, sha1.New)
	clientKey := c.hmac(saltedPassword, "Client Key")
	storedKey := c.h(clientKey)
	clientSignature := c.hmac(storedKey, authMessage)
	clientProof := c.xor(clientKey, clientSignature)
	serverKey := c.hmac(saltedPassword, "Server Key")
	c.serverSignature = c.hmac(serverKey, authMessage)

	proof := "p=" + base64.StdEncoding.EncodeToString(clientProof)
	clientFinalMessage := clientFinalMessageWithoutProof + "," + proof

	return []byte(clientFinalMessage), nil
}

func (c *scramSaslClient) step2(challenge []byte) ([]byte, error) {
	var v, e bool
	fields := bytes.Split(challenge, []byte{','})
	if len(fields) == 1 {
		v = bytes.HasPrefix(fields[0], []byte("v="))
		e = bytes.HasPrefix(fields[0], []byte("e="))
	}
	if e {
		return nil, fmt.Errorf(string(fields[0][2:]))
	}
	if !v {
		return nil, fmt.Errorf("invalid final message")
	}

	if !bytes.Equal(c.serverSignature, fields[0][2:]) {
		return nil, fmt.Errorf("invalid server signature")
	}

	return nil, nil
}

func (c *scramSaslClient) generateClientNonce(n uint8) error {
	if c.nonceGenerator != nil {
		c.clientNonce = make([]byte, n)
		return c.nonceGenerator(c.clientNonce)
	}

	buf := make([]byte, n)
	rand.Read(buf)

	c.clientNonce = make([]byte, base64.StdEncoding.EncodedLen(int(n)))
	base64.StdEncoding.Encode(c.clientNonce, buf)
	return nil
}

func (c *scramSaslClient) h(data []byte) []byte {
	h := sha1.New()
	h.Write(data)
	return h.Sum(nil)
}

func (c *scramSaslClient) hmac(data []byte, key string) []byte {
	h := hmac.New(sha1.New, data)
	io.WriteString(h, key)
	return h.Sum(nil)
}

func (c *scramSaslClient) xor(a []byte, b []byte) []byte {
	result := make([]byte, len(a))
	for i := 0; i < len(a); i++ {
		result[i] = a[i] ^ b[i]
	}
	return result
}
