package migrations_test

import (
	"context"
	"testing"
	"database/sql"
	"database/sql/driver"

	"github.com/stretchr/testify/assert"
	"github.com/whicu/hsa/internal/infrastructure/storage/migrations"
)

type dummyDriver struct{}

func (dummyDriver) Open(name string) (driver.Conn, error) {
	return nil, sql.ErrConnDone
}

func init() {
	sql.Register("dummy", dummyDriver{})
}

// uncovered: testing the SQL operations of Reset() properly requires a running
// Postgres database, which is typically spun up via testcontainers. However,
// testcontainers fails to mount overlayfs in the current environment.

func TestReset_DummyDB_Errors(t *testing.T) {
	db, _ := sql.Open("dummy", "")
	err := migrations.Reset(context.Background(), db)
	assert.Error(t, err)
}
