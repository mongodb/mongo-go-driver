package main

const retryExecuteTmpl = `
// RetryExecute retries this operation against the provided server. This method should only be
// called after a retryable error is returned from either SelectAndExecute or Execute.
func ({{.Receiver}} *InsertOperation) RetryExecute(ctx context.Context, srvr Server, original error) error {
	conn, err := srvr.Connection(ctx)
	// Return original error if connection retrieval fails or new server does not support retryable writes.
	{{ with $writeConcern := or .WriteConcernSelector "nil"}}
	if err != nil || conn == nil || !retrySupported({{$.Receiver}}.{{$.Deployment}}.Description(), conn.Description(), {{$.Receiver}}.{{$.ClientSession}}, {{$writeConcern}}) {
		return original
	}
	{{ end }}
	defer conn.Close()

	return {{.Receiver}}.execute(ctx, conn)
}
`

type RetryExecuteData struct {
	Receiver      string // receiver name
	Type          string // type name
	ClientSession string // ClientSession name
	WriteConcern  string // write concern name
	Deployment    string // Deployment name
}

func (data RetryExecuteData) WriteConcernSelector() string {
	if data.WriteConcern == "" {
		return ""
	}
	return data.Receiver + "." + data.WriteConcern
}
