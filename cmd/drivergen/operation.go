package main

import "errors"

type Operation struct {
	Receiver    string
	TypeName    string
	Constructor Constructor

	// required fields
	Deployment     *Field
	ServerSelector *Field

	Namespace      *Field
	WriteConcern   *Field
	ReadConcern    *Field
	ReadPreference *Field
	ClientSession  *Field
	ClusterClock   *Field
	RetryWrite     *Field
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
