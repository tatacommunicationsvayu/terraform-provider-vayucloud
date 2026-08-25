package network_lb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/client"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/common"
)

// LoadBalancerEnablementRequest matches LoadBalancerEnablementVO for enable/modify LB.
type LoadBalancerEnablementRequest struct {
	LBBandwidth  string `json:"lbBandwidth"`
	Flavor       string `json:"flavor,omitempty"`
	PricingModel string `json:"pricingModel,omitempty"`
}

// EnableLoadBalancer calls POST /loadbalancer/enable/{firewallCiMasterId} to create a new LB.
func EnableLoadBalancer(c *client.Client, ctx context.Context, firewallCiMasterID int64, req *LoadBalancerEnablementRequest) (*client.AuditResponse, error) {
	tflog.Info(ctx, "Enabling Load Balancer", map[string]any{
		"firewall_ci_id": firewallCiMasterID,
		"lb_bandwidth":   req.LBBandwidth,
	})

	path := fmt.Sprintf(common.LoadBalancerServicePath+"/enable/%d", firewallCiMasterID)
	respBody, err := c.DoRequest(ctx, http.MethodPost, path, req)
	if err != nil {
		return nil, fmt.Errorf("failed to enable load balancer: %w", err)
	}

	tflog.Debug(ctx, "Enable Load Balancer response", map[string]any{
		"response": string(respBody),
	})

	var result client.AuditResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse enable LB response: %w", err)
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("load balancer enablement failed: %s (code: %d)", result.Message, result.ResponseCode)
	}

	tflog.Info(ctx, "Load Balancer enablement initiated", map[string]any{
		"audit_id": result.Data.Audit.AuditID,
		"status":   result.Status,
	})

	return &result, nil
}

// ModifyLoadBalancer calls POST /loadbalancer/modifyLb/{firewallCi} to change LB bandwidth.
func ModifyLoadBalancer(c *client.Client, ctx context.Context, firewallCiMasterID int64, req *LoadBalancerEnablementRequest) (*client.AuditResponse, error) {
	tflog.Info(ctx, "Modifying Load Balancer bandwidth", map[string]any{
		"firewall_ci_id": firewallCiMasterID,
		"lb_bandwidth":   req.LBBandwidth,
	})

	path := fmt.Sprintf(common.LoadBalancerServicePath+"/modifyLb/%d", firewallCiMasterID)
	respBody, err := c.DoRequest(ctx, http.MethodPost, path, req)
	if err != nil {
		return nil, fmt.Errorf("failed to modify load balancer: %w", err)
	}

	var result client.AuditResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse modify LB response: %w", err)
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("load balancer modification failed: %s (code: %d)", result.Message, result.ResponseCode)
	}

	tflog.Info(ctx, "Load Balancer modification initiated", map[string]any{
		"audit_id": result.Data.Audit.AuditID,
		"status":   result.Status,
	})

	return &result, nil
}

// DisableLoadBalancer calls DELETE /loadbalancer/disable/{firewallCi} to disable LB on a firewall.
func DisableLoadBalancer(c *client.Client, ctx context.Context, firewallCiMasterID int64) (*client.AuditResponse, error) {
	tflog.Info(ctx, "Disabling Load Balancer", map[string]any{
		"firewall_ci_id": firewallCiMasterID,
	})

	path := fmt.Sprintf(common.LoadBalancerServicePath+"/disable/%d", firewallCiMasterID)
	respBody, err := c.DoRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to disable load balancer: %w", err)
	}

	tflog.Debug(ctx, "Disable Load Balancer response", map[string]any{
		"response": string(respBody),
	})

	var result client.AuditResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse disable LB response: %w", err)
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("load balancer disablement failed: %s (code: %d)", result.Message, result.ResponseCode)
	}

	tflog.Info(ctx, "Load Balancer disablement initiated", map[string]any{
		"audit_id": result.Data.Audit.AuditID,
		"status":   result.Status,
	})

	return &result, nil
}

