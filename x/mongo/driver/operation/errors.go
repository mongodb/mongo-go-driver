package operation

import "errors"

var (
	errUnacknowledgedHint = errors.New("the 'hint' command parameter cannot be used with unacknowledged writes")
)
