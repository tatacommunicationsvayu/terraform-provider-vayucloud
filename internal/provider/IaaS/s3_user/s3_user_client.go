// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package s3_user

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/client"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/common"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/s3_bucket"
)

// CatalystResponse is the standard ICS operations envelope.
type CatalystResponse struct {
	Status       string          `json:"status"`
	Message      string          `json:"message"`
	ResponseCode int             `json:"responseCode"`
	Data         json.RawMessage `json:"data"`
}

// UserPayload is the create/read/list item shape from ICSS3OperationsController.
type UserPayload struct {
	ID       string `json:"id"`
	DomainID string `json:"domain_id"`
	Username string `json:"username"`
	AuditID  string `json:"audit_id,omitempty"`
}

// CreateUserRequest is the POST body for user create.
type CreateUserRequest struct {
	Username string `json:"username"`
}

const (
	userIDSeparator             = ":"
	StorageClassStandard        = "STANDARD"
	StorageClassHighPerformance = "HIGH_PERFORMANCE"
	StorageClassAIStandard      = "AI_STANDARD"
)

func usersBasePath(domainID string) string {
	return fmt.Sprintf("%s/domain/%s/users", common.ICSOperationsPath, domainID)
}

func userItemPath(domainID, username string) string {
	return fmt.Sprintf("%s/%s", usersBasePath(domainID), url.PathEscape(username))
}

// FormatUserID returns the composite Terraform resource id "{domain_id}:{username}".
func FormatUserID(domainID, username string) string {
	return domainID + userIDSeparator + username
}

// ParseUserID splits a composite id into domain_id and username.
func ParseUserID(id string) (domainID, username string, err error) {
	idx := strings.Index(id, userIDSeparator)
	if idx <= 0 || idx >= len(id)-1 {
		return "", "", fmt.Errorf("invalid user id %q: expected format {domain_id}:{username}", id)
	}
	return id[:idx], id[idx+1:], nil
}

func isUserNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "status 404") ||
		strings.Contains(msg, `"responsecode":404`) ||
		strings.Contains(msg, "user not found")
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

	tflog.Debug(ctx, "S3 user API response", map[string]any{
		"method": method,
		"path":   path,
		"body":   string(respBody),
	})

	return parseEnvelope(respBody)
}

// ResolveDomainStorageClass returns storage_class from the first bucket in the domain when available.
// Empty string when unknown (no buckets yet).
func ResolveDomainStorageClass(c *client.Client, ctx context.Context, domainID string) (string, error) {
	buckets, _, err := s3_bucket.ListBuckets(c, ctx, domainID)
	if err != nil {
		if isDomainNotFoundError(err) {
			return "", fmt.Errorf("domain %s not found or inactive: %w", domainID, err)
		}
		return "", err
	}
	for _, b := range buckets {
		if b.StorageClass != "" {
			return b.StorageClass, nil
		}
	}
	return "", nil
}

// UserExists checks whether a username is already mapped in the domain.
// Used only for pre-create availability (ValidateUserNameAvailable): GET single-user;
// 404 means the name is free, 200 means taken.
func UserExists(c *client.Client, ctx context.Context, domainID, username string) (bool, error) {
	_, _, err := GetUser(c, ctx, domainID, username)
	if err == nil {
		return true, nil
	}
	if isDomainNotFoundError(err) {
		return false, fmt.Errorf("domain %s not found or inactive: %w", domainID, err)
	}
	if isUserNotFoundError(err) {
		return false, nil
	}
	return false, err
}

// ValidateUserNameAvailable ensures the username is not already mapped.
// Called from ModifyPlan (create) and Create only — not for existing-resource checks.
func ValidateUserNameAvailable(c *client.Client, ctx context.Context, domainID, username string) error {
	exists, err := UserExists(c, ctx, domainID, username)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("username %q already exists in domain %s", username, domainID)
	}
	return nil
}

// ValidateUserExists checks that a managed user still exists (existing resource / refresh).
// 404 is an error here — opposite of pre-create availability in ValidateUserNameAvailable.
func ValidateUserExists(c *client.Client, ctx context.Context, domainID, username string) error {
	_, _, err := GetUser(c, ctx, domainID, username)
	if err != nil {
		return fmt.Errorf("user %q in domain %s is not available: %w", username, domainID, err)
	}
	return nil
}

