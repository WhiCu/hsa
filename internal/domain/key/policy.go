package key

import "errors"

var ErrTooManyWrappedKeys = errors.New("key: maximum number of wrapped keys exceeded")

type Policy struct {
	maxWrappedKeys int
}

func NewPolicy(maxWrappedKeys int) Policy {
	return Policy{maxWrappedKeys: maxWrappedKeys}
}

func (p Policy) ValidateCount(count int) error {
	if count > p.maxWrappedKeys {
		return ErrTooManyWrappedKeys
	}
	return nil
}
