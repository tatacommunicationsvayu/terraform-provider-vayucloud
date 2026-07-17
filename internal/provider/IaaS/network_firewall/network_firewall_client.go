// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package network_firewall

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/client"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/common"
)

func stripBandwidthUnit(value string) string {
	value = strings.TrimSpace(value)
	lower := strings.ToLower(value)

	suffixes := []string{"gbps", "mbps", "kbps", "bps"}
	for _, suffix := range suffixes {
		if strings.HasSuffix(lower, suffix) {
			return strings.TrimSpace(value[:len(value)-len(suffix)])
		}
	}
	return value
}

func stripMinimumCommitmentUnit(value string) string {
	value = strings.TrimSpace(value)
	lower := strings.ToLower(value)

	suffixes := []string{"gb", "mb", "tb"}
	for _, suffix := range suffixes {
		if strings.HasSuffix(lower, suffix) {
			return strings.TrimSpace(value[:len(value)-len(suffix)])
		}
	}
	return value
}

type NetworkFirewallCreateRequest struct {
	EngagementID          int64   `json:"engagement_id"`
	EndpointID            int64   `json:"endpoint_id"`
	FirewallDisplayName   string  `json:"firewall_name"`
	Bandwidth             string  `json:"bandwidth"`
	MinimumCommitment     string  `json:"minimum_commitment"`
	FirewallThroughput    string  `json:"firewall_throughput"`
	AccessType            string  `json:"access_type"`
	NonDistributedEnabled bool    `json:"non_distributed_firewall"`
	Hypervisor            string  `json:"hypervisor"`
	FirewallPricingModel  *string `json:"firewall_pricing_model,omitempty"`
	InternetPricingModel  *string `json:"internet_pricing_model,omitempty"`
	IsInternetEnabled     bool    `json:"is_internet_enabled"`
}

type NetworkFirewallUpdateRequest struct {
	SelfProvisioning   bool   `json:"self_provisioning"`
	FirewallThroughput string `json:"firewall_throughput"`
	Bandwidth          string `json:"bandwidth"`
	AccessType         string `json:"access_type"`
	MinimumCommitment  string `json:"minimum_commitment"`
}

func CreateNetworkFirewall(c *client.Client, ctx context.Context, req *NetworkFirewallCreateRequest) (*client.AuditResponse, error) {
	tflog.Debug(ctx, "Creating network firewall", map[string]any{
		"engagement_id": req.EngagementID,
		"endpoint_id":   req.EndpointID,
		"firewall_name": req.FirewallDisplayName,
		"firewall_type": req.Hypervisor,
		"hypervisor":    req.Hypervisor,
	})

	respBody, err := c.DoRequest(ctx, http.MethodPost, common.NetworkOperationsPath+"/firewall", req)
	if err != nil {
		return nil, fmt.Errorf("failed to create network firewall: %w", err)
	}

	tflog.Debug(ctx, "Create network firewall response", map[string]any{
		"response": string(respBody),
	})

	var result client.AuditResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse create network firewall response: %w", err)
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("network firewall creation failed: %s (code: %d)", result.Message, result.ResponseCode)
	}

	tflog.Info(ctx, "Network firewall creation initiated", map[string]any{
		"audit_id": result.Data.Audit.AuditID,
		"status":   result.Status,
	})

	return &result, nil
}

func CreateNetworkFirewallAndWait(c *client.Client, ctx context.Context, req *NetworkFirewallCreateRequest) (*client.AuditLogResponse, error) {
	tflog.Info(ctx, "Creating network firewall and waiting for completion", map[string]any{
		"firewall_name": req.FirewallDisplayName,
	})

	createResp, err := CreateNetworkFirewall(c, ctx, req)
	if err != nil {
		return nil, err
	}

	var requestBody any = nil

	auditLog, err := c.WaitForAuditCompletion(ctx, createResp.Data.Audit.AuditID, "create", "firewall", requestBody)
	if err != nil {
		return nil, fmt.Errorf("network firewall creation failed: %w", err)
	}

	tflog.Info(ctx, "Network firewall created successfully", map[string]any{
		"audit_id":      createResp.Data.Audit.AuditID,
		"firewall_name": req.FirewallDisplayName,
	})

	return auditLog, nil
}

