// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package keypair

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/client"
)

// KeypairResponse represents the standard API response for keypair operations.
type KeypairResponse struct {
	// Status is the response status (e.g., "success")
	Status string `json:"status"`

	// Data contains the response data or message
	Data string `json:"data"`

	// Message contains additional information
	Message string `json:"message"`

	// ResponseCode is the API response code (0 = success)
	ResponseCode int `json:"responseCode"`
}

// KeypairCreateResponse represents the response from creating a keypair.
// When creating a keypair, the API returns the private key file content.
type KeypairCreateResponse struct {
	// PrivateKeyContent contains the private key file content (.pem or .ppk)
	PrivateKeyContent string

	// Filename is the name of the private key file
	Filename string

	// Format is the file format (pem or ppk)
	Format string
}

// Keypair represents a keypair resource from the list API.
type Keypair struct {
	// ID is the unique identifier of the keypair
	ID int64 `json:"id"`

	// Name is the keypair name (API returns "keypair_name")
	Name string `json:"keypair_name"`

	// KeypairType is the type of keypair (e.g., "RSA")
	KeypairType string `json:"keypair_type,omitempty"`

	// EngagementID is the engagement ID associated with the keypair
	EngagementID int64 `json:"engagement_id,omitempty"`

	// Fingerprint is the key fingerprint
	Fingerprint string `json:"fingerprint,omitempty"`

	// FingerprintAlgorithm is the algorithm used for fingerprint
	FingerprintAlgorithm string `json:"fingerprint_algorithm,omitempty"`

	// IsActive indicates if keypair is active (1 = active)
	IsActive int `json:"is_active,omitempty"`

	// AssociatedVMs is the number of VMs using this keypair
	AssociatedVMs int `json:"associated_vms,omitempty"`

	// Type indicates how keypair was created ("Created" or "Uploaded")
	Type string `json:"type,omitempty"`

	// CreatedDate is the creation timestamp
	CreatedDate string `json:"created_date,omitempty"`

	// CreatedBy is the user who created the keypair
	CreatedBy string `json:"created_by,omitempty"`
}

// KeypairListData represents the nested data object in list keypairs response.
type KeypairListData struct {
	TotalCount   int       `json:"total_count"`
	EngagementID int64     `json:"engagement_id"`
	Keypairs     []Keypair `json:"keypairs"`
}

// KeypairListResponse represents the response from listing keypairs.
type KeypairListResponse struct {
	// Status is the response status (e.g., "success")
	Status string `json:"status"`

	// Data contains the keypair list data object
	Data KeypairListData `json:"data"`

	// Message contains additional information
	Message string `json:"message"`

	// ResponseCode is the API response code (0 = success)
	ResponseCode int `json:"responseCode"`
}

// KeypairCreateRequest contains parameters for creating a keypair.
type KeypairCreateRequest struct {
	// KeypairName is the name for the new keypair
	KeypairName string

	// EngagementID is the engagement ID
	EngagementID string

	// KeypairType is the type of keypair (e.g., "RSA", "ED25519")
	KeypairType string

	// PrivateKeyFileFormat is the format for the private key file ("pem" or "ppk")
	PrivateKeyFileFormat string
}

// KeypairUploadRequest contains parameters for uploading a keypair.
type KeypairUploadRequest struct {
	// KeypairName is the name for the keypair
	KeypairName string

	// EncryptedPublicKey is the public key content to upload
	EncryptedPublicKey string

	// EngagementID is the engagement ID
	EngagementID string
}

