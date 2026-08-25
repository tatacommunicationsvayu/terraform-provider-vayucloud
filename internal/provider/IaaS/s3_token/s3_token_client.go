// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package s3_token

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/client"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/common"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/s3_user"
)

// CatalystResponse is the standard ICS operations envelope.
type CatalystResponse struct {
	Status       string          `json:"status"`
	Message      string          `json:"message"`
	ResponseCode int             `json:"responseCode"`
	Data         json.RawMessage `json:"data"`
}

// TokenPayload is the create/read/list item shape from ICSS3OperationsController.
type TokenPayload struct {
	ID               string `json:"id"`
	DomainID         string `json:"domain_id"`
	Username         string `json:"username"`
	AccessToken      string `json:"access_token"`
	SecretKey        string `json:"secret_key,omitempty"`
	ExpiryDate       string `json:"expiry_date,omitempty"`
	TokenDescription string `json:"token_description,omitempty"`
	AuditID          string `json:"audit_id,omitempty"`
}

// CreateTokenRequest is the POST body for token create.
type CreateTokenRequest struct {
	TokenExpiryDate  string `json:"token_expiry_date,omitempty"`
	TokenDescription string `json:"token_description,omitempty"`
}

const tokenIDSeparator = ":"

func userTokensBasePath(domainID, username string) string {
	return fmt.Sprintf("%s/domain/%s/users/%s/tokens", common.ICSOperationsPath, domainID, url.PathEscape(username))
}

func domainTokensBasePath(domainID string) string {
	return fmt.Sprintf("%s/domain/%s/tokens", common.ICSOperationsPath, domainID)
}

func tokenItemPath(domainID, username, accessToken string) string {
	return fmt.Sprintf("%s/%s", userTokensBasePath(domainID, username), url.PathEscape(accessToken))
}

// FormatTokenID returns the composite Terraform resource id "{domain_id}:{username}:{access_token}".
func FormatTokenID(domainID, username, accessToken string) string {
	return domainID + tokenIDSeparator + username + tokenIDSeparator + accessToken
}

// ParseTokenID splits a composite id into domain_id, username, and access_token.
// Username must not contain ":" (enforced by API validation).
func ParseTokenID(id string) (domainID, username, accessToken string, err error) {
	parts := strings.Split(id, tokenIDSeparator)
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return "", "", "", fmt.Errorf("invalid token id %q: expected format {domain_id}:{username}:{access_token}", id)
	}
	return parts[0], parts[1], parts[2], nil
}

func isTokenNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "status 404") ||
		strings.Contains(msg, `"responsecode":404`) ||
		strings.Contains(msg, "token not found")
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

	tflog.Debug(ctx, "S3 token API response", map[string]any{
		"method": method,
		"path":   path,
		"body":   string(respBody),
	})

	return parseEnvelope(respBody)
}

// NormalizeTokenExpiryDate converts API values (e.g. 2027-06-16T00:00:00Z) to yyyy-MM-dd
// so refresh does not drift against user config and trigger ForceNew replacement.
func NormalizeTokenExpiryDate(expiry string) string {
	expiry = strings.TrimSpace(expiry)
	if expiry == "" {
		return ""
	}
	if t, err := time.Parse("2006-01-02", expiry); err == nil {
		return t.Format("2006-01-02")
	}
	layouts := []string{
		time.RFC3339,
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05.000Z",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, expiry); err == nil {
			return t.UTC().Format("2006-01-02")
		}
	}
	return expiry
}

// ValidateTokenExpiryDate checks yyyy-MM-dd format, not in past, and within one year.
func ValidateTokenExpiryDate(expiry string) error {
	expiry = NormalizeTokenExpiryDate(expiry)
	if expiry == "" {
		return fmt.Errorf("token_expiry_date is required for STANDARD storage class domains")
	}

	parsed, err := time.Parse("2006-01-02", expiry)
	if err != nil {
		return fmt.Errorf("token_expiry_date must be yyyy-MM-dd: %w", err)
	}

	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	if parsed.Before(today) {
		return fmt.Errorf("token_expiry_date %q must not be in the past", expiry)
	}

	maxExpiry := today.AddDate(1, 0, 0)
	if parsed.After(maxExpiry) {
		return fmt.Errorf("token_expiry_date %q must be within one year from today", expiry)
	}

	return nil
}

// ValidateCreateTokenRequest validates token create fields for the domain storage class when known.
func ValidateCreateTokenRequest(storageClass, expiry, description string) error {
	switch storageClass {
	case s3_user.StorageClassStandard:
		if strings.TrimSpace(description) == "" {
			return fmt.Errorf("token_description is required for STANDARD storage class domains")
		}
		if len(description) > 256 {
			return fmt.Errorf("token_description must be at most 256 characters")
		}
		return ValidateTokenExpiryDate(expiry)
	case s3_user.StorageClassHighPerformance, s3_user.StorageClassAIStandard:
		if description != "" && len(description) > 256 {
			return fmt.Errorf("token_description must be at most 256 characters")
		}
		if expiry != "" {
			return ValidateTokenExpiryDate(expiry)
		}
		return nil
	default:
		// Unknown storage class — validate format when fields are provided.
		if description != "" && len(description) > 256 {
			return fmt.Errorf("token_description must be at most 256 characters")
		}
		if expiry != "" {
			return ValidateTokenExpiryDate(expiry)
		}
		return nil
	}
}

