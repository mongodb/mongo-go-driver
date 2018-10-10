package options

import "github.com/mongodb/mongo-go-driver/core/readpref"

// RunCmdOptions represents all possible options for a runCommand operation.
type RunCmdOptions struct {
	ReadPreference *readpref.ReadPref // The read preference for the operation.
}

// RunCmd creates a new *RunCmdOptions
func RunCmd() *RunCmdOptions {
	return &RunCmdOptions{}
}

// SetReadPreference sets the read preference for the operation.
func (rc *RunCmdOptions) SetReadPreference(rp *readpref.ReadPref) *RunCmdOptions {
	rc.ReadPreference = rp
	return rc
}

// ToRunCmdOptions combines the given *RunCmdOptions into one *RunCmdOptions in a last one wins fashion.
func ToRunCmdOptions(opts ...*RunCmdOptions) *RunCmdOptions {
	rc := RunCmd()
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if opt.ReadPreference != nil {
			rc.ReadPreference = opt.ReadPreference
		}
	}

	return rc
}
