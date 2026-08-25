// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package network_zone

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/client"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/common"
)

type DataPlane struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type NetworkZoneCreateRequest struct {
	Name          string    `json:"name"`
	EnvironmentID int64     `json:"environmentId"`
	FirewallCI    int64     `json:"firewallCi"`
	NoOfIPs       *int64    `json:"noOfIPs,omitempty"`
	Purpose       string    `json:"purpose"`
	DataPlane     DataPlane `json:"dataPlane"`
	ZoneType      string    `json:"zoneType"`
	DualStackMode string    `json:"dualStackMode"`
	NoOfV6IPs     *int64    `json:"noOfv6IPs,omitempty"`
}

type NetworkZoneInputConfig struct {
	Name          string
	EnvironmentID int64
	FirewallID    int64
	NoOfIPs       *int64
	Purpose       string
	DataPlaneType string
	CIDR          string
	ZoneType      string
	DualStackMode string
	NoOfV6IPs     *int64
}

func BuildNetworkZoneCreateRequest(config *NetworkZoneInputConfig) *NetworkZoneCreateRequest {
	req := &NetworkZoneCreateRequest{
		Name:          config.Name,
		EnvironmentID: config.EnvironmentID,
		FirewallCI:    config.FirewallID,
		Purpose:       config.Purpose,
		ZoneType:      config.ZoneType,
	}

	req.DataPlane = DataPlane{
		Type:  "Auto IPAM",
		Value: "Auto IPAM",
	}
	req.NoOfIPs = config.NoOfIPs

	if config.NoOfV6IPs != nil {
		req.NoOfV6IPs = config.NoOfV6IPs
		req.DualStackMode = config.DualStackMode
	} else {
		req.DualStackMode = config.DualStackMode
		req.NoOfV6IPs = nil
	}

	return req
}

type NetworkZoneUpdateRequest struct {
	Name string `json:"name"`
}

func CreateNetworkZone(c *client.Client, ctx context.Context, req *NetworkZoneCreateRequest) (*client.AuditResponse, error) {
	tflog.Debug(ctx, "Creating network zone", map[string]any{
		"name":            req.Name,
		"environment_id":  req.EnvironmentID,
		"firewall_ci":     req.FirewallCI,
		"zone_type":       req.ZoneType,
		"data_plane_type": req.DataPlane.Type,
		"dual_stack_mode": req.DualStackMode,
	})

	respBody, err := c.DoRequest(ctx, http.MethodPost, fmt.Sprintf("%s/create", common.NetworkServicePath), req)
	if err != nil {
		return nil, fmt.Errorf("failed to create network zone: %w", err)
	}

	tflog.Debug(ctx, "Create network zone response", map[string]any{
		"response": string(respBody),
	})

	var result client.AuditResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse create network zone response: %w", err)
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("zone creation failed: %s (code: %d)", result.Message, result.ResponseCode)
	}

	tflog.Info(ctx, "Zone creation initiated", map[string]any{
		"audit_id": result.Data.Audit.AuditID,
		"status":   result.Status,
	})

	return &result, nil
}

func CreateNetworkZoneAndWait(c *client.Client, ctx context.Context, req *NetworkZoneCreateRequest) (*client.AuditLogResponse, error) {
	tflog.Info(ctx, "Creating network zone and waiting for completion", map[string]any{
		"network_zone_name": req.Name,
	})

	createResp, err := CreateNetworkZone(c, ctx, req)
	if err != nil {
		return nil, err
	}

	var requestBody any = nil

	auditLog, err := c.WaitForAuditCompletion(ctx, createResp.Data.Audit.AuditID, "create", "zone", requestBody)
	if err != nil {
		return nil, fmt.Errorf("network zone creation failed: %w", err)
	}

	tflog.Info(ctx, "Network Zone created successfully", map[string]any{
		"audit_id":          createResp.Data.Audit.AuditID,
		"network_zone_name": req.Name,
	})

	return auditLog, nil
}

