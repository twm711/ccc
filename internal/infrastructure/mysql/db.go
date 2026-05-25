package mysql

import (
	"os"
	"strconv"
	"time"

	"github.com/jmoiron/sqlx"

	_ "github.com/go-sql-driver/mysql"
)

func envInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func NewDB(dsn string) (*sqlx.DB, error) {
	db, err := sqlx.Connect("mysql", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(envInt("DB_MAX_OPEN_CONNS", 50))
	db.SetMaxIdleConns(envInt("DB_MAX_IDLE_CONNS", 10))
	db.SetConnMaxLifetime(time.Duration(envInt("DB_CONN_MAX_LIFETIME_MIN", 30)) * time.Minute)
	db.SetConnMaxIdleTime(time.Duration(envInt("DB_CONN_MAX_IDLE_TIME_MIN", 5)) * time.Minute)
	return db, nil
}
