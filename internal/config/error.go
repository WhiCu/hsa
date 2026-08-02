package config

import (
	"errors"
)

var (
	ErrPathNotSet   = errors.New("environment variable is not set")
	ErrPathNotExist = errors.New("does not exist")
)
