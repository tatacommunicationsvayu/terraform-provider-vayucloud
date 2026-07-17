// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package virtualmachine

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

// DiskPartition represents a disk partition configuration.
type DiskPartition struct {
	Partition string `json:"partition"`
	Size      int64  `json:"size"`
}

// AdditionalDisk represents an additional disk to attach during VM creation.
type AdditionalDisk struct {
	Size int64 `json:"size"`
	IOPS int64 `json:"iops"`
}

// VirtualMachineCreateRequest contains the request body for creating a virtual machine.
//
// POST {APIURL}/<InstanceServicePath>/createinstance (common.InstanceServicePath)
type VirtualMachineCreateRequest struct {
	Name                        string           `json:"name"`
	VMPurpose                   string           `json:"vmPurpose"`
	ImageID                     int64            `json:"imageId"`
	FlavorID                    int64            `json:"flavorId"`
	ZoneID                      int64            `json:"zoneId"`
	IOPS                        int64            `json:"iops"`
	IsKdumpOrPageEnabled        string           `json:"isKdumpOrPageEnabled"`
	UsageType                   string           `json:"usageType,omitempty"`
	PricingModel                string           `json:"pricingModel,omitempty"`
	RootDiskSize                int64            `json:"rootDiskSize,omitempty"`
	AssignPublicIp              string           `json:"assignpublicIp,omitempty"`
	RetainPublicIPOnTermination string           `json:"retainPublicIPOnTermination,omitempty"`
	PublicIpPricingModel        string           `json:"publicIpPricingModel,omitempty"`
	DiskPartitions              []DiskPartition  `json:"diskPartitions,omitempty"`
	AdditionalDisk              []AdditionalDisk `json:"additionalDisk,omitempty"`
}

// PreCreateValidateResponse represents the API response for pre-create validation.
type PreCreateValidateResponse struct {
	Status       string          `json:"status"`
	Data         json.RawMessage `json:"data"`
	Message      string          `json:"message"`
	ResponseCode int             `json:"responseCode"`
}

// PreCreateValidateInstance validates a virtual machine creation request before actually creating it.
//
// POST {APIURL}/<InstanceServicePath>/precreateinstancevalidate (common.InstanceServicePath)
func PreCreateValidateInstance(c *client.Client, ctx context.Context, req *VirtualMachineCreateRequest) error {
	tflog.Debug(ctx, "Running pre-create validation for virtual machine", map[string]any{
		"name": req.Name, "zone_id": req.ZoneID,
		"image_id": req.ImageID, "flavor_id": req.FlavorID,
	})

	respBody, err := c.DoRequest(ctx, http.MethodPost, fmt.Sprintf("%s/precreateinstancevalidate", common.InstanceServicePath), req)
	if err != nil {
		return fmt.Errorf("pre-create validation request failed: %w", err)
	}

	tflog.Debug(ctx, "Pre-create validation response", map[string]any{
		"response": string(respBody),
	})

	var result PreCreateValidateResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return fmt.Errorf("failed to parse pre-create validation response: %w", err)
	}

	if result.Status != "success" {
		return fmt.Errorf("pre-create validation failed (code: %d): %s", result.ResponseCode, result.Message)
	}

	// Extract success message from data
	var successData struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(result.Data, &successData); err == nil && successData.Message != "" {
		tflog.Info(ctx, "Pre-create validation passed", map[string]any{
			"message":       successData.Message,
			"response_code": result.ResponseCode,
		})
	}

	return nil
}

// ResizeVolumeValidateRequest is the body for volume resize validation.
//
// POST {APIURL}/<InstanceServicePath>/{instanceId}/volumes/{diskId}/resize-volume/validate (common.InstanceServicePath)
type ResizeVolumeValidateRequest struct {
	Size int64 `json:"size"`
}

