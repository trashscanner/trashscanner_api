package errlocal

import (
	"fmt"
	"strings"
)

const (
	messagePrefix = "message: "
	systemPrefix  = "system: "
	detailsPrefix = "details: "
)

type LocalError interface {
	error
	Message() string
	System() string
	Details() map[string]any
	Code() int
	Base() *BaseError
}

type BaseError struct {
	Msg        string         `json:"message,omitempty"`
	Sys        string         `json:"system,omitempty"`
	DetailsMap map[string]any `json:"details,omitempty"`
}

func (e *BaseError) Error() string {
	b := strings.Builder{}
	if e.Msg != "" {
		b.WriteString(messagePrefix + e.Msg)
	}
	if e.Sys != "" {
		b.WriteByte(' ')
		b.WriteString(systemPrefix + e.Sys + " ")
	}
	if len(e.DetailsMap) > 0 {
		b.WriteString(detailsPrefix)
		for key, value := range e.DetailsMap {
			b.WriteString(key + ": " + fmt.Sprintf("%v", value) + "\n")
		}
	}
	return b.String()
}

func (e *BaseError) Message() string {
	return e.Msg
}

func (e *BaseError) System() string {
	return e.Sys
}

func (e *BaseError) Details() map[string]any {
	return e.DetailsMap
}

func (e *BaseError) Code() int {
	return 500
}

func (e *BaseError) Base() *BaseError {
	return e
}
