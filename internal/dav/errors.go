package dav

import (
	"errors"
)

var (
	ErrNotFound           = errors.New("dav: not found")
	ErrConflict           = errors.New("dav: conflict")
	ErrPreconditionFailed = errors.New("dav: precondition failed")
)
