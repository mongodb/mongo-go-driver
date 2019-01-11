package main

const selectAndExecuteTmpl = `
// SelectAndExecute selects a server and runs this operation against it.
func ({{.Receiver}} *{{.Type}}) SelectAndExecute(ctx context.Context) error {
	srvr, err := {{.Receiver}}.Select(ctx)
	if err != nil {
		return err
	}

	return {{.Receiver}}.Execute(ctx, srvr)
}
`

type SelectAndExecuteData struct {
	Receiver string // receiver name
	Type     string // type name
}
