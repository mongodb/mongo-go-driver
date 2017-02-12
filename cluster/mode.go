package cluster

// ConnectionMode determines the initial cluster type.
type ConnectionMode uint8

// ConnectionMode constants.
const (
	AutomaticMode ConnectionMode = iota
	SingleMode
)