// ValidateVolume checks whether a volume can be resized to the requested size before performing the resize.
//
// POST {APIURL}/<InstanceServicePath>/{instanceId}/volumes/{diskId}/resize-volume/validate (common.InstanceServicePath)
func ValidateVolume(c *client.Client, ctx context.Context, instanceID string, diskID int64, size int64) error {
	tflog.Debug(ctx, "Running resize validation for volume", map[string]any{
		"instance_id": instanceID,
		"disk_id":     diskID,
		"size":        size,
	})

	path := fmt.Sprintf(
		"%s/%s/volumes/%d/resize-volume/validate",
		common.InstanceServicePath, instanceID, diskID,
	)
	body := ResizeVolumeValidateRequest{Size: size}

	respBody, err := c.DoRequest(ctx, http.MethodPost, path, body)
	if err != nil {
		return fmt.Errorf("volume resize validation request failed: %w", err)
	}

	tflog.Debug(ctx, "Volume resize validation response", map[string]any{
		"response": string(respBody),
	})

	var result PreCreateValidateResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return fmt.Errorf("failed to parse volume resize validation response: %w", err)
	}

	if result.Status != "success" {
		return fmt.Errorf("volume resize validation failed (code: %d): %s", result.ResponseCode, result.Message)
	}

	var successData struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(result.Data, &successData); err == nil && successData.Message != "" {
		tflog.Info(ctx, "Volume resize validation passed", map[string]any{
			"message":       successData.Message,
			"response_code": result.ResponseCode,
			"instance_id":   instanceID,
			"disk_id":       diskID,
		})
	}

	return nil
}

// ResizeVolume initiates a volume resize and returns the audit ID.
//
// POST {APIURL}/<InstanceServicePath>/{instanceId}/volumes/{diskId}/resize-volume (common.InstanceServicePath)
func ResizeVolume(c *client.Client, ctx context.Context, instanceID string, diskID int64, size int64) (*client.AuditResponse, error) {
	tflog.Debug(ctx, "Resizing volume", map[string]any{
		"instance_id": instanceID,
		"disk_id":     diskID,
		"size":        size,
	})

	path := fmt.Sprintf(
		"%s/%s/volumes/%d/resize-volume",
		common.InstanceServicePath, instanceID, diskID,
	)
	body := ResizeVolumeValidateRequest{Size: size}

	respBody, err := c.DoRequest(ctx, http.MethodPut, path, body)
	if err != nil {
		return nil, fmt.Errorf("volume resize request failed: %w", err)
	}

	tflog.Debug(ctx, "Volume resize response", map[string]any{
		"response": string(respBody),
	})

	var result client.AuditResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse volume resize response: %w", err)
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("volume resize failed: %s (code: %d)", result.Message, result.ResponseCode)
	}

	tflog.Info(ctx, "Volume resize initiated", map[string]any{
		"audit_id":    result.Data.Audit.AuditID,
		"instance_id": instanceID,
		"disk_id":     diskID,
	})

	return &result, nil
}

// UpdateVolumeSizeAndWait resizes a volume and waits for the operation to complete.
func UpdateVolumeSizeAndWait(c *client.Client, ctx context.Context, instanceID string, diskID int64, size int64) (*client.AuditLogResponse, error) {
	tflog.Info(ctx, "Resizing volume and waiting for completion", map[string]any{
		"instance_id": instanceID,
		"disk_id":     diskID,
		"size":        size,
	})

	resizeResp, err := ResizeVolume(c, ctx, instanceID, diskID, size)
	if err != nil {
		return nil, err
	}

	requestBody := map[string]any{
		"instanceAction": "resize-disk",
		"diskId":         diskID,
		"size":           size,
	}
	auditLog, err := c.WaitForAuditCompletion(ctx, resizeResp.Data.Audit.AuditID, "update", "instance", requestBody)
	if err != nil {
		return nil, fmt.Errorf("volume resize failed: %w", err)
	}

	tflog.Info(ctx, "Volume resize completed successfully", map[string]any{
		"audit_id":    resizeResp.Data.Audit.AuditID,
		"instance_id": instanceID,
		"disk_id":     diskID,
		"size":        size,
	})

	return auditLog, nil
}

// ResizeFlavorRequest is the body for flavor resize validation and resize.
//
// POST {APIURL}/<InstanceServicePath>/{instanceId}/resize/validate
// POST {APIURL}/<InstanceServicePath>/{instanceId}/resize
type ResizeFlavorRequest struct {
	FlavorID int64 `json:"flavorId"`
}

