// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package s3_bucket

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/client"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/common"
)

// CatalystResponse is the standard ICS operations envelope (snake_case fields in data payloads).
type CatalystResponse struct {
	Status       string          `json:"status"`
	Message      string          `json:"message"`
	ResponseCode int             `json:"responseCode"`
	Data         json.RawMessage `json:"data"`
}

// BucketPayload is the create/read/update/list item shape from ICSS3OperationsController.
type BucketPayload struct {
	ID               string `json:"id"`
	DomainID         string `json:"domain_id"`
	BucketName       string `json:"bucket_name"`
	CreatedAt        string `json:"created_at,omitempty"`
	StorageClass     string `json:"storage_class,omitempty"`
	VersioningStatus string `json:"versioning_status"`
	AuditID          string `json:"audit_id,omitempty"`
}

// BucketAvailability is the availability check response data object.
type BucketAvailability struct {
	DomainID   string `json:"domain_id"`
	BucketName string `json:"bucket_name"`
	Available  bool   `json:"available"`
}

// CreateBucketRequest is the POST body for bucket create.
// VersioningEnabled is always sent; omit in HCL defaults to false in the provider.
type CreateBucketRequest struct {
	BucketName        string `json:"bucket_name"`
	VersioningEnabled bool   `json:"versioning_enabled"`
}

// UpdateBucketRequest is the PUT body for bucket update (versioning only).
type UpdateBucketRequest struct {
	VersioningEnabled bool `json:"versioning_enabled"`
}

// BucketDeleteData is the delete-success payload (audit_id only).
type BucketDeleteData struct {
	AuditID string `json:"audit_id"`
}

// BucketAPIError is a structured ICS bucket API failure for Terraform diagnostics.
type BucketAPIError struct {
	Operation     bucketOperation
	HTTPCode      int
	ResponseCode  int
	Message       string
	AuditIDHeader string
	PartialCreate bool
}

func (e *BucketAPIError) Error() string {
	code := e.effectiveCode()
	msg := fmt.Sprintf("Bucket %s failed: HTTP %d — %s", e.Operation, code, e.Message)
	if e.AuditIDHeader != "" {
		msg += fmt.Sprintf(" (audit_id header: %s)", e.AuditIDHeader)
	}
	if e.PartialCreate {
		msg += ". Backend may have created the bucket before versioning failed; check backend or import"
	}
	return msg
}

func (e *BucketAPIError) effectiveCode() int {
	if e.ResponseCode != 0 {
		return e.ResponseCode
	}
	return e.HTTPCode
}

type bucketOperation string

const (
	auditIDResponseHeader = "AUDIT_ID"

	bucketOpCreate       bucketOperation = "create"
	bucketOpUpdate       bucketOperation = "update"
	bucketOpDelete       bucketOperation = "delete"
	bucketOpRead         bucketOperation = "read"
	bucketOpList         bucketOperation = "list"
	bucketOpAvailability bucketOperation = "availability"
)

const bucketIDSeparator = ":"

func bucketsBasePath(domainID string) string {
	return fmt.Sprintf("%s/domain/%s/buckets", common.ICSOperationsPath, domainID)
}

func bucketItemPath(domainID, bucketName string) string {
	return fmt.Sprintf("%s/%s", bucketsBasePath(domainID), url.PathEscape(bucketName))
}

func availabilityPath(domainID, bucketName string) string {
	return fmt.Sprintf("%s/availability?bucketName=%s", bucketsBasePath(domainID), url.QueryEscape(bucketName))
}

// FormatBucketID returns the composite Terraform resource id "{domain_id}:{bucket_name}".
func FormatBucketID(domainID, bucketName string) string {
	return domainID + bucketIDSeparator + bucketName
}

// ParseBucketID splits a composite id into domain_id and bucket_name.
// Only the first ":" separates domain_id from bucket_name.
func ParseBucketID(id string) (domainID, bucketName string, err error) {
	idx := strings.Index(id, bucketIDSeparator)
	if idx <= 0 || idx >= len(id)-1 {
		return "", "", fmt.Errorf("invalid bucket id %q: expected format {domain_id}:{bucket_name}", id)
	}
	return id[:idx], id[idx+1:], nil
}

func versioningEnabledFromStatus(status string) bool {
	return status == "Enabled"
}

