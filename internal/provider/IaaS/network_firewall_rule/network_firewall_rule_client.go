// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package network_firewall_rule

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/client"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/common"
)

const actionStateModuleFirewallRules = "firewallRule"

var scheduleDateTimeAPIRegex = regexp.MustCompile(`^\d{1,2}:\d{2}\s+(\d{4})/(\d{2})/(\d{2})$`)
var expandedPortServiceRegex = regexp.MustCompile(`^(?i)(tcp|udp)_\d+$`)

// normalizeScheduleDateForState converts API schedule values to Terraform YYYY-MM-DD.
func normalizeScheduleDateForState(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	matched, err := regexp.MatchString(`^\d{4}-\d{2}-\d{2}$`, value)
 	if err != nil {
 		return ""
 	}
 	if matched {
 		return value
 	}
	if m := scheduleDateTimeAPIRegex.FindStringSubmatch(value); len(m) == 4 {
		return fmt.Sprintf("%s-%s-%s", m[1], m[2], m[3])
	}
	return value
}

// normalizeServicesForState removes API-expanded low-level port services for NAS rules.
// The API auto-adds tcp_/udp_ entries when NAS is involved; Terraform config uses named services (e.g. HTTP).
func normalizeServicesForState(services []string, source, destination string) []string {
	if len(services) == 0 {
		return services
	}
	src := strings.ToLower(strings.TrimSpace(source))
	dst := strings.ToLower(strings.TrimSpace(destination))
	if src != "nas" && dst != "nas" {
		return services
	}

	filtered := make([]string, 0, len(services))
	for _, service := range services {
		if expandedPortServiceRegex.MatchString(strings.TrimSpace(service)) {
			continue
		}
		filtered = append(filtered, service)
	}
	if len(filtered) > 0 {
		return filtered
	}
	return services
}

// NetworkFirewallRuleRequest is the request body for validate, create, and update APIs.
type NetworkFirewallRuleRequest struct {
	RuleName             string   `json:"ruleName"`
	FirewallID           int64    `json:"firewallId"`
	Source               string   `json:"source"`
	Action               string   `json:"action"`
	SourceZoneID         *int64   `json:"sourceZoneId,omitempty"`
	SourceAddresses      []string `json:"sourceAddresses"`
	Destination          string   `json:"destination"`
	DestinationZoneID    *int64   `json:"destinationZoneId,omitempty"`
	DestinationAddresses []string `json:"destinationAddresses"`
	ScheduleStartDate    string   `json:"scheduleStartDate,omitempty"`
	ScheduleEndDate      string   `json:"scheduleEndDate,omitempty"`
	Services             []string `json:"services"`
	RuleID               *int64   `json:"ruleid,omitempty"`
}

type firewallRuleValidateResponse struct {
	Status       string `json:"status"`
	Message      string `json:"message"`
	ResponseCode int    `json:"responseCode"`
	Data         struct {
		Failure      []string `json:"failure"`
		Success      []string `json:"success"`
		Warnings     []string `json:"warnings"`
		AutoCreateSG bool     `json:"autoCreateSG"`
	} `json:"data"`
}

// ValidateNetworkFirewallRule calls the network-rules validate API.
func ValidateNetworkFirewallRule(c *client.Client, ctx context.Context, req *NetworkFirewallRuleRequest) ([]string, error) {
	validateReq := *req
	validateReq.RuleID = nil

	tflog.Debug(ctx, "Validating network firewall rule", map[string]any{
		"rule_name":   validateReq.RuleName,
		"firewall_id": validateReq.FirewallID,
		"request":     validateReq,
	})

	path := common.FirewallConfigPath + "/network-rules/validate"
	respBody, err := c.DoRequest(ctx, http.MethodPost, path, &validateReq)
	if err != nil {
		return nil, fmt.Errorf("firewall rule validation request failed: %w", err)
	}

	tflog.Debug(ctx, "Firewall rule validation response", map[string]any{
		"response": string(respBody),
	})

	var result firewallRuleValidateResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse firewall rule validation response: %w", err)
	}

	// if result.Status != "success" {
	// 	return nil, fmt.Errorf("firewall rule validation failed: %s (code: %d)", result.Message, result.ResponseCode)
	// }

	if len(result.Data.Failure) > 0 {
		return nil, fmt.Errorf("firewall rule validation failed: %s", strings.Join(result.Data.Failure, "; "))
	}

	var messages []string
	messages = append(messages, result.Data.Success...)
	messages = append(messages, result.Data.Warnings...)
	return messages, nil
}

