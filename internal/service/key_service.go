package service

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	secretcrypto "keymanager/internal/crypto"
	"keymanager/internal/model"
	"keymanager/internal/repository"
	"keymanager/internal/tester"
)

const maxBatchImportRows = 500
const maxBatchDeleteRows = 500
const maxBatchTestRows = 100
const maxBatchExportRows = 500
const batchTestInterval = 5 * time.Second
const defaultPageSize = 20
const maxPageSize = 100

type KeyService struct {
	keys    *repository.APIKeyRepository
	tests   *repository.TestResultRepository
	audits  *repository.AuditRepository
	secrets *secretcrypto.SecretStore
	tester  *tester.ModelTester
}

type batchImportRow struct {
	line  int
	input model.CreateAPIKeyInput
}

type apiKeysTXTExport struct {
	content  string
	exported int
	skipped  int
}

func (e apiKeysTXTExport) Content() string {
	return e.content
}

func (e apiKeysTXTExport) Exported() int {
	return e.exported
}

func (e apiKeysTXTExport) Skipped() int {
	return e.skipped
}

func NewKeyService(keys *repository.APIKeyRepository, tests *repository.TestResultRepository, audits *repository.AuditRepository, secrets *secretcrypto.SecretStore, tester *tester.ModelTester) *KeyService {
	return &KeyService{keys: keys, tests: tests, audits: audits, secrets: secrets, tester: tester}
}

func (s *KeyService) ListAPIKeys(filter model.APIKeyFilter) (model.APIKeyListResult, error) {
	page := filter.Page
	if page <= 0 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize <= 0 {
		pageSize = defaultPageSize
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}
	items, total, err := s.keys.List(filter, pageSize, (page-1)*pageSize)
	if err != nil {
		return model.APIKeyListResult{}, err
	}
	return model.APIKeyListResult{Items: items, Total: total, Page: page, PageSize: pageSize}, nil
}

func (s *KeyService) GetAPIKeyDetail(id int64) (*model.APIKeyDetail, error) {
	key, err := s.keys.Get(id)
	if err != nil {
		return nil, err
	}
	latest, err := s.tests.Latest(id)
	if err != nil {
		return nil, err
	}
	return &model.APIKeyDetail{Key: *key, LatestTestResult: latest}, nil
}

func (s *KeyService) CreateAPIKey(input model.CreateAPIKeyInput) (*model.APIKey, error) {
	return s.createAPIKey(input, true)
}

func (s *KeyService) BatchDeleteAPIKeys(input model.BatchDeleteAPIKeysInput) (model.BatchDeleteAPIKeysResult, error) {
	result := model.BatchDeleteAPIKeysResult{Items: []model.BatchDeleteAPIKeySnapshot{}}
	ids := uniquePositiveIDs(input.IDs)
	result.Total = len(input.IDs)
	if len(ids) == 0 {
		return result, errors.New("请选择要删除的 API Key")
	}
	if len(ids) > maxBatchDeleteRows {
		return result, fmt.Errorf("一次最多删除 %d 个 API Key", maxBatchDeleteRows)
	}

	snapshots, err := s.keys.ListSnapshotsByIDs(ids)
	if err != nil {
		return result, err
	}
	result.Items = snapshots
	if len(snapshots) == 0 {
		result.Skipped = result.Total
		return result, nil
	}

	result.Skipped = len(ids) - len(snapshots) + (len(input.IDs) - len(ids))
	_ = s.audit(nil, "key.batch_deleted", fmt.Sprintf("批量永久删除完成：删除 %d 个 API Key", len(snapshots)), map[string]any{
		"requested": result.Total,
		"deleted":   len(snapshots),
		"skipped":   result.Skipped,
		"items":     snapshots,
	})
	deleted, err := s.keys.BatchDelete(ids)
	if err != nil {
		return result, err
	}
	result.Deleted = int(deleted)
	result.Skipped = result.Total - result.Deleted
	return result, nil
}

