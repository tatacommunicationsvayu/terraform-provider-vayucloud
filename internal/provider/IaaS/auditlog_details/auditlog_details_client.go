// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package auditlog_details

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/client"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/common"
)

// AuditLogComment represents a single comment in the audit log details.
type AuditLogComment struct {
	UpdatedTime string      `json:"updated_time"`
	CommentType string      `json:"comment_type"`
	UpdatedBy   string      `json:"updated_by"`
	CommentID   json.Number `json:"commentId"`
	Comment     string      `json:"comment"`
}

// AuditLogDetailsResponse represents the API response for audit log details.
type AuditLogDetailsResponse struct {
	AuditID          string            `json:"auditID"`
	UpdatedTime      string            `json:"updatedTime"`
	ResourceID       json.Number       `json:"resourceId"`
	UpdatedBy        string            `json:"updatedBy"`
	Comments         []AuditLogComment `json:"comments"`
	ResponseStatus   string            `json:"responseStatus"`
	ResourceCategory string            `json:"resourceCategory"`
	Output           string            `json:"output"`
	Input            string            `json:"input"`
	Environment      string            `json:"environment"`
	CreatedBy        string            `json:"createdBy"`
	Action           string            `json:"action"`
	CreatedTime      string            `json:"createdTime"`
	EngagementID     json.Number       `json:"engagementId"`
	ResourceType     string            `json:"resourceType"`
	Status           string            `json:"status"`
}

// GetAuditLogDetails retrieves detailed information for a specific audit log entry.
//
// GET {APIURL}/auditlogservice/auditlog/info/{auditId}
func GetAuditLogDetails(c *client.Client, ctx context.Context, auditID string) (*AuditLogDetailsResponse, error) {
	tflog.Debug(ctx, "Fetching audit log details", map[string]any{
		"audit_id": auditID,
	})

	path := fmt.Sprintf("%s/info/%s", common.AuditLogServicePath, auditID)

	respBody, err := c.DoRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get audit log details: %w", err)
	}

	tflog.Debug(ctx, "Get audit log details response", map[string]any{
		"response": string(respBody),
	})

	var result AuditLogDetailsResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse audit log details response: %w", err)
	}

	tflog.Info(ctx, "Audit log details fetched successfully", map[string]any{
		"audit_id": result.AuditID,
		"status":   result.Status,
		"action":   result.Action,
	})

	return &result, nil
}
