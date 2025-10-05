package errlocal

import "net/http"

type ErrBadRequest struct {
	BaseError
}

func NewErrBadRequest(msg string, system string, details map[string]any) LocalError {
	return &ErrBadRequest{
		BaseError: BaseError{
			Msg:        msg,
			Sys:        system,
			DetailsMap: details,
		},
	}
}

func (e *ErrBadRequest) Code() int {
	return http.StatusBadRequest
}
