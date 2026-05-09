package errors

import "fmt"

// CodeError is a typed error carrying a stable code for API responses.
type CodeError struct {
	Code    string
	Message string
	Cause   error
}

func (e *CodeError) Error() string {
	if e == nil {
		return ""
	}
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *CodeError) Unwrap() error { return e.Cause }

func New(code, message string, cause error) *CodeError {
	return &CodeError{Code: code, Message: message, Cause: cause}
}
