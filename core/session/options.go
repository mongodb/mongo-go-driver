package session

// ClientOptioner is the interface implemented by types that can be used as options for configuring a client session.
type ClientOptioner interface {
	Option(*Client) error
}

// OptCausalConsistency specifies if a session should be causally consistent.
type OptCausalConsistency bool

// Option implements the ClientOptioner interface.
func (opt OptCausalConsistency) Option(c *Client) error {
	c.Consistent = bool(opt)
	return nil
}
