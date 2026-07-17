// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/common"
)

// AuditResponse represents the response from async operations that return an audit ID.
type AuditResponse struct {
	Status string `json:"status"`
	Data   struct {
		Audit struct {
			AuditID string `json:"auditId"`
		} `json:"audit"`
	} `json:"data"`
	Message      string `json:"message"`
	ResponseCode int    `json:"responseCode"`
}

// AuditComment represents a comment in the audit log.
type AuditComment struct {
	UpdatedTime string `json:"updated_time"`
	CommentType string `json:"comment_type"`
	UpdatedBy   string `json:"updated_by"`
	CommentID   int64  `json:"commentId"`
	Comment     string `json:"comment"`
}

// AuditLogResponse represents the response from the audit log API.
type AuditLogResponse struct {
	AuditID          string         `json:"auditID"`
	UpdatedTime      string         `json:"updatedTime"`
	ResourceID       json.Number    `json:"resourceId"`
	UpdatedBy        string         `json:"updatedBy"`
	Comments         []AuditComment `json:"comments"`
	ResponseStatus   string         `json:"responseStatus"`
	ResourceCategory string         `json:"resourceCategory"`
	Output           string         `json:"output"`
	Input            string         `json:"input"`
	Environment      string         `json:"environment"`
	CreatedBy        string         `json:"createdBy"`
	Action           string         `json:"action"`
	CreatedTime      string         `json:"createdTime"`
	EngagementID     int64          `json:"engagementId"`
	ResourceType     string         `json:"resourceType"`
	Status           string         `json:"status"`
}

// GetAuditLog retrieves the audit log for a given audit ID.
//
// GET /auditlogservice/auditlog/info/{auditId}
// Returns the current status of the async operation.
func (c *Client) GetAuditLog(ctx context.Context, auditID string) (*AuditLogResponse, error) {
	tflog.Debug(ctx, "Getting audit log", map[string]any{
		"audit_id": auditID,
	})

	path := fmt.Sprintf("auditlogservice/auditlog/info/%s", auditID)
	respBody, err := c.DoRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get audit log: %w", err)
	}

	tflog.Debug(ctx, "Audit log response", map[string]any{
		"response": string(respBody),
	})

	var result AuditLogResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse audit log response: %w", err)
	}

	tflog.Debug(ctx, "Audit log status", map[string]any{
		"audit_id": auditID,
		"status":   result.Status,
		"action":   result.Action,
	})

	return &result, nil
}

// WaitForAuditCompletion polls the audit log until the operation completes or fails.
// It polls every 20 seconds with a maximum of 90 attempts (30 minutes total).
func (c *Client) WaitForAuditCompletion(ctx context.Context, auditID string, action string, module string, requestBody any) (*AuditLogResponse, error) {
	tflog.Info(ctx, "Waiting for audit completion", map[string]any{
		"audit_id":      auditID,
		"poll_interval": AuditPollInterval.String(),
		"max_attempts":  AuditPollMaxAttempts,
		"action":        action,
		"module":        module,
		"request_body":  requestBody,
	})

	for attempt := 1; attempt <= AuditPollMaxAttempts; attempt++ {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		tflog.Debug(ctx, "Polling audit status", map[string]any{
			"audit_id": auditID,
			"attempt":  attempt,
			"max":      AuditPollMaxAttempts,
		})

		auditLog, err := c.GetAuditLog(ctx, auditID)
		if err != nil {
			return nil, fmt.Errorf("failed to get audit log (attempt %d/%d): %w", attempt, AuditPollMaxAttempts, err)
		}

		switch strings.ToUpper(auditLog.Status) {
		case AuditStatusCompleted:
			tflog.Info(ctx, "Audit completed successfully", map[string]any{
				"audit_id": auditID,
				"status":   auditLog.Status,
				"attempts": attempt,
			})

			// // Merge resourceId into the requestBody for action-state API
			actionStateBody := make(map[string]any)
			if requestBody != nil {
				if reqMap, ok := requestBody.(map[string]any); ok && len(reqMap) > 0 {
					for k, v := range reqMap {
						actionStateBody[k] = v
					}
				}
			}
			actionStateBody["resourceId"] = auditLog.ResourceID

			// Call action-state API to update the resource state
			actionStateResp, stateErr := common.UpdateActionState(ctx, c, module, action, actionStateBody)
			if stateErr != nil {
				tflog.Error(ctx, "Failed to update action state", map[string]any{
					"audit_id":    auditID,
					"resource_id": auditLog.ResourceID,
					"error":       stateErr.Error(),
				})
				// Mark audit log status as failed and return error
				auditLog.Status = AuditStatusFailed
				return auditLog, fmt.Errorf("audit completed but action state update failed: %w", stateErr)
			}

			// Log the response body if available
			if actionStateResp != nil {
				tflog.Debug(ctx, "Action state response received", map[string]any{
					"audit_id":      auditID,
					"status":        actionStateResp.Status,
					"response_code": actionStateResp.ResponseCode,
					"data":          actionStateResp.Data,
				})
			}

			return auditLog, nil

		case AuditStatusFailed:
			// Try to get error details from comments
			var errorMsg string
			if len(auditLog.Comments) > 0 {
				errorMsg = auditLog.Comments[1].Comment // The first comment is the error message
			}
			return auditLog, fmt.Errorf("audit operation failed: %s", errorMsg)

		case AuditStatusInProgress:
			tflog.Debug(ctx, "Audit still in progress, waiting...", map[string]any{
				"audit_id":     auditID,
				"attempt":      attempt,
				"next_poll_in": AuditPollInterval.String(),
			})

		default:
			tflog.Warn(ctx, "Unknown audit status", map[string]any{
				"audit_id": auditID,
				"status":   auditLog.Status,
			})
		}

		// Wait before next poll (unless this is the last attempt)
		if attempt < AuditPollMaxAttempts {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(AuditPollInterval):
			}
		}
	}

	return nil, fmt.Errorf("audit operation timed out after %d attempts (%v)", AuditPollMaxAttempts, time.Duration(AuditPollMaxAttempts)*AuditPollInterval)
}