// ListLoadBalancers retrieves all LBs for engagement+endpoint via action-state API.
// Backend returns FAILED when no LBs exist; that is treated as an empty list.
func ListLoadBalancers(c *client.Client, ctx context.Context, engagementID int64, endpointID int64) ([]map[string]interface{}, *common.ActionStateResponse, error) {
	tflog.Debug(ctx, "Listing load balancers", map[string]any{
		"engagement_id": engagementID,
		"endpoint_id":   endpointID,
	})

	body := map[string]any{
		"engagementId": engagementID,
		"endpointId":   endpointID,
	}

	path := fmt.Sprintf(common.ConfigServicePath+"/action-state?module=%s&action=%s", "loadbalancer", "list")
	respBody, err := c.DoRequestNoTimeout(ctx, http.MethodPost, path, body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list load balancers: %w", err)
	}

	var resp common.ActionStateResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, nil, fmt.Errorf("failed to parse load balancer list response: %w", err)
	}

	status := strings.ToLower(strings.TrimSpace(resp.Status))
	if status != "success" {
		if isEmptyLBListResponse(resp.Status, resp.Message) {
			tflog.Info(ctx, "No load balancers found", map[string]any{
				"engagement_id": engagementID,
				"endpoint_id":   endpointID,
				"status":        resp.Status,
				"message":       resp.Message,
			})
			return []map[string]interface{}{}, &resp, nil
		}
		return nil, &resp, fmt.Errorf("load balancer list failed: status %s, message: %s", resp.Status, resp.Message)
	}

	if len(resp.Data) == 0 || string(resp.Data) == "null" {
		return []map[string]interface{}{}, &resp, nil
	}

	var items []map[string]interface{}
	if err := json.Unmarshal(resp.Data, &items); err != nil {
		return nil, &resp, fmt.Errorf("failed to parse load balancer list data: %w", err)
	}

	tflog.Info(ctx, "Load Balancers listed successfully", map[string]any{
		"count":         len(items),
		"engagement_id": engagementID,
		"endpoint_id":   endpointID,
	})

	return items, &resp, nil
}

// GetLoadBalancerDetails fetches LB state via action-state read, with network getDetails fallback.
func GetLoadBalancerDetails(c *client.Client, ctx context.Context, lbCiMasterId int64) (map[string]interface{}, error) {
	details, err := getLoadBalancerDetailsFromActionState(c, ctx, lbCiMasterId)
	if err == nil {
		return details, nil
	}

	tflog.Debug(ctx, "Action-state read failed, trying network getDetails", map[string]any{
		"lb_ci_id": lbCiMasterId,
		"error":    err.Error(),
	})

	fallback, fallbackErr := getLoadBalancerDetailsFromNetwork(c, ctx, lbCiMasterId)
	if fallbackErr == nil {
		return fallback, nil
	}

	return nil, fmt.Errorf("action-state read failed: %w; getDetails fallback failed: %v", err, fallbackErr)
}

// ValidateLoadBalancerExists checks that the load balancer CI is readable from the platform.
func ValidateLoadBalancerExists(c *client.Client, ctx context.Context, lbCiID int64) error {
	if lbCiID <= 0 {
		return fmt.Errorf("load_balancer_id must be a positive integer")
	}
	details, err := GetLoadBalancerDetails(c, ctx, lbCiID)
	if err != nil {
		return fmt.Errorf("load balancer %d is not available: %w", lbCiID, err)
	}
	if len(details) == 0 {
		return fmt.Errorf("load balancer %d is not available", lbCiID)
	}
	return nil
}

// ResolveFirewallCIFromLB returns the firewall CI linked to a load balancer.
func ResolveFirewallCIFromLB(c *client.Client, ctx context.Context, lbCiID int64) (int64, error) {
	details, err := GetLoadBalancerDetails(c, ctx, lbCiID)
	if err != nil {
		return 0, fmt.Errorf("could not read load balancer %d: %w", lbCiID, err)
	}
	if firewallCI, ok := int64FromActionStateMap(details, "firewall_ci", "firewallCI"); ok && firewallCI > 0 {
		return firewallCI, nil
	}
	if v := stringFromActionStateMap(details, "firewall_ci", "firewallCI"); v != "" {
		parsed, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
		if err == nil && parsed > 0 {
			return parsed, nil
		}
	}
	return 0, fmt.Errorf("load balancer %d response missing firewall_id", lbCiID)
}

