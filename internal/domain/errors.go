package domain

import (
	"errors"
	"fmt"
)

var ErrValidation = errors.New("domain: validation error")

func ErrInvalidArgument(err error) error {
	return Wrap(ErrValidation, err)
}

func Wrap(mainErr, subErr error) error {
	return fmt.Errorf("%w: %w", mainErr, subErr)
}
