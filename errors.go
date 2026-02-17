package jspacker

import "errors"

// Errors.
var (
	ErrInvalidExport = errors.New("invalid export")
	ErrInvalidURL    = errors.New("added files must be absolute URLs")
	ErrNoFiles       = errors.New("no files")
)