// icsErrorMessageAndCode extracts the best available ICS/Catalyst error text from a response body.
func icsErrorMessageAndCode(respBody []byte, fallbackMsg string, fallbackCode int) (string, int) {
	msg := strings.TrimSpace(fallbackMsg)
	code := fallbackCode

	var fields map[string]json.RawMessage
	if json.Unmarshal(respBody, &fields) == nil {
		for _, key := range []string{"message", "errorMessage", "error_message", "detail"} {
			if v, ok := fields[key]; ok {
				var s string
				if json.Unmarshal(v, &s) == nil {
					s = strings.TrimSpace(s)
					if s != "" && (len(s) > len(msg) || isGenericICSErrorText(msg)) && !isGenericICSErrorText(s) {
						msg = s
					}
				}
			}
		}
		for _, key := range []string{"responseCode", "response_code", "code"} {
			if v, ok := fields[key]; ok {
				var c int
				if json.Unmarshal(v, &c) == nil && c != 0 {
					code = c
					break
				}
			}
		}
	}

	if msg == "" || isGenericICSErrorText(msg) {
		raw := strings.TrimSpace(string(respBody))
		if raw != "" {
			if better := longestICSMessageFromJSON(raw); better != "" {
				msg = better
			} else if isGenericICSErrorText(msg) {
				msg = raw
			}
		}
	}
	if msg == "" {
		msg = "unknown error"
	}
	return msg, code
}

func longestICSMessageFromJSON(raw string) string {
	var fields map[string]json.RawMessage
	if json.Unmarshal([]byte(raw), &fields) != nil {
		return ""
	}
	best := ""
	for _, key := range []string{"message", "errorMessage", "error_message", "detail"} {
		if v, ok := fields[key]; ok {
			var s string
			if json.Unmarshal(v, &s) == nil {
				s = strings.TrimSpace(s)
				if len(s) > len(best) && !isGenericICSErrorText(s) {
					best = s
				}
			}
		}
	}
	return best
}

func isGenericICSErrorText(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "internal server error", "error", "bad request":
		return true
	default:
		return false
	}
}

func auditIDFromHeaders(headers http.Header) string {
	if headers == nil {
		return ""
	}
	return strings.TrimSpace(headers.Get(auditIDResponseHeader))
}

func parseBucketEnvelope(respBody []byte, httpStatus int) (*CatalystResponse, error) {
	var envelope CatalystResponse
	if err := json.Unmarshal(respBody, &envelope); err != nil {
		if httpStatus >= 400 {
			return nil, fmt.Errorf("failed to parse API error response (HTTP %d): %w", httpStatus, err)
		}
		return nil, fmt.Errorf("failed to parse API response: %w", err)
	}

	if envelope.ResponseCode == 0 {
		var alt struct {
			ResponseCode int `json:"response_code"`
		}
		if json.Unmarshal(respBody, &alt) == nil && alt.ResponseCode != 0 {
			envelope.ResponseCode = alt.ResponseCode
		}
	}
	if envelope.ResponseCode == 0 && httpStatus != 0 {
		envelope.ResponseCode = httpStatus
	}

	if httpStatus >= 400 || strings.EqualFold(envelope.Status, "error") {
		msg, code := icsErrorMessageAndCode(respBody, envelope.Message, envelope.ResponseCode)
		return &envelope, &bucketEnvelopeError{message: msg, responseCode: code}
	}

	if envelope.Status != "" && !strings.EqualFold(envelope.Status, "success") {
		msg, code := icsErrorMessageAndCode(respBody, envelope.Message, envelope.ResponseCode)
		return &envelope, &bucketEnvelopeError{message: msg, responseCode: code}
	}

	return &envelope, nil
}

type bucketEnvelopeError struct {
	message      string
	responseCode int
}

func (e *bucketEnvelopeError) Error() string {
	return fmt.Sprintf("%s (code: %d)", e.message, e.responseCode)
}

func isPartialCreateFailure(op bucketOperation, code int, message string) bool {
	if op != bucketOpCreate || code != http.StatusInternalServerError {
		return false
	}
	lower := strings.ToLower(message)
	return strings.Contains(lower, "error configuring version mode") ||
		(strings.Contains(lower, "version") && strings.Contains(lower, "bucket"))
}

