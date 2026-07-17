// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package account_engagement

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/client"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/common"
)

// AccountEngagement represents a single engagement entry from the API response.
type AccountEngagement struct {
	EngagementName string `json:"engagementName"`
	ID             int64  `json:"id"`
	EngagementType string `json:"engagementType"`
	CustomerName   string `json:"customerName"`
}

// AccountEngagementResponse represents the API response for getting user engagements.
type AccountEngagementResponse struct {
	Status       string              `json:"status"`
	Data         []AccountEngagement `json:"data"`
	Message      string              `json:"message"`
	ResponseCode int                 `json:"responseCode"`
}

// GetAccountEngagements retrieves the list of user engagements from the API.
//
// GET {APIURL}/uat-portalservice/configservice/getuserengagements
// Returns the parsed engagement list and any error that occurred.
func GetAccountEngagements(c *client.Client, ctx context.Context) (*AccountEngagementResponse, error) {
	tflog.Debug(ctx, "Fetching account engagements")

	respBody, err := c.DoRequest(ctx, http.MethodGet, common.ConfigServicePath+"/getuserengagements", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get account engagements: %w", err)
	}

	tflog.Debug(ctx, "Get account engagements response", map[string]any{
		"response": string(respBody),
	})

	var result AccountEngagementResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse account engagements response: %w", err)
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("get account engagements failed: %s (code: %d)", result.Message, result.ResponseCode)
	}

	tflog.Info(ctx, "Account engagements fetched successfully", map[string]any{
		"count":  len(result.Data),
		"status": result.Status,
	})

	return &result, nil
}

// ValidateEngagementExists checks whether an engagement with the given ID
// exists in the user's account. Returns nil if found, or an error if not.
func ValidateEngagementExists(c *client.Client, ctx context.Context, engagementID int64) error {
	tflog.Debug(ctx, "Validating engagement exists", map[string]any{
		"engagement_id": engagementID,
	})

	engagementResp, err := GetAccountEngagements(c, ctx)
	if err != nil {
		return fmt.Errorf("failed to validate engagement ID: %w", err)
	}

	for _, engagement := range engagementResp.Data {
		if engagement.ID == engagementID {
			tflog.Debug(ctx, "Engagement ID validated successfully", map[string]any{
				"engagement_id":   engagementID,
				"engagement_name": engagement.EngagementName,
			})
			return nil
		}
	}

	return fmt.Errorf("Engagement ID %d is not available", engagementID)
}
