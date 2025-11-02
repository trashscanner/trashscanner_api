package models

import (
	"io"

	"github.com/google/uuid"
)

type File struct {
	ID    uuid.UUID
	Name  string
	Size  int64
	Entry io.ReadCloser
}
