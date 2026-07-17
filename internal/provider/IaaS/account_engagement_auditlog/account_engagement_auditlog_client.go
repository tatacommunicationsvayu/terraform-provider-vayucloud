// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package account_engagement_auditlog

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/client"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/common"
)

// AuditLogRequest is the request body for fetching engagement audit logs.
// All fields are pointers so that unset values serialize as JSON null.
type AuditLogRequest struct {
	StartDate        *string  `json:"startDate"`
	EndDate          *string  `json:"endDate"`
	Action           []string `json:"action"`
	CreatedBy        *string  `json:"created_by"`
	Status           *string  `json:"status"`
	AuditID          *string  `json:"audit_id"`
	ResourceCategory *string  `json:"resource_category"`
	ResourceID       *string  `json:"resource_id"`
}

// AuditLogEntry represents a single audit log record from the API response.
type AuditLogEntry struct {
	AuditID          string      `json:"auditId"`
	EngagementID     int64       `json:"engagementId"`
	ResourceType     string      `json:"resourceType"`
	ResourceID       json.Number `json:"resourceId"`
	ResourceCategory string      `json:"resourceCategory"`
	Action           string      `json:"action"`
	Description      string      `json:"description"`
	CreatedBy        string      `json:"createdBy"`
	CreatedTime      string      `json:"createdTime"`
	UpdatedBy        string      `json:"updatedBy"`
	UpdatedTime      string      `json:"updatedTime"`
	Status           string      `json:"status"`
	SnowTicketID     *string     `json:"snowTicketId"`
	ClusterName      *string     `json:"clusterName"`
	ClusterStatus    *string     `json:"clusterStatus"`
}

// AuditLogPageable represents the pagination metadata.
type AuditLogPageable struct {
	PageNumber int `json:"pageNumber"`
	PageSize   int `json:"pageSize"`
	Offset     int `json:"offset"`
}

// AuditLogData represents the paginated data wrapper.
type AuditLogData struct {
	Content       []AuditLogEntry  `json:"content"`
	Pageable      AuditLogPageable `json:"pageable"`
	TotalPages    int              `json:"totalPages"`
	TotalElements int              `json:"totalElements"`
	Last          bool             `json:"last"`
	Size          int              `json:"size"`
	Number        int              `json:"number"`
	First         bool             `json:"first"`
	Empty         bool             `json:"empty"`
}

// EngagementAuditLogResponse represents the top-level API response for audit logs.
type EngagementAuditLogResponse struct {
	Data   AuditLogData `json:"data"`
	Status string       `json:"status"`
}

// GetAccountEngagementAuditLogs retrieves audit logs for a given engagement.
//
// POST {APIURL}/uat-auditlogservice/auditlog/audits/{engagementId}
func GetAccountEngagementAuditLogs(c *client.Client, ctx context.Context, engagementID int64, request *AuditLogRequest) (*EngagementAuditLogResponse, error) {
	tflog.Debug(ctx, "Fetching engagement audit logs", map[string]any{
		"engagement_id": engagementID,
	})

	path := fmt.Sprintf("%s/audits/%d", common.AuditLogServicePath, engagementID)
	tflog.Debug(ctx, "Audit log path", map[string]any{
		"engagement_id": engagementID,
		"request":       request,
		"path":          path,
	})

	respBody, err := c.DoRequest(ctx, http.MethodPost, path, request)
	if err != nil {
		return nil, fmt.Errorf("failed to get engagement audit logs: %w", err)
	}

	tflog.Debug(ctx, "Get engagement audit logs response", map[string]any{
		"response": string(respBody),
	})

	var result EngagementAuditLogResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse engagement audit logs response: %w", err)
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("get engagement audit logs failed with status: %s", result.Status)
	}

	tflog.Info(ctx, "Engagement audit logs fetched successfully", map[string]any{
		"count":          len(result.Data.Content),
		"total_elements": result.Data.TotalElements,
		"total_pages":    result.Data.TotalPages,
		"status":         result.Status,
	})

	return &result, nil
}
