// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package s3_object

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/client"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/common"
)

// CatalystResponse is the standard ICS operations envelope.
type CatalystResponse struct {
	Status       string          `json:"status"`
	Message      string          `json:"message"`
	ResponseCode int             `json:"responseCode"`
	Data         json.RawMessage `json:"data"`
}

// ObjectPayload is the object metadata shape from ICSS3OperationsController.
type ObjectPayload struct {
	ID           string `json:"id"`
	DomainID     string `json:"domain_id"`
	BucketName   string `json:"bucket_name"`
	ObjectKey    string `json:"object_key"`
	Prefix       string `json:"prefix,omitempty"`
	FileName     string `json:"file_name,omitempty"`
	Size         int64  `json:"size"`
	ETag         string `json:"etag"`
	ContentType  string `json:"content_type,omitempty"`
	LastModified string `json:"last_modified,omitempty"`
	AuditID      string `json:"audit_id,omitempty"`
}

// ListObjectsData is the paginated list payload in envelope data.
type ListObjectsData struct {
	Objects           []ObjectPayload `json:"objects"`
	ContinuationToken string          `json:"continuation_token,omitempty"`
}

const (
	objectIDSeparator       = ":"
	defaultObjectContentType = "application/octet-stream"
)

func objectsBasePath(domainID, bucketName string) string {
	return fmt.Sprintf("%s/domain/%s/buckets/%s/objects", common.ICSOperationsPath, domainID, url.PathEscape(bucketName))
}

// objectKeyQueryPath builds GET/PUT/DELETE URL with required objectKey query parameter.
// Encode() percent-encodes "/" as %2F so gateways do not mangle nested keys.
func objectKeyQueryPath(domainID, bucketName, objectKey string) string {
	key := normalizeObjectKey(objectKey)
	params := url.Values{}
	params.Set("objectKey", key)
	return objectsBasePath(domainID, bucketName) + "?" + params.Encode()
}

func listObjectsPath(domainID, bucketName, prefix, continuationToken, objectKey string) string {
	params := url.Values{}
	if prefix != "" {
		params.Set("prefix", prefix)
	}
	if continuationToken != "" {
		params.Set("continuationToken", continuationToken)
	}
	if objectKey != "" {
		params.Set("objectKey", objectKey)
	}
	path := objectsBasePath(domainID, bucketName)
	encoded := params.Encode()
	if encoded == "" {
		return path
	}
	return path + "?" + encoded
}

// FormatObjectID returns composite Terraform id "{domain_id}:{bucket_name}:{object_key}".
func FormatObjectID(domainID, bucketName, objectKey string) string {
	return domainID + objectIDSeparator + bucketName + objectIDSeparator + normalizeObjectKey(objectKey)
}

// ParseObjectID splits composite id into domain_id, bucket_name, and object_key.
// object_key may contain ":" — only the first two colons are separators.
func ParseObjectID(id string) (domainID, bucketName, objectKey string, err error) {
	idx1 := strings.Index(id, objectIDSeparator)
	if idx1 <= 0 {
		return "", "", "", fmt.Errorf("invalid object id %q: expected format {domain_id}:{bucket_name}:{object_key}", id)
	}
	remainder := id[idx1+1:]
	idx2 := strings.Index(remainder, objectIDSeparator)
	if idx2 <= 0 || idx2 >= len(remainder)-1 {
		return "", "", "", fmt.Errorf("invalid object id %q: expected format {domain_id}:{bucket_name}:{object_key}", id)
	}
	return id[:idx1], remainder[:idx2], remainder[idx2+1:], nil
}

func normalizeObjectKey(objectKey string) string {
	return strings.TrimPrefix(strings.TrimSpace(objectKey), "/")
}

func isObjectNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "status 404") ||
		strings.Contains(msg, `"responsecode":404`) ||
		strings.Contains(msg, "object not found")
}

func isBucketNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "status 404") ||
		strings.Contains(msg, `"responsecode":404`) ||
		strings.Contains(msg, "bucket not found")
}

func isDomainNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "status 404") ||
		strings.Contains(msg, `"responsecode":404`) ||
		strings.Contains(msg, "domain not found")
}

func parseEnvelope(respBody []byte) (*CatalystResponse, error) {
	var envelope CatalystResponse
	if err := json.Unmarshal(respBody, &envelope); err != nil {
		return nil, fmt.Errorf("failed to parse API response: %w", err)
	}
	if envelope.Status != "success" {
		return &envelope, fmt.Errorf("API request failed: %s (code: %d)", envelope.Message, envelope.ResponseCode)
	}
	return &envelope, nil
}

func doICSRequest(c *client.Client, ctx context.Context, method, path string, body interface{}) (*CatalystResponse, error) {
	respBody, err := c.DoRequest(ctx, method, path, body)
	if err != nil {
		return nil, err
	}

	tflog.Debug(ctx, "S3 object API response", map[string]any{
		"method": method,
		"path":   path,
		"body":   string(respBody),
	})

	return parseEnvelope(respBody)
}

