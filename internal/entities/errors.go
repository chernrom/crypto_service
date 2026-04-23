package entities

import (
	"github.com/pkg/errors"
)

var (
	ErrInvalidParam = errors.New("invalid param")
	ErrInternal     = errors.New("internal error")
	ErrNotFound     = errors.New("not found")
)
