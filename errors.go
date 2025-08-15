package nano

import "errors"

// Errors that could be occurred during message handling.
var (
	ErrCloseClosedGroup   = errors.New("close closed group")
	ErrClosedGroup        = errors.New("group already closed")
	ErrSessionDuplication = errors.New("session already exists in the group")
)