func parseObjectPayload(envelope *CatalystResponse) (*ObjectPayload, error) {
	if envelope == nil || len(envelope.Data) == 0 || string(envelope.Data) == "null" {
		return nil, fmt.Errorf("empty object payload in API response")
	}

	// GET ?objectKey= may return either a single object or a one-element array in data.
	var payload ObjectPayload
	if err := json.Unmarshal(envelope.Data, &payload); err == nil {
		return &payload, nil
	}

	var items []ObjectPayload
	if err := json.Unmarshal(envelope.Data, &items); err != nil {
		return nil, fmt.Errorf("failed to parse object payload: %w", err)
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("empty object payload in API response")
	}
	return &items[0], nil
}

func auditIDFromData(data json.RawMessage) string {
	var row struct {
		AuditID string `json:"audit_id"`
	}
	if len(data) == 0 || string(data) == "null" {
		return ""
	}
	err := json.Unmarshal(data, &row)
	if err != nil {
		return ""
	}
	return row.AuditID
}

// UploadObject sends PUT with raw bytes and objectKey query parameter.
func UploadObject(c *client.Client, ctx context.Context, domainID, bucketName, objectKey, contentType string, body io.Reader, contentLength int64) (*ObjectPayload, *CatalystResponse, error) {
	key := normalizeObjectKey(objectKey)
	if key == "" {
		return nil, nil, fmt.Errorf("object_key is required")
	}
	if strings.HasSuffix(key, "/") {
		return nil, nil, fmt.Errorf("object_key must refer to a file, not a folder prefix (trailing '/' is not allowed)")
	}
	if contentType == "" {
		contentType = defaultObjectContentType
	}

	tflog.Debug(ctx, "Uploading S3 object", map[string]any{
		"domain_id":      domainID,
		"bucket_name":    bucketName,
		"object_key":     key,
		"content_type":   contentType,
		"content_length": contentLength,
	})

	token, err := c.GetToken(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get authentication token: %w", err)
	}

	apiURL := client.APIURL + objectKeyQueryPath(domainID, bucketName, key)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPut, apiURL, body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create upload request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+token)
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("Content-Type", contentType)
	httpReq.Header.Set("X-Vayu-Client-Id", "vayu_iac")
	if contentLength >= 0 {
		httpReq.ContentLength = contentLength
	}

	resp, err := c.GetHTTPClient().Do(httpReq)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to execute upload request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read upload response body: %w", err)
	}

	tflog.Debug(ctx, "S3 object upload response", map[string]any{
		"status_code": resp.StatusCode,
		"body":        string(respBody),
	})

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var apiErr client.APIError
		if json.Unmarshal(respBody, &apiErr) == nil && (apiErr.Message != "" || apiErr.Code != "") {
			apiErr.StatusCode = resp.StatusCode
			return nil, nil, &apiErr
		}
		return nil, nil, fmt.Errorf("upload object failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	envelope, err := parseEnvelope(respBody)
	if err != nil {
		return nil, envelope, fmt.Errorf("failed to upload S3 object: %w", err)
	}

	payload, err := parseObjectPayload(envelope)
	if err != nil {
		return nil, envelope, err
	}

	tflog.Info(ctx, "S3 object uploaded successfully", map[string]any{
		"id":         payload.ID,
		"object_key": payload.ObjectKey,
		"etag":       payload.ETag,
		"size":       payload.Size,
	})

	return payload, envelope, nil
}

// GetObject reads object metadata via GET .../objects?objectKey=.
// Response data is a single object (same shape as the former path-based get).
func GetObject(c *client.Client, ctx context.Context, domainID, bucketName, objectKey string) (*ObjectPayload, *CatalystResponse, error) {
	key := normalizeObjectKey(objectKey)
	tflog.Debug(ctx, "Reading S3 object", map[string]any{
		"domain_id":   domainID,
		"bucket_name": bucketName,
		"object_key":  key,
	})

	envelope, err := doICSRequest(c, ctx, http.MethodGet, objectKeyQueryPath(domainID, bucketName, key), nil)
	if err != nil {
		return nil, envelope, fmt.Errorf("failed to read S3 object: %w", err)
	}

	payload, err := parseObjectPayload(envelope)
	if err != nil {
		return nil, envelope, err
	}

	return payload, envelope, nil
}

// ListObjectsPage returns one page of objects. Pass continuationToken for subsequent pages.
func ListObjectsPage(c *client.Client, ctx context.Context, domainID, bucketName, prefix, continuationToken, objectKey string) (*ListObjectsData, *CatalystResponse, error) {
	tflog.Debug(ctx, "Listing S3 objects", map[string]any{
		"domain_id":           domainID,
		"bucket_name":         bucketName,
		"prefix":              prefix,
		"continuation_token":  continuationToken,
		"object_key":          objectKey,
	})

	envelope, err := doICSRequest(c, ctx, http.MethodGet, listObjectsPath(domainID, bucketName, prefix, continuationToken, objectKey), nil)
	if err != nil {
		return nil, envelope, fmt.Errorf("failed to list S3 objects: %w", err)
	}

	if len(envelope.Data) == 0 || string(envelope.Data) == "null" {
		return &ListObjectsData{Objects: []ObjectPayload{}}, envelope, nil
	}

	// GET ?objectKey= returns a single object in data (same as GetObject), not a list page.
	if objectKey != "" {
		payload, parseErr := parseObjectPayload(envelope)
		if parseErr != nil {
			return nil, envelope, parseErr
		}
		return &ListObjectsData{Objects: []ObjectPayload{*payload}}, envelope, nil
	}

	var page ListObjectsData
	if err := json.Unmarshal(envelope.Data, &page); err != nil {
		var items []ObjectPayload
		if arrayErr := json.Unmarshal(envelope.Data, &items); arrayErr != nil {
			return nil, envelope, fmt.Errorf("failed to parse S3 object list response: %w", err)
		}
		page.Objects = items
	}

	if page.Objects == nil {
		page.Objects = []ObjectPayload{}
	}

	return &page, envelope, nil
}