// ValidateFlavor checks whether the instance can be resized to the requested flavor before performing the resize.
//
// POST {APIURL}/<InstanceServicePath>/{instanceId}/resize/validate (common.InstanceServicePath)
func ValidateFlavor(c *client.Client, ctx context.Context, instanceID string, flavorID int64) error {
	tflog.Debug(ctx, "Running resize validation for flavor", map[string]any{
		"instance_id": instanceID,
		"flavor_id":   flavorID,
	})

	path := fmt.Sprintf("%s/%s/resize/validate", common.InstanceServicePath, instanceID)
	body := ResizeFlavorRequest{FlavorID: flavorID}

	respBody, err := c.DoRequest(ctx, http.MethodPost, path, body)
	if err != nil {
		return fmt.Errorf("flavor resize validation request failed: %w", err)
	}

	tflog.Debug(ctx, "Flavor resize validation response", map[string]any{
		"response": string(respBody),
	})

	var result PreCreateValidateResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return fmt.Errorf("failed to parse flavor resize validation response: %w", err)
	}

	if result.Status != "success" {
		return fmt.Errorf("flavor resize validation failed (code: %d): %s", result.ResponseCode, result.Message)
	}

	var successData struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(result.Data, &successData); err == nil && successData.Message != "" {
		tflog.Info(ctx, "Flavor resize validation passed", map[string]any{
			"message":       successData.Message,
			"response_code": result.ResponseCode,
			"instance_id":   instanceID,
			"flavor_id":     flavorID,
		})
	}

	return nil
}

// ResizeFlavor initiates a flavor resize and returns the audit ID.
//
// POST {APIURL}/<InstanceServicePath>/{instanceId}/resize (common.InstanceServicePath)
func ResizeFlavor(c *client.Client, ctx context.Context, instanceID string, flavorID int64) (*client.AuditResponse, error) {
	tflog.Debug(ctx, "Resizing instance flavor", map[string]any{
		"instance_id": instanceID,
		"flavor_id":   flavorID,
	})

	path := fmt.Sprintf("%s/%s/resize", common.InstanceServicePath, instanceID)
	body := ResizeFlavorRequest{FlavorID: flavorID}

	respBody, err := c.DoRequest(ctx, http.MethodPost, path, body)
	if err != nil {
		return nil, fmt.Errorf("flavor resize request failed: %w", err)
	}

	tflog.Debug(ctx, "Flavor resize response", map[string]any{
		"response": string(respBody),
	})

	var result client.AuditResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse flavor resize response: %w", err)
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("flavor resize failed: %s (code: %d)", result.Message, result.ResponseCode)
	}

	tflog.Info(ctx, "Flavor resize initiated", map[string]any{
		"audit_id":    result.Data.Audit.AuditID,
		"instance_id": instanceID,
		"flavor_id":   flavorID,
	})

	return &result, nil
}

// UpdateFlavorAndWait validates the flavor resize, performs the resize, waits for audit completion,
// then updates action state (module=instance, action=update) with instanceAction resize-flavor and the new flavorId.
func UpdateFlavorAndWait(c *client.Client, ctx context.Context, instanceID string, flavorID int64) (*client.AuditLogResponse, error) {
	tflog.Info(ctx, "Resizing flavor and waiting for completion", map[string]any{
		"instance_id": instanceID,
		"flavor_id":   flavorID,
	})

	if err := ValidateFlavor(c, ctx, instanceID, flavorID); err != nil {
		return nil, err
	}

	resizeResp, err := ResizeFlavor(c, ctx, instanceID, flavorID)
	if err != nil {
		return nil, err
	}

	requestBody := map[string]any{
		"instanceAction": "resize-flavor",
		"flavorId":       flavorID,
	}
	auditLog, err := c.WaitForAuditCompletion(ctx, resizeResp.Data.Audit.AuditID, "update", "instance", requestBody)
	if err != nil {
		return nil, fmt.Errorf("flavor resize failed: %w", err)
	}

	tflog.Info(ctx, "Flavor resize completed successfully", map[string]any{
		"audit_id":    resizeResp.Data.Audit.AuditID,
		"instance_id": instanceID,
		"flavor_id":   flavorID,
	})

	return auditLog, nil
}

