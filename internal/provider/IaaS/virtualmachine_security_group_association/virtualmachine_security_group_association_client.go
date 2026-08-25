// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package virtualmachine_security_group_association

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/client"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/common"
)

// VMSecurityGroupAssociationRequest is the body for attach/detach security groups on a VM.
type VMSecurityGroupAssociationRequest struct {
	SecurityGroupIDs []string `json:"security_group_ids"`
}

// AttachVMSecurityGroups associates security groups with a VM and returns the async audit response.
//
// POST {SecurityGroupServicePath}/vm/{instanceId}
func AttachVMSecurityGroups(c *client.Client, ctx context.Context, instanceID int64, securityGroupIDs []string) (*client.AuditResponse, error) {
	tflog.Debug(ctx, "Attaching security groups to VM", map[string]any{
		"instance_id":        instanceID,
		"security_group_ids": securityGroupIDs,
	})

	path := fmt.Sprintf("%s/vm/%d", common.SecurityGroupServicePath, instanceID)
	req := &VMSecurityGroupAssociationRequest{SecurityGroupIDs: securityGroupIDs}
	respBody, err := c.DoRequest(ctx, http.MethodPost, path, req)
	if err != nil {
		return nil, fmt.Errorf("failed to attach security groups to VM %d: %w", instanceID, err)
	}

	var result client.AuditResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse attach VM security groups response: %w", err)
	}
	if result.Status != "success" {
		return nil, fmt.Errorf("attach VM security groups failed: %s (code: %d)", result.Message, result.ResponseCode)
	}
	return &result, nil
}

// AttachVMSecurityGroupsAndWait attaches security groups and waits for audit completion (no action-state).
func AttachVMSecurityGroupsAndWait(c *client.Client, ctx context.Context, instanceID int64, securityGroupIDs []string) (*client.AuditLogResponse, error) {
	resp, err := AttachVMSecurityGroups(c, ctx, instanceID, securityGroupIDs)
	if err != nil {
		return nil, err
	}
	return c.WaitForAuditCompletionNoActionState(ctx, resp.Data.Audit.AuditID)
}

// DetachVMSecurityGroups dissociates security groups from a VM and returns the async audit response.
//
// DELETE {SecurityGroupServicePath}/vm/{instanceId}
func DetachVMSecurityGroups(c *client.Client, ctx context.Context, instanceID int64, securityGroupIDs []string) (*client.AuditResponse, error) {
	tflog.Debug(ctx, "Detaching security groups from VM", map[string]any{
		"instance_id":        instanceID,
		"security_group_ids": securityGroupIDs,
	})

	path := fmt.Sprintf("%s/vm/%d", common.SecurityGroupServicePath, instanceID)
	req := &VMSecurityGroupAssociationRequest{SecurityGroupIDs: securityGroupIDs}
	respBody, err := c.DoRequest(ctx, http.MethodDelete, path, req)
	if err != nil {
		return nil, fmt.Errorf("failed to detach security groups from VM %d: %w", instanceID, err)
	}

	var result client.AuditResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse detach VM security groups response: %w", err)
	}
	if result.Status != "success" {
		return nil, fmt.Errorf("detach VM security groups failed: %s (code: %d)", result.Message, result.ResponseCode)
	}
	return &result, nil
}

// DetachVMSecurityGroupsAndWait detaches security groups and waits for audit completion (no action-state).
func DetachVMSecurityGroupsAndWait(c *client.Client, ctx context.Context, instanceID int64, securityGroupIDs []string) (*client.AuditLogResponse, error) {
	resp, err := DetachVMSecurityGroups(c, ctx, instanceID, securityGroupIDs)
	if err != nil {
		return nil, err
	}
	return c.WaitForAuditCompletionNoActionState(ctx, resp.Data.Audit.AuditID)
}