// CreateToken sends POST /ics-operations/domain/{domain_id}/users/{username}/tokens.
func CreateToken(c *client.Client, ctx context.Context, domainID, username string, req *CreateTokenRequest) (*TokenPayload, *CatalystResponse, error) {
	tflog.Debug(ctx, "Creating S3 token", map[string]any{
		"domain_id": domainID,
		"username":  username,
	})

	var body interface{}
	if req != nil && (req.TokenExpiryDate != "" || req.TokenDescription != "") {
		body = req
	} else {
		body = map[string]any{}
	}

	envelope, err := doICSRequest(c, ctx, http.MethodPost, userTokensBasePath(domainID, username), body)
	if err != nil {
		return nil, envelope, fmt.Errorf("failed to create S3 token: %w", err)
	}

	payload, err := parseTokenPayload(envelope)
	if err != nil {
		return nil, envelope, err
	}

	tflog.Info(ctx, "S3 token created successfully", map[string]any{
		"id":           payload.ID,
		"domain_id":    payload.DomainID,
		"username":     payload.Username,
		"access_token": payload.AccessToken,
	})

	return payload, envelope, nil
}

// GetToken reads a single token via GET .../users/{username}/tokens/{access_token}.
func GetToken(c *client.Client, ctx context.Context, domainID, username, accessToken string) (*TokenPayload, *CatalystResponse, error) {
	tflog.Debug(ctx, "Reading S3 token", map[string]any{
		"domain_id":    domainID,
		"username":     username,
		"access_token": accessToken,
	})

	envelope, err := doICSRequest(c, ctx, http.MethodGet, tokenItemPath(domainID, username, accessToken), nil)
	if err != nil {
		return nil, envelope, fmt.Errorf("failed to read S3 token: %w", err)
	}

	payload, err := parseTokenPayload(envelope)
	if err != nil {
		return nil, envelope, err
	}

	return payload, envelope, nil
}

// ListUserTokens returns tokens for a user. accessToken filter is optional (query param).
func ListUserTokens(c *client.Client, ctx context.Context, domainID, username, accessToken string) ([]TokenPayload, *CatalystResponse, error) {
	path := userTokensBasePath(domainID, username)
	if accessToken != "" {
		path = fmt.Sprintf("%s?accessToken=%s", path, url.QueryEscape(accessToken))
	}

	tflog.Debug(ctx, "Listing S3 tokens for user", map[string]any{
		"domain_id":    domainID,
		"username":     username,
		"access_token": accessToken,
	})

	envelope, err := doICSRequest(c, ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, envelope, fmt.Errorf("failed to list S3 tokens: %w", err)
	}

	return parseTokenList(ctx, envelope, domainID)
}

// ListDomainTokens returns all tokens in a domain. username and accessToken filters are optional.
func ListDomainTokens(c *client.Client, ctx context.Context, domainID, username, accessToken string) ([]TokenPayload, *CatalystResponse, error) {
	path := domainTokensBasePath(domainID)
	params := url.Values{}
	if username != "" {
		params.Set("username", username)
	}
	if accessToken != "" {
		params.Set("accessToken", accessToken)
	}
	if encoded := params.Encode(); encoded != "" {
		path = path + "?" + encoded
	}

	tflog.Debug(ctx, "Listing S3 tokens in domain", map[string]any{
		"domain_id":    domainID,
		"username":     username,
		"access_token": accessToken,
	})

	envelope, err := doICSRequest(c, ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, envelope, fmt.Errorf("failed to list S3 domain tokens: %w", err)
	}

	return parseTokenList(ctx, envelope, domainID)
}

// DeleteToken sends DELETE .../users/{username}/tokens/{access_token}.
// Treats HTTP 404 as success (idempotent destroy).
func DeleteToken(c *client.Client, ctx context.Context, domainID, username, accessToken string) (*CatalystResponse, error) {
	tflog.Debug(ctx, "Deleting S3 token", map[string]any{
		"domain_id":    domainID,
		"username":     username,
		"access_token": accessToken,
	})

	envelope, err := doICSRequest(c, ctx, http.MethodDelete, tokenItemPath(domainID, username, accessToken), nil)
	if err != nil {
		if isTokenNotFoundError(err) {
			tflog.Warn(ctx, "S3 token not found on DELETE; treating as already deleted", map[string]any{
				"domain_id":    domainID,
				"username":     username,
				"access_token": accessToken,
			})
			return nil, nil
		}
		return envelope, fmt.Errorf("failed to delete S3 token: %w", err)
	}

	tflog.Info(ctx, "S3 token deleted successfully", map[string]any{
		"domain_id":    domainID,
		"username":     username,
		"access_token": accessToken,
		"audit_id":     auditIDFromData(envelope.Data),
	})

	return envelope, nil
}

func parseTokenList(ctx context.Context, envelope *CatalystResponse, domainID string) ([]TokenPayload, *CatalystResponse, error) {
	if len(envelope.Data) == 0 || string(envelope.Data) == "null" {
		return []TokenPayload{}, envelope, nil
	}

	var items []TokenPayload
	if err := json.Unmarshal(envelope.Data, &items); err != nil {
		return nil, envelope, fmt.Errorf("failed to parse S3 token list response: %w", err)
	}

	tflog.Info(ctx, "S3 tokens listed successfully", map[string]any{
		"domain_id": domainID,
		"count":     len(items),
	})

	return items, envelope, nil
}

func parseTokenPayload(envelope *CatalystResponse) (*TokenPayload, error) {
	if envelope == nil || len(envelope.Data) == 0 || string(envelope.Data) == "null" {
		return nil, fmt.Errorf("empty token payload in API response")
	}

	var payload TokenPayload
	if err := json.Unmarshal(envelope.Data, &payload); err != nil {
		return nil, fmt.Errorf("failed to parse token payload: %w", err)
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
