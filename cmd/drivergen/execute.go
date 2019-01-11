package main

const executeTmpl = `
{{.Documentation}}
func ({{.Receiver}} *{{.Type}}) Execute(ctx context.Context, srvr Server) error {
	if {{.Receiver}}.{{.Deployment}} == nil {
		return errors.New("{{.Type}} must have a Deployment set before Execute can be called.")
	}
	conn, err := srvr.Connection(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	{{ if $writeConcern := .WriteConcernSelector }}
	if {{$.Receiver}}.{{$.ClientSession}} != nil && !writeconcern.AckWrite({{$writeConcern}}) {
		return errors.New("session provided for an unacknowledged write")
	}
	{{ end }}

	{{ if .Retry }}
	desc := conn.Description()
	{{ with $writeConcern := or .WriteConcernSelector "nil" }}
	retryable := ({{$.Receiver}}.{{$.Retry}} != nil && *{{$.Receiver}}.{{$.Retry}} == true) && retrySupported({{$.Receiver}}.{{$.Deployment}}.Description(), desc, {{$.Receiver}}.{{$.ClientSession}}, {{$writeConcern}})
	if retryable {
		// {{$.Receiver}}.{{$.ClientSession}} must not be nil or retrySupported would have returned false
		{{$.Receiver}}.{{$.ClientSession}}.RetryWrite = true
		{{$.Receiver}}.{{$.ClientSession}}.IncrementTxnNumber()
	}
	{{ end }}
	{{ end }}
	return {{.Receiver}}.execute(ctx, conn)
}
`

type ExecuteData struct {
	Receiver      string // receiver name
	Type          string // type name
	Retry         string // retry field name
	ClientSession string // ClientSession name
	WriteConcern  string // write concern name
	Deployment    string // Deployment name
}

func (data ExecuteData) WriteConcernSelector() string {
	if data.WriteConcern == "" {
		return ""
	}
	return data.Receiver + "." + data.WriteConcern
}

func (data ExecuteData) Documentation() string {
	str := `// Execute runs this operation against the provided server.`
	if data.Retry != "" {
		str += `
// If the error returned is retryable, either SelectAndRetryExecute or Select followed by
// RetryExecute can be called to retry this operation.`
	}
	return str
}
