package main

import (
	"errors"
	"fmt"
	"go/types"
	"log"
	"strings"
)

type Operation struct {
	Pkg         *types.Package
	Receiver    string
	TypeName    string
	Constructor Constructor

	Fields map[string]Field

	// required fields
	Database   *Field
	CommandFn  *Field
	Deployment *Field
	Server     *Field

	// optional fields
	Collection        *Field
	ProcessResponseFn string // name of the response handler
	Selector          *Field // description.ServerSelector
	ReadPreference    *Field
	ReadConcern       *Field
	WriteConcern      *Field
	ClientSession     *Field
	ClusterClock      *Field
	RetryMode         *Field
	Documents         *Field
	Ordered           *Field
}

func NewOperation(typename, constructorName string) (Operation, error) {
	t, syntaxStruct, pkg, pkgs, err := initialize(typename)
	if err != nil {
		return Operation{}, err
	}

	err = LoadKnownTypes(pkgs)
	if err != nil {
		return Operation{}, fmt.Errorf("Couldn't load known types: %v", err)
	}

	n := t.NumFields()
	fields := make(map[string]Field, n)
	docSequences := make(map[string]Field, 1)
	syntaxFields := syntaxStruct.Fields.List
	if len(syntaxFields) != n {
		return Operation{}, fmt.Errorf("Mismatch number of fields between syntax and type. syntax=%d type=%d", len(syntaxFields), n)
	}

	constructor := Constructor{Name: constructorName}
	operation := Operation{Pkg: pkg, Receiver: receiverName(typename), TypeName: typename, Constructor: constructor}
	for i := 0; i < n; i++ {
		tag := parseTag(t.Tag(i))
		if tag.Skip {
			continue
		}
		f := t.Field(i)
		sf := syntaxFields[i]
		field := Field{
			ftype:         tag.Type,
			name:          f.Name(),
			field:         f,
			pointerExempt: tag.PointerExempt,
			variadic:      tag.Variadic,
		}

		if tag.ConstructorArg {
			constructor.ParamName = f.Name()
			constructor.ParamType = f
			constructor.Variadic = tag.Variadic
			operation.Constructor = constructor
		}
		switch field.ftype {
		case Setter:
			field.fnname = strings.Title(f.Name())
			if tag.Name != "" {
				field.fnname = tag.Name
			}
			if sf.Doc != nil && len(sf.Doc.List) > 0 {
				for _, line := range sf.Doc.List {
					field.doc = append(field.doc, line.Text)
				}
			}
			if _, exists := fields[field.fnname]; exists {
				log.Fatalf("Duplicate name found: %s", field.fnname)
			}

			t := f.Type()
			if ptr, ok := t.(*types.Pointer); ok {
				t = ptr.Elem()
			}
			switch t {
			case Deployment:
				field.knownType = KTDeployment
				operation.Deployment = &field
			case ServerSelector:
				field.knownType = KTServerSelector
				operation.ServerSelector = &field
			case Namespace:
				field.knownType = KTNamespace
				operation.Namespace = &field
			case RetryMode:
				field.knownType = KTRetryMode
				if !tag.RetryableWrite {
					log.Fatalf("Field %s is of known type RetryMode but does not specify retryableWrite", field.name)
				}
				operation.RetryWrite = &field
			case WriteConcern:
				field.knownType = KTWriteConcern
				field.pointerExempt = true
				operation.WriteConcern = &field
			case ReadConcern:
				field.knownType = KTReadConcern
				field.pointerExempt = true
				operation.ReadConcern = &field
			case ReadPreference:
				field.knownType = KTReadPreference
				field.pointerExempt = true
				operation.ReadPreference = &field
			case ClientSession:
				field.knownType = KTClientSession
				field.pointerExempt = true
				operation.ClientSession = &field
			case ClusterClock:
				field.knownType = KTClusterClock
				field.pointerExempt = true
				operation.ClusterClock = &field
			}
			fields[field.fnname] = field
		case CommandName:
		case DocumentSequence:
			if tag.Name != "" {
				field.name = tag.Name
			}
			if _, exists := docSequences[field.name]; exists {
				log.Fatalf("Duplicate document sequence name found: %s", field.name)
			}
			docSequences[field.name] = field
		}
	}

	operation.Fields = fields

	err = operation.Validate()
	if err != nil {
		log.Fatalf("Couldn't validate the %s Operation: %v", typename, err)
	}

	return operation, nil
}

func (op Operation) Validate() error {
	if op.Deployment == nil {
		return errors.New("an Operation must have a Deployment field")
	}
	if op.ServerSelector == nil {
		return errors.New("an Operation must have a ServerSelector field")
	}
	if op.RetryWrite != nil && op.ClientSession == nil {
		return errors.New("a write Operation must have a ClientSession field to be retryable")
	}
	return nil
}
