package nano

import "errors"

// Errors that could be occurred during message handling.
var (
	ErrCloseClosedGroup   = errors.New("close closed group")
	ErrClosedGroup        = errors.New("group closed")
	ErrMemberNotFound     = errors.New("member not found in the group")
	ErrSessionDuplication = errors.New("session has existed in the current group")
)
