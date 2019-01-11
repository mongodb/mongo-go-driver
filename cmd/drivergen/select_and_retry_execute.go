package main

const selectAndRetryExecuteTmpl = `
// SelectAndExecute selects a server and retries this operation against it. This is a convenience
// method and should only be called after a retryable error is returned from SelectAndExecute or
// Execute.
func ({{.Receiver}} *{{.Type}}) SelectAndRetryExecute(ctx context.Context, original error) error {
	if {{.Receiver}}.d == nil {
		return errors.New("{{.Type}} must have a Deployment set before RetryExecute can be called.")
	}
	srvr, err := {{.Receiver}}.Select(ctx)

	// Return original error if server selection fails.
	if err != nil {
		return original
	}

	return {{.Receiver}}.RetryExecute(ctx, srvr, original)
}
`

type SelectAndRetryExecuteData struct {
	Receiver string // receiver name
	Type     string // type name
}
