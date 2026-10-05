// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package resource_group_business_unit

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

// ResourceGroupBusinessUnitCreateRequest contains the request body for creating a business unit component.
type ResourceGroupBusinessUnitCreateRequest struct {
	FirewallID   int64    `json:"firewallId"`
	BusinessUnit string   `json:"businessUnit"`
	Users        []string `json:"users"`
}

// CreateResourceGroupBusinessUnit initiates resource group business unit creation and returns the audit ID.
//
// POST {APIControlPath}/securityservice/business_unit_component
// Returns audit ID for tracking the async operation.
func CreateResourceGroupBusinessUnit(c *client.Client, ctx context.Context, req *ResourceGroupBusinessUnitCreateRequest) (*client.AuditResponse, error) {
	tflog.Debug(ctx, "Creating resource group business unit", map[string]any{
		"firewall_id":   req.FirewallID,
		"business_unit": req.BusinessUnit,
		"users":         req.Users,
	})

	respBody, err := c.DoRequest(ctx, http.MethodPost, common.SecurityServicePath+"/business_unit_component", req)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource group business unit: %w", err)
	}

	tflog.Debug(ctx, "Create resource group business unit response", map[string]any{
		"response": string(respBody),
	})

	var result client.AuditResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse create resource group business unit response: %w", err)
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("resource group business unit creation failed: %s (code: %d)", result.Message, result.ResponseCode)
	}

	tflog.Info(ctx, "Resource group business unit creation initiated", map[string]any{
		"audit_id": result.Data.Audit.AuditID,
		"status":   result.Status,
	})

	return &result, nil
}

// CreateResourceGroupBusinessUnitAndWait creates a business unit and waits for the operation to complete.
// This is a convenience method that combines CreateResourceGroupBusinessUnit and WaitForAuditCompletion.
func CreateResourceGroupBusinessUnitAndWait(c *client.Client, ctx context.Context, req *ResourceGroupBusinessUnitCreateRequest) (*client.AuditLogResponse, error) {
	tflog.Info(ctx, "Creating resource group business unit and waiting for completion", map[string]any{
		"business_unit": req.BusinessUnit,
		"firewall_id":   req.FirewallID,
	})

	// Initiate business unit creation
	createResp, err := CreateResourceGroupBusinessUnit(c, ctx, req)
	if err != nil {
		return nil, err
	}

	// Create an empty requestBody object for future extensibility, currently unused.
	requestBody := map[string]any{
		"resourceType": "BU",
	}

	// Wait for the audit to complete
	auditLog, err := c.WaitForAuditCompletion(ctx, createResp.Data.Audit.AuditID, "create", "engagementComponents", requestBody)
	if err != nil {
		return nil, fmt.Errorf("business unit creation failed: %w", err)
	}

	tflog.Info(ctx, "Business unit created successfully", map[string]any{
		"audit_id":      createResp.Data.Audit.AuditID,
		"business_unit": req.BusinessUnit,
	})

	return auditLog, nil
}

// UpdateResourceGroupBusinessUnit initiates resource group business unit update and returns the audit ID.
//
// PUT {APIControlPath}/securityservice/business_unit_component/{businessUnitId}
// Returns audit ID for tracking the async operation.
func UpdateResourceGroupBusinessUnit(c *client.Client, ctx context.Context, businessUnitID string, req *ResourceGroupBusinessUnitCreateRequest) (*client.AuditResponse, error) {
	tflog.Debug(ctx, "Updating resource group business unit", map[string]any{
		"business_unit_id": businessUnitID,
		"business_unit":    req.BusinessUnit,
	})

	path := fmt.Sprintf(common.SecurityServicePath+"/business_unit_component/%s", businessUnitID)
	respBody, err := c.DoRequest(ctx, http.MethodPut, path, req)
	if err != nil {
		return nil, fmt.Errorf("failed to update resource group business unit: %w", err)
	}

	tflog.Debug(ctx, "Update resource group business unit response", map[string]any{
		"response": string(respBody),
	})

	var result client.AuditResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse update resource group business unit response: %w", err)
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("resource group business unit update failed: %s (code: %d)", result.Message, result.ResponseCode)
	}

	tflog.Info(ctx, "Resource group business unit update initiated", map[string]any{
		"audit_id":         result.Data.Audit.AuditID,
		"status":           result.Status,
		"business_unit_id": businessUnitID,
	})

	return &result, nil
}