// CreateKeypair creates a new keypair and returns the private key content.
//
// POST /portalservice/keypair/create-or-upload?keypairName=X&engagementId=Y&keypairType=Z&privateKeyFileFormat=W
// Returns the private key file as binary content.
func CreateKeypair(c *client.Client, ctx context.Context, req *KeypairCreateRequest) (*KeypairCreateResponse, error) {
	tflog.Debug(ctx, "Creating keypair", map[string]any{
		"keypair_name":            req.KeypairName,
		"engagement_id":           req.EngagementID,
		"keypair_type":            req.KeypairType,
		"private_key_file_format": req.PrivateKeyFileFormat,
	})

	token, err := c.GetToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get authentication token: %w", err)
	}

	// Build URL with query parameters
	params := url.Values{}
	params.Set("keypairName", req.KeypairName)
	params.Set("engagementId", req.EngagementID)
	params.Set("keypairType", req.KeypairType)
	params.Set("privateKeyFileFormat", req.PrivateKeyFileFormat)

	apiURL := client.APIURL + "portalservice/keypair/create-or-upload?" + params.Encode()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+token)
	httpReq.Header.Set("Accept", "application/octet-stream")

	tflog.Debug(ctx, "Executing create keypair request", map[string]any{
		"url": apiURL,
	})

	resp, err := c.GetHTTPClient().Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Handle non-2xx status codes
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var apiErr client.APIError
		if json.Unmarshal(respBody, &apiErr) == nil && (apiErr.Message != "" || apiErr.Code != "") {
			apiErr.StatusCode = resp.StatusCode
			return nil, &apiErr
		}
		return nil, fmt.Errorf("create keypair failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	// Extract filename from Content-Disposition header
	// Format: attachment; filename="keypairName.pem" or attachment; filename="keypairName.ppk"
	contentDisposition := resp.Header.Get("Content-Disposition")
	filename := req.KeypairName + "." + req.PrivateKeyFileFormat
	format := req.PrivateKeyFileFormat

	if contentDisposition != "" {
		// Parse filename from header
		re := regexp.MustCompile(`filename="?([^"]+)"?`)
		matches := re.FindStringSubmatch(contentDisposition)
		if len(matches) > 1 {
			filename = matches[1]
		}
	}

	result := &KeypairCreateResponse{
		PrivateKeyContent: string(respBody),
		Filename:          filename,
		Format:            format,
	}

	tflog.Info(ctx, "Keypair created successfully", map[string]any{
		"keypair_name": req.KeypairName,
		"filename":     filename,
		"format":       format,
	})

	return result, nil
}

// UploadKeypair uploads a public key to create a keypair.
//
// POST /portalservice/keypair/create-or-upload?keypairName=X&encryptedPublicKey=Y&engagementId=Z
// Returns JSON response with status.
func UploadKeypair(c *client.Client, ctx context.Context, req *KeypairUploadRequest) (*KeypairResponse, error) {
	tflog.Debug(ctx, "Uploading keypair", map[string]any{
		"keypair_name":   req.KeypairName,
		"engagement_id":  req.EngagementID,
		"has_public_key": req.EncryptedPublicKey != "",
	})

	token, err := c.GetToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get authentication token: %w", err)
	}

	// Build URL with query parameters
	params := url.Values{}
	params.Set("keypairName", req.KeypairName)
	params.Set("encryptedPublicKey", req.EncryptedPublicKey)
	params.Set("engagementId", req.EngagementID)

	apiURL := client.APIURL + "portalservice/keypair/create-or-upload?" + params.Encode()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+token)
	httpReq.Header.Set("Accept", "application/json")

	tflog.Debug(ctx, "Executing upload keypair request", map[string]any{
		"url": apiURL,
	})

	resp, err := c.GetHTTPClient().Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Handle non-2xx status codes
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var apiErr client.APIError
		if json.Unmarshal(respBody, &apiErr) == nil && (apiErr.Message != "" || apiErr.Code != "") {
			apiErr.StatusCode = resp.StatusCode
			return nil, &apiErr
		}
		return nil, fmt.Errorf("upload keypair failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var result KeypairResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse upload keypair response: %w", err)
	}

	tflog.Info(ctx, "Keypair uploaded successfully", map[string]any{
		"keypair_name": req.KeypairName,
		"status":       result.Status,
	})

	return &result, nil
}

