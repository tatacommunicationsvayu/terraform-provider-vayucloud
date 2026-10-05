// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package resource_group_environment

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/client"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/common"
)

// ResourceGroupEnvironmentCreateRequest contains the request body for creating an environment component.
type ResourceGroupEnvironmentCreateRequest struct {
	FirewallID     int64  `json:"firewallId"`
	Environment    string `json:"environment"`
	BusinessUnitID int64  `json:"businessUnitId"`
}

// CreateResourceGroupEnvironment initiates environment creation and returns the audit ID.
func CreateResourceGroupEnvironment(c *client.Client, ctx context.Context, req *ResourceGroupEnvironmentCreateRequest) (*client.AuditResponse, error) {
	tflog.Debug(ctx, "Creating resource group environment", map[string]any{
		"firewall_id":      req.FirewallID,
		"environment":      req.Environment,
		"business_unit_id": req.BusinessUnitID,
	})

	respBody, err := c.DoRequest(ctx, http.MethodPost, common.SecurityServicePath+"/environment_component", req)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource group environment: %w", err)
	}

	tflog.Debug(ctx, "Create resource group environment response", map[string]any{
		"response": string(respBody),
	})

	var result client.AuditResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse create resource group environment response: %w", err)
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("resource group environment creation failed: %s (code: %d)", result.Message, result.ResponseCode)
	}

	tflog.Info(ctx, "Resource group environment creation initiated", map[string]any{
		"audit_id": result.Data.Audit.AuditID,
		"status":   result.Status,
	})

	return &result, nil
}

// CreateResourceGroupEnvironmentAndWait creates an environment and waits for the operation to complete.
func CreateResourceGroupEnvironmentAndWait(c *client.Client, ctx context.Context, req *ResourceGroupEnvironmentCreateRequest) (*client.AuditLogResponse, error) {
	tflog.Info(ctx, "Creating resource group environment and waiting for completion", map[string]any{
		"firewall_id": req.FirewallID,
		"environment": req.Environment,
	})

	createResp, err := CreateResourceGroupEnvironment(c, ctx, req)
	if err != nil {
		return nil, err
	}

	requestBody := map[string]any{
		"resourceType": "ENV",
	}

	auditLog, err := c.WaitForAuditCompletion(ctx, createResp.Data.Audit.AuditID, "create", "engagementComponents", requestBody)
	if err != nil {
		return nil, fmt.Errorf("resource group environment creation failed: %w", err)
	}

	tflog.Info(ctx, "Environment created successfully", map[string]any{
		"audit_id":    createResp.Data.Audit.AuditID,
		"environment": req.Environment,
	})

	return auditLog, nil
}

// UpdateResourceGroupEnvironment initiates environment update and returns the audit ID.
func UpdateResourceGroupEnvironment(c *client.Client, ctx context.Context, environmentID string, req *ResourceGroupEnvironmentCreateRequest) (*client.AuditResponse, error) {
	tflog.Debug(ctx, "Updating resource group environment", map[string]any{
		"environment_id":   environmentID,
		"environment":      req.Environment,
		"business_unit_id": req.BusinessUnitID,
	})

	path := fmt.Sprintf(common.SecurityServicePath+"/environment_component/%s", environmentID)
	respBody, err := c.DoRequest(ctx, http.MethodPut, path, req)
	if err != nil {
		return nil, fmt.Errorf("failed to update resource group environment: %w", err)
	}

	tflog.Debug(ctx, "Update resource group environment response", map[string]any{
		"response": string(respBody),
	})

	var result client.AuditResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse update resource group environment response: %w", err)
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("resource group environment update failed: %s (code: %d)", result.Message, result.ResponseCode)
	}

	tflog.Info(ctx, "Resource group environment update initiated", map[string]any{
		"audit_id":       result.Data.Audit.AuditID,
		"status":         result.Status,
		"environment_id": environmentID,
	})

	return &result, nil
}

