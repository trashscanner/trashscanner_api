package models

import "io"

type File struct {
	Name  string
	Size  int64
	Entry io.ReadCloser
}
