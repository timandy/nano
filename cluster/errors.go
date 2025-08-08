package cluster

import "errors"

// Errors that could be occurred during message handling.
var (
	ErrSessionOnNotify    = errors.New("current session working on notify mode")
	ErrCloseClosedSession = errors.New("close closed session")
	ErrInvalidRegisterReq = errors.New("invalid register request")
)
