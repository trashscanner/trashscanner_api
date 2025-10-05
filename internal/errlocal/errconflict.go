package errlocal

import "net/http"

type ErrConflict struct {
	BaseError
}

func NewErrConflict(msg string, system string, details map[string]any) LocalError {
	return &ErrConflict{
		BaseError: BaseError{
			Msg:        msg,
			Sys:        system,
			DetailsMap: details,
		},
	}
}

func (e *ErrConflict) Code() int {
	return http.StatusConflict
}