// CreateVirtualMachine initiates virtual machine creation and returns the audit ID.
func CreateVirtualMachine(c *client.Client, ctx context.Context, req *VirtualMachineCreateRequest) (*client.AuditResponse, error) {
	tflog.Debug(ctx, "Creating virtual machine", map[string]any{
		"name": req.Name, "zone_id": req.ZoneID,
		"image_id": req.ImageID, "flavor_id": req.FlavorID, "vm_purpose": req.VMPurpose,
	})

	respBody, err := c.DoRequest(ctx, http.MethodPost, fmt.Sprintf("%s/createinstance", common.InstanceServicePath), req)
	if err != nil {
		return nil, fmt.Errorf("failed to create virtual machine: %w", err)
	}

	tflog.Debug(ctx, "Create virtual machine response", map[string]any{
		"response": string(respBody),
	})

	var result client.AuditResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse create virtual machine response: %w", err)
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("virtual machine creation failed: %s (code: %d)", result.Message, result.ResponseCode)
	}

	tflog.Info(ctx, "Virtual machine creation initiated", map[string]any{
		"audit_id": result.Data.Audit.AuditID, "status": result.Status,
	})

	return &result, nil
}

// CreateVirtualMachineAndWait creates a virtual machine and waits for the operation to complete.
func CreateVirtualMachineAndWait(c *client.Client, ctx context.Context, req *VirtualMachineCreateRequest) (*client.AuditLogResponse, error) {
	tflog.Info(ctx, "Creating virtual machine and waiting for completion", map[string]any{"vm_name": req.Name})

	createResp, err := CreateVirtualMachine(c, ctx, req)
	if err != nil {
		return nil, err
	}

	var requestBody any = nil
	auditLog, err := c.WaitForAuditCompletion(ctx, createResp.Data.Audit.AuditID, "create", "instance", requestBody)
	if err != nil {
		return nil, fmt.Errorf("virtual machine creation failed: %w", err)
	}

	tflog.Info(ctx, "Virtual machine created successfully", map[string]any{
		"audit_id": createResp.Data.Audit.AuditID, "vm_name": req.Name,
	})

	return auditLog, nil
}

// DeleteVirtualMachine initiates virtual machine deletion and returns the audit ID.
//
// publicIPRetentionPolicy must be "release" or "retain" (DELETE query publicIpRetentionPolicy).
func DeleteVirtualMachine(c *client.Client, ctx context.Context, instanceID string, publicIPRetentionPolicy string) (*client.AuditResponse, error) {
	if publicIPRetentionPolicy != "Release" && publicIPRetentionPolicy != "Retain" {
		publicIPRetentionPolicy = "Retain"
	}

	q := url.Values{}
	q.Set("publicIpRetentionPolicy", publicIPRetentionPolicy)
	path := fmt.Sprintf("%s/%s?%s", common.InstanceServicePath, instanceID, q.Encode())

	tflog.Debug(ctx, "Deleting virtual machine", map[string]any{
		"instance_id":                instanceID,
		"public_ip_retention_policy": publicIPRetentionPolicy,
	})

	respBody, err := c.DoRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to delete virtual machine: %w", err)
	}

	var result client.AuditResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse delete virtual machine response: %w", err)
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("virtual machine deletion failed: %s (code: %d)", result.Message, result.ResponseCode)
	}

	tflog.Info(ctx, "Virtual machine deletion initiated", map[string]any{
		"audit_id": result.Data.Audit.AuditID, "status": result.Status, "instance_id": instanceID,
	})

	return &result, nil
}

// DeleteVirtualMachineAndWait deletes a virtual machine and waits for the operation to complete.
//
// publicIPRetentionPolicy must be "release" or "retain" (see DeleteVirtualMachine).
func DeleteVirtualMachineAndWait(c *client.Client, ctx context.Context, instanceID string, publicIPRetentionPolicy string) (*client.AuditLogResponse, error) {
	tflog.Info(ctx, "Deleting virtual machine and waiting for completion", map[string]any{
		"instance_id":                instanceID,
		"public_ip_retention_policy": publicIPRetentionPolicy,
	})

	deleteResp, err := DeleteVirtualMachine(c, ctx, instanceID, publicIPRetentionPolicy)
	if err != nil {
		return nil, err
	}

	var requestBody any = nil
	auditLog, err := c.WaitForAuditCompletion(ctx, deleteResp.Data.Audit.AuditID, "delete", "instance", requestBody)
	if err != nil {
		return nil, fmt.Errorf("virtual machine deletion failed: %w", err)
	}

	tflog.Info(ctx, "Virtual machine deleted successfully", map[string]any{
		"audit_id": deleteResp.Data.Audit.AuditID, "instance_id": instanceID,
	})

	return auditLog, nil
}