// UpdateResourceGroupBusinessUnitAndWait updates a resource group business unit and waits for the operation to complete.
// This is a convenience method that combines UpdateResourceGroupBusinessUnit and WaitForAuditCompletion.
func UpdateResourceGroupBusinessUnitAndWait(c *client.Client, ctx context.Context, businessUnitID string, req *ResourceGroupBusinessUnitCreateRequest) (*client.AuditLogResponse, error) {
	tflog.Info(ctx, "Updating resource group business unit and waiting for completion", map[string]any{
		"business_unit_id": businessUnitID,
		"business_unit":    req.BusinessUnit,
	})

	// Initiate business unit update
	updateResp, err := UpdateResourceGroupBusinessUnit(c, ctx, businessUnitID, req)
	if err != nil {
		return nil, err
	}

	// Pass the update request body for the action-state API
	requestBody := map[string]any{
		"businessUnit": req.BusinessUnit,
		"resourceType": "BU",
	}

	// Wait for the audit to complete
	auditLog, err := c.WaitForAuditCompletion(ctx, updateResp.Data.Audit.AuditID, "update", "engagementComponents", requestBody)
	if err != nil {
		return nil, fmt.Errorf("business unit update failed: %w", err)
	}

	tflog.Info(ctx, "Business unit updated successfully", map[string]any{
		"audit_id":         updateResp.Data.Audit.AuditID,
		"business_unit_id": businessUnitID,
	})

	return auditLog, nil
}

// DeleteResourceGroupBusinessUnit initiates resource group business unit deletion and returns the audit ID.
//
// DELETE {APIControlPath}/securityservice/business_unit_component/{businessUnitId}
// Returns audit ID for tracking the async operation.
func DeleteResourceGroupBusinessUnit(c *client.Client, ctx context.Context, businessUnitID string) (*client.AuditResponse, error) {
	tflog.Debug(ctx, "Deleting resource group business unit", map[string]any{
		"business_unit_id": businessUnitID,
	})

	path := fmt.Sprintf(common.SecurityServicePath+"/business_unit_component/%s", businessUnitID)
	respBody, err := c.DoRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to delete resource group business unit: %w", err)
	}

	tflog.Debug(ctx, "Delete resource group business unit response", map[string]any{
		"response": string(respBody),
	})

	var result client.AuditResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse delete resource group business unit response: %w", err)
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("resource group business unit deletion failed: %s (code: %d)", result.Message, result.ResponseCode)
	}

	tflog.Info(ctx, "Resource group business unit deletion initiated", map[string]any{
		"audit_id":         result.Data.Audit.AuditID,
		"status":           result.Status,
		"business_unit_id": businessUnitID,
	})

	return &result, nil
}

// DeleteResourceGroupBusinessUnitAndWait deletes a resource group business unit and waits for the operation to complete.
// This is a convenience method that combines DeleteResourceGroupBusinessUnit and WaitForAuditCompletion.
func DeleteResourceGroupBusinessUnitAndWait(c *client.Client, ctx context.Context, businessUnitID string) (*client.AuditLogResponse, error) {
	tflog.Info(ctx, "Deleting resource group business unit and waiting for completion", map[string]any{
		"business_unit_id": businessUnitID,
	})

	// Initiate business unit deletion
	deleteResp, err := DeleteResourceGroupBusinessUnit(c, ctx, businessUnitID)
	if err != nil {
		return nil, err
	}

	// Create an empty requestBody object for future extensibility, currently unused.
	requestBody := map[string]any{
		"resourceType": "BU",
	}

	// Wait for the audit to complete
	auditLog, err := c.WaitForAuditCompletion(ctx, deleteResp.Data.Audit.AuditID, "delete", "engagementComponents", requestBody)
	if err != nil {
		return nil, fmt.Errorf("business unit deletion failed: %w", err)
	}

	tflog.Info(ctx, "Business unit deleted successfully", map[string]any{
		"audit_id":         deleteResp.Data.Audit.AuditID,
		"business_unit_id": businessUnitID,
	})

	return auditLog, nil
}

// BusinessUnitListItem represents a single business unit returned by the list action.
type BusinessUnitListItem struct {
	ID           int64       `json:"id"`
	BusinessUnit string      `json:"business_unit"`
	Users        interface{} `json:"users"`
}

