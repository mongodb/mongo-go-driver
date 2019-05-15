package drivergen

import "text/template"

// commandCollectionDatabaseTmpl is the template used to set the command name when the command can
// apply to either a collection or a database.
var commandCollectionDatabaseTmpl = template.Must(template.New("").Parse(
	`header := bsoncore.Value{Type: bsontype.String, Data: bsoncore.AppendString(nil, {{$.ShortName}}.collection)}
if {{$.ShortName}}.collection == "" {
	header = bsoncore.Value{Type: bsontype.Int32, Data: []byte{0x01, 0x00, 0x00, 0x00}}
}
dst = bsoncore.AppendValueElement(dst, "{{$.Command.Name}}", header)
`))

// commandCollectionTmpl is the template used to set the command name when the parameter will be a
// collection.
var commandCollectionTmpl = template.Must(template.New("").Parse(
	`dst = bsoncore.AppendStringElement(dst, "{{$.Command.Name}}", {{$.ShortName}}.collection)
`))

const commandMinWireVersionTmpl = `{{define "minWireVersion"}}{{if $.MinWireVersion}} &&
(desc.WireVersion != nil && desc.WireVersion.Includes({{$.MinWireVersion}})){{end}}{{end}}`

const commandMinWireVersionRequiredTmpl = `{{define "minWireVersionRequired"}}{{if $.MinWireVersionRequired}}
if desc.WireVersion == nil || !desc.WireVersion.Includes({{$.MinWireVersionRequired}}) {
	return nil, errors.New("the '{{$.ParameterName}}' command parameter requires a minimum server wire version of {{$.MinWireVersionRequired}}")
}{{end}}{{end}}`

// parseTemplates parses each tmpl into a single unnamed *template.Template.
func parseTemplates(tmpls ...string) *template.Template {
	t := template.New("")
	for _, tmpl := range tmpls {
		t = template.Must(t.Parse(tmpl))
	}
	return t
}

// parseCommandParamTemplate parses a commandParam template and adds definitions for the minWireVersion and minWireVersionRequired templates.
func parseCommandParamTemplate(tmpl string) *template.Template {
	tmpls := [3]string{tmpl, commandMinWireVersionTmpl, commandMinWireVersionRequiredTmpl}
	return parseTemplates(tmpls[:]...)
}

var commandParamDocumentTmpl = parseCommandParamTemplate(`if {{$.ShortName}}.{{$.Name}} != nil {{template "minWireVersion" $}} {
{{template "minWireVersionRequired" $}}
	dst = bsoncore.AppendDocumentElement(dst, "{{$.ParameterName}}", {{$.ShortName}}.{{$.Name}})
}
`)

var commandParamArrayImpl = parseCommandParamTemplate(`if {{$.ShortName}}.{{$.Name}} != nil {{template "minWireVersion" $}} {
{{template "minWireVersionRequired" $}}
	dst = bsoncore.AppendArrayElement(dst, "{{$.ParameterName}}", {{$.ShortName}}.{{$.Name}})
}
`)

var commandParamValueTmpl = parseCommandParamTemplate(`if {{$.ShortName}}.{{$.Name}}.Type != bsontype.Type(0) {{template "minWireVersion" $}} {
{{template "minWireVersionRequired" $}}
	dst = bsoncore.AppendValueElement(dst, "{{$.ParameterName}}", {{$.ShortName}}.{{$.Name}})
}
`)

var commandParamInt32Tmpl = parseCommandParamTemplate(`if {{$.ShortName}}.{{$.Name}} != nil {{template "minWireVersion" $}} {
{{template "minWireVersionRequired" $}}
	dst = bsoncore.AppendInt32Element(dst, "{{$.ParameterName}}", *{{$.ShortName}}.{{$.Name}})
}
`)

var commandParamInt64Tmpl = parseCommandParamTemplate(`if {{$.ShortName}}.{{$.Name}} != nil {{template "minWireVersion" $}} {
{{template "minWireVersionRequired" $}}
	dst = bsoncore.AppendInt64Element(dst, "{{$.ParameterName}}", *{{$.ShortName}}.{{$.Name}})
}
`)

