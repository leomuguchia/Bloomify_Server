package booking

import "fmt"

type MatchError struct {
	Code    string
	Message string
}

func (e *MatchError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func NewMatchError(msg string) error {
	return &MatchError{
		Code:    "matchError",
		Message: msg,
	}
}