// validateNetworkFirewallRuleLenient calls validate but does not block create/update on HTTP 500.
func validateNetworkFirewallRuleLenient(c *client.Client, ctx context.Context, req *NetworkFirewallRuleRequest) ([]string, error) {
	messages, err := ValidateNetworkFirewallRule(c, ctx, req)
	if err == nil {
		return messages, nil
	}
	var apiErr *client.APIError
	if (errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusInternalServerError) ||
		strings.Contains(err.Error(), "HTTP 500") {
		tflog.Warn(ctx, "Firewall rule validate API returned HTTP 500; proceeding with create/update", map[string]any{
			"rule_name":   req.RuleName,
			"firewall_id": req.FirewallID,
			"error":       err.Error(),
		})
		return append(messages, "validate API returned HTTP 500; proceeding with create/update"), nil
	}
	return nil, err
}

func CreateNetworkFirewallRule(c *client.Client, ctx context.Context, req *NetworkFirewallRuleRequest) (*client.AuditResponse, error) {
	tflog.Debug(ctx, "Creating network firewall rule", map[string]any{
		"rule_name":   req.RuleName,
		"firewall_id": req.FirewallID,
	})

	path := common.FirewallConfigPath + "/network-rule"
	respBody, err := c.DoRequest(ctx, http.MethodPost, path, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create network firewall rule: %w", err)
	}

	tflog.Debug(ctx, "Create network firewall rule response", map[string]any{
		"response": string(respBody),
	})

	var result client.AuditResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse create network firewall rule response: %w", err)
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("network firewall rule creation failed: %s (code: %d)", result.Message, result.ResponseCode)
	}

	tflog.Info(ctx, "Network firewall rule creation initiated", map[string]any{
		"audit_id": result.Data.Audit.AuditID,
		"status":   result.Status,
	})

	return &result, nil
}

func UpdateNetworkFirewallRule(c *client.Client, ctx context.Context, req *NetworkFirewallRuleRequest) (*client.AuditResponse, error) {
	tflog.Debug(ctx, "Updating network firewall rule", map[string]any{
		"rule_name":   req.RuleName,
		"firewall_id": req.FirewallID,
		"rule_id":     req.RuleID,
	})

	path := common.FirewallConfigPath + "/network-rule"
	respBody, err := c.DoRequest(ctx, http.MethodPost, path, req)
	if err != nil {
		return nil, fmt.Errorf("failed to update network firewall rule: %w", err)
	}

	tflog.Debug(ctx, "Update network firewall rule response", map[string]any{
		"response": string(respBody),
	})

	var result client.AuditResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse update network firewall rule response: %w", err)
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("network firewall rule update failed: %s (code: %d)", result.Message, result.ResponseCode)
	}

	tflog.Info(ctx, "Network firewall rule update initiated", map[string]any{
		"audit_id": result.Data.Audit.AuditID,
		"status":   result.Status,
	})

	return &result, nil
}

func DeleteNetworkFirewallRule(c *client.Client, ctx context.Context, firewallID int64, ruleID string) (*client.AuditResponse, error) {
	tflog.Debug(ctx, "Deleting network firewall rule", map[string]any{
		"firewall_id": firewallID,
		"rule_id":     ruleID,
	})

	req := map[string]any{
		"firewallId": firewallID,
		"ruleId":     ruleID,
	}
	path := fmt.Sprintf("%s/%d/rule/%s", common.FirewallConfigPath, firewallID, ruleID)
	respBody, err := c.DoRequest(ctx, http.MethodDelete, path, req)
	if err != nil {
		return nil, fmt.Errorf("failed to delete network firewall rule: %w", err)
	}

	tflog.Debug(ctx, "Delete network firewall rule response", map[string]any{
		"response": string(respBody),
	})

	var result client.AuditResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse delete network firewall rule response: %w", err)
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("network firewall rule deletion failed: %s (code: %d)", result.Message, result.ResponseCode)
	}

	tflog.Info(ctx, "Network firewall rule deletion initiated", map[string]any{
		"audit_id": result.Data.Audit.AuditID,
		"status":   result.Status,
	})

	return &result, nil
}

// ReadNetworkFirewallRule calls the FirewallRules action-state read API.
func ReadNetworkFirewallRule(c *client.Client, ctx context.Context, firewallID int64, ruleID string) (*common.ActionStateResponse, error) {
	actionStateBody := map[string]any{
		"resourceId": fmt.Sprintf("%d", firewallID),
		"ruleId":     ruleID,
	}

	return common.UpdateActionState(ctx, c, actionStateModuleFirewallRules, "read", actionStateBody)
}

