// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package virtualmachine_blockstorage

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/client"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/common"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/virtualmachine"
)

// portalValidateResponse matches the attach/resize validate APIs (same shape as pre-create validation).
type portalValidateResponse struct {
	Status       string          `json:"status"`
	Data         json.RawMessage `json:"data"`
	Message      string          `json:"message"`
	ResponseCode int             `json:"responseCode"`
}

// AttachVolumeRequest is the body for attach-volume validate and attach.
//
// POST {APIURL}/<InstanceServicePath>/{instanceId}/attach-volume/validate
// POST {APIURL}/<InstanceServicePath>/{instanceId}/attach-volume (common.InstanceServicePath)
type AttachVolumeRequest struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
	IOPS int64  `json:"iops"`
}

// ValidateAttachVolume calls the attach-volume validate API before attaching a volume.
//
// POST {APIURL}/<InstanceServicePath>/{instanceId}/attach-volume/validate (common.InstanceServicePath)
func ValidateAttachVolume(c *client.Client, ctx context.Context, instanceID int64, req *AttachVolumeRequest) error {
	tflog.Debug(ctx, "Running attach-volume validation", map[string]any{
		"instance_id": instanceID,
		"name":        req.Name,
		"size":        req.Size,
		"iops":        req.IOPS,
	})

	path := fmt.Sprintf("%s/%d/attach-volume/validate", common.InstanceServicePath, instanceID)
	respBody, err := c.DoRequest(ctx, http.MethodPost, path, req)
	if err != nil {
		return fmt.Errorf("attach-volume validation request failed: %w", err)
	}

	tflog.Debug(ctx, "Attach-volume validation response", map[string]any{
		"response": string(respBody),
	})

	var result portalValidateResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return fmt.Errorf("failed to parse attach-volume validation response: %w", err)
	}

	if result.Status != "success" {
		return fmt.Errorf("attach-volume validation failed (code: %d): %s", result.ResponseCode, result.Message)
	}

	var successData struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(result.Data, &successData); err == nil && successData.Message != "" {
		tflog.Info(ctx, "Attach-volume validation passed", map[string]any{
			"message":       successData.Message,
			"response_code": result.ResponseCode,
			"instance_id":   instanceID,
		})
	}

	return nil
}

// AttachVolume initiates attach-volume and returns the audit ID.
//
// POST {APIURL}/<InstanceServicePath>/{instanceId}/attach-volume (common.InstanceServicePath)
func AttachVolume(c *client.Client, ctx context.Context, instanceID int64, req *AttachVolumeRequest) (*client.AuditResponse, error) {
	tflog.Debug(ctx, "Attaching volume to instance", map[string]any{
		"instance_id": instanceID,
		"name":        req.Name,
		"size":        req.Size,
		"iops":        req.IOPS,
	})

	path := fmt.Sprintf("%s/%d/attach-volume", common.InstanceServicePath, instanceID)
	respBody, err := c.DoRequest(ctx, http.MethodPost, path, req)
	if err != nil {
		return nil, fmt.Errorf("attach-volume request failed: %w", err)
	}

	tflog.Debug(ctx, "Attach-volume response", map[string]any{
		"response": string(respBody),
	})

	var result client.AuditResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse attach-volume response: %w", err)
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("attach-volume failed: %s (code: %d)", result.Message, result.ResponseCode)
	}

	tflog.Info(ctx, "Attach-volume initiated", map[string]any{
		"audit_id":    result.Data.Audit.AuditID,
		"instance_id": instanceID,
	})

	return &result, nil
}

// AttachVolumeAndWait attaches a volume and waits for audit completion (same pattern as virtual machine creation).
func AttachVolumeAndWait(c *client.Client, ctx context.Context, instanceID int64, req *AttachVolumeRequest) (*client.AuditLogResponse, error) {
	tflog.Info(ctx, "Attaching volume and waiting for completion", map[string]any{
		"instance_id": instanceID,
		"name":        req.Name,
	})

	attachResp, err := AttachVolume(c, ctx, instanceID, req)
	if err != nil {
		return nil, err
	}

	// Create an empty requestBody object for future extensibility, currently unused.
	requestBody := map[string]any{
		"instanceAction": "attach-disk",
		"size":           req.Size,
	}
	auditLog, err := c.WaitForAuditCompletion(ctx, attachResp.Data.Audit.AuditID, "update", "instance", requestBody)
	if err != nil {
		return nil, fmt.Errorf("attach-volume failed: %w", err)
	}

	tflog.Info(ctx, "Attach-volume completed successfully", map[string]any{
		"audit_id": attachResp.Data.Audit.AuditID,
	})

	return auditLog, nil
}