func (s *KeyService) BatchTestAPIKeys(ctx context.Context, input model.BatchTestAPIKeysInput) (model.BatchTestAPIKeysResult, error) {
	result := model.BatchTestAPIKeysResult{Results: []model.BatchTestAPIKeyItemResult{}}
	ids := uniquePositiveIDs(input.IDs)
	result.Total = len(input.IDs)
	if len(ids) == 0 {
		return result, errors.New("请选择要测试的 API Key")
	}
	if len(ids) > maxBatchTestRows {
		return result, fmt.Errorf("一次最多测试 %d 个 API Key", maxBatchTestRows)
	}
	result.Skipped = len(input.IDs) - len(ids)
	for index, id := range ids {
		if index > 0 {
			select {
			case <-ctx.Done():
				return result, ctx.Err()
			case <-time.After(batchTestInterval):
			}
		}
		detail, err := s.TestAPIKey(ctx, id)
		item := model.BatchTestAPIKeyItemResult{ID: id}
		if detail != nil {
			item.Name = detail.Key.Name
			item.Status = string(detail.Key.Status)
		}
		if err != nil {
			item.Success = false
			item.Message = err.Error()
			result.Failed++
			result.Results = append(result.Results, item)
			continue
		}
		item.Success = true
		item.Message = "测试通过"
		result.Success++
		result.Results = append(result.Results, item)
	}
	return result, nil
}

func (s *KeyService) BuildAPIKeysTXTExport(input model.BatchExportAPIKeysInput) (apiKeysTXTExport, error) {
	export := apiKeysTXTExport{}
	ids := uniquePositiveIDs(input.IDs)
	if len(ids) == 0 {
		return export, errors.New("请选择要导出的 API Key")
	}
	if len(ids) > maxBatchExportRows {
		return export, fmt.Errorf("一次最多导出 %d 个 API Key", maxBatchExportRows)
	}

	var builder strings.Builder
	for _, id := range ids {
		key, err := s.keys.Get(id)
		if err != nil {
			export.skipped++
			continue
		}
		plaintext, err := s.secrets.DecryptSecret(key.KeyCiphertext, key.KeyNonce)
		if err != nil {
			return export, err
		}
		if export.exported > 0 {
			builder.WriteString("\n")
		}
		builder.WriteString("baseurl=")
		builder.WriteString(key.BaseURL)
		builder.WriteString("\napiKey=")
		builder.WriteString(plaintext)
		builder.WriteString("\n")
		export.exported++
	}
	export.skipped += len(input.IDs) - len(ids)
	if export.exported == 0 {
		return export, errors.New("没有可导出的 API Key")
	}
	export.content = builder.String()
	return export, nil
}

func (s *KeyService) AuditAPIKeysTXTExport(requested int, exported int, skipped int) {
	_ = s.audit(nil, "credentials.batch_exported", fmt.Sprintf("批量导出完成：导出 %d 个 API Key", exported), map[string]any{
		"requested": requested,
		"exported":  exported,
		"skipped":   skipped,
	})
}

func (s *KeyService) AuditDataBackup(path string, size int64) {
	_ = s.audit(nil, "data.backup_created", "数据备份已创建", map[string]any{
		"path": path,
		"size": size,
	})
}

func (s *KeyService) AuditDataRestore(path string) {
	_ = s.audit(nil, "data.backup_restored", "数据已从备份还原", map[string]any{
		"path": path,
	})
}

