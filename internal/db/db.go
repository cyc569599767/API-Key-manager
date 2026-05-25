package db

import (
	"database/sql"
	"fmt"

	"keymanager/internal/storage"
)

import _ "modernc.org/sqlite"

func Open() (*sql.DB, error) {
	paths, err := storage.AppPaths()
	if err != nil {
		return nil, err
	}

	database, err := sql.Open("sqlite", paths.Database)
	if err != nil {
		return nil, err
	}
	database.SetMaxOpenConns(1)

	if _, err := database.Exec("PRAGMA foreign_keys = ON"); err != nil {
		database.Close()
		return nil, err
	}
	if err := migrate(database); err != nil {
		database.Close()
		return nil, err
	}
	return database, nil
}

func migrate(database *sql.DB) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS api_keys (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			provider TEXT NOT NULL,
			environment TEXT NOT NULL,
			group_name TEXT NOT NULL DEFAULT '',
			budget_tag TEXT NOT NULL DEFAULT '',
			base_url TEXT NOT NULL,
			key_ciphertext BLOB NOT NULL,
			key_nonce BLOB NOT NULL,
			key_fingerprint TEXT NOT NULL,
			masked_key TEXT NOT NULL,
			status TEXT NOT NULL,
			test_endpoint TEXT NOT NULL,
			timeout_ms INTEGER NOT NULL,
			rate_limit_rpm INTEGER NOT NULL,
			failure_strategy TEXT NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			disabled_at TEXT
		)`,
		`CREATE INDEX IF NOT EXISTS idx_api_keys_status ON api_keys(status)`,
		`CREATE INDEX IF NOT EXISTS idx_api_keys_provider ON api_keys(provider)`,
		`CREATE INDEX IF NOT EXISTS idx_api_keys_environment ON api_keys(environment)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_api_keys_fingerprint ON api_keys(key_fingerprint)`,
		`CREATE TABLE IF NOT EXISTS test_results (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			api_key_id INTEGER NOT NULL,
			endpoint TEXT NOT NULL,
			status TEXT NOT NULL,
			http_status INTEGER NOT NULL,
			latency_ms INTEGER NOT NULL,
			model_count INTEGER NOT NULL,
			error_rate REAL NOT NULL,
			error_message TEXT NOT NULL,
			request_headers_json TEXT NOT NULL DEFAULT '{}',
			request_body_json TEXT NOT NULL DEFAULT '{}',
			response_headers_json TEXT NOT NULL DEFAULT '{}',
			response_body_json TEXT NOT NULL DEFAULT '',
			response_truncated INTEGER NOT NULL DEFAULT 0,
			tested_at TEXT NOT NULL,
			FOREIGN KEY(api_key_id) REFERENCES api_keys(id) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_test_results_key_time ON test_results(api_key_id, tested_at DESC)`,
		`CREATE TABLE IF NOT EXISTS audit_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			api_key_id INTEGER,
			action TEXT NOT NULL,
			summary TEXT NOT NULL,
			metadata_json TEXT NOT NULL,
			created_at TEXT NOT NULL,
			FOREIGN KEY(api_key_id) REFERENCES api_keys(id) ON DELETE SET NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_logs_time ON audit_logs(created_at DESC)`,
		`CREATE TABLE IF NOT EXISTS app_settings (
			key TEXT PRIMARY KEY,
			value_json TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
	}
	for _, statement := range statements {
		if _, err := database.Exec(statement); err != nil {
			return err
		}
	}
	return ensureTestResultColumns(database)
}

func ensureTestResultColumns(database *sql.DB) error {
	columns := map[string]string{
		"request_headers_json":  "TEXT NOT NULL DEFAULT '{}'",
		"request_body_json":     "TEXT NOT NULL DEFAULT '{}'",
		"response_headers_json": "TEXT NOT NULL DEFAULT '{}'",
		"response_body_json":    "TEXT NOT NULL DEFAULT ''",
		"response_truncated":    "INTEGER NOT NULL DEFAULT 0",
	}
	for column, definition := range columns {
		exists, err := columnExists(database, "test_results", column)
		if err != nil {
			return err
		}
		if exists {
			continue
		}
		if _, err := database.Exec(fmt.Sprintf("ALTER TABLE test_results ADD COLUMN %s %s", column, definition)); err != nil {
			return err
		}
	}
	return nil
}

func columnExists(database *sql.DB, table string, column string) (bool, error) {
	rows, err := database.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return false, err
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name string
		var dataType string
		var notNull int
		var defaultValue sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &pk); err != nil {
			return false, err
		}
		if name == column {
			return true, nil
		}
	}
	return false, rows.Err()
}