// PerformVMPowerAction performs a power action on a virtual machine.
//
// POST {APIURL}/<InstanceServicePath>/{instanceId}/{action} (common.InstanceServicePath)
// Valid actions: poweroff, poweron, suspend, hardreboot, softreboot
func PerformVMPowerAction(c *client.Client, ctx context.Context, instanceID string, powerAction string) (*client.AuditResponse, error) {
	tflog.Debug(ctx, "Performing VM power action", map[string]any{
		"instance_id": instanceID,
		"action":      powerAction,
	})

	apiPath := fmt.Sprintf("%s/%s/%s", common.InstanceServicePath, instanceID, powerAction)
	respBody, err := c.DoRequest(ctx, http.MethodPost, apiPath, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to perform power action %s: %w", powerAction, err)
	}

	tflog.Debug(ctx, "VM power action response", map[string]any{
		"response": string(respBody),
	})

	var result client.AuditResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse power action response: %w", err)
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("power action %s failed: %s (code: %d)", powerAction, result.Message, result.ResponseCode)
	}

	tflog.Info(ctx, "VM power action initiated", map[string]any{
		"audit_id":    result.Data.Audit.AuditID,
		"instance_id": instanceID,
		"action":      powerAction,
	})

	return &result, nil
}

// PerformVMPowerActionAndWait performs a power action and waits for completion.
func PerformVMPowerActionAndWait(c *client.Client, ctx context.Context, instanceID string, powerAction string) (*client.AuditLogResponse, error) {
	tflog.Info(ctx, "Performing VM power action and waiting for completion", map[string]any{
		"instance_id": instanceID,
		"action":      powerAction,
	})

	actionResp, err := PerformVMPowerAction(c, ctx, instanceID, powerAction)
	if err != nil {
		return nil, err
	}

	// Create an empty requestBody object for future extensibility, currently unused.
	requestBody := map[string]any{
		"instanceAction": powerAction,
	}
	auditLog, err := c.WaitForAuditCompletion(ctx, actionResp.Data.Audit.AuditID, "update", "instance", requestBody)
	if err != nil {
		return nil, fmt.Errorf("VM power action %s failed: %w", powerAction, err)
	}

	tflog.Info(ctx, "VM power action completed successfully", map[string]any{
		"audit_id":    actionResp.Data.Audit.AuditID,
		"instance_id": instanceID,
		"action":      powerAction,
	})

	return auditLog, nil
}

// VirtualMachineListVolume represents a volume in the list response.
type VirtualMachineListVolume struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Size        int64  `json:"size"`
	DiskType    string `json:"diskType"`
	CreatedDate string `json:"createdDate"`
	IOPS        int64  `json:"iops"`
}

// VirtualMachineListItem represents a single virtual machine in the list response.
// Unlike VirtualMachineDetail, this does not include flavorId/imageId but provides
// flavor_name and image_name instead.
type VirtualMachineListItem struct {
	ID            int64                      `json:"id"`
	Name          string                     `json:"name"`
	Hostname      string                     `json:"hostname"`
	ZoneID        int64                      `json:"zoneId"`
	IP            string                     `json:"ip"`
	RootDiskSize  int64                      `json:"rootDisk"`
	PowerStatus   string                     `json:"powerStatus"`
	FlavorName    string                     `json:"flavorName"`
	ImageName     string                     `json:"imageName"`
	OsType        string                     `json:"osType"`
	OsVersion     string                     `json:"osVersion"`
	OsModel       string                     `json:"osModel"`
	OsMake        string                     `json:"osMake"`
	OsServicePack string                     `json:"osServicePack"`
	PricingModel  string                     `json:"pricingModel"`
	Volumes       []VirtualMachineListVolume `json:"volumes"`
	VCpu          int64                      `json:"vCpu"`
	VRam          int64                      `json:"vRam"`
}