func getLoadBalancerDetailsFromActionState(c *client.Client, ctx context.Context, lbCiMasterId int64) (map[string]interface{}, error) {
	body := map[string]any{
		"resourceId": lbCiMasterId,
	}

	resp, err := common.UpdateActionState(ctx, c, "loadbalancer", "read", body)
	if err != nil {
		return nil, err
	}

	if resp.Status != "SUCCESS" && resp.Status != "success" {
		msg := resp.Message
		if msg == "" {
			msg = "no message"
		}
		return nil, fmt.Errorf("status %s: %s", resp.Status, msg)
	}

	var details map[string]interface{}
	if err := json.Unmarshal(resp.Data, &details); err != nil {
		return nil, fmt.Errorf("failed to parse action-state data: %w", err)
	}

	return details, nil
}

// ErrLoadBalancerNotFoundOnFirewall is returned when no load balancer is enabled on the firewall.
var ErrLoadBalancerNotFoundOnFirewall = errors.New("no load balancer CI found for firewall")

// FindLoadBalancerCIOnFirewall returns the LB CI when a load balancer is already enabled on the firewall.
func FindLoadBalancerCIOnFirewall(c *client.Client, ctx context.Context, firewallCiID int64) (int64, error) {
	lbCiID, err := ResolveLbCiIDFromFirewall(c, ctx, firewallCiID)
	if err != nil {
		if errors.Is(err, ErrLoadBalancerNotFoundOnFirewall) {
			return 0, nil
		}
		return 0, err
	}
	return lbCiID, nil
}

// ResolveLbCiIDFromFirewall finds the Load Balancer CI linked to a firewall CI.
// Audit logs for enable/modify/disable LB use firewall CI as resourceId, not LB CI.
func ResolveLbCiIDFromFirewall(c *client.Client, ctx context.Context, firewallCiID int64) (int64, error) {
	tflog.Debug(ctx, "Resolving LB CI from firewall", map[string]any{
		"firewall_ci_id": firewallCiID,
	})

	fwBody := map[string]any{
		"resourceId": firewallCiID,
	}
	fwResp, err := common.UpdateActionState(ctx, c, "firewall", "read", fwBody)
	if err != nil {
		return 0, fmt.Errorf("failed to read firewall %d: %w", firewallCiID, err)
	}

	var fwDetails map[string]interface{}
	if err := json.Unmarshal(fwResp.Data, &fwDetails); err != nil {
		return 0, fmt.Errorf("failed to parse firewall read response: %w", err)
	}

	engagementID, ok := int64FromActionStateMap(fwDetails, "engagement_id", "engagementId")
	if !ok {
		return 0, fmt.Errorf("firewall %d response missing engagement_id", firewallCiID)
	}
	endpointID, ok := int64FromActionStateMap(fwDetails, "endpoint_id", "endpointId")
	if !ok {
		return 0, fmt.Errorf("firewall %d response missing endpoint_id", firewallCiID)
	}

	lbs, _, err := ListLoadBalancers(c, ctx, engagementID, endpointID)
	if err != nil {
		return 0, fmt.Errorf("failed to list load balancers for firewall %d: %w", firewallCiID, err)
	}

	firewallIDStr := strconv.FormatInt(firewallCiID, 10)
	for _, lb := range lbs {
		if lbFirewallID, ok := int64FromActionStateMap(lb, "firewall_ci", "firewallCI"); ok && lbFirewallID == firewallCiID {
			if lbID, ok := int64FromActionStateMap(lb, "id"); ok {
				tflog.Info(ctx, "Resolved LB CI from firewall", map[string]any{
					"firewall_ci_id": firewallCiID,
					"lb_ci_id":       lbID,
				})
				return lbID, nil
			}
		}
		if fwStr := stringFromActionStateMap(lb, "firewall_ci", "firewallCI"); fwStr == firewallIDStr {
			if lbID, ok := int64FromActionStateMap(lb, "id"); ok {
				tflog.Info(ctx, "Resolved LB CI from firewall", map[string]any{
					"firewall_ci_id": firewallCiID,
					"lb_ci_id":       lbID,
				})
				return lbID, nil
			}
		}
	}

	return 0, fmt.Errorf("%w %d", ErrLoadBalancerNotFoundOnFirewall, firewallCiID)
}

