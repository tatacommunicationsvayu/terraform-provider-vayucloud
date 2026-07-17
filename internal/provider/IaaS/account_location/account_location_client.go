// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package account_location

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/client"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/common"
)

// AccountLocation represents a single endpoint/location entry from the API response.
type AccountLocation struct {
	EndpointID          int64  `json:"endpointId"`
	EndpointDisplayName string `json:"endpointDisplayName"`
}

// AccountLocationResponse represents the API response for getting endpoints by engagement.
type AccountLocationResponse struct {
	Status       string            `json:"status"`
	Data         []AccountLocation `json:"data"`
	Message      *string           `json:"message"`
	ResponseCode int               `json:"responseCode"`
}

// GetAccountLocations retrieves the list of endpoints/locations for a given engagement.
//
// GET {APIURL}/portalservice/configservice/getEndpointsByEngagement/{engagementId}
// Returns the parsed location list and any error that occurred.
func GetAccountLocations(c *client.Client, ctx context.Context, engagementID int64) (*AccountLocationResponse, error) {
	tflog.Debug(ctx, "Fetching account locations", map[string]any{
		"engagement_id": engagementID,
	})

	path := fmt.Sprintf("%s/getEndpointsByEngagement/%d", common.ConfigServicePath, engagementID)
	respBody, err := c.DoRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get account locations: %w", err)
	}

	tflog.Debug(ctx, "Get account locations response", map[string]any{
		"response": string(respBody),
	})

	var result AccountLocationResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse account locations response: %w", err)
	}

	if result.Status != "success" {
		msg := ""
		if result.Message != nil {
			msg = *result.Message
		}
		return nil, fmt.Errorf("get account locations failed: %s (code: %d)", msg, result.ResponseCode)
	}

	tflog.Info(ctx, "Account locations fetched successfully", map[string]any{
		"engagement_id": engagementID,
		"count":         len(result.Data),
		"status":        result.Status,
	})

	return &result, nil
}

// ValidateEndpointExists checks whether an endpoint with the given ID
// exists under the specified engagement. Returns nil if found, or an error if not.
func ValidateEndpointExists(c *client.Client, ctx context.Context, engagementID int64, endpointID int64) error {
	tflog.Debug(ctx, "Validating endpoint exists", map[string]any{
		"engagement_id": engagementID,
		"endpoint_id":   endpointID,
	})

	locationResp, err := GetAccountLocations(c, ctx, engagementID)
	if err != nil {
		return fmt.Errorf("failed to validate endpoint ID: %w", err)
	}

	for _, location := range locationResp.Data {
		if location.EndpointID == endpointID {
			tflog.Debug(ctx, "Endpoint ID validated successfully", map[string]any{
				"endpoint_id":           endpointID,
				"endpoint_display_name": location.EndpointDisplayName,
			})
			return nil
		}
	}

	return fmt.Errorf("Endpoint ID %d is not available for engagement %d", endpointID, engagementID)
}