func (s *KeyService) BatchImportAPIKeys(input model.BatchImportAPIKeysInput) (model.BatchImportAPIKeysResult, error) {
	result := model.BatchImportAPIKeysResult{
		Failures:    []model.BatchImportAPIKeyFailure{},
		SkippedRows: []model.BatchImportAPIKeySkipped{},
		CreatedKeys: []model.APIKey{},
	}
	rows, format, err := parseBatchImportRows(input)
	if err != nil {
		return result, err
	}
	result.Total = len(rows)
	if len(rows) == 0 {
		return result, errors.New("没有可导入的 API Key")
	}
	if len(rows) > maxBatchImportRows {
		return result, fmt.Errorf("一次最多导入 %d 行", maxBatchImportRows)
	}

	strategy := strings.ToLower(strings.TrimSpace(input.DuplicateStrategy))
	if strategy == "" {
		strategy = "skip"
	}
	seen := map[string]struct{}{}
	for _, row := range rows {
		row.input = mergeBatchDefaults(row.input, input.Defaults, row.line)
		masked := secretcrypto.MaskSecret(row.input.APIKey)
		name := strings.TrimSpace(row.input.Name)
		if strings.TrimSpace(row.input.APIKey) == "" {
			result.Failed++
			result.Failures = append(result.Failures, model.BatchImportAPIKeyFailure{Line: row.line, Name: name, MaskedKey: masked, Reason: "API Key 不能为空"})
			continue
		}
		fingerprint := s.secrets.FingerprintSecret(strings.TrimSpace(row.input.APIKey))
		if _, ok := seen[fingerprint]; ok {
			s.recordDuplicate(strategy, &result, row.line, name, masked, "本批次重复，已跳过")
			continue
		}
		exists, err := s.keys.ExistsByFingerprint(fingerprint)
		if err != nil {
			result.Failed++
			result.Failures = append(result.Failures, model.BatchImportAPIKeyFailure{Line: row.line, Name: name, MaskedKey: masked, Reason: "检查重复 key 失败"})
			continue
		}
		if exists {
			s.recordDuplicate(strategy, &result, row.line, name, masked, "已存在相同 API Key，已跳过")
			continue
		}
		key, err := s.createAPIKey(row.input, false)
		if err != nil {
			result.Failed++
			result.Failures = append(result.Failures, model.BatchImportAPIKeyFailure{Line: row.line, Name: name, MaskedKey: masked, Reason: safeBatchReason(err)})
			continue
		}
		seen[fingerprint] = struct{}{}
		result.Created++
		result.CreatedKeys = append(result.CreatedKeys, *key)
	}

	_ = s.audit(nil, "key.batch_imported", fmt.Sprintf("批量导入完成：成功 %d 个，跳过 %d 个，失败 %d 个", result.Created, result.Skipped, result.Failed), map[string]any{
		"total":             result.Total,
		"created":           result.Created,
		"skipped":           result.Skipped,
		"failed":            result.Failed,
		"format":            format,
		"duplicateStrategy": strategy,
	})
	return result, nil
}

func (s *KeyService) recordDuplicate(strategy string, result *model.BatchImportAPIKeysResult, line int, name string, masked string, reason string) {
	if strategy == "error" {
		result.Failed++
		result.Failures = append(result.Failures, model.BatchImportAPIKeyFailure{Line: line, Name: name, MaskedKey: masked, Reason: strings.Replace(reason, "已跳过", "", 1)})
		return
	}
	result.Skipped++
	result.SkippedRows = append(result.SkippedRows, model.BatchImportAPIKeySkipped{Line: line, Name: name, MaskedKey: masked, Reason: reason})
}

