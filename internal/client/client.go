// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

// Package client provides the HTTP client for interacting with the VayuCloud API.
//
// This package handles:
// - Authentication via Spring Boot REST API (Bearer token)
// - Token storage and automatic refresh
// - HTTP request/response handling with proper error handling
// - All API operations for VayuCloud resources
package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Config holds the configuration for the VayuCloud API client.
type Config struct {
	// Username is sent as "email" in the portal getAuthToken request body.
	Username string
	Password string
	Timeout  int64
}

// Client is the VayuCloud API client.
// It handles authentication and all API operations.
type Client struct {
	// config holds the client configuration
	config *Config

	// httpClient is the underlying HTTP client
	httpClient *http.Client

	// token holds the current authentication token
	token *AuthToken

	// tokenMutex protects token access for concurrent operations
	tokenMutex sync.RWMutex
}

// AuthToken represents the authentication token returned by the Spring Boot API.
type AuthToken struct {
	// AccessToken is the Bearer token used for API requests
	AccessToken string `json:"accessToken"`

	// TokenType is typically "Bearer"
	TokenType string `json:"token_type"`

	// ExpiresIn is the token validity duration in seconds
	ExpiresIn int64 `json:"expiresIn"`

	ExpiresAt time.Time `json:"-"`

	// ExpiresAt is the calculated expiration time
	RefreshExpiresIn int64     `json:"refreshExpiresIn"`
	RefreshExpiresAt time.Time `json:"-"`

	// RefreshToken can be used to obtain a new access token (optional)
	RefreshToken string `json:"refreshToken,omitempty"`
}

// portalAuthRequest is the JSON body for getAuthToken.
type portalAuthRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// APIError represents an error response from the VayuCloud API.
type APIError struct {
	StatusCode int    `json:"-"`
	Code       string `json:"code,omitempty"`
	Message    string `json:"message,omitempty"`
	Details    string `json:"details,omitempty"`
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("VayuCloud API error (HTTP %d): %s - %s", e.StatusCode, e.Code, e.Message)
	}
	return fmt.Sprintf("VayuCloud API error (HTTP %d): %s", e.StatusCode, e.Code)
}

// NewClient creates a new VayuCloud API client and authenticates.
//
// This function:
// 1. Creates an HTTP client with the configured timeout and TLS settings
// 2. Calls the authentication endpoint to obtain a Bearer token
// 3. Returns a client ready for API operations
func NewClient(ctx context.Context, config *Config) (*Client, error) {
	tflog.Debug(ctx, "Creating new VayuCloud API client")

	timeout := config.Timeout
	if timeout <= 0 {
		timeout = DefaultTimeout
	}

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
		IdleConnTimeout:     90 * time.Second,
	}

	httpClient := &http.Client{
		Transport: transport,
		Timeout:   time.Duration(timeout) * time.Second,
	}

	client := &Client{
		config:     config,
		httpClient: httpClient,
	}

	// Authenticate and obtain token
	if err := client.Authenticate(ctx); err != nil {
		return nil, fmt.Errorf("failed to authenticate: %w", err)
	}

	tflog.Info(ctx, "VayuCloud API client created and authenticated successfully")
	return client, nil
}

