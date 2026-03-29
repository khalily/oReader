package testutil

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// SetupTestDB creates a MySQL database connection for testing.
// It uses TEST_DATABASE_URL env var (default: mysql://oreader:oreader@tcp(localhost:3306)/oreader_test?charset=utf8mb4&parseTime=True&loc=Local)
// It automatically cleans up after the test.
//
// Usage:
//
//	func TestSomething(t *testing.T) {
//	    db := testutil.SetupTestDB(t)
//
//	    repo := repository.NewFeedRepository(db)
//	    // ... test code
//	}
func SetupTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := GetTestDatabaseURL()

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err, "Failed to connect to test database (set TEST_DATABASE_URL env var)")

	// Clean all business tables to prevent data pollution between parallel tests.
	// Order matters due to foreign key constraints: child tables first.
	cleanTables := []string{
		"paper_collection_items",
		"paper_tags",
		"user_item_states",
		"user_feeds",
		"import_jobs",
		"refresh_tokens",
		"pending_oauths",
		"oauth_states",
		"items",
		"papers",
		"paper_collections",
		"feeds",
		"users",
	}
	for _, table := range cleanTables {
		db.Exec("DELETE FROM " + table)
	}

	t.Cleanup(func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	})

	return db
}

// AssertEventually asserts that a condition becomes true within a timeout.
func AssertEventually(t *testing.T, condition func() bool, timeout time.Duration, interval time.Duration, msgAndArgs ...any) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(interval)
	}

	require.Fail(t, "Condition never satisfied", msgAndArgs...)
}

// ContextWithTimeout creates a context with timeout for testing.
func ContextWithTimeout(t *testing.T, timeout time.Duration) context.Context {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	t.Cleanup(cancel)

	return ctx
}

// TestClock provides a controllable clock for testing time-dependent code.
type TestClock struct {
	current time.Time
}

func NewTestClock(start time.Time) *TestClock {
	return &TestClock{current: start}
}

func (c *TestClock) Now() time.Time {
	return c.current
}

func (c *TestClock) Advance(d time.Duration) {
	c.current = c.current.Add(d)
}

func (c *TestClock) Set(t time.Time) {
	c.current = t
}