func (s *KeyService) createAPIKey(input model.CreateAPIKeyInput, auditSingle bool) (*model.APIKey, error) {
	if err := validateCreate(input); err != nil {
		return nil, err
	}
	ciphertext, nonce, err := s.secrets.EncryptSecret(strings.TrimSpace(input.APIKey))
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	key := &model.APIKey{
		Name:            strings.TrimSpace(input.Name),
		Provider:        strings.TrimSpace(input.Provider),
		Environment:     "Default",
		GroupName:       strings.TrimSpace(input.GroupName),
		BudgetTag:       "",
		BaseURL:         strings.TrimRight(strings.TrimSpace(input.BaseURL), "/"),
		KeyCiphertext:   ciphertext,
		KeyNonce:        nonce,
		KeyFingerprint:  s.secrets.FingerprintSecret(strings.TrimSpace(input.APIKey)),
		MaskedKey:       secretcrypto.MaskSecret(input.APIKey),
		Status:          model.StatusUntested,
		TestEndpoint:    defaultString(strings.TrimSpace(input.TestEndpoint), defaultTestModel(input.Provider)),
		TimeoutMS:       defaultInt(input.TimeoutMS, 10000),
		RateLimitRPM:    defaultInt(input.RateLimitRPM, 120),
		FailureStrategy: defaultString(strings.TrimSpace(input.FailureStrategy), "连续 2 次告警"),
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := s.keys.Create(key); err != nil {
		return nil, err
	}
	if auditSingle {
		_ = s.audit(&key.ID, "key.created", key.Name+" 已创建", map[string]any{"provider": key.Provider, "environment": key.Environment})
	}
	return key, nil
}

func (s *KeyService) UpdateAPIKey(id int64, input model.UpdateAPIKeyInput) (*model.APIKey, error) {
	key, err := s.keys.Get(id)
	if err != nil {
		return nil, err
	}
	if err := validateUpdate(input); err != nil {
		return nil, err
	}
	key.Name = strings.TrimSpace(input.Name)
	key.Provider = strings.TrimSpace(input.Provider)
	key.Environment = "Default"
	key.GroupName = strings.TrimSpace(input.GroupName)
	key.BudgetTag = ""
	key.BaseURL = strings.TrimRight(strings.TrimSpace(input.BaseURL), "/")
	key.TestEndpoint = defaultString(strings.TrimSpace(input.TestEndpoint), defaultTestModel(input.Provider))
	key.TimeoutMS = defaultInt(input.TimeoutMS, 10000)
	key.RateLimitRPM = defaultInt(input.RateLimitRPM, 120)
	key.FailureStrategy = defaultString(strings.TrimSpace(input.FailureStrategy), "连续 2 次告警")
	if strings.TrimSpace(input.APIKey) != "" {
		ciphertext, nonce, err := s.secrets.EncryptSecret(strings.TrimSpace(input.APIKey))
		if err != nil {
			return nil, err
		}
		key.KeyCiphertext = ciphertext
		key.KeyNonce = nonce
		key.KeyFingerprint = s.secrets.FingerprintSecret(strings.TrimSpace(input.APIKey))
		key.MaskedKey = secretcrypto.MaskSecret(input.APIKey)
		key.Status = model.StatusUntested
	}
	key.UpdatedAt = time.Now().UTC()
	if err := s.keys.Update(key); err != nil {
		return nil, err
	}
	_ = s.audit(&key.ID, "key.updated", key.Name+" 已更新", map[string]any{"provider": key.Provider, "environment": key.Environment})
	return key, nil
}

func (s *KeyService) DisableAPIKey(id int64) (*model.APIKey, error) {
	key, err := s.keys.Get(id)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	key.Status = model.StatusDisabled
	key.DisabledAt = &now
	key.UpdatedAt = now
	if err := s.keys.Update(key); err != nil {
		return nil, err
	}
	_ = s.audit(&key.ID, "key.disabled", key.Name+" 已禁用", nil)
	return key, nil
}

func (s *KeyService) EnableAPIKey(id int64) (*model.APIKey, error) {
	key, err := s.keys.Get(id)
	if err != nil {
		return nil, err
	}
	key.Status = model.StatusUntested
	key.DisabledAt = nil
	key.UpdatedAt = time.Now().UTC()
	if err := s.keys.Update(key); err != nil {
		return nil, err
	}
	_ = s.audit(&key.ID, "key.enabled", key.Name+" 已启用", nil)
	return key, nil
}

func (s *KeyService) TestAPIKey(ctx context.Context, id int64) (*model.APIKeyDetail, error) {
	key, err := s.keys.Get(id)
	if err != nil {
		return nil, err
	}
	if key.Status == model.StatusDisabled {
		return nil, errors.New("已禁用的 key 不能测试")
	}
	plaintext, err := s.secrets.DecryptSecret(key.KeyCiphertext, key.KeyNonce)
	if err != nil {
		return nil, err
	}
	result := s.tester.Test(ctx, key.BaseURL, key.TestEndpoint, plaintext, key.TimeoutMS)
	testResult := &model.TestResult{
		APIKeyID:            key.ID,
		Endpoint:            key.TestEndpoint,
		Status:              result.Status,
		HTTPStatus:          result.HTTPStatus,
		LatencyMS:           result.LatencyMS,
		ModelCount:          result.ModelCount,
		ErrorRate:           result.ErrorRate,
		ErrorMessage:        result.ErrorMessage,
		RequestHeadersJSON:  result.RequestHeadersJSON,
		RequestBodyJSON:     result.RequestBodyJSON,
		ResponseHeadersJSON: result.ResponseHeadersJSON,
		ResponseBodyJSON:    result.ResponseBodyJSON,
		ResponseTruncated:   result.ResponseTruncated,
		TestedAt:            time.Now().UTC(),
	}
	if err := s.tests.Create(testResult); err != nil {
		return nil, err
	}
	if result.Status == "success" {
		key.Status = model.StatusAvailable
		_ = s.audit(&key.ID, "key.tested", key.Name+" 测试通过", map[string]any{"latencyMs": result.LatencyMS, "httpStatus": result.HTTPStatus})
	} else {
		key.Status = model.StatusError
		_ = s.audit(&key.ID, "key.test_failed", key.Name+" 测试失败", map[string]any{"httpStatus": result.HTTPStatus})
	}
	key.UpdatedAt = time.Now().UTC()
	if err := s.keys.Update(key); err != nil {
		return nil, err
	}
	return &model.APIKeyDetail{Key: *key, LatestTestResult: testResult}, nil
}

func (s *KeyService) GetDashboardStats() (model.DashboardStats, error) {
	return s.keys.Stats()
}

func (s *KeyService) GetFilterOptions() (model.FilterOptions, error) {
	return s.keys.FilterOptions()
}

func (s *KeyService) ListAuditLogs(limit int) ([]model.AuditLog, error) {
	return s.audits.List(limit)
}

func (s *KeyService) CopyBaseURLAudit(id int64) error {
	key, err := s.keys.Get(id)
	if err != nil {
		return err
	}
	return s.audit(&key.ID, "base_url.copied", key.Name+" 的 Base URL 已复制", nil)
}

func (s *KeyService) GetCopyCredentials(id int64) (string, error) {
	key, err := s.keys.Get(id)
	if err != nil {
		return "", err
	}
	plaintext, err := s.secrets.DecryptSecret(key.KeyCiphertext, key.KeyNonce)
	if err != nil {
		return "", err
	}
	_ = s.audit(&key.ID, "credentials.copied", key.Name+" 的 API Key 已复制", nil)
	return plaintext, nil
}

func (s *KeyService) audit(apiKeyID *int64, action string, summary string, metadata map[string]any) error {
	metadataJSON := "{}"
	if metadata != nil {
		encoded, err := json.Marshal(metadata)
		if err != nil {
			return err
		}
		metadataJSON = string(encoded)
	}
	return s.audits.Create(&model.AuditLog{APIKeyID: apiKeyID, Action: action, Summary: summary, MetadataJSON: metadataJSON, CreatedAt: time.Now().UTC()})
}

func uniquePositiveIDs(ids []int64) []int64 {
	seen := map[int64]struct{}{}
	unique := []int64{}
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
	}
	return unique
}

