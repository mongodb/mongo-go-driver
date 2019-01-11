package main

const constructorTmpl = `
// {{.Name}} constructs and returns a new {{.Type}}.
func {{.Name}}({{if .Parameter}}{{.ParameterName}} {{.ParameterType}}{{end}}) *{{.Type}} {
	return &{{.Type}}{
	{{if .Parameter}}{{.ParameterName}}: {{.ParameterName}},{{end}}
	}
}
`

type ConstructorData struct {
	Name          string // Constructor name
	Type          string // type being constructed
	Parameter     bool   // enables a parameter
	ParameterName string // parameter name
	ParameterType string // parameter type
}
