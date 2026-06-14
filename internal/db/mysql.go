package db

import (
	"database/sql"
	"fmt"
	"math/rand"
	"os"
	"regexp"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// Helper to build connection string from Env
func getRootDSN() string {
	pass := os.Getenv("DB_ROOT_PASS")
	// If empty, assume no password or default
	if pass == "" {
		return "root:root@tcp(127.0.0.1:3306)/"
	}
	return fmt.Sprintf("root:%s@tcp(127.0.0.1:3306)/", pass)
}

type DBResult struct {
	DBName   string
	Username string
	Password string
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

func GeneratePassword(n int) string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#%^&*")
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func CreateDatabase(name string, user string) (*DBResult, error) {
	isAlphaNumeric := regexp.MustCompile(`^[a-zA-Z0-9_]+$`).MatchString
	if !isAlphaNumeric(name) || !isAlphaNumeric(user) {
		return nil, fmt.Errorf("invalid characters")
	}

	// USE DYNAMIC DSN
	db, err := sql.Open("mysql", getRootDSN())
	if err != nil {
		return nil, fmt.Errorf("driver error: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("MySQL connection failed. Check DB_ROOT_PASS env var.")
	}

	password := GeneratePassword(16)
	queries := []string{
		fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s`;", name),
		fmt.Sprintf("CREATE USER '%s'@'localhost' IDENTIFIED BY '%s';", user, password),
		fmt.Sprintf("GRANT ALL PRIVILEGES ON `%s`.* TO '%s'@'localhost';", name, user),
		"FLUSH PRIVILEGES;",
	}

	for _, q := range queries {
		if _, err := db.Exec(q); err != nil {
			return nil, fmt.Errorf("SQL Error: %v", err)
		}
	}

	return &DBResult{DBName: name, Username: user, Password: password}, nil
}
