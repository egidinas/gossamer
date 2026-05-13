//go:build noduckdb

package api

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
)

func init() {
	sql.Register("duckdb", stubDriver{})
}

type stubDriver struct{}
type stubConn struct{}
type stubStmt struct{}
type stubRows struct{}
type stubTx struct{}

func (stubDriver) Open(string) (driver.Conn, error)  { return stubConn{}, nil }
func (stubConn) Prepare(string) (driver.Stmt, error) { return stubStmt{}, nil }
func (stubConn) Close() error                        { return nil }
func (stubConn) Begin() (driver.Tx, error)           { return stubTx{}, nil }
func (stubStmt) Close() error                        { return nil }
func (stubStmt) NumInput() int                       { return -1 }
func (stubStmt) Exec([]driver.Value) (driver.Result, error) {
	return nil, errors.New("duckdb: built without DuckDB support")
}
func (stubStmt) Query([]driver.Value) (driver.Rows, error) { return stubRows{}, nil }
func (stubRows) Columns() []string                         { return nil }
func (stubRows) Close() error                              { return nil }
func (stubRows) Next([]driver.Value) error                 { return io.EOF }
func (stubTx) Commit() error                               { return nil }
func (stubTx) Rollback() error                             { return nil }