func UpdateNetworkFirewall(c *client.Client, ctx context.Context, firewallID int64, firewallThroughput, bandwidth string, minimumCommitment string) (*client.AuditResponse, error) {
	tflog.Debug(ctx, "Updating network firewall", map[string]any{
		"firewall_id":         firewallID,
		"firewall_throughput": firewallThroughput,
		"bandwidth":           bandwidth,
		"minimum_commitment":  minimumCommitment,
	})

	throughputValue := stripBandwidthUnit(firewallThroughput)
	bandwidthValue := ""
	if bandwidth!="" {
		bandwidthValue = stripBandwidthUnit(bandwidth)
	}
	

	req := &NetworkFirewallUpdateRequest{
		SelfProvisioning:   true,
		FirewallThroughput: throughputValue,
	}
	if minimumCommitment!="" {
		req.Bandwidth = bandwidthValue
		req.MinimumCommitment = minimumCommitment
		req.AccessType = "DataTransfer"
	} else {
		req.Bandwidth = bandwidthValue
		req.AccessType = "Bandwidth"
	}

	path := fmt.Sprintf(common.NetworkOperationsPath+"/firewall/%d", firewallID)
	respBody, err := c.DoRequest(ctx, http.MethodPut, path, req)
	if err != nil {
		return nil, fmt.Errorf("failed to update network firewall: %w", err)
	}

	tflog.Debug(ctx, "Update network firewall response", map[string]any{
		"response": string(respBody),
	})

	var result client.AuditResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse update network firewall response: %w", err)
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("network firewall update failed: %s (code: %d)", result.Message, result.ResponseCode)
	}

	tflog.Info(ctx, "Network firewall update initiated", map[string]any{
		"audit_id":    result.Data.Audit.AuditID,
		"status":      result.Status,
		"firewall_id": firewallID,
	})

	return &result, nil
}

func UpdateNetworkFirewallAndWait(c *client.Client, ctx context.Context, firewallID int64, firewallThroughput, bandwidth string, minimumCommitment string) (*client.AuditLogResponse, error) {
	tflog.Info(ctx, "Updating network firewall and waiting for completion", map[string]any{
		"firewall_id":         firewallID,
		"firewall_throughput": firewallThroughput,
		"bandwidth":           bandwidth,
		"minimum_commitment":  minimumCommitment,
	})

	updateResp, err := UpdateNetworkFirewall(c, ctx, firewallID, firewallThroughput, bandwidth, minimumCommitment)
	if err != nil {
		return nil, err
	}

	requestBody := map[string]any{
		"firewallThroughput": firewallThroughput,
		"bandwidth":           bandwidth,
		"minimum_commitment":  minimumCommitment,
	}

	auditLog, err := c.WaitForAuditCompletion(ctx, updateResp.Data.Audit.AuditID, "update", "firewall", requestBody)
	if err != nil {
		return nil, fmt.Errorf("network firewall update failed: %w", err)
	}

	tflog.Info(ctx, "Network firewall updated successfully", map[string]any{
		"audit_id":    updateResp.Data.Audit.AuditID,
		"firewall_id": firewallID,
	})

	return auditLog, nil
}


func UpdateNetworkFirewallDisplayName(c *client.Client, ctx context.Context, firewallID int64, displayName string) error {
	tflog.Debug(ctx, "Updating firewall display name", map[string]any{
		"firewall_id":  firewallID,
		"display_name": displayName,
	})

	req := map[string]any{
		"firewallName": displayName,
	}

	path := fmt.Sprintf(common.ConfigServicePath+"/createorupdatename/%d", firewallID)
	respBody, err := c.DoRequest(ctx, http.MethodPut, path, req)
	if err != nil {
		return fmt.Errorf("failed to update network firewall display name: %w", err)
	}

	tflog.Debug(ctx, "Update network firewall display name response", map[string]any{
		"response": string(respBody),
	})

	var result common.ActionStateResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return fmt.Errorf("failed to parse update network firewall display name response: %w", err)
	}

	if result.Status != "success" {
		return fmt.Errorf("network firewall display name update failed: %s (code: %d)", result.Message, result.ResponseCode)
	}

	tflog.Info(ctx, "Network firewall display name updated successfully", map[string]any{
		"status":       result.Status,
		"firewall_id":  firewallID,
		"display_name": displayName,
		"message":      result.Message,
	})

	return nil
}