// UpdateResourceGroupEnvironmentAndWait updates an environment and waits for the operation to complete.
func UpdateResourceGroupEnvironmentAndWait(c *client.Client, ctx context.Context, environmentID string, req *ResourceGroupEnvironmentCreateRequest) (*client.AuditLogResponse, error) {
	tflog.Info(ctx, "Updating resource group environment and waiting for completion", map[string]any{
		"environment_id": environmentID,
		"environment":    req.Environment,
	})

	updateResp, err := UpdateResourceGroupEnvironment(c, ctx, environmentID, req)
	if err != nil {
		return nil, err
	}

	requestBody := map[string]any{
		"environment":    req.Environment,
		"businessUnitId": req.BusinessUnitID,
		"resourceType":   "ENV",
	}

	auditLog, err := c.WaitForAuditCompletion(ctx, updateResp.Data.Audit.AuditID, "update", "engagementComponents", requestBody)
	if err != nil {
		return nil, fmt.Errorf("resource group environment update failed: %w", err)
	}

	tflog.Info(ctx, "Resource group environment updated successfully", map[string]any{
		"audit_id":       updateResp.Data.Audit.AuditID,
		"environment_id": environmentID,
	})

	return auditLog, nil
}

// DeleteResourceGroupEnvironment initiates environment deletion and returns the audit ID.
func DeleteResourceGroupEnvironment(c *client.Client, ctx context.Context, environmentID string) (*client.AuditResponse, error) {
	tflog.Debug(ctx, "Deleting resource group environment", map[string]any{
		"environment_id": environmentID,
	})

	path := fmt.Sprintf(common.SecurityServicePath+"/environment_component/%s", environmentID)
	respBody, err := c.DoRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to delete resource group environment: %w", err)
	}

	tflog.Debug(ctx, "Delete resource group environment response", map[string]any{
		"response": string(respBody),
	})

	var result client.AuditResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse delete resource group environment response: %w", err)
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("resource group environment deletion failed: %s (code: %d)", result.Message, result.ResponseCode)
	}

	tflog.Info(ctx, "Resource group environment deletion initiated", map[string]any{
		"audit_id":       result.Data.Audit.AuditID,
		"status":         result.Status,
		"environment_id": environmentID,
	})

	return &result, nil
}

// DeleteResourceGroupEnvironmentAndWait deletes an environment and waits for the operation to complete.
func DeleteResourceGroupEnvironmentAndWait(c *client.Client, ctx context.Context, environmentID string) (*client.AuditLogResponse, error) {
	tflog.Info(ctx, "Deleting resource group environment and waiting for completion", map[string]any{
		"environment_id": environmentID,
	})

	deleteResp, err := DeleteResourceGroupEnvironment(c, ctx, environmentID)
	if err != nil {
		return nil, err
	}

	requestBody := map[string]any{
		"resourceType": "ENV",
	}

	auditLog, err := c.WaitForAuditCompletion(ctx, deleteResp.Data.Audit.AuditID, "delete", "engagementComponents", requestBody)
	if err != nil {
		return nil, fmt.Errorf("resource group environment deletion failed: %w", err)
	}

	tflog.Info(ctx, "Resource group environment deleted successfully", map[string]any{
		"audit_id":       deleteResp.Data.Audit.AuditID,
		"environment_id": environmentID,
	})

	return auditLog, nil
}

// EnvironmentListItem represents a single environment returned by the list action.
type EnvironmentListItem struct {
	ID             int64   `json:"id"`
	Environment    string  `json:"environment"`
	BusinessUnitID float64 `json:"business_unit_id"`
	Status         string  `json:"status"`
}

// ListEnvironments retrieves all environments for a given business unit.
//
// GET {SecurityServicePath}/list-environment-state/{businessUnitId}
// Returns the same response envelope as the legacy action-state list API (module=engagementComponents, action=list).
func ListEnvironments(c *client.Client, ctx context.Context, businessUnitID int64) ([]EnvironmentListItem, *common.ActionStateResponse, error) {
	tflog.Debug(ctx, "Listing environments", map[string]any{
		"business_unit_id": businessUnitID,
	})

	path := fmt.Sprintf("%s/environments-state/%d", common.SecurityServicePath, businessUnitID)
	respBody, err := c.DoRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list environments: %w", err)
	}

	tflog.Debug(ctx, "List environments response", map[string]any{
		"response": string(respBody),
	})

	var resp common.ActionStateResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, nil, fmt.Errorf("failed to parse environment list response: %w", err)
	}

	if len(resp.Data) == 0 {
		return []EnvironmentListItem{}, &resp, nil
	}

	var items []EnvironmentListItem
	if err := json.Unmarshal(resp.Data, &items); err != nil {
		return nil, &resp, fmt.Errorf("failed to parse environment list response: %w", err)
	}

	tflog.Info(ctx, "Environments listed successfully", map[string]any{
		"count":            len(items),
		"business_unit_id": businessUnitID,
	})

	return items, &resp, nil
}