// ListNetworkFirewallRules calls the FirewallRules action-state list API for a firewall.
func ListNetworkFirewallRules(c *client.Client, ctx context.Context, firewallID int64) ([]map[string]interface{}, *common.ActionStateResponse, error) {
	tflog.Debug(ctx, "Listing network firewall rules", map[string]any{
		"firewall_id": firewallID,
	})

	body := map[string]any{
		"resourceId": fmt.Sprintf("%d", firewallID),
	}

	resp, err := common.UpdateActionState(ctx, c, actionStateModuleFirewallRules, "list", body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list network firewall rules: %w", err)
	}

	if len(resp.Data) == 0 {
		return []map[string]interface{}{}, resp, nil
	}

	var items []map[string]interface{}
	if err := json.Unmarshal(resp.Data, &items); err != nil {
		return nil, resp, fmt.Errorf("failed to parse network firewall rule list response: %w", err)
	}

	tflog.Info(ctx, "Network firewall rules listed successfully", map[string]any{
		"count":       len(items),
		"firewall_id": firewallID,
	})

	return items, resp, nil
}

// FirewallRuleActionState is normalized rule fields from action-state read/list payloads.
type FirewallRuleActionState struct {
	ID                   string
	RuleName             string
	FirewallID           int64
	Source               string
	Action               string
	SourceZoneID         *int64
	Destination          string
	DestinationZoneID    *int64
	SourceAddresses      []string
	DestinationAddresses []string
	ScheduleStartDate    string
	ScheduleEndDate      string
	Services             []string
	Status               string
}

// MapFirewallRuleFromResponseMap normalizes one action-state rule object for data sources.
func MapFirewallRuleFromResponseMap(responseMap map[string]interface{}) FirewallRuleActionState {
	out := FirewallRuleActionState{}

	if v := stringFromMap(responseMap, "ruleId", "rule_id", "id", "ruleid"); v != "" {
		out.ID = v
	} else if ruleID := optionalInt64FromMap(responseMap, "ruleId", "rule_id", "id", "ruleid"); ruleID != nil {
		out.ID = strconv.FormatInt(*ruleID, 10)
	}

	out.RuleName = stringFromMap(responseMap, "ruleName", "rule_name")
	if firewallID := optionalInt64FromMap(responseMap, "firewallId", "firewall_id"); firewallID != nil {
		out.FirewallID = *firewallID
	}

	source := stringFromMap(responseMap, "source")
	if source != "" {
		out.Source = canonicalTerraformSource(source)
	}
	out.Action = canonicalTerraformAction(stringFromMap(responseMap, "action"))
	out.SourceZoneID = optionalInt64FromMap(responseMap, "sourceZoneId", "source_zone_id")

	destination := stringFromMap(responseMap, "destination")
	if destination != "" {
		out.Destination = canonicalTerraformDestination(destination)
	}
	out.DestinationZoneID = optionalInt64FromMap(responseMap, "destinationZoneId", "destination_zone_id")

	out.SourceAddresses = stringSliceFromMap(responseMap, "sourceAddresses", "source_addresses")
	out.DestinationAddresses = stringSliceFromMap(responseMap, "destinationAddresses", "destination_addresses")

	if v := normalizeScheduleDateForState(stringFromMap(responseMap, "scheduleStartDate", "schedule_start_date")); v != "" {
		out.ScheduleStartDate = v
	}
	if v := normalizeScheduleDateForState(stringFromMap(responseMap, "scheduleEndDate", "schedule_end_date")); v != "" {
		out.ScheduleEndDate = v
	}

	services := stringSliceFromMap(responseMap, "services")
	out.Services = normalizeServicesForState(services, out.Source, out.Destination)
	out.Status = stringFromMap(responseMap, "status")

	return out
}

func firewallRuleAuditRequestBody(firewallID int64) map[string]any {
	return map[string]any{
		"resourceId": fmt.Sprintf("%d", firewallID),
	}
}

// CreateNetworkFirewallRuleAndWait validates (best-effort), creates, and waits for the rule to be provisioned.
func CreateNetworkFirewallRuleAndWait(c *client.Client, ctx context.Context, req *NetworkFirewallRuleRequest) (*client.AuditLogResponse, []string, error) {
	messages, err := validateNetworkFirewallRuleLenient(c, ctx, req)
	if err != nil {
		return nil, nil, err
	}

	createResp, err := CreateNetworkFirewallRule(c, ctx, req)
	if err != nil {
		return nil, messages, err
	}

	auditLog, err := c.WaitForAuditCompletion(ctx, createResp.Data.Audit.AuditID, "read", actionStateModuleFirewallRules, firewallRuleAuditRequestBody(req.FirewallID))
	if err != nil {
		return nil, messages, fmt.Errorf("network firewall rule creation failed: %w", err)
	}

	return auditLog, messages, nil
}