// ListObjects returns all objects, following continuation_token until exhausted.
func ListObjects(c *client.Client, ctx context.Context, domainID, bucketName, prefix string) ([]ObjectPayload, *CatalystResponse, error) {
	var (
		all       []ObjectPayload
		token     string
		envelope  *CatalystResponse
	)

	for {
		page, env, err := ListObjectsPage(c, ctx, domainID, bucketName, prefix, token, "")
		if err != nil {
			return nil, env, err
		}
		envelope = env
		all = append(all, page.Objects...)
		if page.ContinuationToken == "" {
			break
		}
		token = page.ContinuationToken
	}

	tflog.Info(ctx, "S3 objects listed successfully", map[string]any{
		"domain_id":   domainID,
		"bucket_name": bucketName,
		"count":       len(all),
	})

	return all, envelope, nil
}

// DeleteObject removes an object via DELETE .../objects?objectKey=.
// Treats 404 as success (idempotent destroy).
// Uses a dedicated HTTP call (no JSON Content-Type) so objectKey is bound from the query string only.
func DeleteObject(c *client.Client, ctx context.Context, domainID, bucketName, objectKey string) (*CatalystResponse, error) {
	key := normalizeObjectKey(objectKey)
	if key == "" {
		return nil, fmt.Errorf("object_key is required")
	}

	tflog.Debug(ctx, "Deleting S3 object", map[string]any{
		"domain_id":   domainID,
		"bucket_name": bucketName,
		"object_key":  key,
	})

	token, err := c.GetToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get authentication token: %w", err)
	}

	// Build URL via url.URL; Encode() turns "/" into %2F in objectKey.
	parsed, err := url.Parse(client.APIURL + objectsBasePath(domainID, bucketName))
	if err != nil {
		return nil, fmt.Errorf("failed to parse delete URL: %w", err)
	}
	q := url.Values{}
	q.Set("objectKey", key)
	parsed.RawQuery = q.Encode()
	apiURL := parsed.String()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodDelete, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create delete request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+token)
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("X-Vayu-Client-Id", "vayu_iac")
	// Intentionally omit Content-Type: no body; objectKey is query-only.

	tflog.Debug(ctx, "Executing S3 object DELETE", map[string]any{
		"method":     http.MethodDelete,
		"url":        apiURL,
		"raw_query":  httpReq.URL.RawQuery,
		"object_key": key,
	})

	resp, err := c.GetHTTPClient().Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute delete request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read delete response body: %w", err)
	}

	tflog.Debug(ctx, "S3 object delete response", map[string]any{
		"status_code": resp.StatusCode,
		"body":        string(respBody),
		"url":         apiURL,
	})

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if isObjectNotFoundError(fmt.Errorf("status %d: %s", resp.StatusCode, string(respBody))) {
			tflog.Warn(ctx, "S3 object not found on DELETE; treating as already deleted", map[string]any{
				"domain_id":   domainID,
				"bucket_name": bucketName,
				"object_key":  key,
			})
			return nil, nil
		}
		var apiErr client.APIError
		if json.Unmarshal(respBody, &apiErr) == nil && (apiErr.Message != "" || apiErr.Code != "") {
			apiErr.StatusCode = resp.StatusCode
			if isObjectNotFoundError(&apiErr) {
				tflog.Warn(ctx, "S3 object not found on DELETE; treating as already deleted", map[string]any{
					"domain_id":   domainID,
					"bucket_name": bucketName,
					"object_key":  key,
				})
				return nil, nil
			}
			return nil, fmt.Errorf("failed to delete S3 object (url=%s objectKey=%q): %w", apiURL, key, &apiErr)
		}
		return nil, fmt.Errorf("failed to delete S3 object (url=%s objectKey=%q): status %d: %s", apiURL, key, resp.StatusCode, string(respBody))
	}

	envelope, err := parseEnvelope(respBody)
	if err != nil {
		return envelope, fmt.Errorf("failed to delete S3 object: %w", err)
	}

	tflog.Info(ctx, "S3 object deleted successfully", map[string]any{
		"domain_id":   domainID,
		"bucket_name": bucketName,
		"object_key":  key,
		"audit_id":    auditIDFromData(envelope.Data),
	})

	return envelope, nil
}