func parseBatchImportRows(input model.BatchImportAPIKeysInput) ([]batchImportRow, string, error) {
	raw := strings.TrimSpace(strings.TrimPrefix(input.RawText, "\ufeff"))
	if raw == "" {
		return nil, "", errors.New("导入内容不能为空")
	}
	format := strings.ToLower(strings.TrimSpace(input.Format))
	if format == "" || format == "auto" {
		format = detectBatchFormat(raw)
	}
	switch format {
	case "lines":
		return parsePlainLines(raw), format, nil
	case "jsonl":
		rows, err := parseJSONLines(raw)
		return rows, format, err
	case "csv":
		rows, err := parseDelimited(raw, ',')
		return rows, format, err
	case "tsv":
		rows, err := parseDelimited(raw, '\t')
		return rows, format, err
	default:
		return nil, format, errors.New("不支持的导入格式")
	}
}

func detectBatchFormat(raw string) string {
	first := firstNonEmptyLine(raw)
	if strings.HasPrefix(strings.TrimSpace(first), "{") {
		return "jsonl"
	}
	if strings.Contains(first, "\t") {
		return "tsv"
	}
	if strings.Contains(first, ",") && headerHasAPIKey(first, ',') {
		return "csv"
	}
	return "lines"
}

func parsePlainLines(raw string) []batchImportRow {
	rows := []batchImportRow{}
	for index, line := range strings.Split(raw, "\n") {
		key := strings.TrimSpace(line)
		if key == "" {
			continue
		}
		rows = append(rows, batchImportRow{line: index + 1, input: model.CreateAPIKeyInput{APIKey: key}})
	}
	return rows
}

