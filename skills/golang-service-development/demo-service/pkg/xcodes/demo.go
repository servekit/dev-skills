// Package xcodes defines demo-service error codes. Group domain errors into
// per-domain files (one per major capability area); generic codes live in
// go-common's xerr package and are re-exported here as needed.
//
// Usage:
//
//	if err != nil {
//	    return nil, xcodes.ErrDemoNotFound.Wrapf(err, "demo id=%d", id)
//	}
//
// See go-common-usage skill §xerr for the full xerr API.
package xcodes

import "github.com/servekit/go-common/xerr"

// Demo-domain error codes.
var (
	ErrDemoInternal     = xerr.New("DEMO_INTERNAL", xerr.CategoryInternal, 500, "internal error")
	ErrDemoInvalidInput = xerr.New("DEMO_INVALID_INPUT", xerr.CategoryBadRequest, 400, "invalid input")
	ErrDemoNotFound     = xerr.New("DEMO_NOT_FOUND", xerr.CategoryNotFound, 404, "demo not found")
	ErrDemoConflict     = xerr.New("DEMO_CONFLICT", xerr.CategoryConflict, 409, "demo already exists")
)
