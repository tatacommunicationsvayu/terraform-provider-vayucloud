// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package virtualmachine_image

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/client"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/common"
)

// VirtualMachineImage represents a single image entry from the API response.
type VirtualMachineImage struct {
	ID                  int64   `json:"id"`
	OsType              string  `json:"osType"`
	Name                string  `json:"name"`
	DisplayName			string  `json:"displayName"`
	OsMake              string  `json:"osMake"`
	OsModel             string  `json:"osModel"`
	OsVersion           string  `json:"osVersion"`
	OsServicePack       *string `json:"osServicePack"`
}

// VirtualMachineImageResponseData holds the nested data object containing the image list.
type VirtualMachineImageResponseData struct {
	Image []VirtualMachineImage `json:"image"`
}

// VirtualMachineImageResponse represents the API response for getting VM images.
type VirtualMachineImageResponse struct {
	Status       string                          `json:"status"`
	Data         VirtualMachineImageResponseData `json:"data"`
	Message      string                          `json:"message"`
	ResponseCode int                             `json:"responseCode"`
}

// GetVirtualMachineImages retrieves the list of VM images/templates for a given zone.
//
// GET {APIURL}/<InstanceServicePath>/{zoneId}/templates?type={type}&osMake={osMake} (common.InstanceServicePath)
// Returns the parsed image list and any error that occurred.
func GetVirtualMachineImages(c *client.Client, ctx context.Context, zoneID string, imageType string, osMake string) (*VirtualMachineImageResponse, error) {
	tflog.Debug(ctx, "Fetching virtual machine images", map[string]any{
		"zone_id":    zoneID,
		"image_type": imageType,
		"os_make":    osMake,
	})

	params := url.Values{}
	if imageType != "" {
		params.Set("type", imageType)
	}
	if osMake != "" {
		params.Set("osMake", osMake)
	}

	path := fmt.Sprintf("%s/%s/templates", common.InstanceServicePath, zoneID)
	if encoded := params.Encode(); encoded != "" {
		path += "?" + encoded
	}
	respBody, err := c.DoRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get virtual machine images: %w", err)
	}

	tflog.Debug(ctx, "Get virtual machine images response", map[string]any{
		"response": string(respBody),
	})

	var result VirtualMachineImageResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse virtual machine images response: %w", err)
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("get virtual machine images failed: %s (code: %d)", result.Message, result.ResponseCode)
	}

	tflog.Info(ctx, "Virtual machine images fetched successfully", map[string]any{
		"zone_id": zoneID,
		"count":   len(result.Data.Image),
		"status":  result.Status,
	})

	return &result, nil
}
