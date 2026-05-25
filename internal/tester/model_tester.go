package tester

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const maxStoredResponseBodyBytes = 32 * 1024
const testUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/147.0.0.0 Safari/537.36"

type ModelTestResult struct {
	Status              string
	HTTPStatus          int
	LatencyMS           int64
	ModelCount          int
	ErrorRate           float64
	ErrorMessage        string
	RequestHeadersJSON  string
	RequestBodyJSON     string
	ResponseHeadersJSON string
	ResponseBodyJSON    string
	ResponseTruncated   bool
}

type ModelTester struct{}

func NewModelTester() *ModelTester {
	return &ModelTester{}
}

func (t *ModelTester) Test(ctx context.Context, baseURL string, model string, apiKey string, timeoutMS int) ModelTestResult {
	if timeoutMS <= 0 {
		timeoutMS = 10000
	}
	url := chatCompletionsURL(baseURL)
	payload, err := json.Marshal(map[string]any{
		"model": model,
		"messages": []map[string]string{
			{"role": "user", "content": "hi"},
		},
	})
	requestBody := formatJSONOrText(payload, apiKey)
	if err != nil {
		return failedResult(0, 0, err.Error(), "{}", "{}", "{}", "", false, apiKey)
	}
	requestCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutMS)*time.Millisecond)
	defer cancel()

	request, err := http.NewRequestWithContext(requestCtx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return failedResult(0, 0, err.Error(), "{}", requestBody, "{}", "", false, apiKey)
	}
	request.Header.Set("Authorization", "Bearer "+apiKey)
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("User-Agent", testUserAgent)
	requestHeaders := sanitizedHeaders(request.Header)

	start := time.Now()
	response, err := http.DefaultClient.Do(request)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		return failedResult(0, latency, err.Error(), requestHeaders, requestBody, "{}", "", false, apiKey)
	}
	defer response.Body.Close()

	body, truncated, err := readLimitedBody(response.Body)
	responseHeaders := sanitizedHeaders(response.Header)
	responseBody := formatJSONOrText(body, apiKey)
	if err != nil {
		return failedResult(response.StatusCode, latency, err.Error(), requestHeaders, requestBody, responseHeaders, responseBody, truncated, apiKey)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return failedResult(response.StatusCode, latency, string(body), requestHeaders, requestBody, responseHeaders, responseBody, truncated, apiKey)
	}
	return ModelTestResult{
		Status:              "success",
		HTTPStatus:          response.StatusCode,
		LatencyMS:           latency,
		ModelCount:          0,
		ErrorRate:           0,
		RequestHeadersJSON:  requestHeaders,
		RequestBodyJSON:     requestBody,
		ResponseHeadersJSON: responseHeaders,
		ResponseBodyJSON:    responseBody,
		ResponseTruncated:   truncated,
	}
}

func chatCompletionsURL(baseURL string) string {
	baseURL = strings.TrimRight(baseURL, "/")
	if strings.HasSuffix(baseURL, "/v1") {
		return baseURL + "/chat/completions"
	}
	return baseURL + "/v1/chat/completions"
}

func failedResult(httpStatus int, latencyMS int64, message string, requestHeaders string, requestBody string, responseHeaders string, responseBody string, truncated bool, apiKey string) ModelTestResult {
	return ModelTestResult{
		Status:              "failed",
		HTTPStatus:          httpStatus,
		LatencyMS:           latencyMS,
		ErrorRate:           1,
		ErrorMessage:        truncate(maskSecret(message, apiKey), 500),
		RequestHeadersJSON:  defaultString(requestHeaders, "{}"),
		RequestBodyJSON:     defaultString(requestBody, "{}"),
		ResponseHeadersJSON: defaultString(responseHeaders, "{}"),
		ResponseBodyJSON:    responseBody,
		ResponseTruncated:   truncated,
	}
}

func readLimitedBody(reader io.Reader) ([]byte, bool, error) {
	body, err := io.ReadAll(io.LimitReader(reader, maxStoredResponseBodyBytes+1))
	if err != nil {
		return nil, false, err
	}
	if len(body) > maxStoredResponseBodyBytes {
		return body[:maxStoredResponseBodyBytes], true, nil
	}
	return body, false, nil
}

func sanitizedHeaders(headers http.Header) string {
	values := map[string][]string{}
	for key, headerValues := range headers {
		if isSensitiveHeader(key) {
			values[key] = []string{"***"}
			continue
		}
		values[key] = headerValues
	}
	encoded, err := json.MarshalIndent(values, "", "  ")
	if err != nil {
		return "{}"
	}
	return string(encoded)
}

func isSensitiveHeader(name string) bool {
	normalized := strings.ToLower(strings.TrimSpace(name))
	return normalized == "authorization" ||
		normalized == "proxy-authorization" ||
		normalized == "cookie" ||
		normalized == "set-cookie" ||
		strings.Contains(normalized, "api-key") ||
		strings.Contains(normalized, "apikey") ||
		strings.Contains(normalized, "token") ||
		strings.Contains(normalized, "secret")
}

func formatJSONOrText(body []byte, apiKey string) string {
	masked := []byte(maskSecret(string(body), apiKey))
	var pretty bytes.Buffer
	if err := json.Indent(&pretty, masked, "", "  "); err == nil {
		return truncate(pretty.String(), maxStoredResponseBodyBytes)
	}
	return truncate(string(masked), maxStoredResponseBodyBytes)
}

func maskSecret(value string, secret string) string {
	if strings.TrimSpace(secret) == "" {
		return value
	}
	return strings.ReplaceAll(value, secret, "***")
}

func truncate(value string, limit int) string {
	value = strings.TrimSpace(value)
	if len(value) <= limit {
		return value
	}
	return fmt.Sprintf("%s...", value[:limit])
}

func defaultString(value string, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
