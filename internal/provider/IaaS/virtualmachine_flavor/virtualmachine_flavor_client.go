// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package virtualmachine_flavor

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/client"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/common"
)

type PricingModel struct {
	PPU      []string `json:"ppu"`
	Reserved []string `json:"reserved"`
}

type LinuxPartitionDetails struct {
	Usr   int64 `json:"usr"`
	Swap  int64 `json:"swap"`
	Root  int64 `json:"root"`
	Boot  int64 `json:"boot"`
	Kdump int64 `json:"kdump"`
}

type WindowsPartitionDetails struct {
	CDrive int64 `json:"cDrive"`
	Page   int64 `json:"page"`
}

type VirtualMachineFlavor struct {
	ID                          int64                   `json:"id"`
	Name                        string                  `json:"name"`
	ArtifactType                string                  `json:"artifactType"`
	OsModel                     string                  `json:"osModel"`
	P2RPricingModel             PricingModel            `json:"p2rPricingModel"`
	VCpu                        int64                   `json:"vCpu"`
	VRam                        int64                   `json:"vRam"`
	VGPU                        int64                   `json:"vGPU"`
	VDiskL                      int64                   `json:"vDisk.L"`
	VDiskW                      int64                   `json:"vDisk.W"`
	RootStoragePartitionLinux   LinuxPartitionDetails   `json:"rootStoragePartionDetails.L"`
	RootStoragePartitionWindows WindowsPartitionDetails `json:"rootStoragePartionDetails.W"`
}

type VirtualMachineFlavorResponseData struct {
	Flavors []VirtualMachineFlavor `json:"flavors"`
}

type VirtualMachineFlavorResponse struct {
	Status       string                           `json:"status"`
	Data         VirtualMachineFlavorResponseData `json:"data"`
	Message      string                           `json:"message"`
	ResponseCode int                              `json:"responseCode"`
}

// GetVirtualMachineFlavors retrieves the list of VM flavors for a given zone.
//
// GET {APIURL}/<InstanceServicePath>/{zoneId}/flavors (common.InstanceServicePath)
func GetVirtualMachineFlavors(c *client.Client, ctx context.Context, zoneID string) (*VirtualMachineFlavorResponse, error) {
	tflog.Debug(ctx, "Fetching virtual machine flavors", map[string]any{
		"zone_id": zoneID,
	})

	path := fmt.Sprintf("%s/%s/flavors", common.InstanceServicePath, zoneID)
	respBody, err := c.DoRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get virtual machine flavors: %w", err)
	}

	tflog.Debug(ctx, "Get virtual machine flavors response", map[string]any{
		"response": string(respBody),
	})

	var result VirtualMachineFlavorResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse virtual machine flavors response: %w", err)
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("get virtual machine flavors failed: %s (code: %d)", result.Message, result.ResponseCode)
	}

	tflog.Info(ctx, "Virtual machine flavors fetched successfully", map[string]any{
		"zone_id": zoneID,
		"count":   len(result.Data.Flavors),
		"status":  result.Status,
	})

	return &result, nil
}
