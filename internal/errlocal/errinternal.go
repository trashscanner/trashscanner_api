package errlocal

import "net/http"

type ErrInternal struct {
	BaseError
}

func NewErrInternal(msg string, system string, details map[string]any) LocalError {
	return &ErrInternal{
		BaseError: BaseError{
			Msg:        msg,
			Sys:        system,
			DetailsMap: details,
		},
	}
}

func (e *ErrInternal) Code() int {
	return http.StatusInternalServerError
}
