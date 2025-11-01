package errlocal

import "net/http"

type ErrToManyRequests struct {
	BaseError
}

func NewErrToManyRequests(msg string) LocalError {
	return &ErrToManyRequests{
		BaseError: BaseError{
			Msg: msg,
		},
	}
}

func (e *ErrToManyRequests) Code() int {
	return http.StatusTooManyRequests
}