func UpdateNetworkZone(c *client.Client, ctx context.Context, zoneID string, name string) (*client.AuditResponse, error) {
	tflog.Debug(ctx, "Updating network zone", map[string]any{
		"zone_id": zoneID,
		"name":    name,
	})

	req := &NetworkZoneUpdateRequest{
		Name: name,
	}

	path := fmt.Sprintf("%s/zone/%s", common.NetworkServicePath, zoneID)
	respBody, err := c.DoRequest(ctx, http.MethodPut, path, req)
	if err != nil {
		return nil, fmt.Errorf("failed to update network zone: %w", err)
	}

	tflog.Debug(ctx, "Update network zone response", map[string]any{
		"response": string(respBody),
	})

	var result client.AuditResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse update network zone response: %w", err)
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("network zone update failed: %s (code: %d)", result.Message, result.ResponseCode)
	}

	tflog.Info(ctx, "Network zone update initiated", map[string]any{
		"audit_id": result.Data.Audit.AuditID,
		"status":   result.Status,
		"zone_id":  zoneID,
	})

	return &result, nil
}

func UpdateNetworkZoneAndWait(c *client.Client, ctx context.Context, zoneID string, name string) (*client.AuditLogResponse, error) {
	tflog.Info(ctx, "Updating network zone and waiting for completion", map[string]any{
		"zone_id": zoneID,
		"name":    name,
	})

	updateResp, err := UpdateNetworkZone(c, ctx, zoneID, name)
	if err != nil {
		return nil, err
	}

	requestBody := map[string]any{
		"name": name,
	}

	auditLog, err := c.WaitForAuditCompletion(ctx, updateResp.Data.Audit.AuditID, "update", "zone", requestBody)
	if err != nil {
		return nil, fmt.Errorf("network zone update failed: %w", err)
	}

	tflog.Info(ctx, "Network zone updated successfully", map[string]any{
		"audit_id": updateResp.Data.Audit.AuditID,
		"zone_id":  zoneID,
	})

	return auditLog, nil
}

func DeleteNetworkZone(c *client.Client, ctx context.Context, networkZoneID string) (*client.AuditResponse, error) {
	tflog.Debug(ctx, "Deleting zone", map[string]any{
		"zone_id": networkZoneID,
	})

	path := fmt.Sprintf("%s/delete/%s", common.NetworkServicePath, networkZoneID)
	respBody, err := c.DoRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to delete network zone: %w", err)
	}

	tflog.Debug(ctx, "Delete network zone response", map[string]any{
		"response": string(respBody),
	})

	var result client.AuditResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse delete network zone response: %w", err)
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("network zone deletion failed: %s (code: %d)", result.Message, result.ResponseCode)
	}

	tflog.Info(ctx, "Zone deletion initiated", map[string]any{
		"audit_id": result.Data.Audit.AuditID,
		"status":   result.Status,
		"zone_id":  networkZoneID,
	})

	return &result, nil
}

func DeleteNetworkZoneAndWait(c *client.Client, ctx context.Context, networkZoneID string) (*client.AuditLogResponse, error) {
	tflog.Info(ctx, "Deleting network zone and waiting for completion", map[string]any{
		"zone_id": networkZoneID,
	})

	deleteResp, err := DeleteNetworkZone(c, ctx, networkZoneID)
	if err != nil {
		return nil, err
	}

	var requestBody any = nil

	auditLog, err := c.WaitForAuditCompletion(ctx, deleteResp.Data.Audit.AuditID, "delete", "zone", requestBody)
	if err != nil {
		return nil, fmt.Errorf("network zone deletion failed: %w", err)
	}

	tflog.Info(ctx, "Zone deleted successfully", map[string]any{
		"audit_id": deleteResp.Data.Audit.AuditID,
		"zone_id":  networkZoneID,
	})

	return auditLog, nil
}

// NetworkZoneListItem represents a single network zone returned by the list action.
type NetworkZoneListItem struct {
	ID            int64   `json:"id"`
	Name          string  `json:"name"`
	EnvironmentID float64 `json:"environment_id"`
	FirewallID    float64 `json:"firewall_id"`
	NoOfIPs       float64 `json:"no_of_ips"`
	Purpose       string  `json:"purpose"`
	ZoneType      string  `json:"zone_type"`
	NoOfV6IPs     float64 `json:"no_of_v6_ips"`
	IPv6CIDR      string  `json:"ipv6_cidr"`
}