// FindNonRootVolumeIDByName returns the volume ID for a non-root volume with the given name on the instance.
func FindNonRootVolumeIDByName(c *client.Client, ctx context.Context, instanceID, volumeName string) (int64, error) {
	detail, err := virtualmachine.GetVirtualMachineDetail(c, ctx, instanceID)
	if err != nil {
		return 0, err
	}

	for _, v := range detail.Data.Volumes {
		if strings.EqualFold(strings.TrimSpace(v.Name), strings.TrimSpace(volumeName)) && strings.ToLower(v.DiskType) != "root" {
			return v.ID, nil
		}
	}

	return 0, fmt.Errorf("no non-root volume named %q found on instance %s", volumeName, instanceID)
}

// GetVolumeByID returns volume details from the instance detail API.
func GetVolumeByID(c *client.Client, ctx context.Context, instanceID int64, volumeID int64) (*virtualmachine.VirtualMachineVolume, error) {
	detail, err := virtualmachine.GetVirtualMachineDetail(c, ctx, fmt.Sprintf("%d", instanceID))
	if err != nil {
		return nil, err
	}

	for i := range detail.Data.Volumes {
		if detail.Data.Volumes[i].ID == volumeID {
			v := detail.Data.Volumes[i]
			return &v, nil
		}
	}

	return nil, fmt.Errorf("volume id %d not found on instance %s", volumeID, instanceID)
}

// DeleteAttachedVolume deletes a volume attached to an instance.
//
// DELETE {APIURL}/<InstanceServicePath>/{instanceId}/volumes/{diskId} (common.InstanceServicePath)
func DeleteAttachedVolume(c *client.Client, ctx context.Context, instanceID int64, diskID int64) (*client.AuditResponse, error) {
	tflog.Debug(ctx, "Deleting attached volume", map[string]any{
		"instance_id": instanceID,
		"disk_id":     diskID,
	})

	path := fmt.Sprintf("%s/%d/volumes/%d", common.InstanceServicePath, instanceID, diskID)
	respBody, err := c.DoRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return nil, fmt.Errorf("delete volume request failed: %w", err)
	}

	tflog.Debug(ctx, "Delete volume response", map[string]any{
		"response": string(respBody),
	})

	var result client.AuditResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse delete volume response: %w", err)
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("delete volume failed: %s (code: %d)", result.Message, result.ResponseCode)
	}

	tflog.Info(ctx, "Delete volume initiated", map[string]any{
		"audit_id":    result.Data.Audit.AuditID,
		"instance_id": instanceID,
		"disk_id":     diskID,
	})

	return &result, nil
}

// DeleteAttachedVolumeAndWait deletes a volume and waits for audit completion.
func DeleteAttachedVolumeAndWait(c *client.Client, ctx context.Context, instanceID int64, diskID int64) (*client.AuditLogResponse, error) {
	tflog.Info(ctx, "Deleting volume and waiting for completion", map[string]any{
		"instance_id": instanceID,
		"disk_id":     diskID,
	})

	delResp, err := DeleteAttachedVolume(c, ctx, instanceID, diskID)
	if err != nil {
		return nil, err
	}

	// Create an empty requestBody object for future extensibility, currently unused.
	requestBody := map[string]any{
		"instanceAction": "delete-disk",
	}
	auditLog, err := c.WaitForAuditCompletion(ctx, delResp.Data.Audit.AuditID, "update", "instance", requestBody)
	if err != nil {
		return nil, fmt.Errorf("delete volume failed: %w", err)
	}

	tflog.Info(ctx, "Volume deleted successfully", map[string]any{
		"audit_id": delResp.Data.Audit.AuditID,
	})

	return auditLog, nil
}