// Authenticate performs authentication against the portal getAuthToken API.
//
// POST JSON: { "email", "password" } → { "access_token" } (HTTP 200).
// Token lifetime is not returned; ExpiresAt defaults to common.DefaultAuthTokenLifetime.
func (c *Client) Authenticate(ctx context.Context) error {
	tflog.Debug(ctx, "Authenticating with VayuCloud API", map[string]any{
		"auth_url": AuthURL,
		"email":    c.config.Username,
	})

	body, err := json.Marshal(portalAuthRequest{
		Email:    c.config.Username,
		Password: c.config.Password,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal auth request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, AuthURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create auth request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute auth request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read auth response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var apiErr APIError
		if json.Unmarshal(respBody, &apiErr) == nil && apiErr.Message != "" {
			apiErr.StatusCode = resp.StatusCode
			return &apiErr
		}
		return fmt.Errorf("authentication failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var token AuthToken
	if err := json.Unmarshal(respBody, &token); err != nil {
		return fmt.Errorf("failed to parse auth response: %w", err)
	}
	if token.AccessToken == "" {
		return fmt.Errorf("authentication response missing access_token")
	}
	if token.TokenType == "" {
		token.TokenType = "Bearer"
	}

	// API does not return expires_in; refresh before this deadline.
	if token.ExpiresIn == 0 {
		token.ExpiresIn = int64(DefaultAuthTokenLifetime / time.Second)
	}
	if token.RefreshExpiresIn == 0 {
		token.RefreshExpiresIn = int64(DefaultAuthTokenLifetime / time.Second)
	}
	token.ExpiresAt = time.Now().Add(time.Duration(token.ExpiresIn) * time.Second)
	token.RefreshExpiresAt = time.Now().Add(time.Duration(token.RefreshExpiresIn) * time.Second)

	c.tokenMutex.Lock()
	c.token = &token
	c.tokenMutex.Unlock()

	tflog.Debug(ctx, "Authentication successful", map[string]any{
		"token_type": token.TokenType,
		"expires_at": token.RefreshExpiresAt.Format(time.RFC3339),
	})

	return nil
}

// GetUsername returns the configured username for this client.
func (c *Client) GetUsername() string {
	return c.config.Username
}

// GetToken returns the current access token, refreshing if necessary.
func (c *Client) GetToken(ctx context.Context) (string, error) {
	c.tokenMutex.RLock()
	token := c.token
	c.tokenMutex.RUnlock()

	// Check if token needs refresh
	if token == nil || time.Now().After(token.RefreshExpiresAt) {
		tflog.Debug(ctx, "Token expired or missing, re-authenticating")
		if err := c.Authenticate(ctx); err != nil {
			return "", err
		}
		c.tokenMutex.RLock()
		token = c.token
		c.tokenMutex.RUnlock()
	}

	return token.AccessToken, nil
}

// GetHTTPClient returns the underlying HTTP client for direct use by domain packages.
func (c *Client) GetHTTPClient() *http.Client {
	return c.httpClient
}

// redactedRequestHeaders returns a copy of HTTP headers safe for debug logging.
func redactedRequestHeaders(h http.Header) map[string]string {
	out := make(map[string]string, len(h))
	for key, values := range h {
		if http.CanonicalHeaderKey(key) == "Authorization" {
			out[key] = "Bearer [REDACTED]"
			continue
		}
		out[key] = strings.Join(values, ", ")
	}
	return out
}

// DoICSRequest performs an authenticated request to ICS operations endpoints and returns
// the response body for any HTTP status. ICS callers parse Catalyst {"status","message"} envelopes.
// Other modules should continue using DoRequest.
func (c *Client) DoICSRequest(ctx context.Context, method, path string, body interface{}) ([]byte, error) {
	for attempt := 0; attempt < 2; attempt++ {
		token, err := c.GetToken(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get authentication token: %w", err)
		}

		var bodyReader io.Reader
		if body != nil {
			bodyBytes, err := json.Marshal(body)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal request body: %w", err)
			}
			bodyReader = bytes.NewReader(bodyBytes)
		}

		req, err := http.NewRequestWithContext(ctx, method, APIURL+path, bodyReader)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		req.Header.Set("X-Vayu-Client-Id", "vayu_iac")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to execute request: %w", err)
		}

		respBody, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return nil, fmt.Errorf("failed to read response body: %w", readErr)
		}
		closeErr := resp.Body.Close()
		if closeErr != nil {
			return nil, fmt.Errorf("failed to close response body: %w", closeErr)
		}
		

		if resp.StatusCode == http.StatusUnauthorized && attempt == 0 {
			tflog.Debug(ctx, "ICS request received 401, attempting to re-authenticate")
			if err := c.Authenticate(ctx); err != nil {
				return nil, fmt.Errorf("re-authentication failed: %w", err)
			}
			continue
		}

		return respBody, nil
	}

	return nil, fmt.Errorf("ICS request failed after re-authentication")
}

// ICSHTTPResponse is the raw HTTP result from an ICS operations call (status, headers, body).
// ICS modules that need response headers (e.g. AUDIT_ID on failures) use DoICSRequestEx.
type ICSHTTPResponse struct {
	StatusCode int
	Headers    http.Header
	Body       []byte
}

