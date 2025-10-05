package errlocal

import "net/http"

type ErrForbidden struct {
	BaseError
}

func NewErrForbidden(msg string, system string, details map[string]any) LocalError {
	return &ErrForbidden{
		BaseError: BaseError{
			Msg:        msg,
			Sys:        system,
			DetailsMap: details,
		},
	}
}

func (e *ErrForbidden) Code() int {
	return http.StatusForbidden
}
