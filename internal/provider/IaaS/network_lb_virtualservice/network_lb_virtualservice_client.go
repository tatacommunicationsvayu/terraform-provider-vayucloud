package network_lb_virtualservice

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/client"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/common"
)

// VirtualServicePoolMember matches LBNodeVO pool member fields for HAProxy.
type VirtualServicePoolMember struct {
	Name      string `json:"name"`
	IPAddress string `json:"ipAddress"`
	Port      int    `json:"port"`
}

// VirtualServiceRequest matches VirtualServerVO for HAProxy create/edit.
type VirtualServiceRequest struct {
	VirtualServerName string                     `json:"virtualServerName"`
	VirtualServerPort string                     `json:"virtualServerport"`
	Protocol          string                     `json:"protocol"`
	ZoneID            int                        `json:"zoneId"`
	VipIP             string                     `json:"vipIp,omitempty"`
	PoolAlgorithm     string                     `json:"poolAlgorithm"`
	Monitors          []string                   `json:"monitor"`
	PersistenceType   string                     `json:"persistenceType,omitempty"`
	PersistenceValue  string                     `json:"persistenceValue,omitempty"`
	CertificateName   string                     `json:"certificateName,omitempty"`
	PoolMembers       []VirtualServicePoolMember `json:"poolMembers"`
}

func parseLBServiceEnvelope(respBody []byte) (map[string]interface{}, error) {
	var envelope map[string]interface{}
	if err := json.Unmarshal(respBody, &envelope); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	status, _ := envelope["status"].(string)
	if !strings.EqualFold(strings.TrimSpace(status), "success") {
		msg, _ := envelope["message"].(string)
		if msg == "" {
			msg = "request failed"
		}
		return nil, fmt.Errorf("status %s: %s", status, msg)
	}
	return envelope, nil
}

func extractLBServiceDataArray(envelope map[string]interface{}) ([]map[string]interface{}, error) {
	raw, ok := envelope["data"]
	if !ok || raw == nil {
		return []map[string]interface{}{}, nil
	}
	switch data := raw.(type) {
	case []interface{}:
		items := make([]map[string]interface{}, 0, len(data))
		for _, item := range data {
			if m, ok := item.(map[string]interface{}); ok {
				items = append(items, m)
			}
		}
		return items, nil
	case map[string]interface{}:
		return []map[string]interface{}{data}, nil
	default:
		return nil, fmt.Errorf("unexpected data type %T", raw)
	}
}

func extractLBServiceProfileValues(envelope map[string]interface{}) ([]map[string]interface{}, error) {
	data, err := extractLBServiceDataObject(envelope)
	if err != nil {
		return nil, err
	}
	if strings.EqualFold(stringFromActionStateMap(data, "status"), "error") {
		msg := stringFromActionStateMap(data, "response", "message")
		if msg == "" {
			msg = "profile list failed"
		}
		return nil, fmt.Errorf("%s", msg)
	}
	raw, ok := data["values"]
	if !ok || raw == nil {
		return []map[string]interface{}{}, nil
	}
	items, ok := raw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected values type %T", raw)
	}
	result := make([]map[string]interface{}, 0, len(items))
	for _, item := range items {
		if m, ok := item.(map[string]interface{}); ok {
			result = append(result, m)
		}
	}
	return result, nil
}

func extractLBServiceDataObject(envelope map[string]interface{}) (map[string]interface{}, error) {
	raw, ok := envelope["data"]
	if !ok || raw == nil {
		return map[string]interface{}{}, nil
	}
	data, ok := raw.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected data type %T", raw)
	}
	return data, nil
}

func getLBServiceList(ctx context.Context, c *client.Client, path string) ([]map[string]interface{}, error) {
	respBody, err := c.DoRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	envelope, err := parseLBServiceEnvelope(respBody)
	if err != nil {
		return nil, err
	}
	return extractLBServiceDataArray(envelope)
}

// ListLBProtocols returns HAProxy protocol options for virtual service create/edit.
func ListLBProtocols(c *client.Client, ctx context.Context, lbCiMasterID int64) ([]map[string]interface{}, error) {
	path := fmt.Sprintf(common.LoadBalancerServicePath+"/list/protocols/%d", lbCiMasterID)
	return getLBServiceList(ctx, c, path)
}

