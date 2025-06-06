package query_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/canonical/lxd/lxd/db/query"
)

// Any error happening when beginning the transaction will be propagated.
func TestTransaction_BeginError(t *testing.T) {
	db := newDB(t)
	err := db.Close()
	require.NoError(t, err)

	err = query.Transaction(context.TODO(), db, func(ctx context.Context, tx *sql.Tx) error { return nil })
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Failed to begin transaction")
}

// Any error happening when in the transaction function will cause a rollback.
func TestTransaction_FunctionError(t *testing.T) {
	db := newDB(t)

	err := query.Transaction(context.TODO(), db, func(ctx context.Context, tx *sql.Tx) error {
		_, err := tx.Exec("CREATE TABLE test (id INTEGER)")
		assert.NoError(t, err)
		return errors.New("boom")
	})
	assert.EqualError(t, err, "boom")

	tx, err := db.Begin()
	assert.NoError(t, err)

	tables, err := query.SelectStrings(context.Background(), tx, "SELECT name FROM sqlite_master WHERE type = 'table'")
	assert.NoError(t, err)
	assert.NotContains(t, tables, "test")
}

// Return a new in-memory SQLite database.
func newDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	assert.NoError(t, err)
	return db
}