// DoICSRequestEx performs the same authenticated ICS request as DoICSRequest but also
// returns HTTP status and response headers. DoICSRequest is unchanged for existing callers.
func (c *Client) DoICSRequestEx(ctx context.Context, method, path string, body interface{}) (*ICSHTTPResponse, error) {
	for attempt := 0; attempt < 2; attempt++ {
		token, err := c.GetToken(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get authentication token: %w", err)
		}

		var bodyReader io.Reader
		if body != nil {
			bodyBytes, err := json.Marshal(body)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal request body: %w", err)
			}
			bodyReader = bytes.NewReader(bodyBytes)
		}

		req, err := http.NewRequestWithContext(ctx, method, APIURL+path, bodyReader)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		req.Header.Set("X-Vayu-Client-Id", "vayu_iac")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to execute request: %w", err)
		}

		respBody, readErr := io.ReadAll(resp.Body)
		closeErr := resp.Body.Close()
		if closeErr != nil {
			return nil, fmt.Errorf("failed to close response body: %w", closeErr)
		}
		if readErr != nil {
			return nil, fmt.Errorf("failed to read response body: %w", readErr)
		}

		if resp.StatusCode == http.StatusUnauthorized && attempt == 0 {
			tflog.Debug(ctx, "ICS request received 401, attempting to re-authenticate")
			if err := c.Authenticate(ctx); err != nil {
				return nil, fmt.Errorf("re-authentication failed: %w", err)
			}
			continue
		}

		return &ICSHTTPResponse{
			StatusCode: resp.StatusCode,
			Headers:    resp.Header,
			Body:       respBody,
		}, nil
	}

	return nil, fmt.Errorf("ICS request failed after re-authentication")
}

// DoRequest performs an HTTP request with authentication.
// It automatically adds the Bearer token and handles common error cases.
func (c *Client) DoRequest(ctx context.Context, method, path string, body interface{}) ([]byte, error) {
	token, err := c.GetToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get authentication token: %w", err)
	}

	var bodyReader io.Reader
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	url := APIURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Vayu-Client-Id", "vayu_iac")

	tflog.Debug(ctx, "Executing API request", map[string]any{
		"method":  method,
		"url":     url,
		"headers": fmt.Sprintf("%v", req.Header),
	})

	resp, err := c.httpClient.Do(req)
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
		// Check if it's an authentication error (401) - token might have expired
		if resp.StatusCode == http.StatusUnauthorized {
			tflog.Debug(ctx, "Received 401, attempting to re-authenticate")
			if err := c.Authenticate(ctx); err != nil {
				return nil, fmt.Errorf("re-authentication failed: %w", err)
			}
			// Retry the request once after re-authentication
			return c.DoRequestNoRetry(ctx, method, path, body)
		}

		var apiErr APIError
		if json.Unmarshal(respBody, &apiErr) == nil && (apiErr.Message != "" || apiErr.Code != "") {
			apiErr.StatusCode = resp.StatusCode
			return nil, &apiErr
		}
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// DoRequestWithHTTPStatus performs an authenticated request and returns the HTTP status and raw body.
// Unlike DoRequest, non-2xx responses are not returned as errors — callers parse portal envelopes (e.g. CatalystResponse).
func (c *Client) DoRequestWithHTTPStatus(ctx context.Context, method, path string, body interface{}) (int, []byte, error) {
	token, err := c.GetToken(ctx)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to get authentication token: %w", err)
	}

	var bodyReader io.Reader
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return 0, nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	url := APIURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Vayu-Client-Id", "vayu_iac")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, nil, fmt.Errorf("failed to read response body: %w", err)
	}
	return resp.StatusCode, respBody, nil
}

// DoRequestNoTimeout performs an HTTP request without timeout (waits indefinitely).
// Use this for long-running API calls that may take extended time to respond.
func (c *Client) DoRequestNoTimeout(ctx context.Context, method, path string, body interface{}) ([]byte, error) {
	token, err := c.GetToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get authentication token: %w", err)
	}

	var bodyReader io.Reader
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	url := APIURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Vayu-Client-Id", "vayu_iac")

	tflog.Debug(ctx, "Executing API request (no timeout)", map[string]any{
		"method": method,
		"url":    url,
	})

	// Create a new HTTP client with no timeout for this specific request
	noTimeoutClient := &http.Client{
		Transport: c.httpClient.Transport,
		Timeout:   0, // No timeout - wait indefinitely
	}

	resp, err := noTimeoutClient.Do(req)
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
		var apiErr APIError
		if json.Unmarshal(respBody, &apiErr) == nil && (apiErr.Message != "" || apiErr.Code != "") {
			apiErr.StatusCode = resp.StatusCode
			return nil, &apiErr
		}
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// DoRequestNoRetry performs an HTTP request without automatic retry on 401.
// Used internally to prevent infinite retry loops.
func (c *Client) DoRequestNoRetry(ctx context.Context, method, path string, body interface{}) ([]byte, error) {
	token, err := c.GetToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get authentication token: %w", err)
	}

	var bodyReader io.Reader
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	url := APIURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Vayu-Client-Id", "vayu_iac")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var apiErr APIError
		if json.Unmarshal(respBody, &apiErr) == nil && (apiErr.Message != "" || apiErr.Code != "") {
			apiErr.StatusCode = resp.StatusCode
			return nil, &apiErr
		}
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}
