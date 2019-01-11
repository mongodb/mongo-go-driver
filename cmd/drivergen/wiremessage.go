package main

const commandArgDocumentTmpl = `
if {{.Receiver}}.{{.Name}} != nil {
	dst = bsoncore.AppendDocumentElement(dst, "{{.CommandArgName}}", {{.Receiver}}.{{.Name}})
}
`

const commandArgValueTmpl = `
if {{.Receiver}}.{{.Name}}.Type != bsontype.Type(0) {
	dst = bsoncore.AppendHeader(dst, {{.Receiver}}.{{.Name}}.Type, "{{.CommandArgName}}")
	dst = append(dst, {{.Receiver}}.{{.Name}}.Value...)
}
`

const commandArgInt64Tmpl = `
if {{.Receiver}}.{{.Name}} != nil {
	dst = bsoncore.AppendInt64Element(dst, "{{.CommandArgName}}", *{{.Receiver}}.{{.Name}})
}
`

const commandArgBoolTmpl = `
if {{.Receiver}}.{{.Name}} != nil {
	dst = bsoncore.AppendBooleanElement(dst, "{{.CommandArgName}}", *{{.Receiver}}.{{.Name}})
}
`

const commandArgStringTmpl = `
if {{.Receiver}}.{{.Name}} != nil {
	dst = bsoncore.AppendStringElement(dst, "{{.CommandArgName}}", *{{.Receiver}}.{{.Name}})
}
`

const commandHeaderCollectionTmpl = `
dst = bsoncore.AppendStringElement(dst, "{{.CommandName}}", {{.Receiver}}.{{.Namespace}}.Collection)
`

const commandHeaderDatabaseTmpl = `
header := bsoncore.Value{Type: bsontype.String, Data: bsoncore.AppendString(dst, {{.Receiver}}.{{.Namespace}}.Collection)}
if {{.Receiver}}.{{.Namespace}}.Collection == "" {
	header = bsoncore.Value{Type: bsontype.Int32, Data: []byte{0x01, 0x00, 0x00, 0x00}}
}
dst = bsoncore.AppendHeader(dst, header.Type, {{.CommandName}})
dst = append(dst, header.Data...)
`

type CommandHeaderData struct {
	Receiver    string // receiver name
	Namespace   string // namespace field name
	CommandName string // command name
}

type CommandArgData struct {
	Receiver       string // receiver name
	Name           string // field
	CommandArgName string // name of this argument when provided to a command
}

const createMsgWireMessageTmpl = `
func ({{.Receiver}} {{.TypeName}}) createMsgWireMessage(dst []byte, desc description.SelectedServer) ([]byte, error) {
	var flags wiremessage.MsgFlag
	var wmindex int32
	wmindex, dst = wiremessagex.AppendHeaderStart(dst, wiremessage.NextRequestID(), 0, wiremessage.OpMsg)
	dst = wiremessagex.AppendMsgFlags(dst, flags)

	dst = wiremessagex.AppendMsgSectionType(dst, wiremessage.SingleDocument)
	idx, dst := bsoncore.AppendDocumentStart(dst)

	// TODO(GODRIVER-617): Replace this with a subtemplate for setting the command name and
	// collection. This will differ for some commands, e.g. aggregate on a database has the value as
	// an int32 of 1.
	dst = bsoncore.AppendStringElement(dst, "{{.CommandName}}", {{.Receiver}}.{{.Namespace}}.Collection)

	{{ range .Parameters }}
	{{ end }}
	{{ if .ReadConcern }}
	{{ with $session := or .SessionSelector "nil" }}
	dst, err := addReadConcern(dst, {{$.Receiver}}.{{$.ReadConcern}}, {{$session}}, desc){{end}}
	if err != nil {
		return dst, err
	}
	{{ end }}

	{{ if .WriteConcern }}
	dst, err := addWriteConcern(dst, {{.Receiver}}.{{.WriteConcern}})
	if err != nil {
		return dst, err
	}
	{{ end }}
	{{ if $session := .SessionSelector }}
	dst, err = addSession(dst, {{$session}}, desc)
	if err != nil {
		return dst, err
	}
	{{ if .RetryWrite }}
	if {{$session}}.Retry {
		dst = bsoncore.AppendInt64Element(dst, "txnNumber", {{$session}}.TxnNumber)
	}
	{{ with $clock := or .Clock "nil" }}
	dst = addClusterTime(dst, {{$session}}, {{ $clock }}, desc)
	{{ end }}
	{{ end }}

	dst = bsoncore.AppendStringElement(dst, "$db", {{.Receiver}}.{{.Namespace}}.DB)

	dst, _ = bsoncore.AppendDocumentEnd(dst, idx)

	{{ range .Sequences }}
	dst = wiremessagex.AppendMsgSectionType(dst, wiremessage.DocumentSequence)
	idx, dst = bsoncore.ReserveLength(dst)

	dst = append(dst, {{.Name}}[:]...)

	for _, doc := range {{$.Receiver}}.{{.Field}} {
		dst = append(dst, doc...)
	}

	dst = bsoncore.UpdateLength(dst, idx, int32(len(dst[idx:])))
	{{ end }}

	return bsoncore.UpdateLength(dst, wmindex, int32(len(dst[wmindex:]))), nil
}
`

type CreateMsgWireMessageData struct {
	Receiver      string                    // receiver name
	Type          string                    // type name
	CommandHeader string                    // command name and value, e.g. collection name or 1 for database commands
	CommandArgs   []string                  // arguments for the command, excluding the known types and document sequences.
	WriteConcern  string                    // write concern field name
	ReadConcern   string                    // read concern field name
	ClientSession string                    // session.Client field name
	ClusterClock  string                    // session.ClusterClock field name
	Retry         string                    // retryable field name
	Sequences     []MsgDocumentSequenceData // document sequences
}

func (data CreateMsgWireMessageData) SessionSelector() string {
	if data.ClientSession == "" {
		return ""
	}
	return data.Receiver + "." + data.ClientSession
}

type MsgDocumentSequenceData struct {
	Identifier string // identifier for the document sequence
	Name       string // field name for document sequence
}
