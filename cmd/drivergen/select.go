package main

const selectTmpl = `
// Select retrieves a server to be used when executing an operation.
func ({{.Receiver}} *{{.Type}}) Select(ctx context.Context) (Server, error) {
	if {{.Receiver}}.{{.Deployment}} == nil {
		return nil, errors.New("{{.Type}} must have a Deployment set before Select can be called.")
	}
	{{ with $readPref := or .ReadPrefSelector "readpref.Primary()" }}
	return {{$.Receiver}}.{{$.Deployment}}.SelectServer(ctx, createReadPrefSelector({{$readPref}}, {{$.Receiver}}.{{$.Selector}})){{ end }}
}
`

type SelectData struct {
	Receiver   string // receiver name
	Type       string // type name
	Deployment string // field name of Deployment
	Selector   string // field name of ServerSelector
	ReadPref   string // field name of *readpref.ReadPref
}

func (sd SelectData) ReadPrefSelector() string {
	if sd.ReadPref == "" {
		return ""
	}
	return sd.Receiver + "." + sd.ReadPref
}