// ListLBAlgorithms returns HAProxy pool algorithm options.
func ListLBAlgorithms(c *client.Client, ctx context.Context, lbCiMasterID int64) ([]map[string]interface{}, error) {
	path := fmt.Sprintf(common.LoadBalancerServicePath+"/list/algorithms/%d", lbCiMasterID)
	return getLBServiceList(ctx, c, path)
}

// ListLBPersistenceTypes returns HAProxy persistence options.
func ListLBPersistenceTypes(c *client.Client, ctx context.Context, lbCiMasterID int64) ([]map[string]interface{}, error) {
	path := fmt.Sprintf(common.LoadBalancerServicePath+"/list/persistence/%d", lbCiMasterID)
	return getLBServiceList(ctx, c, path)
}

// ListLBMonitors returns health check options filtered by protocol family (http|tcp).
func ListLBMonitors(c *client.Client, ctx context.Context, lbCiMasterID int64, protocolType string) ([]map[string]interface{}, error) {
	protocolType = strings.TrimSpace(strings.ToLower(protocolType))
	if protocolType == "" {
		protocolType = "http"
	}
	path := fmt.Sprintf(common.LoadBalancerServicePath+"/list/node/monitors/%d?type=%s", lbCiMasterID, protocolType)
	respBody, err := c.DoRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	envelope, err := parseLBServiceEnvelope(respBody)
	if err != nil {
		return nil, err
	}
	return extractLBServiceProfileValues(envelope)
}

// ListLBZones returns zones available for virtual service placement on the load balancer.
func ListLBZones(c *client.Client, ctx context.Context, lbCiMasterID int64) ([]map[string]interface{}, error) {
	path := fmt.Sprintf(common.LoadBalancerServicePath+"/list/zones/%d", lbCiMasterID)
	return getLBServiceList(ctx, c, path)
}

// ListClientSSLProfiles returns uploaded client SSL certificates on the load balancer.
func ListClientSSLProfiles(c *client.Client, ctx context.Context, lbCiMasterID int64) ([]map[string]interface{}, error) {
	path := fmt.Sprintf(common.LoadBalancerServicePath+"/list/client/sslprofile/%d", lbCiMasterID)
	respBody, err := c.DoRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	envelope, err := parseLBServiceEnvelope(respBody)
	if err != nil {
		return nil, err
	}
	return extractLBServiceProfileValues(envelope)
}

func postVirtualServiceAudit(c *client.Client, ctx context.Context, method, path string, req *VirtualServiceRequest) (*client.AuditResponse, error) {
	respBody, err := c.DoRequest(ctx, method, path, req)
	if err != nil {
		return nil, err
	}
	var result client.AuditResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse virtual service response: %w", err)
	}
	if result.Status != "success" {
		return nil, fmt.Errorf("virtual service request failed: %s (code: %d)", result.Message, result.ResponseCode)
	}
	return &result, nil
}

// CreateVirtualService calls POST /create/virtual/service/{loadbalancerCI}.
func CreateVirtualService(c *client.Client, ctx context.Context, lbCiMasterID int64, req *VirtualServiceRequest) (*client.AuditResponse, error) {
	tflog.Info(ctx, "Creating Virtual Service", map[string]any{
		"name":             req.VirtualServerName,
		"load_balancer_id": lbCiMasterID,
		"port":             req.VirtualServerPort,
		"protocol":         req.Protocol,
		"zone_id":          req.ZoneID,
	})
	path := fmt.Sprintf(common.LoadBalancerServicePath+"/create/virtual/service/%d", lbCiMasterID)
	result, err := postVirtualServiceAudit(c, ctx, http.MethodPost, path, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create virtual service: %w", err)
	}
	tflog.Info(ctx, "Virtual Service creation initiated", map[string]any{
		"audit_id": result.Data.Audit.AuditID,
	})
	return result, nil
}

// EditVirtualService calls POST /edit/virtual/service/{loadbalancerCI}.
func EditVirtualService(c *client.Client, ctx context.Context, lbCiMasterID int64, req *VirtualServiceRequest) (*client.AuditResponse, error) {
	tflog.Info(ctx, "Editing Virtual Service", map[string]any{
		"name":             req.VirtualServerName,
		"load_balancer_id": lbCiMasterID,
	})
	path := fmt.Sprintf(common.LoadBalancerServicePath+"/edit/virtual/service/%d", lbCiMasterID)
	result, err := postVirtualServiceAudit(c, ctx, http.MethodPost, path, req)
	if err != nil {
		return nil, fmt.Errorf("failed to edit virtual service: %w", err)
	}
	tflog.Info(ctx, "Virtual Service edit initiated", map[string]any{
		"audit_id": result.Data.Audit.AuditID,
	})
	return result, nil
}