// UpdateNetworkFirewallRuleAndWait validates (best-effort), updates, and waits for the rule to be provisioned.
func UpdateNetworkFirewallRuleAndWait(c *client.Client, ctx context.Context, req *NetworkFirewallRuleRequest) (*client.AuditLogResponse, []string, error) {
	messages, err := validateNetworkFirewallRuleLenient(c, ctx, req)
	if err != nil {
		return nil, nil, err
	}

	updateResp, err := UpdateNetworkFirewallRule(c, ctx, req)
	if err != nil {
		return nil, messages, err
	}

	auditLog, err := c.WaitForAuditCompletion(ctx, updateResp.Data.Audit.AuditID, "read", actionStateModuleFirewallRules, firewallRuleAuditRequestBody(req.FirewallID))
	if err != nil {
		return nil, messages, fmt.Errorf("network firewall rule update failed: %w", err)
	}

	return auditLog, messages, nil
}

// DeleteNetworkFirewallRuleAndWait deletes a rule and waits for audit completion.
func DeleteNetworkFirewallRuleAndWait(c *client.Client, ctx context.Context, firewallID int64, ruleID string) (*client.AuditLogResponse, error) {
	deleteResp, err := DeleteNetworkFirewallRule(c, ctx, firewallID, ruleID)
	if err != nil {
		return nil, err
	}

	auditLog, err := c.WaitForAuditCompletion(ctx, deleteResp.Data.Audit.AuditID, "delete", actionStateModuleFirewallRules, firewallRuleAuditRequestBody(firewallID))
	if err != nil {
		return nil, fmt.Errorf("network firewall rule deletion failed: %w", err)
	}

	return auditLog, nil
}

// canonicalTerraformSource maps API or config values to the Terraform schema value.
func canonicalTerraformSource(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "internet":
		return "internet"
	case "zone":
		return "zone"
	case "nas":
		return "nas"
	default:
		return strings.ToLower(strings.TrimSpace(value))
	}
}

// canonicalTerraformDestination maps API or config values to the Terraform schema value.
func canonicalTerraformDestination(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "internet":
		return "internet"
	case "zone":
		return "zone"
	case "nas":
		return "nas"
	case "vcs":
		return "vcs"
	case "load_balancer":
		return "load_balancer"
	default:
		return strings.ToLower(strings.TrimSpace(value))
	}
}

// apiSource maps a Terraform schema value to the API request value.
func apiSource(value string) string {
	switch canonicalTerraformSource(value) {
	case "internet":
		return "Internet"
	case "zone":
		return "Zone"
	case "nas":
		return "NAS"
	default:
		return value
	}
}

// apiDestination maps a Terraform schema value to the API request value.
func apiDestination(value string) string {
	switch canonicalTerraformDestination(value) {
	case "internet":
		return "Internet"
	case "zone":
		return "Zone"
	case "nas":
		return "NAS"
	case "vcs":
		return "VCS"
	case "load_balancer":
		return "Load_Balancer"
	default:
		return value
	}
}

// apiAction maps a Terraform schema value to the API request value.
func apiAction(value string) string {
	switch canonicalTerraformAction(value) {
	case "allow":
		return "ALLOW"
	case "deny":
		return "DENY"
	default:
		return strings.ToUpper(strings.TrimSpace(value))
	}
}

// canonicalTerraformAction maps API or config values to the Terraform schema value.
func canonicalTerraformAction(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "allow", "accept":
		return "allow"
	case "deny", "reject", "drop":
		return "deny"
	default:
		return strings.ToLower(strings.TrimSpace(value))
	}
}

func optionalInt64FromMap(m map[string]interface{}, keys ...string) *int64 {
	for _, key := range keys {
		v, ok := m[key]
		if !ok || v == nil {
			continue
		}
		switch n := v.(type) {
		case float64:
			val := int64(n)
			return &val
		case int64:
			return &n
		case int:
			val := int64(n)
			return &val
		case json.Number:
			if i, err := n.Int64(); err == nil {
				return &i
			}
		case string:
			if i, err := strconv.ParseInt(n, 10, 64); err == nil {
				return &i
			}
		}
	}
	return nil
}

func stringFromMap(m map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if v, ok := m[key]; ok && v != nil {
			if s, ok := v.(string); ok {
				return s
			}
		}
	}
	return ""
}

func stringSliceFromMap(m map[string]interface{}, keys ...string) []string {
	for _, key := range keys {
		v, ok := m[key]
		if !ok || v == nil {
			continue
		}
		switch arr := v.(type) {
		case []interface{}:
			out := make([]string, 0, len(arr))
			for _, item := range arr {
				if s, ok := item.(string); ok {
					out = append(out, s)
				}
			}
			return out
		case []string:
			return arr
		}
	}
	return nil
}