func DeleteNetworkFirewall(c *client.Client, ctx context.Context, firewallID string) (*client.AuditResponse, error) {
	tflog.Debug(ctx, "Deleting network firewall", map[string]any{
		"firewall_id": firewallID,
	})

	path := fmt.Sprintf(common.NetworkOperationsPath+"/firewall/%s", firewallID)
	respBody, err := c.DoRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to delete network firewall: %w", err)
	}

	tflog.Debug(ctx, "Delete network firewall response", map[string]any{
		"response": string(respBody),
	})

	var result client.AuditResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse delete network firewall response: %w", err)
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("network firewall deletion failed: %s (code: %d)", result.Message, result.ResponseCode)
	}

	tflog.Info(ctx, "Network firewall deletion initiated", map[string]any{
		"audit_id":    result.Data.Audit.AuditID,
		"status":      result.Status,
		"firewall_id": firewallID,
	})

	return &result, nil
}

func DeleteNetworkFirewallAndWait(c *client.Client, ctx context.Context, firewallID string) (*client.AuditLogResponse, error) {
	tflog.Info(ctx, "Deleting network firewall and waiting for completion", map[string]any{
		"firewall_id": firewallID,
	})

	deleteResp, err := DeleteNetworkFirewall(c, ctx, firewallID)
	if err != nil {
		return nil, err
	}

	var requestBody any = nil

	auditLog, err := c.WaitForAuditCompletion(ctx, deleteResp.Data.Audit.AuditID, "delete", "firewall", requestBody)
	if err != nil {
		return nil, fmt.Errorf("network firewall deletion failed: %w", err)
	}

	tflog.Info(ctx, "Network firewall deleted successfully", map[string]any{
		"audit_id":    deleteResp.Data.Audit.AuditID,
		"firewall_id": firewallID,
	})

	return auditLog, nil
}

// NetworkFirewallListItem represents a single firewall returned by the list action.
type NetworkFirewallListItem struct {
	ID                   int64  `json:"id"`
	FirewallDisplayName  string `json:"firewall_display_name"`
	FirewallThroughput   string `json:"firewall_throughput"`
	Bandwidth            string `json:"bandwidth"`
	FirewallPricingModel string `json:"firewall_pricing_model"`
	InternetPricingModel string `json:"internet_pricing_model"`
	EngagementID         int64  `json:"engagement_id"`
	EndpointID           int64  `json:"endpoint_id"`
	Hypervisor           string `json:"hypervisor"`
}

// ListNetworkFirewalls retrieves all firewalls for a given engagement and endpoint
// via the action-state API with module=firewall&action=list.
func ListNetworkFirewalls(c *client.Client, ctx context.Context, engagementID int64, endpointID int64) ([]NetworkFirewallListItem, *common.ActionStateResponse, error) {
	tflog.Debug(ctx, "Listing network firewalls", map[string]any{
		"engagement_id": engagementID,
		"endpoint_id":   endpointID,
	})

	body := map[string]any{
		"engagementId": engagementID,
		"endpointId":   endpointID,
	}

	resp, err := common.UpdateActionState(ctx, c, "firewall", "list", body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list network firewalls: %w", err)
	}

	if len(resp.Data) == 0 {
		return []NetworkFirewallListItem{}, resp, nil
	}

	var items []NetworkFirewallListItem
	if err := json.Unmarshal(resp.Data, &items); err != nil {
		return nil, resp, fmt.Errorf("failed to parse network firewall list response: %w", err)
	}

	tflog.Info(ctx, "Network firewalls listed successfully", map[string]any{
		"count":         len(items),
		"engagement_id": engagementID,
		"endpoint_id":   endpointID,
	})

	return items, resp, nil
}

func ValidateFirewallExists(c *client.Client, ctx context.Context, firewallID int64) error {
	tflog.Debug(ctx, "Validating firewall exists", map[string]any{
		"firewall_id": firewallID,
	})

	actionStateBody := map[string]any{
		"resourceId": fmt.Sprintf("%d", firewallID),
	}

	actionStateResponse, err := common.UpdateActionState(ctx, c, "firewall", "read", actionStateBody)
	if err != nil {
		return fmt.Errorf("failed to validate firewall ID: %w", err)
	}

	if len(actionStateResponse.Data) == 0 {
		return fmt.Errorf("Firewall ID %d is not available", firewallID)
	}

	var responseMap map[string]interface{}
	if err := json.Unmarshal(actionStateResponse.Data, &responseMap); err != nil {
		return fmt.Errorf("Firewall ID %d is not available", firewallID)
	}

	if len(responseMap) == 0 {
		return fmt.Errorf("Firewall ID %d is not available", firewallID)
	}

	tflog.Debug(ctx, "Firewall ID validated successfully", map[string]any{
		"firewall_id": firewallID,
	})

	return nil
}
