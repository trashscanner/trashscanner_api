package errlocal

import "net/http"

type ErrUnauthorized struct {
	BaseError
}

func NewErrUnauthorized(msg string, system string, details map[string]any) LocalError {
	return &ErrUnauthorized{
		BaseError: BaseError{
			Msg:        msg,
			Sys:        system,
			DetailsMap: details,
		},
	}
}

func (e *ErrUnauthorized) Code() int {
	return http.StatusUnauthorized
}
