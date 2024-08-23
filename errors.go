package jspacker

import "errors"

// Errors.
var (
	ErrInvalidExport = errors.New("invalid export")
	ErrNoFiles       = errors.New("no files")
	ErrInvalidURL    = errors.New("added files must be absolute URLs")
)
