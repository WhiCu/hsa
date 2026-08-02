package application

import (
	"github.com/google/uuid"
)

type IDGenerator interface{ NewID() uuid.UUID }
type Hasher interface {
	hash([]byte) (string, error)
}
