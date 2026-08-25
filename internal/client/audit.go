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
	AuditID             string          `json:"auditID"`
	UpdatedTime         string          `json:"updatedTime"`
	ResourceID          json.Number     `json:"resourceId"`
	UpdatedBy           string          `json:"updatedBy"`
	Comments            []AuditComment  `json:"comments"`
	ResponseStatus      string          `json:"responseStatus"`
	ResourceCategory    string          `json:"resourceCategory"`
	Output              string          `json:"output"`
	Input               string          `json:"input"`
	Environment         string          `json:"environment"`
	CreatedBy           string          `json:"createdBy"`
	Action              string          `json:"action"`
	CreatedTime         string          `json:"createdTime"`
	EngagementID        int64           `json:"engagementId"`
	ResourceType        string          `json:"resourceType"`
	Status              string          `json:"status"`
	ProvisioningDetails json.RawMessage `json:"provisioningDetails"`
}

// ProvisioningResourceID extracts resourceId from provisioningDetails JSON.
// provisioningDetails is expected as {"resourceId": "<id>"} (string or number).
func (a *AuditLogResponse) ProvisioningResourceID() (string, error) {
	if a == nil || len(a.ProvisioningDetails) == 0 || string(a.ProvisioningDetails) == "null" {
		return "", fmt.Errorf("audit response missing provisioningDetails")
	}

	raw := a.ProvisioningDetails
	// Some APIs return provisioningDetails as a JSON-encoded string.
	var asString string
	if err := json.Unmarshal(raw, &asString); err == nil && asString != "" {
		raw = json.RawMessage(asString)
	}

	var details struct {
		ResourceID json.RawMessage `json:"resourceId"`
	}
	if err := json.Unmarshal(raw, &details); err != nil {
		return "", fmt.Errorf("failed to parse provisioningDetails: %w", err)
	}
	if len(details.ResourceID) == 0 || string(details.ResourceID) == "null" {
		return "", fmt.Errorf("provisioningDetails missing resourceId")
	}

	var idString string
	if err := json.Unmarshal(details.ResourceID, &idString); err == nil && idString != "" {
		return idString, nil
	}
	var idNumber json.Number
	if err := json.Unmarshal(details.ResourceID, &idNumber); err == nil && idNumber.String() != "" {
		return idNumber.String(), nil
	}

	return "", fmt.Errorf("provisioningDetails resourceId has unsupported type: %s", string(details.ResourceID))
}

// GetAuditLog retrieves the audit log for a given audit ID.
//
// GET {AuditLogServicePath}/info/{auditId}
// Returns the current status of the async operation.
func (c *Client) GetAuditLog(ctx context.Context, auditID string) (*AuditLogResponse, error) {
	tflog.Debug(ctx, "Getting audit log", map[string]any{
		"audit_id": auditID,
	})

	path := fmt.Sprintf("%s/info/%s", common.AuditLogServicePath, auditID)
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

// WaitForAuditCompletionNoActionState polls until the audit completes without calling action-state.
// Use for portal flows (e.g. security groups) that are not wired to config action-state.
func (c *Client) WaitForAuditCompletionNoActionState(ctx context.Context, auditID string) (*AuditLogResponse, error) {
	return c.waitForAuditCompletion(ctx, auditID, "", "", nil, false)
}

// WaitForAuditCompletion polls the audit log until the operation completes or fails.
// On success it also calls the config action-state API. It polls every 20 seconds with a maximum of 90 attempts (30 minutes total).
func (c *Client) WaitForAuditCompletion(ctx context.Context, auditID string, action string, module string, requestBody any) (*AuditLogResponse, error) {
	return c.waitForAuditCompletion(ctx, auditID, action, module, requestBody, true)
}

func (c *Client) waitForAuditCompletion(ctx context.Context, auditID string, action string, module string, requestBody any, updateActionState bool) (*AuditLogResponse, error) {
	tflog.Info(ctx, "Waiting for audit completion", map[string]any{
		"audit_id":            auditID,
		"poll_interval":       AuditPollInterval.String(),
		"max_attempts":        AuditPollMaxAttempts,
		"action":              action,
		"module":              module,
		"request_body":        requestBody,
		"update_action_state": updateActionState,
	})

	for attempt := 1; attempt <= AuditPollMaxAttempts; attempt++ {
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

			if !updateActionState {
				return auditLog, nil
			}

			actionStateBody := make(map[string]any)
			if requestBody != nil {
				if reqMap, ok := requestBody.(map[string]any); ok && len(reqMap) > 0 {
					for k, v := range reqMap {
						actionStateBody[k] = v
					}
				}
			}
			if _, ok := actionStateBody["resourceId"]; ok {
				actionStateBody["ruleId"] = auditLog.ResourceID
			} else {
				actionStateBody["resourceId"] = auditLog.ResourceID
			}

			actionStateResp, stateErr := common.UpdateActionState(ctx, c, module, action, actionStateBody)
			if stateErr != nil {
				tflog.Error(ctx, "Failed to update action state", map[string]any{
					"audit_id":    auditID,
					"resource_id": auditLog.ResourceID,
					"error":       stateErr.Error(),
				})
				auditLog.Status = AuditStatusFailed
				return auditLog, fmt.Errorf("audit completed but action-state update failed (module=%s action=%s): %w", module, action, stateErr)
			}

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
			var errorMsg string
			if len(auditLog.Comments) > 0 {
				errorMsg = auditLog.Comments[1].Comment
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