// ValidateVirtualServiceCreation calls POST /validate/virtualServerCreation/{loadbalancerCI}.
func ValidateVirtualServiceCreation(c *client.Client, ctx context.Context, lbCiMasterID int64, req *VirtualServiceRequest) error {
	path := fmt.Sprintf(common.LoadBalancerServicePath+"/validate/virtualServerCreation/%d", lbCiMasterID)
	respBody, err := c.DoRequest(ctx, http.MethodPost, path, req)
	if err != nil {
		return fmt.Errorf("failed to validate virtual service: %w", err)
	}

	envelope, err := parseLBServiceEnvelope(respBody)
	if err != nil {
		return err
	}

	data, err := extractLBServiceDataObject(envelope)
	if err != nil {
		return err
	}

	if !validationResultIsValid(data) {
		message := stringFromActionStateMap(data, "message")
		if message == "" {
			message = "virtual service pre-create validation failed"
		}
		return fmt.Errorf("%s", message)
	}

	return nil
}

func validationResultIsValid(data map[string]interface{}) bool {
	raw, ok := data["isValid"]
	if !ok || raw == nil {
		return false
	}
	switch value := raw.(type) {
	case bool:
		return value
	case string:
		return strings.EqualFold(strings.TrimSpace(value), "true")
	default:
		return strings.EqualFold(strings.TrimSpace(fmt.Sprint(value)), "true")
	}
}

// DeleteVirtualService calls DELETE /delete/virtual/service/{loadbalancerCI}?virtualServiceName=...
func DeleteVirtualService(c *client.Client, ctx context.Context, lbCiMasterID int64, virtualServiceName string) (*client.AuditResponse, error) {
	tflog.Info(ctx, "Deleting Virtual Service", map[string]any{
		"load_balancer_id":     lbCiMasterID,
		"virtual_service_name": virtualServiceName,
	})
	path := fmt.Sprintf(
		common.LoadBalancerServicePath+"/delete/virtual/service/%d?virtualServiceName=%s",
		lbCiMasterID,
		url.PathEscape(strings.TrimSpace(virtualServiceName)),
	)
	result, err := postVirtualServiceAudit(c, ctx, http.MethodDelete, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to delete virtual service: %w", err)
	}
	tflog.Info(ctx, "Virtual Service deletion initiated", map[string]any{
		"audit_id": result.Data.Audit.AuditID,
	})
	return result, nil
}

// normalizeProtocol maps user/protocol-list input to lowercase (http, https, tcp).
func normalizeProtocol(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "http", "https", "tcp":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return strings.ToLower(strings.TrimSpace(value))
	}
}

// apiProtocolForCreate maps the Terraform listener protocol to VirtualServerVO.protocol on
// create/edit. User config keeps "https"; the API receives "http" because HaProxyConfigServiceImpl
// passes protocol through as HAProxy frontend mode, which only accepts http or tcp. TLS termination
// is applied on the VIP bind (certificate_name + SSL port). Monitors remain a separate field.
func apiProtocolForCreate(protocol string) string {
	switch normalizeProtocol(protocol) {
	case "https":
		return "http"
	default:
		return normalizeProtocol(protocol)
	}
}

// displayProtocolFromAPI maps list/read API protocol back to the user-facing value.
func displayProtocolFromAPI(apiProtocol, port, certName string) string {
	switch strings.ToUpper(strings.TrimSpace(apiProtocol)) {
	case "HTTPS":
		return "https"
	case "HTTP":
		if strings.TrimSpace(certName) != "" && listenerUsesSSLPort(port) {
			return "https"
		}
		return "http"
	case "TCP":
		return "tcp"
	default:
		return normalizeProtocol(apiProtocol)
	}
}

