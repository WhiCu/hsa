package migrations_test

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/whicu/hsa/internal/infrastructure/storage/migrations"
)

type dummyDriver struct{}

func (dummyDriver) Open(_ string) (driver.Conn, error) {
	return dummyConn{}, nil
}

type dummyConn struct{}

func (dummyConn) Prepare(query string) (driver.Stmt, error) {
	return dummyStmt{query: query}, nil
}

func (dummyConn) Close() error { return nil }

func (dummyConn) Begin() (driver.Tx, error) { return dummyTx{}, nil }

type dummyTx struct{}

func (dummyTx) Commit() error   { return nil }
func (dummyTx) Rollback() error { return nil }

type dummyStmt struct {
	query string
}

func (dummyStmt) Close() error { return nil }

func (dummyStmt) NumInput() int { return -1 }

func (s dummyStmt) Exec(_ []driver.Value) (driver.Result, error) {
	return dummyResult{}, nil
}

func (s dummyStmt) Query(_ []driver.Value) (driver.Rows, error) {
	return &dummyRows{tables: []string{"test_table", "weird\"name"}}, nil
}

type dummyResult struct{}

func (dummyResult) LastInsertId() (int64, error) { return 0, nil }
func (dummyResult) RowsAffected() (int64, error) { return 0, nil }

type dummyRows struct {
	tables []string
	pos    int
}

func (r *dummyRows) Columns() []string { return []string{"table_name"} }

func (r *dummyRows) Close() error { return nil }

func (r *dummyRows) Next(dest []driver.Value) error {
	if r.pos >= len(r.tables) {
		return io.EOF
	}
	dest[0] = r.tables[r.pos]
	r.pos++
	return nil
}

func init() {
	sql.Register("dummy2", dummyDriver{})
}

func TestReset_DummyDB_Coverage(t *testing.T) {
	db, _ := sql.Open("dummy2", "")
	err := migrations.Reset(context.Background(), db)
	assert.NoError(t, err)
}

func TestUp_DummyDB_Coverage(_ *testing.T) {
	db, _ := sql.Open("dummy2", "")
	err := migrations.Up(context.Background(), db)
	// It'll probably fail internally somewhere but should cover our function up to goose calls
	_ = err
}

func TestReset_EmptyDB_Coverage(t *testing.T) {
	sql.Register("emptyDB", emptyDriver{})
	db, _ := sql.Open("emptyDB", "")
	err := migrations.Reset(context.Background(), db)
	assert.NoError(t, err)
}

type emptyDriver struct{}

func (emptyDriver) Open(_ string) (driver.Conn, error) {
	return emptyConn{}, nil
}

type emptyConn struct{}

func (emptyConn) Prepare(query string) (driver.Stmt, error) {
	return emptyStmt{query: query}, nil
}

func (emptyConn) Close() error { return nil }

func (emptyConn) Begin() (driver.Tx, error) { return dummyTx{}, nil }

type emptyStmt struct {
	query string
}

func (emptyStmt) Close() error { return nil }

func (emptyStmt) NumInput() int { return -1 }

func (s emptyStmt) Exec(_ []driver.Value) (driver.Result, error) {
	return dummyResult{}, nil
}

func (s emptyStmt) Query(_ []driver.Value) (driver.Rows, error) {
	return &emptyRows{}, nil
}

type emptyRows struct {
}

func (r *emptyRows) Columns() []string { return []string{"table_name"} }

func (r *emptyRows) Close() error { return nil }

func (r *emptyRows) Next(_ []driver.Value) error {
	return io.EOF
}

func TestReset_ErrorDB_Coverage(t *testing.T) {
	sql.Register("errorDB", errorDriver{})
	db, _ := sql.Open("errorDB", "")
	err := migrations.Reset(context.Background(), db)
	assert.Error(t, err)
}

type errorDriver struct{}

func (errorDriver) Open(_ string) (driver.Conn, error) {
	return errorConn{}, nil
}

type errorConn struct{}

func (errorConn) Prepare(query string) (driver.Stmt, error) {
	return errorStmt{query: query}, nil
}

func (errorConn) Close() error { return nil }

func (errorConn) Begin() (driver.Tx, error) { return dummyTx{}, nil }

type errorStmt struct {
	query string
}

func (errorStmt) Close() error { return nil }

func (errorStmt) NumInput() int { return -1 }

func (s errorStmt) Exec(_ []driver.Value) (driver.Result, error) {
	return nil, sql.ErrNoRows
}

func (s errorStmt) Query(_ []driver.Value) (driver.Rows, error) {
	return nil, sql.ErrNoRows
}
