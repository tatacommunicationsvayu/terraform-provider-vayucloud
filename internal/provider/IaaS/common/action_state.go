// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package common

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// ActionStatePollInterval is the interval between action-state status checks.
const ActionStatePollInterval = 10 * time.Second

// ActionStatePollMaxAttempts is the maximum number of action-state polling attempts.
const ActionStatePollMaxAttempts = 30

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
// When the API returns status "error", polls every ActionStatePollInterval until status is "success"
// or ActionStatePollMaxAttempts is reached.
func UpdateActionState(ctx context.Context, c ActionStateHTTPClient, module string, action string, requestBody any) (*ActionStateResponse, error) {
	readImportRefresh := action == "read" || action == "import" || action == "refresh"

	tflog.Info(ctx, "Calling action-state API", map[string]any{
		"module":        module,
		"action":        action,
		"request_body":  requestBody,
		"poll_interval": ActionStatePollInterval.String(),
		"max_attempts":  ActionStatePollMaxAttempts,
	})

	var lastResult *ActionStateResponse

	for attempt := 1; attempt <= ActionStatePollMaxAttempts; attempt++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		tflog.Debug(ctx, "Action-state API request", map[string]any{
			"module":       module,
			"action":       action,
			"request_body": requestBody,
			"attempt":      attempt,
			"max":          ActionStatePollMaxAttempts,
		})

		result, err := postActionState(ctx, c, module, action, requestBody)
		if err != nil {
			return nil, err
		}
		lastResult = result

		if strings.EqualFold(result.Status, "success") {
			if readImportRefresh {
				tflog.Debug(ctx, "Action state response (read/import/refresh)", map[string]any{
					"response_status": result.Status,
					"attempts":        attempt,
				})
			} else {
				tflog.Info(ctx, "Action state updated successfully", map[string]any{
					"module":        module,
					"action":        action,
					"request_body":  requestBody,
					"response":      string(result.Data),
					"response_code": result.ResponseCode,
					"attempts":      attempt,
				})
			}
			return result, nil
		}

		if strings.EqualFold(result.Status, "error") {
			tflog.Debug(ctx, "Action-state returned error, waiting before retry", map[string]any{
				"module":       module,
				"action":       action,
				"attempt":      attempt,
				"max":          ActionStatePollMaxAttempts,
				"message":      result.Message,
				"next_poll_in": ActionStatePollInterval.String(),
			})
		} else if readImportRefresh {
			tflog.Debug(ctx, "Action state response (read/import/refresh)", map[string]any{
				"response_status": result.Status,
				"attempts":        attempt,
			})
			return result, nil
		} else {
			return nil, fmt.Errorf("action-state API returned non-success status: %s, message: %s", result.Status, result.Message)
		}

		if attempt < ActionStatePollMaxAttempts {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(ActionStatePollInterval):
			}
		}
	}

	if lastResult != nil && strings.EqualFold(lastResult.Status, "error") {
		return nil, fmt.Errorf(
			"action-state API still returned error after %d attempts (%v): %s",
			ActionStatePollMaxAttempts,
			time.Duration(ActionStatePollMaxAttempts)*ActionStatePollInterval,
			lastResult.Message,
		)
	}

	return nil, fmt.Errorf(
		"action-state API timed out after %d attempts (%v)",
		ActionStatePollMaxAttempts,
		time.Duration(ActionStatePollMaxAttempts)*ActionStatePollInterval,
	)
}

func postActionState(ctx context.Context, c ActionStateHTTPClient, module string, action string, requestBody any) (*ActionStateResponse, error) {
	path := fmt.Sprintf(ConfigServicePath+"/action-state?module=%s&action=%s", module, action)

	respBody, err := c.DoRequestNoTimeout(ctx, http.MethodPost, path, requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to call action-state API: %w", err)
	}

	tflog.Debug(ctx, "Action state response body", map[string]any{
		"module":   module,
		"action":   action,
		"response": string(respBody),
	})

	var result ActionStateResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse action-state response: %w", err)
	}

	return &result, nil
}