func parseJSONLines(raw string) ([]batchImportRow, error) {
	rows := []batchImportRow{}
	for index, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var input model.CreateAPIKeyInput
		if err := json.Unmarshal([]byte(line), &input); err != nil {
			return nil, fmt.Errorf("第 %d 行 JSON 格式不正确", index+1)
		}
		rows = append(rows, batchImportRow{line: index + 1, input: input})
	}
	return rows, nil
}

func parseDelimited(raw string, comma rune) ([]batchImportRow, error) {
	reader := csv.NewReader(strings.NewReader(raw))
	reader.Comma = comma
	reader.FieldsPerRecord = -1
	reader.TrimLeadingSpace = true
	records, err := reader.ReadAll()
	if err != nil {
		return nil, errors.New("表格格式不正确")
	}
	if len(records) == 0 {
		return nil, errors.New("导入内容不能为空")
	}
	headers := normalizeHeaders(records[0])
	if _, ok := headers["apikey"]; !ok {
		return nil, errors.New("CSV/TSV 必须包含 apiKey 表头")
	}
	rows := []batchImportRow{}
	for index, record := range records[1:] {
		line := index + 2
		if recordIsEmpty(record) {
			continue
		}
		rows = append(rows, batchImportRow{line: line, input: inputFromRecord(headers, record)})
	}
	return rows, nil
}

func normalizeHeaders(record []string) map[string]int {
	headers := map[string]int{}
	for index, value := range record {
		headers[normalizeFieldName(value)] = index
	}
	return headers
}

func inputFromRecord(headers map[string]int, record []string) model.CreateAPIKeyInput {
	input := model.CreateAPIKeyInput{
		Name:            recordValue(headers, record, "name"),
		Provider:        recordValue(headers, record, "provider"),
		Environment:     recordValue(headers, record, "environment"),
		GroupName:       recordValue(headers, record, "groupname"),
		BudgetTag:       recordValue(headers, record, "budgettag"),
		BaseURL:         recordValue(headers, record, "baseurl"),
		APIKey:          recordValue(headers, record, "apikey"),
		TestEndpoint:    recordValue(headers, record, "testendpoint"),
		FailureStrategy: recordValue(headers, record, "failurestrategy"),
	}
	input.TimeoutMS = parseInt(recordValue(headers, record, "timeoutms"))
	input.RateLimitRPM = parseInt(recordValue(headers, record, "ratelimitrpm"))
	return input
}

func recordValue(headers map[string]int, record []string, name string) string {
	index, ok := headers[name]
	if !ok || index >= len(record) {
		return ""
	}
	return strings.TrimSpace(record[index])
}

