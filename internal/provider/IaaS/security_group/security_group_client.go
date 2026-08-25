// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package security_group

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

// SecurityGroupCreateRequest is the body for POST /security-group/{firewallId}.
type SecurityGroupCreateRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// SecurityGroupRuleCreateRequest is the body for POST /security-group/rules/{firewallId}/{sgId}.
type SecurityGroupRuleCreateRequest struct {
	Protocol             string   `json:"protocol"`
	Direction            string   `json:"direction"`
	EtherType            string   `json:"ether_type"`
	PortType             string   `json:"port_type"`
	RemoteType           string   `json:"remote_type,omitempty"`
	PortRangeList        []string `json:"port_range_list,omitempty"`
	RemoteIPPrefixList   []string `json:"remote_ip_prefix_list,omitempty"`
	RemoteSecurityGroup  string   `json:"remote_security_group,omitempty"`
}

// CatalystResponse is a generic status envelope used by several security-group APIs.
type CatalystResponse struct {
	Status       string          `json:"status"`
	Data         json.RawMessage `json:"data"`
	Message      string          `json:"message"`
	ResponseCode int             `json:"responseCode"`
}

// SecurityGroupListItem is one entry from GET /security-group/{firewallId}.
type SecurityGroupListItem struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Description    string `json:"description"`
	TenantID       string `json:"tenant_id"`
	ProjectID      string `json:"project_id"`
	Shared         bool   `json:"shared"`
	Stateful       bool   `json:"stateful"`
	RevisionNumber int64  `json:"revision_number"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
}

// InstanceSecurityGroupEntry is one port+SG row from GET /security-group/instance/{instanceId}.
type InstanceSecurityGroupEntry struct {
	PortName          string                     `json:"port_name"`
	PortIP            string                     `json:"port_ip"`
	PortID            string                     `json:"port_id"`
	SecurityGroupName string                     `json:"security_group_name"`
	SecurityGroupID   string                     `json:"security_group_id"`
	Rules             []InstanceSecurityGroupRule `json:"rules"`
}

// InstanceSecurityGroupRule is a rule nested under an instance security-group entry.
type InstanceSecurityGroupRule struct {
	RuleID               string  `json:"rule_id"`
	SecurityGroupID      string  `json:"security_group_id"`
	Protocol             *string `json:"protocol"`
	EtherType            string  `json:"ether_type"`
	Direction            string  `json:"direction"`
	PortRangeMin         *int64  `json:"port_range_min"`
	PortRangeMax         *int64  `json:"port_range_max"`
	RemoteIPPrefix       *string `json:"remote_ip_prefix"`
	RemoteGroupID        *string `json:"remote_group_id"`
	RemoteAddressGroupID *string `json:"remote_address_group_id"`
	NormalizedCIDR       *string `json:"normalized_cidr"`
	Description          string  `json:"description"`
	CreatedAt            string  `json:"created_at"`
	UpdatedAt            string  `json:"updated_at"`
	TenantID             string  `json:"tenant_id"`
	ProjectID            string  `json:"project_id"`
}

// ListSecurityGroupsWithRulesRequest is the body for POST /security-group/listsgwithrules/{firewallId}.
type ListSecurityGroupsWithRulesRequest struct {
	IDs []string `json:"ids"`
}

// SecurityGroupWithRules is one security group with its rules from listsgwithrules.
type SecurityGroupWithRules struct {
	ID          string              `json:"id"`
	Name        string              `json:"name"`
	Description string              `json:"description"`
	Rules       []SecurityGroupRule `json:"rules"`
}

// SecurityGroupRule is one rule from listsgwithrules.
type SecurityGroupRule struct {
	ID               string  `json:"id"`
	SecurityGroupID  string  `json:"security_group_id"`
	Protocol         string  `json:"protocol"`
	EtherType        string  `json:"ether_type"`
	Direction        string  `json:"direction"`
	PortRangeMin     *int64  `json:"port_range_min"`
	PortRangeMax     *int64  `json:"port_range_max"`
	RemoteIPPrefix   *string `json:"remote_ip_prefix"`
	RemoteGroupID    *string `json:"remote_group_id"`
	Description      string  `json:"description"`
}

// ListSecurityGroupsWithRules fetches security groups and their rules by IDs.
//
// POST {SecurityGroupServicePath}/listsgwithrules/{firewallId}
func ListSecurityGroupsWithRules(c *client.Client, ctx context.Context, firewallID int64, sgIDs []string) ([]SecurityGroupWithRules, error) {
	tflog.Debug(ctx, "Listing security groups with rules", map[string]any{
		"firewall_id": firewallID,
		"sg_ids":      sgIDs,
	})

	path := fmt.Sprintf("%s/listsgwithrules/%d", common.SecurityGroupServicePath, firewallID)
	req := &ListSecurityGroupsWithRulesRequest{IDs: sgIDs}
	respBody, err := c.DoRequest(ctx, http.MethodPost, path, req)
	if err != nil {
		return nil, fmt.Errorf("failed to list security groups with rules: %w", err)
	}

	var result CatalystResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse listsgwithrules response: %w", err)
	}
	if !strings.EqualFold(result.Status, "success") {
		return nil, fmt.Errorf("listsgwithrules failed: %s (code: %d)", result.Message, result.ResponseCode)
	}

	var items []SecurityGroupWithRules
	if len(result.Data) > 0 && string(result.Data) != "null" {
		if err := json.Unmarshal(result.Data, &items); err != nil {
			return nil, fmt.Errorf("failed to parse listsgwithrules data: %w", err)
		}
	}
	return items, nil
}

// GetSecurityGroupWithRules fetches one security group with rules by ID.
func GetSecurityGroupWithRules(c *client.Client, ctx context.Context, firewallID int64, sgID string) (*SecurityGroupWithRules, error) {
	items, err := ListSecurityGroupsWithRules(c, ctx, firewallID, []string{sgID})
	if err != nil {
		return nil, err
	}
	for i := range items {
		item := &items[i]
		if item.ID == sgID {
			return item, nil
		}
		// Some responses omit top-level id; match via rule security_group_id.
		for _, rule := range item.Rules {
			if rule.SecurityGroupID == sgID {
				if item.ID == "" {
					item.ID = sgID
				}
				return item, nil
			}
		}
	}
	// Single requested ID and exactly one result — treat as match.
	if len(items) == 1 {
		if items[0].ID == "" {
			items[0].ID = sgID
		}
		return &items[0], nil
	}
	return nil, fmt.Errorf("security group %s not found under firewall %d", sgID, firewallID)
}

// CreateSecurityGroup creates a security group under a firewall and returns the async audit response.
func CreateSecurityGroup(c *client.Client, ctx context.Context, firewallID int64, req *SecurityGroupCreateRequest) (*client.AuditResponse, error) {
	tflog.Debug(ctx, "Creating security group", map[string]any{
		"firewall_id": firewallID,
		"name":        req.Name,
	})

	path := fmt.Sprintf("%s/%d", common.SecurityGroupServicePath, firewallID)
	respBody, err := c.DoRequest(ctx, http.MethodPost, path, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create security group: %w", err)
	}

	var result client.AuditResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse create security group response: %w", err)
	}
	if result.Status != "success" {
		return nil, fmt.Errorf("security group creation failed: %s (code: %d)", result.Message, result.ResponseCode)
	}
	return &result, nil
}

// CreateSecurityGroupAndWait creates a security group and waits for audit completion (no action-state).
func CreateSecurityGroupAndWait(c *client.Client, ctx context.Context, firewallID int64, req *SecurityGroupCreateRequest) (*client.AuditLogResponse, error) {
	resp, err := CreateSecurityGroup(c, ctx, firewallID, req)
	if err != nil {
		return nil, err
	}
	return c.WaitForAuditCompletionNoActionState(ctx, resp.Data.Audit.AuditID)
}

// CreateSecurityGroupRule creates a rule and returns the async audit response.
func CreateSecurityGroupRule(c *client.Client, ctx context.Context, firewallID int64, sgID string, req *SecurityGroupRuleCreateRequest) (*client.AuditResponse, error) {
	tflog.Debug(ctx, "Creating security group rule", map[string]any{
		"firewall_id": firewallID,
		"sg_id":       sgID,
		"protocol":    req.Protocol,
		"direction":   req.Direction,
	})

	path := fmt.Sprintf("%s/rules/%d/%s", common.SecurityGroupServicePath, firewallID, sgID)
	respBody, err := c.DoRequest(ctx, http.MethodPost, path, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create security group rule: %w", err)
	}

	var result client.AuditResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse create security group rule response: %w", err)
	}
	if result.Status != "success" {
		return nil, fmt.Errorf("security group rule creation failed: %s (code: %d)", result.Message, result.ResponseCode)
	}
	return &result, nil
}

// CreateSecurityGroupRuleAndWait creates a rule and waits for audit completion (no action-state).
func CreateSecurityGroupRuleAndWait(c *client.Client, ctx context.Context, firewallID int64, sgID string, req *SecurityGroupRuleCreateRequest) (*client.AuditLogResponse, error) {
	resp, err := CreateSecurityGroupRule(c, ctx, firewallID, sgID, req)
	if err != nil {
		return nil, err
	}
	return c.WaitForAuditCompletionNoActionState(ctx, resp.Data.Audit.AuditID)
}

// ValidateSecurityGroupExists checks GET /security-group/rules/{firewallId}/{sgId} for Catalyst success.
func ValidateSecurityGroupExists(c *client.Client, ctx context.Context, firewallID int64, sgID string) error {
	tflog.Debug(ctx, "Validating security group exists", map[string]any{
		"firewall_id": firewallID,
		"sg_id":       sgID,
	})

	path := fmt.Sprintf("%s/rules/%d/%s", common.SecurityGroupServicePath, firewallID, sgID)
	respBody, err := c.DoRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return fmt.Errorf("Security group ID %s is not available under firewall %d: %w", sgID, firewallID, err)
	}

	var result CatalystResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return fmt.Errorf("failed to parse security group validation response: %w", err)
	}
	if !strings.EqualFold(result.Status, "success") {
		return fmt.Errorf("Security group ID %s is not available under firewall %d: %s", sgID, firewallID, result.Message)
	}
	return nil
}

// ListSecurityGroups lists security groups for a firewall.
func ListSecurityGroups(c *client.Client, ctx context.Context, firewallID int64) ([]SecurityGroupListItem, error) {
	tflog.Debug(ctx, "Listing security groups", map[string]any{
		"firewall_id": firewallID,
	})

	path := fmt.Sprintf("%s/%d", common.SecurityGroupServicePath, firewallID)
	respBody, err := c.DoRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list security groups: %w", err)
	}

	var result CatalystResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse list security groups response: %w", err)
	}
	if !strings.EqualFold(result.Status, "success") {
		return nil, fmt.Errorf("list security groups failed: %s (code: %d)", result.Message, result.ResponseCode)
	}

	var items []SecurityGroupListItem
	if len(result.Data) > 0 && string(result.Data) != "null" {
		if err := json.Unmarshal(result.Data, &items); err != nil {
			return nil, fmt.Errorf("failed to parse security group list data: %w", err)
		}
	}
	return items, nil
}

// ListSecurityGroupRules lists rules for a security group under a firewall.
//
// GET {SecurityGroupServicePath}/rules/{firewallId}/{sgId}
func ListSecurityGroupRules(c *client.Client, ctx context.Context, firewallID int64, sgID string) ([]SecurityGroupRule, error) {
	tflog.Debug(ctx, "Listing security group rules", map[string]any{
		"firewall_id": firewallID,
		"sg_id":       sgID,
	})

	path := fmt.Sprintf("%s/rules/%d/%s", common.SecurityGroupServicePath, firewallID, sgID)
	respBody, err := c.DoRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list security group rules: %w", err)
	}

	var result CatalystResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse list security group rules response: %w", err)
	}
	if !strings.EqualFold(result.Status, "success") {
		return nil, fmt.Errorf("list security group rules failed: %s (code: %d)", result.Message, result.ResponseCode)
	}

	var items []SecurityGroupRule
	if len(result.Data) > 0 && string(result.Data) != "null" {
		if err := json.Unmarshal(result.Data, &items); err != nil {
			return nil, fmt.Errorf("failed to parse security group rules data: %w", err)
		}
	}
	return items, nil
}

// ListInstanceSecurityGroups lists security groups (and rules) attached to a VM instance.
//
// GET {SecurityGroupServicePath}/instance/{instanceId}
func ListInstanceSecurityGroups(c *client.Client, ctx context.Context, instanceID int64) ([]InstanceSecurityGroupEntry, error) {
	tflog.Debug(ctx, "Listing instance security groups", map[string]any{
		"instance_id": instanceID,
	})

	path := fmt.Sprintf("%s/instance/%d", common.SecurityGroupServicePath, instanceID)
	respBody, err := c.DoRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list instance security groups: %w", err)
	}

	var result CatalystResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse instance security groups response: %w", err)
	}
	if !strings.EqualFold(result.Status, "success") {
		return nil, fmt.Errorf("list instance security groups failed: %s (code: %d)", result.Message, result.ResponseCode)
	}

	var items []InstanceSecurityGroupEntry
	if len(result.Data) > 0 && string(result.Data) != "null" {
		if err := json.Unmarshal(result.Data, &items); err != nil {
			return nil, fmt.Errorf("failed to parse instance security groups data: %w", err)
		}
	}
	return items, nil
}

// FindSecurityGroupIDByName lists security groups and returns the id for an exact name match.
func FindSecurityGroupIDByName(c *client.Client, ctx context.Context, firewallID int64, name string) (string, error) {
	items, err := ListSecurityGroups(c, ctx, firewallID)
	if err != nil {
		return "", err
	}
	for _, item := range items {
		if item.Name == name {
			return item.ID, nil
		}
	}
	return "", fmt.Errorf("security group %q does not exist under firewall %d; create the security group first and retry", name, firewallID)
}

// DeleteSecurityGroupRule deletes a single rule and returns the async audit response.
func DeleteSecurityGroupRule(c *client.Client, ctx context.Context, firewallID int64, sgID, ruleID string) (*client.AuditResponse, error) {
	tflog.Debug(ctx, "Deleting security group rule", map[string]any{
		"firewall_id": firewallID,
		"sg_id":       sgID,
		"rule_id":     ruleID,
	})

	path := fmt.Sprintf("%s/rules/%d/%s/%s", common.SecurityGroupServicePath, firewallID, sgID, ruleID)
	respBody, err := c.DoRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to delete security group rule: %w", err)
	}

	var result client.AuditResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse delete security group rule response: %w", err)
	}
	if result.Status != "success" {
		return nil, fmt.Errorf("security group rule deletion failed: %s (code: %d)", result.Message, result.ResponseCode)
	}
	return &result, nil
}

// DeleteSecurityGroupRuleAndWait deletes a rule and waits for audit completion (no action-state).
func DeleteSecurityGroupRuleAndWait(c *client.Client, ctx context.Context, firewallID int64, sgID, ruleID string) (*client.AuditLogResponse, error) {
	resp, err := DeleteSecurityGroupRule(c, ctx, firewallID, sgID, ruleID)
	if err != nil {
		return nil, err
	}
	return c.WaitForAuditCompletionNoActionState(ctx, resp.Data.Audit.AuditID)
}

// DeleteSecurityGroup deletes a security group (and its mapping) and returns the async audit response.
func DeleteSecurityGroup(c *client.Client, ctx context.Context, firewallID int64, sgID string) (*client.AuditResponse, error) {
	tflog.Debug(ctx, "Deleting security group", map[string]any{
		"firewall_id": firewallID,
		"sg_id":       sgID,
	})

	path := fmt.Sprintf("%s/%d/%s", common.SecurityGroupServicePath, firewallID, sgID)
	respBody, err := c.DoRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to delete security group: %w", err)
	}

	var result client.AuditResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse delete security group response: %w", err)
	}
	if result.Status != "success" {
		return nil, fmt.Errorf("security group deletion failed: %s (code: %d)", result.Message, result.ResponseCode)
	}
	return &result, nil
}

// DeleteSecurityGroupAndWait deletes a security group and waits for audit completion (no action-state).
func DeleteSecurityGroupAndWait(c *client.Client, ctx context.Context, firewallID int64, sgID string) (*client.AuditLogResponse, error) {
	resp, err := DeleteSecurityGroup(c, ctx, firewallID, sgID)
	if err != nil {
		return nil, err
	}
	return c.WaitForAuditCompletionNoActionState(ctx, resp.Data.Audit.AuditID)
}

// FrameSecurityGroupName builds the system security-group name from type and resource id.
func FrameSecurityGroupName(resourceType string, resourceID int64) (string, error) {
	switch strings.ToLower(resourceType) {
	case "zone":
		return fmt.Sprintf("SG_Zone_%d", resourceID), nil
	case "virtualmachine":
		return fmt.Sprintf("SG_VM_%d", resourceID), nil
	case "firewall":
		return fmt.Sprintf("SG_TR_%d", resourceID), nil
	default:
		return "", fmt.Errorf("unsupported type %q; must be zone, virtualmachine, or firewall", resourceType)
	}
}
