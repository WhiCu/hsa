package application

import (
	"github.com/google/uuid"
)

type IDGenerator interface{ NewID() uuid.UUID }

type TokenGenerator interface {
	GenerateToken(length int) (token string, hash string, err error)
}
