package database

import (
	"fmt"
	"net/url"
	"strings"

	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// NewConnection creates a new database connection based on the DSN
func NewConnection(dsn string) (*gorm.DB, error) {
	if dsn == "" {
		return nil, fmt.Errorf("database DSN cannot be empty")
	}

	// Detect database type from DSN
	if isMySQL(dsn) {
		return newMySQLConnection(dsn)
	}

	// Default to SQLite
	return newSQLiteConnection(dsn)
}

// isMySQL checks if the DSN is for MySQL
func isMySQL(dsn string) bool {
	return strings.HasPrefix(dsn, "mysql://") ||
		strings.HasPrefix(dsn, "mysql:")
}

// newMySQLConnection creates a MySQL connection
func newMySQLConnection(dsn string) (*gorm.DB, error) {
	// Parse mysql://user:password@host:port/database format
	if strings.HasPrefix(dsn, "mysql://") {
		u, err := url.Parse(dsn)
		if err != nil {
			return nil, fmt.Errorf("invalid MySQL DSN: %w", err)
		}

		password, _ := u.User.Password()
		mysqlDSN := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			u.User.Username(),
			password,
			u.Host,
			strings.TrimPrefix(u.Path, "/"),
		)
		dsn = mysqlDSN
	}

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MySQL: %w", err)
	}

	// Configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying database: %w", err)
	}

	// Set connection pool settings
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)

	return db, nil
}

// newSQLiteConnection creates a SQLite connection
func newSQLiteConnection(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SQLite: %w", err)
	}

	// Configure connection pool (SQLite only allows one writer at a time)
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying database: %w", err)
	}

	// SQLite specific settings
	sqlDB.SetMaxIdleConns(1)
	sqlDB.SetMaxOpenConns(1)

	return db, nil
}
