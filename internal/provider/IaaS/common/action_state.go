// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package common

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// ActionStateResponse represents the response from the action-state API.
type ActionStateResponse struct {
	Status       string          `json:"status"`
	Data         json.RawMessage `json:"data"` // Use RawMessage to handle both string and object responses
	Message      string          `json:"message"`
	ResponseCode int             `json:"responseCode"`
}

// ActionStateHTTPClient is the subset of behavior needed to call the action-state API.
type ActionStateHTTPClient interface {
	DoRequestNoTimeout(ctx context.Context, method, path string, body any) ([]byte, error)
}

// UpdateActionState calls the action-state API to update the resource state after audit completion.
// POST {APIControlPath}/action-state?module={module}&action={action}
// Request body: {"resourceId": resourceId}
// Returns the parsed response body and any error that occurred.
func UpdateActionState(ctx context.Context, c ActionStateHTTPClient, module string, action string, requestBody any) (*ActionStateResponse, error) {
	tflog.Debug(ctx, "Updating action state", map[string]any{
		"module":       module,
		"action":       action,
		"request_body": requestBody,
	})

	path := fmt.Sprintf(ConfigServicePath+"/action-state?module=%s&action=%s", module, action)

	respBody, err := c.DoRequestNoTimeout(ctx, http.MethodPost, path, requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to call action-state API: %w", err)
	}

	if action == "read" || action == "import" || action == "refresh" {
		tflog.Debug(ctx, "Action state response (read/import/refresh)", map[string]any{
			"response": string(respBody),
		})

		var result ActionStateResponse
		if err := json.Unmarshal(respBody, &result); err != nil {
			return nil, fmt.Errorf("failed to parse action-state response: %w", err)
		}
		return &result, nil
	}

	tflog.Debug(ctx, "Action state response", map[string]any{
		"response": string(respBody),
	})

	var result ActionStateResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse action-state response: %w", err)
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("action-state API returned non-success status: %s, message: %s", result.Status, result.Message)
	}

	tflog.Info(ctx, "Action state updated successfully", map[string]any{
		"module":        module,
		"action":        action,
		"request_body":  requestBody,
		"response":      string(result.Data),
		"response_code": result.ResponseCode,
	})

	return &result, nil
}
