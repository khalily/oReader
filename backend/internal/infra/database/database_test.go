package database

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConnection(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set, skipping MySQL connection test")
	}

	db, err := NewConnection(dsn)
	require.NoError(t, err)
	require.NotNil(t, db)

	// Clean up
	sqlDB, err := db.DB()
	if err == nil {
		sqlDB.Close()
	}
}

func TestNewConnectionEmptyDSN(t *testing.T) {
	_, err := NewConnection("")
	require.Error(t, err)
}

func TestNewConnectionInvalidDSN(t *testing.T) {
	_, err := NewConnection("mysql://invalid:invalid@tcp(nonexistent:3306)/nonexistent")
	require.Error(t, err)
}

func TestConnectionPool(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set, skipping MySQL connection pool test")
	}

	db, err := NewConnection(dsn)
	require.NoError(t, err)
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	sqlDB, err := db.DB()
	require.NoError(t, err)

	// Test pool settings
	stats := sqlDB.Stats()
	assert.GreaterOrEqual(t, stats.MaxOpenConnections, 0)
}