var commandParamDoubleTmpl = parseCommandParamTemplate(`if {{$.ShortName}}.{{$.Name}} != nil {{template "minWireVersion" $}} {
{{template "minWireVersionRequired" $}}
	dst = bsoncore.AppendDoubleElement(dst, "{{$.ParameterName}}", *{{$.ShortName}}.{{$.Name}})
}
`)

var commandParamBooleanTmpl = parseCommandParamTemplate(`if {{$.ShortName}}.{{$.Name}} != nil {{template "minWireVersion" $}} {
{{template "minWireVersionRequired" $}}
	dst = bsoncore.AppendBooleanElement(dst, "{{$.ParameterName}}", *{{$.ShortName}}.{{$.Name}})
}
`)

var commandParamStringTmpl = parseCommandParamTemplate(`if {{$.ShortName}}.{{$.Name}} != nil {{template "minWireVersion" $}} {
{{template "minWireVersionRequired" $}}
	dst = bsoncore.AppendStringElement(dst, "{{$.ParameterName}}", *{{$.ShortName}}.{{$.Name}})
}
`)

var responseFieldInt64Tmpl = parseTemplates(`
	case "{{$.ResponseName}}":
		var ok bool
		{{$.ResponseShortName}}.{{$.Field}}, ok = element.Value().Int64OK()
		if !ok {
			err = fmt.Errorf("response field '{{$.ResponseName}}' is type int64, but received BSON type %s", element.Value().Type)
		}
`)

var responseFieldInt32Tmpl = parseTemplates(`
	case "{{$.ResponseName}}":
		var ok bool
		{{$.ResponseShortName}}.{{$.Field}}, ok = element.Value().Int32OK()
		if !ok {
			err = fmt.Errorf("response field '{{$.ResponseName}}' is type int32, but received BSON type %s", element.Value().Type)
		}
`)