// ListVirtualMachines retrieves all virtual machines for a given zone
// via the action-state API with module=instance&action=list.
func ListVirtualMachines(c *client.Client, ctx context.Context, zoneID int64) ([]VirtualMachineListItem, *common.ActionStateResponse, error) {
	tflog.Debug(ctx, "Listing virtual machines", map[string]any{
		"zone_id": zoneID,
	})

	body := map[string]any{
		"resourceId": fmt.Sprintf("%d", zoneID),
	}

	resp, err := common.UpdateActionState(ctx, c, "instance", "list", body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list virtual machines: %w", err)
	}

	if len(resp.Data) == 0 {
		return []VirtualMachineListItem{}, resp, nil
	}

	var items []VirtualMachineListItem
	if err := json.Unmarshal(resp.Data, &items); err != nil {
		return nil, resp, fmt.Errorf("failed to parse virtual machine list response: %w", err)
	}

	tflog.Info(ctx, "Virtual machines listed successfully", map[string]any{
		"count":   len(items),
		"zone_id": zoneID,
	})

	return items, resp, nil
}

// VirtualMachineVolume represents a volume attached to a virtual machine.
type VirtualMachineVolume struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Size        int64  `json:"size"`
	DiskType    string `json:"diskType"`
	CreatedDate string `json:"createdDate"`
	IOPS        int64  `json:"iops"`
}

// VirtualMachineDetail represents the full virtual machine detail returned by the API.
type VirtualMachineDetail struct {
	ID                   int64                  `json:"id"`
	Name                 string                 `json:"name"`
	Hostname             string                 `json:"hostname"`
	ZoneID               int64                  `json:"zoneId"`
	IP                   string                 `json:"ip"`
	RootDiskSize         int64                  `json:"rootDisk"`
	PowerStatus          string                 `json:"powerStatus"`
	FlavorID             int64                  `json:"flavorId"`
	ImageID              int64                  `json:"imageId"`
	OsType               string                 `json:"osType"`
	OsVersion            string                 `json:"osVersion"`
	OsModel              string                 `json:"osModel"`
	OsMake               string                 `json:"osMake"`
	OsServicePack        string                 `json:"osServicePack"`
	PublicIP             string                 `json:"publicIp"`
	PublicIPPricingModel string                 `json:"publicIpPricingModel"`
	PricingModel         string                 `json:"pricingModel"`
	Volumes              []VirtualMachineVolume `json:"volumes"`
	VCpu                 int64                  `json:"vCpu"`
	VRam                 int64                  `json:"vRam"`
}

// VirtualMachineDetailResponse represents the API response for getting virtual machine details.
type VirtualMachineDetailResponse struct {
	Status       string               `json:"status"`
	Data         VirtualMachineDetail `json:"data"`
	Message      string               `json:"message"`
	ResponseCode int                  `json:"responseCode"`
}

// GetVirtualMachineDetail retrieves virtual machine details by instance ID.
//
// GET {APIURL}/<InstanceServicePath>/{instanceId} (common.InstanceServicePath)
func GetVirtualMachineDetail(c *client.Client, ctx context.Context, instanceID string) (*VirtualMachineDetailResponse, error) {
	tflog.Debug(ctx, "Fetching virtual machine detail", map[string]any{
		"instance_id": instanceID,
	})

	path := fmt.Sprintf("%s/%s", common.InstanceServicePath, instanceID)
	respBody, err := c.DoRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get virtual machine detail: %w", err)
	}

	tflog.Debug(ctx, "Get virtual machine detail response", map[string]any{
		"response": string(respBody),
	})

	var result VirtualMachineDetailResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse virtual machine detail response: %w", err)
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("get virtual machine detail failed: %s (code: %d)", result.Message, result.ResponseCode)
	}

	tflog.Info(ctx, "Virtual machine detail fetched successfully", map[string]any{
		"instance_id": instanceID,
		"name":        result.Data.Name,
		"status":      result.Status,
	})

	return &result, nil
}

// ValidateInstanceExists checks whether a virtual machine with the given ID exists
// by calling the same detail API used for the VM read operation.
func ValidateInstanceExists(c *client.Client, ctx context.Context, instanceID int64) error {
	tflog.Debug(ctx, "Validating instance exists", map[string]any{
		"instance_id": instanceID,
	})

	_, err := GetVirtualMachineDetail(c, ctx, fmt.Sprintf("%d", instanceID))
	if err != nil {
		return fmt.Errorf("Instance ID %d is not available: %w", instanceID, err)
	}

	tflog.Debug(ctx, "Instance ID validated successfully", map[string]any{
		"instance_id": instanceID,
	})

	return nil
}