func doICSRequest(c *client.Client, ctx context.Context, method, path string, body interface{}, op bucketOperation) (*CatalystResponse, error) {
	httpResp, err := c.DoICSRequestEx(ctx, method, path, body)
	if err != nil {
		return nil, err
	}

	tflog.Debug(ctx, "S3 bucket API response", map[string]any{
		"method":      method,
		"path":        path,
		"http_status": httpResp.StatusCode,
		"audit_id":    auditIDFromHeaders(httpResp.Headers),
		"body":        string(httpResp.Body),
	})

	envelope, parseErr := parseBucketEnvelope(httpResp.Body, httpResp.StatusCode)
	if parseErr != nil {
		message := parseErr.Error()
		responseCode := httpResp.StatusCode

		var envErr *bucketEnvelopeError
		if errors.As(parseErr, &envErr) {
			message = envErr.message
			responseCode = envErr.responseCode
		} else if envelope != nil {
			if trimmed := strings.TrimSpace(envelope.Message); trimmed != "" {
				message = trimmed
			}
			if envelope.ResponseCode != 0 {
				responseCode = envelope.ResponseCode
			}
		}

		apiErr := &BucketAPIError{
			Operation:     op,
			HTTPCode:      httpResp.StatusCode,
			ResponseCode:  responseCode,
			Message:       message,
			AuditIDHeader: auditIDFromHeaders(httpResp.Headers),
		}
		apiErr.PartialCreate = isPartialCreateFailure(op, apiErr.effectiveCode(), message)

		tflog.Error(ctx, "S3 bucket ICS API error response", map[string]any{
			"operation":      op,
			"method":         method,
			"path":           path,
			"http_status":    httpResp.StatusCode,
			"response_code":  apiErr.ResponseCode,
			"message":        apiErr.Message,
			"audit_id":       apiErr.AuditIDHeader,
			"partial_create": apiErr.PartialCreate,
			"body":           string(httpResp.Body),
		})
		return envelope, apiErr
	}

	return envelope, nil
}

func bucketErrorCode(err error) int {
	var apiErr *BucketAPIError
	if errors.As(err, &apiErr) {
		return apiErr.effectiveCode()
	}
	return 0
}

func bucketErrorMessage(err error) string {
	var apiErr *BucketAPIError
	if errors.As(err, &apiErr) {
		return apiErr.Message
	}
	if err != nil {
		return err.Error()
	}
	return ""
}

func isHTTPNotFound(err error) bool {
	code := bucketErrorCode(err)
	if code == http.StatusNotFound {
		return true
	}
	msg := strings.ToLower(bucketErrorMessage(err))
	return strings.Contains(msg, "http 404") || strings.Contains(msg, `"responsecode":404`)
}

func isBucketNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	if isHTTPNotFound(err) {
		msg := strings.ToLower(bucketErrorMessage(err))
		return strings.Contains(msg, "bucket not found") ||
			(strings.Contains(msg, "bucket") && strings.Contains(msg, "not found"))
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "bucket not found")
}

func isDomainNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	if isHTTPNotFound(err) {
		msg := strings.ToLower(bucketErrorMessage(err))
		return strings.Contains(msg, "domain not found") ||
			(strings.Contains(msg, "domain") && strings.Contains(msg, "not found"))
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "domain not found")
}

func isLikelyInvalidDomainError(err error) bool {
	if err == nil || isDomainNotFoundError(err) {
		return false
	}
	code := bucketErrorCode(err)
	if code == http.StatusInternalServerError {
		msg := strings.ToLower(bucketErrorMessage(err))
		return strings.Contains(msg, "internal server error")
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "http 500") && strings.Contains(msg, "internal server error")
}

func isBucketAlreadyExistsError(err error) bool {
	if err == nil {
		return false
	}
	code := bucketErrorCode(err)
	if code == http.StatusConflict {
		return true
	}
	msg := strings.ToLower(bucketErrorMessage(err))
	return strings.Contains(msg, "already exists")
}

func isBucketNotEmptyError(err error) bool {
	if err == nil {
		return false
	}
	code := bucketErrorCode(err)
	if code == http.StatusBadRequest {
		msg := strings.ToLower(bucketErrorMessage(err))
		return strings.Contains(msg, "must be empty") || strings.Contains(msg, "not empty")
	}
	msg := strings.ToLower(bucketErrorMessage(err))
	return strings.Contains(msg, "must be empty before deletion")
}

func parseBucketPayload(envelope *CatalystResponse) (*BucketPayload, error) {
	if envelope == nil || len(envelope.Data) == 0 || string(envelope.Data) == "null" {
		return nil, fmt.Errorf("empty bucket payload in API response")
	}

	var payload BucketPayload
	if err := json.Unmarshal(envelope.Data, &payload); err != nil {
		return nil, fmt.Errorf("failed to parse bucket payload: %w", err)
	}
	return &payload, nil
}

