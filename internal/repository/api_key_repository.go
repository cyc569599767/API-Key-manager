package repository

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"keymanager/internal/model"
)

type APIKeyRepository struct {
	db *sql.DB
}

func NewAPIKeyRepository(db *sql.DB) *APIKeyRepository {
	return &APIKeyRepository{db: db}
}

func (r *APIKeyRepository) Create(key *model.APIKey) error {
	result, err := r.db.Exec(`INSERT INTO api_keys (
		name, provider, environment, group_name, budget_tag, base_url, key_ciphertext, key_nonce,
		key_fingerprint, masked_key, status, test_endpoint, timeout_ms, rate_limit_rpm,
		failure_strategy, created_at, updated_at, disabled_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		key.Name, key.Provider, key.Environment, key.GroupName, key.BudgetTag, key.BaseURL, key.KeyCiphertext, key.KeyNonce,
		key.KeyFingerprint, key.MaskedKey, key.Status, key.TestEndpoint, key.TimeoutMS, key.RateLimitRPM,
		key.FailureStrategy, formatTime(key.CreatedAt), formatTime(key.UpdatedAt), formatNullableTime(key.DisabledAt),
	)
	if err != nil {
		return err
	}
	key.ID, err = result.LastInsertId()
	return err
}

func (r *APIKeyRepository) Update(key *model.APIKey) error {
	_, err := r.db.Exec(`UPDATE api_keys SET
		name = ?, provider = ?, environment = ?, group_name = ?, budget_tag = ?, base_url = ?,
		key_ciphertext = ?, key_nonce = ?, key_fingerprint = ?, masked_key = ?, status = ?,
		test_endpoint = ?, timeout_ms = ?, rate_limit_rpm = ?, failure_strategy = ?, updated_at = ?, disabled_at = ?
		WHERE id = ?`,
		key.Name, key.Provider, key.Environment, key.GroupName, key.BudgetTag, key.BaseURL,
		key.KeyCiphertext, key.KeyNonce, key.KeyFingerprint, key.MaskedKey, key.Status,
		key.TestEndpoint, key.TimeoutMS, key.RateLimitRPM, key.FailureStrategy, formatTime(key.UpdatedAt), formatNullableTime(key.DisabledAt), key.ID,
	)
	return err
}

func (r *APIKeyRepository) Get(id int64) (*model.APIKey, error) {
	row := r.db.QueryRow(`SELECT id, name, provider, environment, group_name, budget_tag, base_url,
		key_ciphertext, key_nonce, key_fingerprint, masked_key, status, test_endpoint, timeout_ms,
		rate_limit_rpm, failure_strategy, created_at, updated_at, disabled_at FROM api_keys WHERE id = ?`, id)
	return scanAPIKey(row)
}

func (r *APIKeyRepository) List(filter model.APIKeyFilter, limit int, offset int) ([]model.APIKey, int, error) {
	where, args := buildFilter(filter)
	countQuery := `SELECT COUNT(*) FROM api_keys`
	if where != "" {
		countQuery += " WHERE " + where
	}
	var total int
	if err := r.db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `SELECT id, name, provider, environment, group_name, budget_tag, base_url,
		key_ciphertext, key_nonce, key_fingerprint, masked_key, status, test_endpoint, timeout_ms,
		rate_limit_rpm, failure_strategy, created_at, updated_at, disabled_at FROM api_keys`
	if where != "" {
		query += " WHERE " + where
	}
	query += " ORDER BY updated_at DESC, id DESC LIMIT ? OFFSET ?"
	queryArgs := append([]any{}, args...)
	queryArgs = append(queryArgs, limit, offset)

	rows, err := r.db.Query(query, queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	keys := []model.APIKey{}
	for rows.Next() {
		key, err := scanAPIKey(rows)
		if err != nil {
			return nil, 0, err
		}
		keys = append(keys, *key)
	}
	return keys, total, rows.Err()
}

func (r *APIKeyRepository) Stats() (model.DashboardStats, error) {
	stats := model.DashboardStats{}
	rows, err := r.db.Query(`SELECT status, COUNT(*) FROM api_keys GROUP BY status`)
	if err != nil {
		return stats, err
	}
	defer rows.Close()
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return stats, err
		}
		stats.Total += count
		switch model.APIKeyStatus(status) {
		case model.StatusAvailable:
			stats.Available = count
		case model.StatusError:
			stats.Error = count
		case model.StatusUntested:
			stats.Untested = count
		case model.StatusDisabled:
			stats.Disabled = count
		}
	}
	return stats, rows.Err()
}

func (r *APIKeyRepository) FilterOptions() (model.FilterOptions, error) {
	options := model.FilterOptions{Providers: []string{}, Environments: []string{}, GroupNames: []string{}}
	providers, err := r.distinct("provider")
	if err != nil {
		return options, err
	}
	environments, err := r.distinct("environment")
	if err != nil {
		return options, err
	}
	groupNames, err := r.distinct("group_name")
	if err != nil {
		return options, err
	}
	options.Providers = providers
	options.Environments = environments
	options.GroupNames = groupNames
	return options, nil
}

func (r *APIKeyRepository) ExistsByFingerprint(fingerprint string) (bool, error) {
	var exists int
	err := r.db.QueryRow(`SELECT 1 FROM api_keys WHERE key_fingerprint = ? LIMIT 1`, fingerprint).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (r *APIKeyRepository) ListSnapshotsByIDs(ids []int64) ([]model.BatchDeleteAPIKeySnapshot, error) {
	if len(ids) == 0 {
		return []model.BatchDeleteAPIKeySnapshot{}, nil
	}
	placeholders, args := placeholdersForIDs(ids)
	rows, err := r.db.Query(`SELECT id, name, provider, base_url, masked_key FROM api_keys WHERE id IN (`+placeholders+`) ORDER BY updated_at DESC, id DESC`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	snapshots := []model.BatchDeleteAPIKeySnapshot{}
	for rows.Next() {
		snapshot := model.BatchDeleteAPIKeySnapshot{}
		if err := rows.Scan(&snapshot.ID, &snapshot.Name, &snapshot.Provider, &snapshot.BaseURL, &snapshot.MaskedKey); err != nil {
			return nil, err
		}
		snapshots = append(snapshots, snapshot)
	}
	return snapshots, rows.Err()
}

func (r *APIKeyRepository) BatchDelete(ids []int64) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	placeholders, args := placeholdersForIDs(ids)
	result, err := r.db.Exec(`DELETE FROM api_keys WHERE id IN (`+placeholders+`)`, args...)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (r *APIKeyRepository) distinct(column string) ([]string, error) {
	rows, err := r.db.Query(fmt.Sprintf(`SELECT DISTINCT %s FROM api_keys WHERE %s != '' ORDER BY %s`, column, column, column))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	values := []string{}
	for rows.Next() {
		var value string
		if err := rows.Scan(&value); err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, rows.Err()
}

func placeholdersForIDs(ids []int64) (string, []any) {
	placeholders := make([]string, len(ids))
	args := make([]any, len(ids))
	for index, id := range ids {
		placeholders[index] = "?"
		args[index] = id
	}
	return strings.Join(placeholders, ","), args
}

func buildFilter(filter model.APIKeyFilter) (string, []any) {
	clauses := []string{}
	args := []any{}
	if strings.TrimSpace(filter.Search) != "" {
		like := "%" + strings.ToLower(strings.TrimSpace(filter.Search)) + "%"
		clauses = append(clauses, `(LOWER(name) LIKE ? OR LOWER(provider) LIKE ? OR LOWER(base_url) LIKE ? OR LOWER(environment) LIKE ? OR LOWER(group_name) LIKE ? OR LOWER(budget_tag) LIKE ? OR LOWER(masked_key) LIKE ?)`)
		args = append(args, like, like, like, like, like, like, like)
	}
	if filter.Provider != "" && filter.Provider != "all" {
		clauses = append(clauses, "provider = ?")
		args = append(args, filter.Provider)
	}
	if filter.Status != "" && filter.Status != "all" {
		clauses = append(clauses, "status = ?")
		args = append(args, filter.Status)
	}
	if filter.Environment != "" && filter.Environment != "all" {
		clauses = append(clauses, "environment = ?")
		args = append(args, filter.Environment)
	}
	if filter.GroupName != "" && filter.GroupName != "all" {
		clauses = append(clauses, "group_name = ?")
		args = append(args, filter.GroupName)
	}
	return strings.Join(clauses, " AND "), args
}

type scanner interface {
	Scan(dest ...any) error
}

func scanAPIKey(row scanner) (*model.APIKey, error) {
	key := model.APIKey{}
	var status string
	var createdAt, updatedAt string
	var disabledAt sql.NullString
	if err := row.Scan(&key.ID, &key.Name, &key.Provider, &key.Environment, &key.GroupName, &key.BudgetTag, &key.BaseURL,
		&key.KeyCiphertext, &key.KeyNonce, &key.KeyFingerprint, &key.MaskedKey, &status, &key.TestEndpoint, &key.TimeoutMS,
		&key.RateLimitRPM, &key.FailureStrategy, &createdAt, &updatedAt, &disabledAt); err != nil {
		return nil, err
	}
	key.Status = model.APIKeyStatus(status)
	key.CreatedAt = parseTime(createdAt)
	key.UpdatedAt = parseTime(updatedAt)
	if disabledAt.Valid {
		t := parseTime(disabledAt.String)
		key.DisabledAt = &t
	}
	return &key, nil
}

func formatTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339Nano)
}

func formatNullableTime(t *time.Time) any {
	if t == nil {
		return nil
	}
	return formatTime(*t)
}

func parseTime(value string) time.Time {
	t, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}
	}
	return t
}
