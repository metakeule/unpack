package lib

import (
	"fmt"
)

type RunError struct {
	Command string
	Options []string
	Err     error
}

func (r *RunError) Error() string {
	return r.Err.Error()
}

type NoExtensionError string

func (n NoExtensionError) Error() string {
	return fmt.Sprintf("file has no extension: %#v", n)
}

type MkDirError string

func (n MkDirError) Error() string {
	return fmt.Sprintf("could not create dir: %#v", n)
}

type UnknownPackerError string

func (n UnknownPackerError) Error() string {
	return fmt.Sprintf("for extension %#v there is no known unpacker", n)
}

type UnpackerRegisteredError string

func (d UnpackerRegisteredError) Error() string {
	return fmt.Sprintf("unpacker for extension %#v is already registered", d)
}
