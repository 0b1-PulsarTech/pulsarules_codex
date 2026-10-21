package loader

import (
	"errors"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config/projectfile"
)

type fileError struct {
	File string
	Lead string
	Err  error
}

func (fault *fileError) Error() string {
	return fault.Err.Error()
}

func (fault *fileError) Unwrap() error {
	return fault.Err
}

type reworded struct {
	text  string
	cause error
}

func (fault *reworded) Error() string { return fault.text }

func (fault *reworded) Unwrap() error { return fault.cause }

func leadOf(err error, file *projectfile.File) string {
	if located, isLocated := errors.AsType[*fileError](err); isLocated && located.Lead != "" {
		return located.Lead
	}
	if file != nil {
		return file.Path
	}
	return ""
}

func faultFile(err error, file *projectfile.File) string {
	if located, isLocated := errors.AsType[*fileError](err); isLocated {
		return located.File
	}
	if file == nil {
		return projectfile.Name
	}
	if len(file.Sources) > 0 {
		return file.Sources[0]
	}
	return file.Path
}
