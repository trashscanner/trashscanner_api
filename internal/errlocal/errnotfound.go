package errlocal

import "net/http"

type ErrNotFound struct {
	BaseError
}

func NewErrNotFound(msg string, system string, details map[string]any) LocalError {
	return &ErrNotFound{
		BaseError: BaseError{
			Msg:        msg,
			Sys:        system,
			DetailsMap: details,
		},
	}
}

func (e *ErrNotFound) Code() int {
	return http.StatusNotFound
}