// CreateUser sends POST /ics-operations/domain/{domain_id}/users.
func CreateUser(c *client.Client, ctx context.Context, domainID string, req *CreateUserRequest) (*UserPayload, *CatalystResponse, error) {
	tflog.Debug(ctx, "Creating S3 user", map[string]any{
		"domain_id": domainID,
		"username":  req.Username,
	})

	envelope, err := doICSRequest(c, ctx, http.MethodPost, usersBasePath(domainID), req)
	if err != nil {
		return nil, envelope, fmt.Errorf("failed to create S3 user: %w", err)
	}

	payload, err := parseUserPayload(envelope)
	if err != nil {
		return nil, envelope, err
	}

	tflog.Info(ctx, "S3 user created successfully", map[string]any{
		"id":        payload.ID,
		"domain_id": payload.DomainID,
		"username":  payload.Username,
	})

	return payload, envelope, nil
}

// GetUser reads a single user via GET /ics-operations/domain/{domain_id}/users/{username}.
func GetUser(c *client.Client, ctx context.Context, domainID, username string) (*UserPayload, *CatalystResponse, error) {
	tflog.Debug(ctx, "Reading S3 user", map[string]any{
		"domain_id": domainID,
		"username":  username,
	})

	envelope, err := doICSRequest(c, ctx, http.MethodGet, userItemPath(domainID, username), nil)
	if err != nil {
		return nil, envelope, fmt.Errorf("failed to read S3 user: %w", err)
	}

	payload, err := parseUserPayload(envelope)
	if err != nil {
		return nil, envelope, err
	}

	return payload, envelope, nil
}

// ListUsers returns users for a domain. username filter is optional (query param).
func ListUsers(c *client.Client, ctx context.Context, domainID, username string) ([]UserPayload, *CatalystResponse, error) {
	path := usersBasePath(domainID)
	if username != "" {
		path = fmt.Sprintf("%s?username=%s", path, url.QueryEscape(username))
	}

	tflog.Debug(ctx, "Listing S3 users", map[string]any{
		"domain_id": domainID,
		"username":  username,
	})

	envelope, err := doICSRequest(c, ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, envelope, fmt.Errorf("failed to list S3 users: %w", err)
	}

	if len(envelope.Data) == 0 || string(envelope.Data) == "null" {
		return []UserPayload{}, envelope, nil
	}

	var items []UserPayload
	if err := json.Unmarshal(envelope.Data, &items); err != nil {
		return nil, envelope, fmt.Errorf("failed to parse S3 user list response: %w", err)
	}

	tflog.Info(ctx, "S3 users listed successfully", map[string]any{
		"domain_id": domainID,
		"count":     len(items),
	})

	return items, envelope, nil
}

// DeleteUser sends DELETE /ics-operations/domain/{domain_id}/users/{username}.
// Cascades token deletion. Treats HTTP 404 as success (idempotent destroy).
func DeleteUser(c *client.Client, ctx context.Context, domainID, username string) (*CatalystResponse, error) {
	tflog.Debug(ctx, "Deleting S3 user", map[string]any{
		"domain_id": domainID,
		"username":  username,
	})

	envelope, err := doICSRequest(c, ctx, http.MethodDelete, userItemPath(domainID, username), nil)
	if err != nil {
		if isUserNotFoundError(err) {
			tflog.Warn(ctx, "S3 user not found on DELETE; treating as already deleted", map[string]any{
				"domain_id": domainID,
				"username":  username,
			})
			return nil, nil
		}
		return envelope, fmt.Errorf("failed to delete S3 user: %w", err)
	}

	tflog.Info(ctx, "S3 user deleted successfully", map[string]any{
		"domain_id": domainID,
		"username":  username,
		"audit_id":  auditIDFromData(envelope.Data),
	})

	return envelope, nil
}

func parseUserPayload(envelope *CatalystResponse) (*UserPayload, error) {
	if envelope == nil || len(envelope.Data) == 0 || string(envelope.Data) == "null" {
		return nil, fmt.Errorf("empty user payload in API response")
	}

	var payload UserPayload
	if err := json.Unmarshal(envelope.Data, &payload); err != nil {
		return nil, fmt.Errorf("failed to parse user payload: %w", err)
	}
	return &payload, nil
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
