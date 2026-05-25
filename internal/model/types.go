package model

import "time"

type APIKeyStatus string

const (
	StatusAvailable APIKeyStatus = "available"
	StatusError     APIKeyStatus = "error"
	StatusUntested  APIKeyStatus = "untested"
	StatusDisabled  APIKeyStatus = "disabled"
)

type APIKey struct {
	ID              int64        `json:"id"`
	Name            string       `json:"name"`
	Provider        string       `json:"provider"`
	Environment     string       `json:"environment"`
	GroupName       string       `json:"groupName"`
	BudgetTag       string       `json:"budgetTag"`
	BaseURL         string       `json:"baseUrl"`
	KeyCiphertext   []byte       `json:"-"`
	KeyNonce        []byte       `json:"-"`
	KeyFingerprint  string       `json:"-"`
	MaskedKey       string       `json:"maskedKey"`
	Status          APIKeyStatus `json:"status"`
	TestEndpoint    string       `json:"testEndpoint"`
	TimeoutMS       int          `json:"timeoutMs"`
	RateLimitRPM    int          `json:"rateLimitRpm"`
	FailureStrategy string       `json:"failureStrategy"`
	CreatedAt       time.Time    `json:"createdAt"`
	UpdatedAt       time.Time    `json:"updatedAt"`
	DisabledAt      *time.Time   `json:"disabledAt,omitempty"`
}

type TestResult struct {
	ID                  int64     `json:"id"`
	APIKeyID            int64     `json:"apiKeyId"`
	Endpoint            string    `json:"endpoint"`
	Status              string    `json:"status"`
	HTTPStatus          int       `json:"httpStatus"`
	LatencyMS           int64     `json:"latencyMs"`
	ModelCount          int       `json:"modelCount"`
	ErrorRate           float64   `json:"errorRate"`
	ErrorMessage        string    `json:"errorMessage"`
	RequestHeadersJSON  string    `json:"requestHeadersJson"`
	RequestBodyJSON     string    `json:"requestBodyJson"`
	ResponseHeadersJSON string    `json:"responseHeadersJson"`
	ResponseBodyJSON    string    `json:"responseBodyJson"`
	ResponseTruncated   bool      `json:"responseTruncated"`
	TestedAt            time.Time `json:"testedAt"`
}

type AuditLog struct {
	ID           int64     `json:"id"`
	APIKeyID     *int64    `json:"apiKeyId,omitempty"`
	APIKeyName   string    `json:"apiKeyName,omitempty"`
	Action       string    `json:"action"`
	Summary      string    `json:"summary"`
	MetadataJSON string    `json:"metadataJson"`
	CreatedAt    time.Time `json:"createdAt"`
}

type APIKeyFilter struct {
	Search      string `json:"search"`
	Provider    string `json:"provider"`
	Status      string `json:"status"`
	Environment string `json:"environment"`
	GroupName   string `json:"groupName"`
	Page        int    `json:"page"`
	PageSize    int    `json:"pageSize"`
}

type APIKeyListResult struct {
	Items    []APIKey `json:"items"`
	Total    int      `json:"total"`
	Page     int      `json:"page"`
	PageSize int      `json:"pageSize"`
}

type CreateAPIKeyInput struct {
	Name            string `json:"name"`
	Provider        string `json:"provider"`
	Environment     string `json:"environment"`
	GroupName       string `json:"groupName"`
	BudgetTag       string `json:"budgetTag"`
	BaseURL         string `json:"baseUrl"`
	APIKey          string `json:"apiKey"`
	TestEndpoint    string `json:"testEndpoint"`
	TimeoutMS       int    `json:"timeoutMs"`
	RateLimitRPM    int    `json:"rateLimitRpm"`
	FailureStrategy string `json:"failureStrategy"`
}

type UpdateAPIKeyInput struct {
	Name            string `json:"name"`
	Provider        string `json:"provider"`
	Environment     string `json:"environment"`
	GroupName       string `json:"groupName"`
	BudgetTag       string `json:"budgetTag"`
	BaseURL         string `json:"baseUrl"`
	APIKey          string `json:"apiKey"`
	TestEndpoint    string `json:"testEndpoint"`
	TimeoutMS       int    `json:"timeoutMs"`
	RateLimitRPM    int    `json:"rateLimitRpm"`
	FailureStrategy string `json:"failureStrategy"`
}

type APIKeyDetail struct {
	Key              APIKey      `json:"key"`
	LatestTestResult *TestResult `json:"latestTestResult,omitempty"`
}

type DashboardStats struct {
	Total     int `json:"total"`
	Available int `json:"available"`
	Error     int `json:"error"`
	Untested  int `json:"untested"`
	Disabled  int `json:"disabled"`
}

type FilterOptions struct {
	Providers    []string `json:"providers"`
	Environments []string `json:"environments"`
	GroupNames   []string `json:"groupNames"`
}

type BatchDeleteAPIKeysInput struct {
	IDs []int64 `json:"ids"`
}

type BatchDeleteAPIKeysResult struct {
	Total   int                         `json:"total"`
	Deleted int                         `json:"deleted"`
	Skipped int                         `json:"skipped"`
	Items   []BatchDeleteAPIKeySnapshot `json:"items"`
}

type BatchDeleteAPIKeySnapshot struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Provider  string `json:"provider"`
	BaseURL   string `json:"baseUrl"`
	MaskedKey string `json:"maskedKey"`
}

type BatchTestAPIKeysInput struct {
	IDs []int64 `json:"ids"`
}

type BatchTestAPIKeysResult struct {
	Total   int                         `json:"total"`
	Success int                         `json:"success"`
	Failed  int                         `json:"failed"`
	Skipped int                         `json:"skipped"`
	Results []BatchTestAPIKeyItemResult `json:"results"`
}

type BatchTestAPIKeyItemResult struct {
	ID      int64  `json:"id"`
	Name    string `json:"name"`
	Success bool   `json:"success"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

type BatchExportAPIKeysInput struct {
	IDs []int64 `json:"ids"`
}

type BatchExportAPIKeysResult struct {
	Total    int    `json:"total"`
	Exported int    `json:"exported"`
	Skipped  int    `json:"skipped"`
	Path     string `json:"path"`
	Canceled bool   `json:"canceled"`
}

type BackupRestoreResult struct {
	Path     string `json:"path"`
	Canceled bool   `json:"canceled"`
}

type BatchImportAPIKeysInput struct {
	RawText           string            `json:"rawText"`
	Defaults          CreateAPIKeyInput `json:"defaults"`
	Format            string            `json:"format"`
	DuplicateStrategy string            `json:"duplicateStrategy"`
}

type BatchImportAPIKeysResult struct {
	Total       int                        `json:"total"`
	Created     int                        `json:"created"`
	Skipped     int                        `json:"skipped"`
	Failed      int                        `json:"failed"`
	Failures    []BatchImportAPIKeyFailure `json:"failures"`
	SkippedRows []BatchImportAPIKeySkipped `json:"skippedRows"`
	CreatedKeys []APIKey                   `json:"createdKeys"`
}

type BatchImportAPIKeyFailure struct {
	Line      int    `json:"line"`
	Name      string `json:"name"`
	MaskedKey string `json:"maskedKey"`
	Reason    string `json:"reason"`
}

type BatchImportAPIKeySkipped struct {
	Line      int    `json:"line"`
	Name      string `json:"name"`
	MaskedKey string `json:"maskedKey"`
	Reason    string `json:"reason"`
}
