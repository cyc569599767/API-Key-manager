package repository

import (
	"database/sql"
	"keymanager/internal/model"
)

type TestResultRepository struct {
	db *sql.DB
}

func NewTestResultRepository(db *sql.DB) *TestResultRepository {
	return &TestResultRepository{db: db}
}

func (r *TestResultRepository) Create(result *model.TestResult) error {
	row, err := r.db.Exec(`INSERT INTO test_results (
		api_key_id, endpoint, status, http_status, latency_ms, model_count, error_rate, error_message,
		request_headers_json, request_body_json, response_headers_json, response_body_json, response_truncated, tested_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		result.APIKeyID, result.Endpoint, result.Status, result.HTTPStatus, result.LatencyMS, result.ModelCount, result.ErrorRate, result.ErrorMessage,
		defaultJSON(result.RequestHeadersJSON), defaultJSON(result.RequestBodyJSON), defaultJSON(result.ResponseHeadersJSON), result.ResponseBodyJSON, boolInt(result.ResponseTruncated), formatTime(result.TestedAt),
	)
	if err != nil {
		return err
	}
	result.ID, err = row.LastInsertId()
	return err
}

func (r *TestResultRepository) Latest(apiKeyID int64) (*model.TestResult, error) {
	row := r.db.QueryRow(`SELECT id, api_key_id, endpoint, status, http_status, latency_ms, model_count, error_rate, error_message,
		request_headers_json, request_body_json, response_headers_json, response_body_json, response_truncated, tested_at
		FROM test_results WHERE api_key_id = ? ORDER BY tested_at DESC, id DESC LIMIT 1`, apiKeyID)
	result, err := scanTestResult(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return result, err
}

func scanTestResult(row scanner) (*model.TestResult, error) {
	result := model.TestResult{}
	var testedAt string
	var responseTruncated int
	if err := row.Scan(
		&result.ID,
		&result.APIKeyID,
		&result.Endpoint,
		&result.Status,
		&result.HTTPStatus,
		&result.LatencyMS,
		&result.ModelCount,
		&result.ErrorRate,
		&result.ErrorMessage,
		&result.RequestHeadersJSON,
		&result.RequestBodyJSON,
		&result.ResponseHeadersJSON,
		&result.ResponseBodyJSON,
		&responseTruncated,
		&testedAt,
	); err != nil {
		return nil, err
	}
	result.ResponseTruncated = responseTruncated != 0
	result.TestedAt = parseTime(testedAt)
	return &result, nil
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func defaultJSON(value string) string {
	if value == "" {
		return "{}"
	}
	return value
}
