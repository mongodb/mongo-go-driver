package scram

import (
	"sync"

	"golang.org/x/crypto/pbkdf2"
)

// Client ...
type Client struct {
	sync.RWMutex
	username string
	password string
	authID   string
	minIters int
	nonceGen NonceGeneratorFcn
	hashGen  HashGeneratorFcn
	cache    map[KeyFactors]DerivedKeys
}

func newClient(username, password, authID string, fcn HashGeneratorFcn) *Client {
	return &Client{
		username: username,
		password: password,
		authID:   authID,
		minIters: 4096,
		nonceGen: defaultNonceGenerator,
		hashGen:  fcn,
		cache:    make(map[KeyFactors]DerivedKeys),
	}
}

// WithMinIterations ...
func (c *Client) WithMinIterations(n int) *Client {
	c.Lock()
	defer c.Unlock()
	c.minIters = n
	return c
}

// WithNonceGenerator ...
func (c *Client) WithNonceGenerator(ng NonceGeneratorFcn) *Client {
	c.Lock()
	defer c.Unlock()
	c.nonceGen = ng
	return c
}

// NewConversation ...
func (c *Client) NewConversation() *ClientConversation {
	c.RLock()
	defer c.RUnlock()
	return &ClientConversation{
		client:   c,
		nonceGen: c.nonceGen,
		hashGen:  c.hashGen,
		minIters: c.minIters,
	}
}

// GetDerivedKeys ...
func (c *Client) GetDerivedKeys(kf KeyFactors) DerivedKeys {
	dk, ok := c.getCache(kf)
	if !ok {
		dk = c.computeKeys(kf)
		c.setCache(kf, dk)
	}
	return dk
}

// GetStoredCredentials ...
func (c *Client) GetStoredCredentials(kf KeyFactors) StoredCredentials {
	dk := c.GetDerivedKeys(kf)
	return StoredCredentials{
		KeyFactors: kf,
		StoredKey:  dk.StoredKey,
		ServerKey:  dk.ServerKey,
	}
}

func (c *Client) computeKeys(kf KeyFactors) DerivedKeys {
	h := c.hashGen()
	saltedPassword := pbkdf2.Key([]byte(c.password), []byte(kf.Salt), kf.Iters, h.Size(), c.hashGen)
	clientKey := computeHMAC(c.hashGen, saltedPassword, []byte("Client Key"))

	return DerivedKeys{
		ClientKey: clientKey,
		StoredKey: computeHash(c.hashGen, clientKey),
		ServerKey: computeHMAC(c.hashGen, saltedPassword, []byte("Server Key")),
	}
}

func (c *Client) getCache(kf KeyFactors) (DerivedKeys, bool) {
	c.RLock()
	defer c.RUnlock()
	dk, ok := c.cache[kf]
	return dk, ok
}

func (c *Client) setCache(kf KeyFactors, dk DerivedKeys) {
	c.Lock()
	defer c.Unlock()
	c.cache[kf] = dk
	return
}