func listenerUsesSSLPort(port string) bool {
	for _, part := range strings.Split(port, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if strings.Contains(part, "-") {
			rangeParts := strings.SplitN(part, "-", 2)
			if len(rangeParts) != 2 {
				continue
			}
			start, err1 := strconv.Atoi(strings.TrimSpace(rangeParts[0]))
			end, err2 := strconv.Atoi(strings.TrimSpace(rangeParts[1]))
			if err1 != nil || err2 != nil {
				continue
			}
			for _, sslPort := range []int{443, 8443, 4433, 9443, 10443} {
				if sslPort >= start && sslPort <= end {
					return true
				}
			}
			continue
		}
		p, err := strconv.Atoi(part)
		if err != nil {
			continue
		}
		switch p {
		case 443, 8443, 4433, 9443, 10443:
			return true
		}
	}
	return false
}

func defaultMonitorForProtocol(protocol string) string {
	switch strings.ToUpper(strings.TrimSpace(protocol)) {
	case "TCP":
		return "tcp-check"
	default:
		return "httpchk"
	}
}

func normalizePoolAlgorithm(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "roundrobin"
	}
	switch strings.ToLower(value) {
	case "round robin":
		return "roundrobin"
	case "least connections":
		return "leastconn"
	case "client ip", "source ip":
		return "source"
	case "first server":
		return "first"
	case "random selection":
		return "random"
	default:
		return value
	}
}

// ListVirtualServicesOnLB returns HAProxy virtual services for a load balancer.
func ListVirtualServicesOnLB(c *client.Client, ctx context.Context, lbCiMasterID, zoneID int64) ([]map[string]interface{}, error) {
	path := fmt.Sprintf(common.LoadBalancerServicePath+"/list/virtualservices/%d", lbCiMasterID)
	if zoneID > 0 {
		path = fmt.Sprintf("%s?zoneId=%d", path, zoneID)
	}
	tflog.Debug(ctx, "Listing virtual services on load balancer", map[string]any{
		"load_balancer_id": lbCiMasterID,
		"zone_id":          zoneID,
		"path":             path,
	})
	return getLBServiceList(ctx, c, path)
}

func compositeVSID(lbCiID int64, vsName string) string {
	return fmt.Sprintf("%d/%s", lbCiID, strings.TrimSpace(vsName))
}

func isCompositeVSID(id string) bool {
	parts := strings.Split(strings.TrimSpace(id), "/")
	return len(parts) == 2 && strings.TrimSpace(parts[0]) != "" && strings.TrimSpace(parts[1]) != ""
}

func findVSInList(list []map[string]interface{}, vsName string) (map[string]interface{}, bool) {
	target := strings.TrimSpace(vsName)
	for _, item := range list {
		name := stringFromActionStateMap(item, "virtualServerName", "name", "virtual_server_name")
		if strings.EqualFold(name, target) {
			return item, true
		}
	}
	return nil, false
}

// GetVirtualServiceDetails fetches virtual service state.
//
// GET {LoadBalancerServicePath}/lb-virtualservice-state/{virtualserviceId}
// Returns the same response envelope as the legacy action-state read API (module=virtualservice, action=read).
func GetVirtualServiceDetails(c *client.Client, ctx context.Context, vsCiMasterId int64) (map[string]interface{}, error) {
	tflog.Debug(ctx, "Getting virtual service state", map[string]any{
		"virtualservice_id": vsCiMasterId,
	})

	path := fmt.Sprintf("%s/lb-virtualservice-state/%d", common.LoadBalancerServicePath, vsCiMasterId)
	respBody, err := c.DoRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get virtual service state: %w", err)
	}

	tflog.Debug(ctx, "Virtual service state response", map[string]any{
		"response": string(respBody),
	})

	var resp common.ActionStateResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse virtual service state response: %w", err)
	}

	var details map[string]interface{}
	if err := json.Unmarshal(resp.Data, &details); err != nil {
		return nil, fmt.Errorf("failed to parse virtual service state data: %w", err)
	}

	return details, nil
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

func inferZoneIDFromVSDetails(vsDetails map[string]interface{}) (int64, bool) {
	if zoneID, ok := int64FromActionStateMap(vsDetails, "zoneId", "zone_id"); ok && zoneID > 0 {
		return zoneID, true
	}
	rawMembers, ok := vsDetails["poolMembers"].([]interface{})
	if !ok {
		return 0, false
	}
	for _, item := range rawMembers {
		memberMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		if zoneID, ok := int64FromActionStateMap(memberMap, "zoneId", "zone_id"); ok && zoneID > 0 {
			return zoneID, true
		}
	}
	return 0, false
}
