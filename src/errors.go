package main

import (
	"errors"
)

var (
	ERR_CANT_SET_TARGET      error = errors.New("Can't set target")
	ERR_CANT_SET_BACKEND     error = errors.New("Can't set backend")
	ERR_BAD_MODEL            error = errors.New("Can't load model")
	ERR_BAD_STREAM           error = errors.New("Can't read from stream")
	ERR_STREAM_ENDED         error = errors.New("Stream ended")
	ERR_CANCELLED_BY_CONTEXT error = errors.New("Cancelled via context")
	ERR_INTERRUPTED_BY_USER  error = errors.New("Interrupted by user")
)