func normalizeFieldName(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "_", "")
	value = strings.ReplaceAll(value, "-", "")
	return value
}

func headerHasAPIKey(header string, comma rune) bool {
	reader := csv.NewReader(strings.NewReader(header))
	reader.Comma = comma
	reader.FieldsPerRecord = -1
	record, err := reader.Read()
	if err != nil {
		return false
	}
	headers := normalizeHeaders(record)
	_, ok := headers["apikey"]
	return ok
}

func recordIsEmpty(record []string) bool {
	for _, value := range record {
		if strings.TrimSpace(value) != "" {
			return false
		}
	}
	return true
}

func mergeBatchDefaults(input model.CreateAPIKeyInput, defaults model.CreateAPIKeyInput, line int) model.CreateAPIKeyInput {
	if strings.TrimSpace(input.Name) == "" {
		provider := defaultString(strings.TrimSpace(input.Provider), strings.TrimSpace(defaults.Provider))
		if provider == "" {
			provider = "API"
		}
		input.Name = fmt.Sprintf("%s Key %d", provider, line)
	}
	if strings.TrimSpace(input.Provider) == "" {
		input.Provider = defaults.Provider
	}
	if strings.TrimSpace(input.Environment) == "" {
		input.Environment = defaults.Environment
	}
	if strings.TrimSpace(input.GroupName) == "" {
		input.GroupName = defaults.GroupName
	}
	if strings.TrimSpace(input.BudgetTag) == "" {
		input.BudgetTag = defaults.BudgetTag
	}
	if strings.TrimSpace(input.BaseURL) == "" {
		input.BaseURL = defaults.BaseURL
	}
	if strings.TrimSpace(input.TestEndpoint) == "" {
		input.TestEndpoint = defaults.TestEndpoint
	}
	if input.TimeoutMS <= 0 {
		input.TimeoutMS = defaults.TimeoutMS
	}
	if input.RateLimitRPM <= 0 {
		input.RateLimitRPM = defaults.RateLimitRPM
	}
	if strings.TrimSpace(input.FailureStrategy) == "" {
		input.FailureStrategy = defaults.FailureStrategy
	}
	return input
}

func firstNonEmptyLine(raw string) string {
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			return line
		}
	}
	return ""
}

func parseInt(value string) int {
	if strings.TrimSpace(value) == "" {
		return 0
	}
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0
	}
	return parsed
}

func safeBatchReason(err error) string {
	reason := err.Error()
	if strings.Contains(strings.ToLower(reason), "unique") || strings.Contains(strings.ToLower(reason), "constraint") {
		return "保存失败或 key 已存在"
	}
	return reason
}

func validateCreate(input model.CreateAPIKeyInput) error {
	if strings.TrimSpace(input.APIKey) == "" {
		return errors.New("API Key 不能为空")
	}
	return validateBase(input.Name, input.Provider, input.TestEndpoint, input.BaseURL)
}

func validateUpdate(input model.UpdateAPIKeyInput) error {
	return validateBase(input.Name, input.Provider, input.TestEndpoint, input.BaseURL)
}

func validateBase(name string, provider string, testModel string, baseURL string) error {
	if strings.TrimSpace(name) == "" {
		return errors.New("名称不能为空")
	}
	if strings.TrimSpace(provider) == "" {
		return errors.New("Provider 不能为空")
	}
	if strings.TrimSpace(testModel) == "" {
		return errors.New("测试模型不能为空")
	}
	parsed, err := url.ParseRequestURI(strings.TrimSpace(baseURL))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return errors.New("Base URL 格式不正确")
	}
	return nil
}

func defaultTestModel(provider string) string {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "openai":
		return "gpt-5.5"
	case "anthropic":
		return "claude-haiku-4-5-20251001"
	default:
		return "gpt-5.5"
	}
}

func defaultString(value string, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func defaultInt(value int, fallback int) int {
	if value <= 0 {
		return fallback
	}
	return value
}
