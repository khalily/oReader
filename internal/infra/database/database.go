package database

import (
	"fmt"
	"net/url"
	"strings"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// NewConnection creates a new MySQL database connection.
// DSN format: mysql://user:password@host:port/database?parseTime=true
// Or raw DSN: user:password@tcp(host:port)/database?charset=utf8mb4&parseTime=True&loc=Local
func NewConnection(dsn string) (*gorm.DB, error) {
	if dsn == "" {
		return nil, fmt.Errorf("database DSN cannot be empty")
	}

	// Parse mysql://user:password@host:port/database format
	if strings.HasPrefix(dsn, "mysql://") {
		u, err := url.Parse(dsn)
		if err != nil {
			return nil, fmt.Errorf("invalid MySQL DSN: %w", err)
		}

		password, _ := u.User.Password()
		dsn = fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			u.User.Username(),
			password,
			u.Host,
			strings.TrimPrefix(u.Path, "/"),
		)
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

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)

	return db, nil
}
