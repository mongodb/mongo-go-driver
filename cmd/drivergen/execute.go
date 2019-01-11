package main

const executeTmpl = `
{{.Documentation}}
func ({{.Receiver}} *{{.Type}}) Execute(ctx context.Context) error {
	{{ if .Deployment }}
	if {{.Receiver}}.{{.Deployment}} == nil {
		return errors.New("{{.Type}} must have a Deployment set before Execute can be called.")
	}{{end}}
	{{ if .Server }}
	if {{.Receiver}}.{{.Server}} == nil {
		return errors.New("{{.Type}} must have a Server set before Execute can be called.")
	}{{end}}

	{{ if .WriteConcern }}
	if {{$.Receiver}}.{{$.ClientSession}} != nil && !writeconcern.AckWrite({{$.Receiver}}.{{$.WriteConcern}}) {
		return errors.New("session provided for an unacknowledged write")
	}{{ end }}
	if {{.Receiver}}.{{.Database}} == "" {
		return errors.New("database must be of non-zero length")
	}
	{{ if .Collection }}
	if {{.Receiver}}.{{.Collection}} == "" {
		return errors.New("collection must be of non-zero length")
	}
	return OperationContext{
		CommandFn: {{.Receiver}}.{{.CommandFn}},
		Database: {{.Receiver}}.{{.Database}},

		{{if .Deployment}}Deployment: {{.Receiver}}.{{.Deployment}},{{end}}
		{{if .Server}}Server: {{.Receiver}}.{{.Server}},{{end}}
		{{if .TopologyKind}}TopologyKind: {{.Receiver}}.{{.TopologyKind}},{{end}}

		{{if .ProcessResponseFn}}ProcessResponseFn: {{.Receiver}}.{{.ProcessResponseFn}},{{end}}
		{{if .RetryableFn}}RetryableFn: {{.Receiver}}.{{.RetryableFn}},{{end}}

		{{if .Selector}}Selector: {{.Receiver}}.{{.Selector}},{{end}}
		{{if .ReadPreference}}ReadPreference: {{.Receiver}}.{{.ReadPreference}},{{end}}
		{{if .ReadConcern}}ReadConcern: {{.Receiver}}.{{.ReadConcern}},{{end}}
		{{if .WriteConcern}}WriteConcern: {{.Receiver}}.{{.WriteConcern}},{{end}}

		{{if .ClientSession}}Client: {{.Receiver}}.{{.ClientSession}},{{end}}
		{{if .ClusterClock}}Clock: {{.Receiver}}.{{.ClusterClock}},{{end}}

		{{if .RetryMode}}RetryMode: {{.Receiver}}.{{.RetryMode}},{{end}}
		{{if .Batches}}Batches: &{{.BatchesType}}{
				Identifier: "{{.Identifier}}",
				Documents: {{.Receiver}}.{{.Documents}},
				Ordered: {{.Receiver}}.{{.Ordered}},
		},{{end}}
	}.Execute(ctx)
}
`

type ExecuteData struct {
	// required fields
	Receiver  string // receiver name
	Database  string // database name
	Type      string // type name
	CommandFn string // command method name

	// Either Deployment OR Server, ServerSelector, & TopologyKind must be set.
	Deployment   string // Deployment name
	Server       string // Server name
	TopologyKind string // TopologyKind name

	// optional fields
	Collection        string // collection name
	ProcessResponseFn string // response processing method
	RetryableFn       string // retryable determination method
	Selector          string // description server selector method
	ReadPreference    string // read preference name
	ReadConcern       string // read concern name
	WriteConcern      string // write concern name
	ClientSession     string // ClientSession name
	ClusterClock      string // ClusterClock name
	RetryMode         string // RetryMode name

	// If Batches is true, then all of the fields in this group must be set
	Batches     bool   // enables batching support
	BatchesType string // The full name for the driver.Batches type, should be Batches if generating w/in package, and driver.Batches for external
	Identifier  string // identifier for type 1 payload or command parameter name
	Documents   string // field name for documents to batch split
	Ordered     string // field name for ordering
}

func (data ExecuteData) Documentation() string {
	str := `// Execute runs this operation.`
	if data.RetryMode != "" {
		str += `
// This method automatically handles retrying retryable errors when retry is enabled.
`
	}
	return str
}
