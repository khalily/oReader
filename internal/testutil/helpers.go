// Package testutil provides shared utilities for testing.
package testutil

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// SetupTestDB creates an in-memory SQLite database for testing.
// It automatically migrates all models and cleans up after the test.
//
// Usage:
//
//	func TestSomething(t *testing.T) {
//	    db := testutil.SetupTestDB(t)
//	    defer db.Close() // Optional - t.Cleanup handles it
//
//	    repo := repository.NewFeedRepository(db)
//	    // ... test code
//	}
func SetupTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err, "Failed to connect to test database")

	// Auto-migrate will be done by the caller based on their models

	t.Cleanup(func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	})

	return db
}

// AssertEventually asserts that a condition becomes true within a timeout.
// Useful for testing async operations or eventual consistency.
//
// Usage:
//
//	testutil.AssertEventually(t, func() bool {
//	    return someAsyncCondition()
//	}, 5*time.Second, 100*time.Millisecond)
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
// The context is automatically canceled when the test completes.
//
// Usage:
//
//	func TestSomething(t *testing.T) {
//	    ctx := testutil.ContextWithTimeout(t, 5*time.Second)
//	    result, err := svc.SomeOperation(ctx, ...)
//	    // ...
//	}
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

// NewTestClock creates a TestClock starting at the given time.
func NewTestClock(start time.Time) *TestClock {
	return &TestClock{current: start}
}

// Now returns the current test time.
func (c *TestClock) Now() time.Time {
	return c.current
}

// Advance moves the clock forward by the given duration.
func (c *TestClock) Advance(d time.Duration) {
	c.current = c.current.Add(d)
}

// Set sets the clock to a specific time.
func (c *TestClock) Set(t time.Time) {
	c.current = t
}