func parseDeleteAuditID(envelope *CatalystResponse) string {
	if envelope == nil {
		return ""
	}
	var data BucketDeleteData
	if len(envelope.Data) == 0 || string(envelope.Data) == "null" {
		return ""
	}
	if err := json.Unmarshal(envelope.Data, &data); err != nil {
		return auditIDFromData(envelope.Data)
	}
	return data.AuditID
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

// CheckBucketAvailability calls GET .../buckets/availability for plan-time validation.
func CheckBucketAvailability(c *client.Client, ctx context.Context, domainID, bucketName string) (*BucketAvailability, *CatalystResponse, error) {
	tflog.Debug(ctx, "Checking S3 bucket name availability", map[string]any{
		"domain_id":   domainID,
		"bucket_name": bucketName,
	})

	envelope, err := doICSRequest(c, ctx, http.MethodGet, availabilityPath(domainID, bucketName), nil, bucketOpAvailability)
	if err != nil {
		return nil, envelope, err
	}

	var data BucketAvailability
	if len(envelope.Data) > 0 && string(envelope.Data) != "null" {
		if err := json.Unmarshal(envelope.Data, &data); err != nil {
			return nil, envelope, fmt.Errorf("failed to parse bucket availability response: %w", err)
		}
	}

	tflog.Debug(ctx, "S3 bucket availability result", map[string]any{
		"domain_id":   domainID,
		"bucket_name": bucketName,
		"available":   data.Available,
	})

	return &data, envelope, nil
}

// ValidateDomainExistsByID checks that an ICS S3 domain with the given ID exists and is active.
// Uses GET .../domain/{domain_id}/buckets (an empty bucket list is OK).
// Exported for cross-resource ModifyPlan validation (bucket, user, token, object).
func ValidateDomainExistsByID(c *client.Client, ctx context.Context, domainID string) error {
	tflog.Debug(ctx, "Validating S3 domain exists by id", map[string]any{
		"domain_id": domainID,
	})

	_, _, err := ListBuckets(c, ctx, domainID)
	if err == nil {
		tflog.Debug(ctx, "S3 domain ID validated successfully", map[string]any{
			"domain_id": domainID,
		})
		return nil
	}
	if isDomainNotFoundError(err) || isLikelyInvalidDomainError(err) {
		return fmt.Errorf("domain %s not found or inactive: %w", domainID, err)
	}
	return fmt.Errorf("failed to validate domain %s: %w", domainID, err)
}

// ValidateBucketNameAvailable ensures the bucket name is free (plan/create pre-check).
// Bucket names are globally unique across all domains.
func ValidateBucketNameAvailable(c *client.Client, ctx context.Context, domainID, bucketName string) error {
	data, _, err := CheckBucketAvailability(c, ctx, domainID, bucketName)
	if err != nil {
		if isDomainNotFoundError(err) || isLikelyInvalidDomainError(err) {
			return fmt.Errorf("domain %s not found or inactive: %w", domainID, err)
		}
		return err
	}
	if !data.Available {
		return fmt.Errorf("bucket name %q already exists", bucketName)
	}
	return nil
}

// ValidateBucketExists checks that a managed bucket still exists via GET single-bucket.
func ValidateBucketExists(c *client.Client, ctx context.Context, domainID, bucketName string) error {
	_, _, err := GetBucket(c, ctx, domainID, bucketName)
	if err == nil {
		return nil
	}
	if isDomainNotFoundError(err) || isLikelyInvalidDomainError(err) {
		return fmt.Errorf("domain %s not found or inactive: %w", domainID, err)
	}
	if isBucketNotFoundError(err) {
		return fmt.Errorf("bucket %q was not found in domain %s", bucketName, domainID)
	}
	return fmt.Errorf("failed to read bucket %q in domain %s: %w", bucketName, domainID, err)
}

// CreateBucket sends POST /ics-operations/domain/{domain_id}/buckets.
func CreateBucket(c *client.Client, ctx context.Context, domainID string, req *CreateBucketRequest) (*BucketPayload, *CatalystResponse, error) {
	tflog.Debug(ctx, "Creating S3 bucket", map[string]any{
		"domain_id":   domainID,
		"bucket_name": req.BucketName,
	})

	envelope, err := doICSRequest(c, ctx, http.MethodPost, bucketsBasePath(domainID), req, bucketOpCreate)
	if err != nil {
		return nil, envelope, err
	}

	payload, err := parseBucketPayload(envelope)
	if err != nil {
		return nil, envelope, err
	}

	tflog.Info(ctx, fmt.Sprintf("Created bucket %s in domain %s (audit_id: %s)",
		payload.BucketName, payload.DomainID, payload.AuditID), map[string]any{
		"id":          payload.ID,
		"domain_id":   payload.DomainID,
		"bucket_name": payload.BucketName,
		"audit_id":    payload.AuditID,
	})

	return payload, envelope, nil
}

// GetBucket reads a single bucket via GET /ics-operations/domain/{domain_id}/buckets/{bucket_name}.
func GetBucket(c *client.Client, ctx context.Context, domainID, bucketName string) (*BucketPayload, *CatalystResponse, error) {
	tflog.Debug(ctx, "Reading S3 bucket", map[string]any{
		"domain_id":   domainID,
		"bucket_name": bucketName,
	})

	envelope, err := doICSRequest(c, ctx, http.MethodGet, bucketItemPath(domainID, bucketName), nil, bucketOpRead)
	if err != nil {
		return nil, envelope, err
	}

	payload, err := parseBucketPayload(envelope)
	if err != nil {
		return nil, envelope, err
	}

	return payload, envelope, nil
}

// ListBuckets returns all buckets for a domain.
func ListBuckets(c *client.Client, ctx context.Context, domainID string) ([]BucketPayload, *CatalystResponse, error) {
	tflog.Debug(ctx, "Listing S3 buckets", map[string]any{
		"domain_id": domainID,
	})

	envelope, err := doICSRequest(c, ctx, http.MethodGet, bucketsBasePath(domainID), nil, bucketOpList)
	if err != nil {
		return nil, envelope, err
	}

	if len(envelope.Data) == 0 || string(envelope.Data) == "null" {
		return []BucketPayload{}, envelope, nil
	}

	var items []BucketPayload
	if err := json.Unmarshal(envelope.Data, &items); err != nil {
		return nil, envelope, fmt.Errorf("failed to parse S3 bucket list response: %w", err)
	}

	tflog.Info(ctx, "S3 buckets listed successfully", map[string]any{
		"domain_id": domainID,
		"count":     len(items),
	})

	return items, envelope, nil
}

// UpdateBucket sends PUT to update bucket versioning.
func UpdateBucket(c *client.Client, ctx context.Context, domainID, bucketName string, req *UpdateBucketRequest) (*BucketPayload, *CatalystResponse, error) {
	tflog.Debug(ctx, "Updating S3 bucket", map[string]any{
		"domain_id":          domainID,
		"bucket_name":        bucketName,
		"versioning_enabled": req.VersioningEnabled,
	})

	envelope, err := doICSRequest(c, ctx, http.MethodPut, bucketItemPath(domainID, bucketName), req, bucketOpUpdate)
	if err != nil {
		return nil, envelope, err
	}

	payload, err := parseBucketPayload(envelope)
	if err != nil {
		return nil, envelope, err
	}

	tflog.Info(ctx, fmt.Sprintf("Updated bucket %s (audit_id: %s)", payload.BucketName, payload.AuditID), map[string]any{
		"id":          payload.ID,
		"domain_id":   payload.DomainID,
		"bucket_name": payload.BucketName,
		"audit_id":    payload.AuditID,
	})

	return payload, envelope, nil
}

// DeleteBucket sends DELETE /ics-operations/domain/{domain_id}/buckets/{bucket_name}.
// Returns nil envelope on success. Treats HTTP 404 as success (idempotent destroy).
func DeleteBucket(c *client.Client, ctx context.Context, domainID, bucketName string) (*CatalystResponse, error) {
	tflog.Debug(ctx, "Deleting S3 bucket", map[string]any{
		"domain_id":   domainID,
		"bucket_name": bucketName,
	})

	envelope, err := doICSRequest(c, ctx, http.MethodDelete, bucketItemPath(domainID, bucketName), nil, bucketOpDelete)
	if err != nil {
		if isBucketNotFoundError(err) {
			tflog.Warn(ctx, "S3 bucket not found on DELETE; treating as already deleted", map[string]any{
				"domain_id":   domainID,
				"bucket_name": bucketName,
			})
			return nil, nil
		}
		return envelope, err
	}

	auditID := parseDeleteAuditID(envelope)
	tflog.Info(ctx, fmt.Sprintf("Deleted bucket %s (audit_id: %s)", bucketName, auditID), map[string]any{
		"domain_id":   domainID,
		"bucket_name": bucketName,
		"audit_id":    auditID,
	})

	return envelope, nil
}
