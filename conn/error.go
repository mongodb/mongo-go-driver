package conn

import "fmt"

type multiError struct {
	message string
	errors  []error
}

func (e *multiError) Message() string {
	return e.message
}

func (e *multiError) Error() string {
	result := e.message
	for _, e := range e.errors {
		result += fmt.Sprintf("\n  %s", e)
	}
	return result
}

func (e *multiError) Errors() []error {
	return e.errors
}
