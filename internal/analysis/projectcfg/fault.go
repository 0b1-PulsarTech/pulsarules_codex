package projectcfg

import (
	"errors"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config"
)

const (
	mergedJoin   = " + "
	mergedSuffix = " (merged)"
)

type FileError struct {
	File string
	Lead string
	Err  error
}

func (fault *FileError) Error() string {
	return fault.Err.Error()
}

func (fault *FileError) Unwrap() error {
	return fault.Err
}

type reworded struct {
	text  string
	cause error
}

func (fault *reworded) Error() string { return fault.text }

func (fault *reworded) Unwrap() error { return fault.cause }

func leadOf(err error, file *config.ProjectFile) string {
	if located, isLocated := errors.AsType[*FileError](err); isLocated && located.Lead != "" {
		return located.Lead
	}
	if file != nil {
		return file.Path
	}
	return ""
}

func faultFile(err error, file *config.ProjectFile) string {
	if located, isLocated := errors.AsType[*FileError](err); isLocated {
		return located.File
	}
	if file == nil {
		return config.ProjectFileName
	}
	if names, isMerged := strings.CutSuffix(file.Path, mergedSuffix); isMerged {
		base, _, _ := strings.Cut(names, mergedJoin)
		return base
	}
	return file.Path
}