// ListBusinessUnits retrieves all business units for a given firewall.
//
// GET {SecurityServicePath}/list-businessunit-state/{firewallId}
// Returns the same response envelope as the legacy action-state list API (module=engagementComponents, action=list).
func ListBusinessUnits(c *client.Client, ctx context.Context, firewallID int64) ([]BusinessUnitListItem, *common.ActionStateResponse, error) {
	tflog.Debug(ctx, "Listing business units", map[string]any{
		"firewall_id": firewallID,
	})

	path := fmt.Sprintf("%s/businessunits-state/%d", common.SecurityServicePath, firewallID)
	respBody, err := c.DoRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list business units: %w", err)
	}

	tflog.Debug(ctx, "List business units response", map[string]any{
		"response": string(respBody),
	})

	var resp common.ActionStateResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, nil, fmt.Errorf("failed to parse business unit list response: %w", err)
	}

	if len(resp.Data) == 0 {
		return []BusinessUnitListItem{}, &resp, nil
	}

	var items []BusinessUnitListItem
	if err := json.Unmarshal(resp.Data, &items); err != nil {
		return nil, &resp, fmt.Errorf("failed to parse business unit list response: %w", err)
	}

	tflog.Info(ctx, "Business units listed successfully", map[string]any{
		"count":       len(items),
		"firewall_id": firewallID,
	})

	return items, &resp, nil
}

// ReadBusinessUnitState retrieves the current business unit state.
//
// GET {SecurityServicePath}/businessunit-state/{businessUnitId}
// Returns the same response envelope as the legacy action-state read API (module=engagementComponents, action=read).
func ReadBusinessUnitState(c *client.Client, ctx context.Context, businessUnitID string) (*common.ActionStateResponse, error) {
	tflog.Debug(ctx, "Reading business unit state", map[string]any{
		"business_unit_id": businessUnitID,
	})

	path := fmt.Sprintf("%s/businessunit-state/%s", common.SecurityServicePath, businessUnitID)
	respBody, err := c.DoRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to read business unit state: %w", err)
	}

	tflog.Debug(ctx, "Business unit state response", map[string]any{
		"response": string(respBody),
	})

	var result common.ActionStateResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse business unit state response: %w", err)
	}

	return &result, nil
}

// getBusinessUnitReadMap reads business unit state used by ValidateBusinessUnitExists.
func getBusinessUnitReadMap(c *client.Client, ctx context.Context, businessUnitID int64) (map[string]interface{}, error) {
	actionStateResponse, err := ReadBusinessUnitState(c, ctx, fmt.Sprintf("%d", businessUnitID))
	if err != nil {
		return nil, fmt.Errorf("failed to validate business unit ID: %w", err)
	}

	if len(actionStateResponse.Data) == 0 {
		return nil, fmt.Errorf("Business Unit ID %d is not available", businessUnitID)
	}

	var responseMap map[string]interface{}
	if err := json.Unmarshal(actionStateResponse.Data, &responseMap); err != nil {
		return nil, fmt.Errorf("Business Unit ID %d is not available", businessUnitID)
	}

	if len(responseMap) == 0 {
		return nil, fmt.Errorf("Business Unit ID %d is not available", businessUnitID)
	}

	return responseMap, nil
}

// firewallIDStringFromBURead returns the firewall id from a business-unit read payload as a decimal string.
// The API returns firewallId as a string; other JSON types are normalized for comparison.
func firewallIDStringFromBURead(m map[string]interface{}) (string, bool) {
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

// ValidateBusinessUnitExists checks whether a business unit with the given ID exists.
// Returns nil if found, or an error if not.
func ValidateBusinessUnitExists(c *client.Client, ctx context.Context, businessUnitID int64) error {
	tflog.Debug(ctx, "Validating business unit exists", map[string]any{
		"business_unit_id": businessUnitID,
	})

	_, err := getBusinessUnitReadMap(c, ctx, businessUnitID)
	if err != nil {
		return err
	}

	tflog.Debug(ctx, "Business unit ID validated successfully", map[string]any{
		"business_unit_id": businessUnitID,
	})

	return nil
}

// ValidateBusinessUnitExistsForFirewall uses the same business unit state API as ValidateBusinessUnitExists,
// then requires a firewall id in the response (string firewallId from API) matching firewallID from config/plan.
func ValidateBusinessUnitExistsForFirewall(c *client.Client, ctx context.Context, businessUnitID, firewallID int64) error {
	tflog.Debug(ctx, "Validating business unit exists for firewall", map[string]any{
		"business_unit_id": businessUnitID,
		"firewall_id":      firewallID,
	})

	responseMap, err := getBusinessUnitReadMap(c, ctx, businessUnitID)
	if err != nil {
		return err
	}

	respFW, ok := firewallIDStringFromBURead(responseMap)
	if !ok {
		return fmt.Errorf("business unit %d: API response does not include a firewall id", businessUnitID)
	}

	planFW := strconv.FormatInt(firewallID, 10)
	if respFW != planFW {
		return fmt.Errorf(
			"business unit %d is bound to firewall %s, which does not match configured firewall_id %s",
			businessUnitID, respFW, planFW,
		)
	}

	tflog.Debug(ctx, "Business unit ID validated for firewall", map[string]any{
		"business_unit_id": businessUnitID,
		"firewall_id":      firewallID,
	})

	return nil
}