// ReadEnvironmentState retrieves the current environment state.
//
// GET {SecurityServicePath}/environment-state/{environmentId}
// Returns the same response envelope as the legacy action-state read API (module=engagementComponents, action=read).
func ReadEnvironmentState(c *client.Client, ctx context.Context, environmentID string) (*common.ActionStateResponse, error) {
	tflog.Debug(ctx, "Reading environment state", map[string]any{
		"environment_id": environmentID,
	})

	path := fmt.Sprintf("%s/environment-state/%s", common.SecurityServicePath, environmentID)
	respBody, err := c.DoRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to read environment state: %w", err)
	}

	tflog.Debug(ctx, "Environment state response", map[string]any{
		"response": string(respBody),
	})

	var result common.ActionStateResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse environment state response: %w", err)
	}

	return &result, nil
}

// getEnvironmentReadMap reads environment state used by ValidateEnvironmentExists.
func getEnvironmentReadMap(c *client.Client, ctx context.Context, environmentID int64) (map[string]interface{}, error) {
	actionStateResponse, err := ReadEnvironmentState(c, ctx, fmt.Sprintf("%d", environmentID))
	if err != nil {
		return nil, fmt.Errorf("failed to validate environment ID: %w", err)
	}

	if len(actionStateResponse.Data) == 0 {
		return nil, fmt.Errorf("Environment ID %d is not available", environmentID)
	}

	var responseMap map[string]interface{}
	if err := json.Unmarshal(actionStateResponse.Data, &responseMap); err != nil {
		return nil, fmt.Errorf("Environment ID %d is not available", environmentID)
	}

	if len(responseMap) == 0 {
		return nil, fmt.Errorf("Environment ID %d is not available", environmentID)
	}

	return responseMap, nil
}

// firewallIDStringFromResponse returns the firewall id from the read payload as a decimal string.
// The API returns firewallId as a string; other JSON types are normalized for comparison.
func firewallIDStringFromResponse(m map[string]interface{}) (string, bool) {
	for _, key := range []string{"firewallId", "firewall_id"} {
		v, ok := m[key]
		if !ok {
			continue
		}
		switch x := v.(type) {
		case string:
			s := strings.TrimSpace(x)
			if s == "" {
				return "", false
			}
			return s, true
		case float64:
			return strconv.FormatInt(int64(x), 10), true
		case int64:
			return strconv.FormatInt(x, 10), true
		case int:
			return strconv.FormatInt(int64(x), 10), true
		case json.Number:
			return strings.TrimSpace(string(x)), true
		default:
			return fmt.Sprint(x), true
		}
	}
	return "", false
}

// ValidateEnvironmentExists checks whether an environment with the given ID exists.
func ValidateEnvironmentExists(c *client.Client, ctx context.Context, environmentID int64) error {
	tflog.Debug(ctx, "Validating environment exists", map[string]any{
		"environment_id": environmentID,
	})

	_, err := getEnvironmentReadMap(c, ctx, environmentID)
	if err != nil {
		return err
	}

	tflog.Debug(ctx, "Environment ID validated successfully", map[string]any{
		"environment_id": environmentID,
	})

	return nil
}

// ValidateEnvironmentExistsForFirewall uses the same environment state API as ValidateEnvironmentExists,
// then requires a firewall id in the response (string firewallId from API) matching firewallID from config/plan.
func ValidateEnvironmentExistsForFirewall(c *client.Client, ctx context.Context, environmentID, firewallID int64) error {
	tflog.Debug(ctx, "Validating environment exists for firewall", map[string]any{
		"environment_id": environmentID,
		"firewall_id":      firewallID,
	})

	responseMap, err := getEnvironmentReadMap(c, ctx, environmentID)
	if err != nil {
		return err
	}

	respFW, ok := firewallIDStringFromResponse(responseMap)
	if !ok {
		return fmt.Errorf("environment %d: API response does not include a firewall id", environmentID)
	}

	planFW := strconv.FormatInt(firewallID, 10)
	if respFW != planFW {
		return fmt.Errorf(
			"environment %d is bound to firewall %s, which does not match configured firewall_id %s",
			environmentID, respFW, planFW,
		)
	}

	tflog.Debug(ctx, "Environment ID validated for firewall", map[string]any{
		"environment_id": environmentID,
		"firewall_id":      firewallID,
	})

	return nil
}