// ListNetworkZones retrieves all network zones for a given environment
// via the action-state API with module=zone&action=list.
func ListNetworkZones(c *client.Client, ctx context.Context, environmentID int64) ([]NetworkZoneListItem, *common.ActionStateResponse, error) {
	tflog.Debug(ctx, "Listing network zones", map[string]any{
		"environment_id": environmentID,
	})

	body := map[string]any{
		"resourceId": fmt.Sprintf("%d", environmentID),
	}

	resp, err := common.UpdateActionState(ctx, c, "zone", "list", body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list network zones: %w", err)
	}

	if len(resp.Data) == 0 {
		return []NetworkZoneListItem{}, resp, nil
	}

	var items []NetworkZoneListItem
	if err := json.Unmarshal(resp.Data, &items); err != nil {
		return nil, resp, fmt.Errorf("failed to parse network zone list response: %w", err)
	}

	tflog.Info(ctx, "Network zones listed successfully", map[string]any{
		"count":          len(items),
		"environment_id": environmentID,
	})

	return items, resp, nil
}

func readNetworkZone(ctx context.Context, c *client.Client, networkZoneID int64) (map[string]interface{}, error) {
	actionStateBody := map[string]any{
		"resourceId": fmt.Sprintf("%d", networkZoneID),
	}

	actionStateResponse, err := common.UpdateActionState(ctx, c, "zone", "read", actionStateBody)
	if err != nil {
		return nil, fmt.Errorf("failed to read network zone %d: %w", networkZoneID, err)
	}

	if len(actionStateResponse.Data) == 0 {
		return nil, fmt.Errorf("network zone ID %d is not available", networkZoneID)
	}

	var responseMap map[string]interface{}
	if err := json.Unmarshal(actionStateResponse.Data, &responseMap); err != nil {
		return nil, fmt.Errorf("network zone ID %d is not available", networkZoneID)
	}

	if len(responseMap) == 0 {
		return nil, fmt.Errorf("network zone ID %d is not available", networkZoneID)
	}

	return responseMap, nil
}

func networkZoneFirewallID(responseMap map[string]interface{}) (int64, bool) {
	if v, ok := responseMap["firewall_id"].(float64); ok {
		return int64(v), true
	}
	return 0, false
}

func ValidateNetworkZoneExists(c *client.Client, ctx context.Context, networkZoneID int64) error {
	tflog.Debug(ctx, "Validating network zone exists", map[string]any{
		"network_zone_id": networkZoneID,
	})

	if _, err := readNetworkZone(ctx, c, networkZoneID); err != nil {
		return err
	}

	tflog.Debug(ctx, "Network Zone ID validated successfully", map[string]any{
		"network_zone_id": networkZoneID,
	})

	return nil
}

// ValidateNetworkZoneOnFirewall ensures the zone exists and belongs to the given firewall.
func ValidateNetworkZoneOnFirewall(c *client.Client, ctx context.Context, networkZoneID, firewallID int64) error {
	if networkZoneID <= 0 {
		return fmt.Errorf("zone_id must be a positive integer")
	}
	if firewallID <= 0 {
		return fmt.Errorf("firewall_id must be a positive integer")
	}

	responseMap, err := readNetworkZone(ctx, c, networkZoneID)
	if err != nil {
		return err
	}

	zoneFirewallID, ok := networkZoneFirewallID(responseMap)
	if !ok {
		return fmt.Errorf("network zone ID %d has no firewall_id in platform response", networkZoneID)
	}
	if zoneFirewallID != firewallID {
		return fmt.Errorf(
			"zone_id %d belongs to firewall %d, not firewall %d",
			networkZoneID,
			zoneFirewallID,
			firewallID,
		)
	}

	tflog.Debug(ctx, "Network zone validated on firewall", map[string]any{
		"network_zone_id": networkZoneID,
		"firewall_id":     firewallID,
	})

	return nil
}