func getLoadBalancerDetailsFromNetwork(c *client.Client, ctx context.Context, lbCiMasterId int64) (map[string]interface{}, error) {
	path := fmt.Sprintf(common.LoadBalancerServicePath+"/getDetails/%d", lbCiMasterId)
	respBody, err := c.DoRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var envelope map[string]interface{}
	if err := json.Unmarshal(respBody, &envelope); err != nil {
		return nil, fmt.Errorf("failed to parse getDetails response: %w", err)
	}

	status, _ := envelope["status"].(string)
	if status != "success" && status != "SUCCESS" {
		msg, _ := envelope["message"].(string)
		if msg == "" {
			msg = "getDetails request failed"
		}
		return nil, fmt.Errorf("status %s: %s", status, msg)
	}

	data, ok := envelope["data"].(map[string]interface{})
	if !ok || data == nil {
		return nil, fmt.Errorf("getDetails response missing data object")
	}

	return data, nil
}
// normalizeLBBandwidth treats the numeric value as Mbps. Accepts "100" or "100Mbps"; API always receives "100Mbps". Gbps is not supported.
// maxLBBandwidthMbps matches backend validateBandwidth (LoadBalancerService).
const maxLBBandwidthMbps = 1000

// parseLBBandwidthMbps returns the numeric Mbps value from a bandwidth string.
func parseLBBandwidthMbps(value string) (int, bool) {
	normalized := normalizeLBBandwidth(value)
	if normalized == "" {
		return 0, false
	}
	numStr := strings.TrimSuffix(strings.TrimSuffix(normalized, "Mbps"), "mbps")
	mbps, err := strconv.Atoi(strings.TrimSpace(numStr))
	if err != nil || mbps <= 0 {
		return 0, false
	}
	return mbps, true
}

func validateLBBandwidthLimit(value string) error {
	mbps, ok := parseLBBandwidthMbps(value)
	if !ok {
		return fmt.Errorf("bandwidth must be a positive value in Mbps (e.g. 100 or 100Mbps)")
	}
	if mbps > maxLBBandwidthMbps {
		return fmt.Errorf("bandwidth %d Mbps exceeds the platform limit of %d Mbps", mbps, maxLBBandwidthMbps)
	}
	return nil
}

func normalizeLBBandwidth(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	numStr := value
	if idx := strings.Index(strings.ToLower(value), "mbps"); idx >= 0 {
		numStr = strings.TrimSpace(value[:idx])
	}
	n, err := strconv.Atoi(numStr)
	if err != nil || n <= 0 {
		return ""
	}
	return strconv.Itoa(n) + "Mbps"
}

func reconcileBandwidthState(preferred, apiValue string) string {
	apiNorm := normalizeLBBandwidth(apiValue)
	if apiNorm == "" {
		if preferred != "" {
			return preferred
		}
		return apiValue
	}
	if preferred != "" && normalizeLBBandwidth(preferred) == apiNorm {
		return preferred
	}
	return apiNorm
}

func resolveLBPricingModel(value types.String) string {
	if value.IsNull() || value.IsUnknown() {
		return "daily"
	}
	v := strings.TrimSpace(strings.ToLower(value.ValueString()))
	if v == "" {
		return "daily"
	}
	return v
}

func isEmptyLBListResponse(status, message string) bool {
	if strings.EqualFold(strings.TrimSpace(status), "success") {
		return false
	}
	message = strings.ToLower(strings.TrimSpace(message))
	for _, signal := range []string{"unavailable", "no load balancers", "not found"} {
		if strings.Contains(message, signal) {
			return true
		}
	}
	return strings.EqualFold(strings.TrimSpace(status), "failed") && message == ""
}

func firewallIDRequiresReplace(plan, state types.String) bool {
	planID := strings.TrimSpace(plan.ValueString())
	stateID := strings.TrimSpace(state.ValueString())
	if planID == "" || stateID == "" {
		return false
	}
	return planID != stateID
}
func stringFromActionStateMap(m map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		v, ok := m[key]
		if !ok || v == nil {
			continue
		}
		switch val := v.(type) {
		case string:
			if val != "" {
				return val
			}
		default:
			s := fmt.Sprint(val)
			if s != "" && s != "<nil>" {
				return s
			}
		}
	}
	return ""
}

func int64FromActionStateMap(m map[string]interface{}, keys ...string) (int64, bool) {
	for _, key := range keys {
		v, ok := m[key]
		if !ok || v == nil {
			continue
		}
		switch n := v.(type) {
		case float64:
			return int64(n), true
		case int:
			return int64(n), true
		case int64:
			return n, true
		case json.Number:
			i, err := n.Int64()
			if err == nil {
				return i, true
			}
		case string:
			i, err := strconv.ParseInt(strings.TrimSpace(n), 10, 64)
			if err == nil {
				return i, true
			}
		}
	}
	return 0, false
}
