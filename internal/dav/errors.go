package dav

import (
	"errors"
	"fmt"
)

var (
	ErrNotFound           = errors.New("dav: not found")
	ErrConflict           = errors.New("dav: conflict")
	ErrPreconditionFailed = errors.New("dav: precondition failed")
)

func mapHTTPError(code int) error {
	switch code {
	case 404:
		return ErrNotFound
	case 409:
		return ErrConflict
	case 412:
		return ErrPreconditionFailed
	default:
		return fmt.Errorf("dav: unexpected HTTP status %d", code)
	}
}