const typeTemplate string = `// Copyright (C) MongoDB, Inc. 2019-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

// Code generated by operationgen. DO NOT EDIT.

package {{$.PackageName}}

import "go.mongodb.org/mongo-driver/x/mongo/driver"

{{$.EscapeDocumentation $.Documentation}}
type {{$.Name}} struct {
{{range $name, $field := $.Request}}	{{$name}} {{$field.DeclarationType}}
{{end}}{{range $builtin := $.Properties.Builtins}}	{{$builtin.ReferenceName}} {{$builtin.Type}}
{{end}}{{if $.Properties.Retryable.Mode}}	retry *driver.RetryMode{{end}}
{{if $.Response.Name}}	result {{$.ResultType}}{{end}}
{{if eq $.Response.Type "batch cursor"}}	result driver.CursorResponse{{end}}
}

{{if $.Response.Name}}
type {{$.Response.Name}} struct {
{{range $name, $field := $.Response.Field}}// {{$field.Documentation}}
	{{$.Title $name}} {{$field.Type}}
{{end}}
}

func build{{$.Response.Name}}(response bsoncore.Document, srvr driver.Server) ({{$.Response.Name}}, error) {
	elements, err := response.Elements()
	if err != nil {
		return {{$.Response.Name}}{}, err
	}
	{{$.Response.ShortName}} := {{$.Response.Name}}{}
	for _, element := range elements {
		switch element.Key() {
		{{$.Response.BuildMethod}}
		}
	}
	return {{$.Response.ShortName}}, nil
}
{{end}}

// New{{$.Name}} constructs and returns a new {{$.Name}}.
func New{{$.Name}}({{$.ConstructorParameters}}) *{{$.Name}} {
	return &{{$.Name}}{
{{range $field := $.ConstructorFields}}		{{$field}}
{{end}}
	}
}

{{if $.Response.Name}}
// Result returns the result of executing this operation.
func ({{$.ShortName}} *{{$.Name}}) Result() {{$.ResultType}} { return {{$.ShortName}}.result }{{end}}
{{if eq $.Response.Type "batch cursor"}}
// Result returns the result of executing this operation.
func ({{$.ShortName}} *{{$.Name}}) Result(opts driver.CursorOptions) (*driver.BatchCursor, error) {
{{if $builtin := $.Properties.IsEnabled "client session"}}
clientSession := {{$.ShortName}}.{{$builtin.ReferenceName}}{{else}}
var clientSession *session.Client{{end}}
{{if $builtin := $.Properties.IsEnabled "cluster clock"}}
clock := {{$.ShortName}}.{{$builtin.ReferenceName}}{{else}}
var clock *session.ClusterClock{{end}}
	return driver.NewBatchCursor({{$.ShortName}}.result, clientSession, clock, opts)
}{{end}}

func ({{$.ShortName}} *{{$.Name}}) processResponse(response bsoncore.Document, srvr driver.Server, desc description.Server) error {
	var err error
{{if $.Response.Name}}
	{{$.ShortName}}.result, err = build{{$.Response.Name}}(response, srvr)
	return err
{{end}}
{{if eq $.Response.Type "batch cursor"}}
	{{$.ShortName}}.result, err = driver.NewCursorResponse(response, srvr, desc)
	return err
{{end}}
}

// Execute runs this operations and returns an error if the operaiton did not execute successfully.
func ({{$.ShortName}} *{{$.Name}}) Execute(ctx context.Context) error {
	if {{$.ShortName}}.deployment == nil {
		return errors.New("the {{$.Name}} operation must have a Deployment set before Execute can be called")
	}
{{if $.Properties.Batches}}		batches := &driver.Batches{
			Identifier: "{{$.Properties.Batches}}",
			Documents: {{$.ShortName}}.{{$.Properties.Batches}},{{if index $.Request "ordered"}}
			Ordered: {{$.ShortName}}.ordered,
{{end}}}{{end}}
{{with $builtins := $.Properties.BuiltinsMap}}
	return driver.Operation{
		CommandFn: {{$.ShortName}}.command,
		ProcessResponseFn: {{$.ShortName}}.processResponse,{{if $.Properties.Batches}}
		Batches: batches,{{end}}
{{if $.Properties.Retryable.Mode}}	RetryMode: {{$.ShortName}}.retry,{{end}}
{{if eq $.Properties.Retryable.Type "writes"}}	RetryType: driver.RetryWrite,
{{end}}{{range $b := $.Properties.ExecuteBuiltins}}{{$b.ExecuteName}}: {{$.ShortName}}.{{$b.ReferenceName}},
{{end}}{{if $.Properties.MinimumWriteConcernWireVersion}}MinimumWriteConcernWireVersion: {{$.Properties.MinimumWriteConcernWireVersion}},
{{end}}{{if $.Properties.MinimumReadConcernWireVersion}}MinimumReadConcernWireVersion: {{$.Properties.MinimumReadConcernWireVersion}},
{{end}}
	}.Execute(ctx, nil)
{{end}}
}

func ({{$.ShortName}} *{{$.Name}}) command(dst []byte, desc description.SelectedServer) ([]byte, error) {
	{{$.CommandMethod}}
	return dst, nil
}

{{range $name, $field := $.Request}}
{{$.EscapeDocumentation $field.Documentation}}
func ({{$.ShortName}} *{{$.Name}}) {{$.Title $name}}({{$name}} {{$field.ParameterType}}) *{{$.Name}} {
	if {{$.ShortName}} == nil {
		{{$.ShortName}} = new({{$.Name}})
	}

	{{$.ShortName}}.{{$name}} = {{if $field.PointerType}}&{{end}}{{$name}}
	return {{$.ShortName}}
}

{{end}}

{{range $builtin := $.Properties.Builtins}}
{{$.EscapeDocumentation $builtin.Documentation}}
func ({{$.ShortName}} *{{$.Name}}) {{$builtin.SetterName}}({{$builtin.ReferenceName}} {{$builtin.Type}}) *{{$.Name}} {
	if {{$.ShortName}} == nil {
		{{$.ShortName}} = new({{$.Name}})
	}

	{{$.ShortName}}.{{$builtin.ReferenceName}} = {{$builtin.ReferenceName}}
	return {{$.ShortName}}
}
{{end}}

{{if $.Properties.Retryable.Mode}}
// Retry enables retryable writes for this operation. Retries are not handled automatically,
// instead a boolean is returned from Execute and SelectAndExecute that indicates if the
// operation can be retried. Retrying is handled by calling RetryExecute.
func ({{$.ShortName}} *{{$.Name}}) Retry(retry driver.RetryMode) *{{$.Name}} {
	if {{$.ShortName}} == nil {
		{{$.ShortName}} = new({{$.Name}})
	}

	{{$.ShortName}}.retry = &retry
	return {{$.ShortName}}
}
{{end}}
`
