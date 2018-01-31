package errors

import "errors"

var (
	NoIPAvailableYetErr = errors.New("no external ip available yet")
)
