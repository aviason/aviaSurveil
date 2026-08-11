package request

import "errors"

var (
	ErrNotFound           = errors.New("request not found")
	ErrDuplicate          = errors.New("you already have an active request for this item")
	ErrInsufficientPoints = errors.New("insufficient points")
	ErrInvalidTransition  = errors.New("invalid status transition")
)
