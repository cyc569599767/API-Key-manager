package repository

import (
	"database/sql"
	"keymanager/internal/model"
)

type AuditRepository struct {
	db *sql.DB
}

func NewAuditRepository(db *sql.DB) *AuditRepository {
	return &AuditRepository{db: db}
}

func (r *AuditRepository) Create(log *model.AuditLog) error {
	result, err := r.db.Exec(`INSERT INTO audit_logs (api_key_id, action, summary, metadata_json, created_at) VALUES (?, ?, ?, ?, ?)`, log.APIKeyID, log.Action, log.Summary, log.MetadataJSON, formatTime(log.CreatedAt))
	if err != nil {
		return err
	}
	log.ID, err = result.LastInsertId()
	return err
}

func (r *AuditRepository) List(limit int) ([]model.AuditLog, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	rows, err := r.db.Query(`SELECT audit_logs.id, audit_logs.api_key_id, COALESCE(api_keys.name, ''), audit_logs.action, audit_logs.summary, audit_logs.metadata_json, audit_logs.created_at
		FROM audit_logs LEFT JOIN api_keys ON api_keys.id = audit_logs.api_key_id
		ORDER BY audit_logs.created_at DESC, audit_logs.id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	logs := []model.AuditLog{}
	for rows.Next() {
		log := model.AuditLog{}
		var apiKeyID sql.NullInt64
		var createdAt string
		if err := rows.Scan(&log.ID, &apiKeyID, &log.APIKeyName, &log.Action, &log.Summary, &log.MetadataJSON, &createdAt); err != nil {
			return nil, err
		}
		if apiKeyID.Valid {
			id := apiKeyID.Int64
			log.APIKeyID = &id
		}
		log.CreatedAt = parseTime(createdAt)
		logs = append(logs, log)
	}
	return logs, rows.Err()
}