// ListKeypairs retrieves all keypairs for an engagement.
//
// GET /portalservice/keypair/list/{engagementId}
// Returns JSON response with list of keypairs.
func ListKeypairs(c *client.Client, ctx context.Context, engagementID string) (*KeypairListResponse, error) {
	tflog.Debug(ctx, "Listing keypairs", map[string]any{
		"engagement_id": engagementID,
	})

	path := fmt.Sprintf("portalservice/keypair/list/%s", engagementID)
	respBody, err := c.DoRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list keypairs: %w", err)
	}

	tflog.Debug(ctx, "List keypairs response", map[string]any{
		"response": string(respBody),
	})

	var result KeypairListResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse list keypairs response: %w", err)
	}

	tflog.Info(ctx, "Keypairs listed successfully", map[string]any{
		"engagement_id": engagementID,
		"count":         len(result.Data.Keypairs),
	})

	return &result, nil
}

// GetKeypairByName retrieves a specific keypair by name from the list.
func GetKeypairByName(c *client.Client, ctx context.Context, engagementID, keypairName string) (*Keypair, error) {
	tflog.Debug(ctx, "Getting keypair by name", map[string]any{
		"engagement_id": engagementID,
		"keypair_name":  keypairName,
	})

	listResp, err := ListKeypairs(c, ctx, engagementID)
	if err != nil {
		return nil, err
	}

	for _, kp := range listResp.Data.Keypairs {
		if kp.Name == keypairName {
			tflog.Info(ctx, "Found keypair by name", map[string]any{
				"keypair_id":   kp.ID,
				"keypair_name": kp.Name,
			})
			return &kp, nil
		}
	}

	return nil, fmt.Errorf("keypair with name '%s' not found", keypairName)
}

// GetKeypairByID retrieves a specific keypair by ID from the list.
func GetKeypairByID(c *client.Client, ctx context.Context, engagementID string, keypairID int64) (*Keypair, error) {
	tflog.Debug(ctx, "Getting keypair by ID", map[string]any{
		"engagement_id": engagementID,
		"keypair_id":    keypairID,
	})

	listResp, err := ListKeypairs(c, ctx, engagementID)
	if err != nil {
		return nil, err
	}

	for _, kp := range listResp.Data.Keypairs {
		if kp.ID == keypairID {
			tflog.Info(ctx, "Found keypair by ID", map[string]any{
				"keypair_id":   kp.ID,
				"keypair_name": kp.Name,
			})
			return &kp, nil
		}
	}

	return nil, fmt.Errorf("keypair with ID '%d' not found", keypairID)
}

// DeleteKeypair deletes a keypair by ID.
//
// DELETE /portalservice/keypair/{keypairId}
// Returns JSON response with status.
func DeleteKeypair(c *client.Client, ctx context.Context, keypairID int64) (*KeypairResponse, error) {
	tflog.Debug(ctx, "Deleting keypair", map[string]any{
		"keypair_id": keypairID,
	})

	path := fmt.Sprintf("portalservice/keypair/%d", keypairID)
	respBody, err := c.DoRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to delete keypair: %w", err)
	}

	tflog.Debug(ctx, "Delete keypair response", map[string]any{
		"response": string(respBody),
	})

	var result KeypairResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse delete keypair response: %w", err)
	}

	tflog.Info(ctx, "Keypair deleted successfully", map[string]any{
		"keypair_id": keypairID,
		"status":     result.Status,
	})

	return &result, nil
}

// ParseKeypairID parses a keypair ID string to int64.
func ParseKeypairID(idStr string) (int64, error) {
	return strconv.ParseInt(idStr, 10, 64)
}

// FormatKeypairID formats a keypair ID as string.
func FormatKeypairID(id int64) string {
	return strconv.FormatInt(id, 10)
}
