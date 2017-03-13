package msg

// NewCommand creates a new RequestMessage to be set as a command.
func NewCommand(requestID int32, dbName string, slaveOK bool, cmd interface{}) Request {
	flags := QueryFlags(0)
	if slaveOK {
		flags |= SlaveOK
	}

	return &Query{
		ReqID:              requestID,
		Flags:              flags,
		FullCollectionName: dbName + ".$cmd",
		NumberToReturn:     -1,
		Query:              cmd,
	}
}
