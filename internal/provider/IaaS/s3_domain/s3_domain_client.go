// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package s3_domain

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

// DomainNameSuffix is appended server-side to the short domain_name to form the FQDN.
const DomainNameSuffix = ".ipstorage.tatacommunications.com"

// actionStateModule is the query param for configservice action-state calls.
const actionStateModule = "domain"

// S3DomainCreateRequest is the POST body sent to /ics-operations/createDomain.
//
// IMPORTANT: the ICS API uses camelCase JSON keys (unlike the Network API which
// uses snake_case). All tags below must stay camelCase — do not rename them.
//
// Optional fields use pointer + omitempty so they are omitted entirely when
// not set, rather than being serialised as zero/null, which would confuse the
// backend.
type S3DomainCreateRequest struct {
	EngagementID        int64   `json:"engagementId"`
	EndpointID          int64   `json:"endpointId"`
	DomainName          string  `json:"domainName"`
	Quota               float64 `json:"quota"`
	StorageClass        string  `json:"storageClass"`
	Variant             string  `json:"variant"`
	SecondaryEndpointID *int64  `json:"secondaryEndpointId,omitempty"`
	PricingModel        *string `json:"pricingModel,omitempty"`
	FirewallID          int64   `json:"firewallId"`
}

// S3DomainPayload is the action-state read/list item shape returned by
// DomainStateServiceImpl.buildDomainStatePayload (snake_case JSON keys).
type S3DomainPayload struct {
	ID                   int64   `json:"id"`
	DomainName           string  `json:"domain_name"`
	EngagementID         int64   `json:"engagement_id"`
	EndpointID           int64   `json:"endpoint_id"`
	Quota                float64 `json:"quota"`
	QuotaUnit            string  `json:"quota_unit"`
	StorageClass         string  `json:"storage_class"`
	Variant              string  `json:"variant"`
	DomainAccessIP       string  `json:"domain_access_ip,omitempty"`
	DomainAccessPublicIP string  `json:"domain_access_public_ip,omitempty"`
}

// S3DomainUpdateRequest is the PUT body for /ics-operations/domain/{domain_id}.
type S3DomainUpdateRequest struct {
	Quota float64 `json:"quota"`
}

func isDomainNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "status 404") ||
		strings.Contains(msg, `"responsecode":404`) ||
		strings.Contains(msg, "domain not found")
}

// ExpectedDomainFQDN returns the FQDN the server assigns for a short domain name.
func ExpectedDomainFQDN(shortName string) string {
	if strings.HasSuffix(shortName, DomainNameSuffix) {
		return shortName
	}
	return shortName + DomainNameSuffix
}

func buildListRequestBody(engagementID, endpointID int64, firewallID *int64) map[string]any {
	body := map[string]any{
		"engagementId": engagementID,
		"endpointId":   endpointID,
	}
	if firewallID != nil {
		body["firewallId"] = *firewallID
	}
	return body
}

func buildReadRequestBody(engagementID int64, resourceID string, firewallID int64) map[string]any {
	return map[string]any{
		"engagementId": engagementID,
		"resourceId":   resourceID,
		"firewallId":   firewallID,
	}
}

// ListS3Domains returns all domains for an IPC engagement and endpoint via
// action-state module=domain, action=list. The backend resolves IPC→ICS engagement.
func ListS3Domains(c *client.Client, ctx context.Context, engagementID, endpointID int64, firewallID *int64) ([]S3DomainPayload, *common.ActionStateResponse, error) {
	tflog.Debug(ctx, "Listing S3 domains", map[string]any{
		"engagement_id": engagementID,
		"endpoint_id":   endpointID,
		"firewall_id":   firewallID,
	})

	resp, err := common.UpdateActionState(ctx, c, actionStateModule, "list", buildListRequestBody(engagementID, endpointID, firewallID))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list S3 domains: %w", err)
	}

	if len(resp.Data) == 0 {
		return []S3DomainPayload{}, resp, nil
	}

	var items []S3DomainPayload
	if err := json.Unmarshal(resp.Data, &items); err != nil {
		return nil, resp, fmt.Errorf("failed to parse S3 domain list response: %w", err)
	}

	tflog.Info(ctx, "S3 domains listed successfully", map[string]any{
		"count":         len(items),
		"engagement_id": engagementID,
		"endpoint_id":   endpointID,
	})

	return items, resp, nil
}

// ReadS3DomainActionState reads a single domain via action-state module=domain,
// action=read and returns the parsed payload plus the full action-state wrapper.
func ReadS3DomainActionState(c *client.Client, ctx context.Context, engagementID int64, resourceID string, firewallID int64) (*S3DomainPayload, *common.ActionStateResponse, error) {
	tflog.Debug(ctx, "Reading S3 domain", map[string]any{
		"engagement_id": engagementID,
		"resource_id":   resourceID,
		"firewall_id":   firewallID,
	})

	resp, err := common.UpdateActionState(ctx, c, actionStateModule, "read", buildReadRequestBody(engagementID, resourceID, firewallID))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read S3 domain: %w", err)
	}

	if len(resp.Data) == 0 {
		return nil, resp, fmt.Errorf("S3 domain id %s is not available", resourceID)
	}

	var payload S3DomainPayload
	if err := json.Unmarshal(resp.Data, &payload); err != nil {
		return nil, resp, fmt.Errorf("failed to parse S3 domain read response: %w", err)
	}

	return &payload, resp, nil
}

// ReadS3Domain reads a single domain via action-state module=domain, action=read.
func ReadS3Domain(c *client.Client, ctx context.Context, engagementID int64, resourceID string, firewallID int64) (*S3DomainPayload, error) {
	payload, _, err := ReadS3DomainActionState(c, ctx, engagementID, resourceID, firewallID)
	return payload, err
}

// ResolveEngagementEndpointFromFirewall reads firewall action-state (module=firewall,
// action=read) and returns the IPC engagement_id and endpoint_id linked to the firewall.
func ResolveEngagementEndpointFromFirewall(c *client.Client, ctx context.Context, firewallID int64) (engagementID, endpointID int64, err error) {
	tflog.Debug(ctx, "Resolving engagement and endpoint from firewall", map[string]any{
		"firewall_id": firewallID,
	})

	actionStateBody := map[string]any{
		"resourceId": fmt.Sprintf("%d", firewallID),
	}

	actionStateResponse, err := common.UpdateActionState(ctx, c, "firewall", "read", actionStateBody)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to read firewall %d: %w", firewallID, err)
	}

	if len(actionStateResponse.Data) == 0 {
		return 0, 0, fmt.Errorf("firewall ID %d is not available", firewallID)
	}

	var responseMap map[string]interface{}
	if err := json.Unmarshal(actionStateResponse.Data, &responseMap); err != nil {
		return 0, 0, fmt.Errorf("firewall ID %d is not available", firewallID)
	}

	if len(responseMap) == 0 {
		return 0, 0, fmt.Errorf("firewall ID %d is not available", firewallID)
	}

	engagementID, ok := int64FromActionStateMap(responseMap, "engagement_id", "engagementId")
	if !ok {
		return 0, 0, fmt.Errorf("firewall %d response missing engagement_id", firewallID)
	}
	endpointID, ok = int64FromActionStateMap(responseMap, "endpoint_id", "endpointId")
	if !ok {
		return 0, 0, fmt.Errorf("firewall %d response missing endpoint_id", firewallID)
	}

	tflog.Debug(ctx, "Resolved engagement and endpoint from firewall", map[string]any{
		"firewall_id":   firewallID,
		"engagement_id": engagementID,
		"endpoint_id":   endpointID,
	})

	return engagementID, endpointID, nil
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
		case int64:
			return n, true
		case int:
			return int64(n), true
		case json.Number:
			i, err := n.Int64()
			if err == nil {
				return i, true
			}
		}
	}
	return 0, false
}

// ValidateDomainExists checks that an S3 domain with the given ID exists via
// action-state read. Exported for cross-resource validation (mirror of
// network_firewall.ValidateFirewallExists).
func ValidateDomainExists(c *client.Client, ctx context.Context, engagementID int64, domainID string, firewallID int64) error {
	tflog.Debug(ctx, "Validating S3 domain exists", map[string]any{
		"engagement_id": engagementID,
		"domain_id":     domainID,
	})

	_, err := ReadS3Domain(c, ctx, engagementID, domainID, firewallID)
	if err != nil {
		return fmt.Errorf("S3 domain ID %s is not available: %w", domainID, err)
	}

	tflog.Debug(ctx, "S3 domain ID validated successfully", map[string]any{
		"domain_id": domainID,
	})

	return nil
}

// ValidateDomainNameAvailable checks that no domain with the same FQDN already
// exists for the engagement and endpoint. Pass the IPC engagement_id; the
// backend resolves ICS engagement internally.
func ValidateDomainNameAvailable(c *client.Client, ctx context.Context, engagementID, endpointID int64, domainName string, firewallID int64) error {
	expectedFQDN := ExpectedDomainFQDN(domainName)

	items, _, err := ListS3Domains(c, ctx, engagementID, endpointID, &firewallID)
	if err != nil {
		return fmt.Errorf("failed to check domain name availability: %w", err)
	}

	for _, item := range items {
		if strings.EqualFold(item.DomainName, expectedFQDN) {
			return fmt.Errorf("domain name %q already exists for engagement %d and endpoint %d (id %d)",
				domainName, engagementID, endpointID, item.ID)
		}
	}

	tflog.Debug(ctx, "Domain name is available", map[string]any{
		"domain_name":   domainName,
		"expected_fqdn": expectedFQDN,
		"engagement_id": engagementID,
		"endpoint_id":   endpointID,
	})

	return nil
}

// CreateS3Domain sends POST /ics-operations/createDomain and returns the initial
// audit response containing the auditId for subsequent polling.
func CreateS3Domain(c *client.Client, ctx context.Context, req *S3DomainCreateRequest) (*client.AuditResponse, error) {
	tflog.Debug(ctx, "Creating S3 domain", map[string]any{
		"engagement_id": req.EngagementID,
		"endpoint_id":   req.EndpointID,
		"domain_name":   req.DomainName,
		"storage_class": req.StorageClass,
		"variant":       req.Variant,
		"quota":         req.Quota,
		"firewall_id":   req.FirewallID,
		"pricing_model": req.PricingModel,
	})

	respBody, err := c.DoRequest(ctx, http.MethodPost, common.ICSOperationsPath+"/createDomain", req)
	if err != nil {
		return nil, fmt.Errorf("failed to create S3 domain: %w", err)
	}

	tflog.Debug(ctx, "Create S3 domain response", map[string]any{
		"response": string(respBody),
	})

	var result client.AuditResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse create S3 domain response: %w", err)
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("S3 domain creation failed: %s (code: %d)", result.Message, result.ResponseCode)
	}

	tflog.Info(ctx, "S3 domain creation initiated", map[string]any{
		"audit_id": result.Data.Audit.AuditID,
		"status":   result.Status,
	})

	return &result, nil
}

// CreateS3DomainAndWait creates an S3 domain and blocks until the audit reaches
// a terminal state (Completed or Failed).
func CreateS3DomainAndWait(c *client.Client, ctx context.Context, req *S3DomainCreateRequest, actionStateBody map[string]any) (*client.AuditLogResponse, error) {
	tflog.Info(ctx, "Creating S3 domain and waiting for completion", map[string]any{
		"domain_name":   req.DomainName,
		"engagement_id": req.EngagementID,
		"endpoint_id":   req.EndpointID,
	})

	createResp, err := CreateS3Domain(c, ctx, req)
	if err != nil {
		return nil, err
	}

	auditLog, err := c.WaitForAuditCompletion(ctx, createResp.Data.Audit.AuditID, "create", actionStateModule, actionStateBody)
	if err != nil {
		return nil, fmt.Errorf("S3 domain creation failed: %w", err)
	}

	if auditLog.ResourceID.String() == "" {
		return auditLog, fmt.Errorf("audit completed but resourceId is empty (audit %s)", createResp.Data.Audit.AuditID)
	}

	tflog.Info(ctx, "S3 domain created successfully", map[string]any{
		"audit_id":    createResp.Data.Audit.AuditID,
		"resource_id": auditLog.ResourceID.String(),
		"domain_name": req.DomainName,
	})

	return auditLog, nil
}

// UpdateS3Domain sends PUT /ics-operations/domain/{domain_id} to update quota.
func UpdateS3Domain(c *client.Client, ctx context.Context, domainID string, quota float64) (*client.AuditResponse, error) {
	tflog.Debug(ctx, "Updating S3 domain quota", map[string]any{
		"domain_id": domainID,
		"quota":     quota,
	})

	path := fmt.Sprintf("%s/domain/%s", common.ICSOperationsPath, domainID)
	req := &S3DomainUpdateRequest{Quota: quota}

	respBody, err := c.DoRequest(ctx, http.MethodPut, path, req)
	if err != nil {
		return nil, fmt.Errorf("failed to update S3 domain: %w", err)
	}

	tflog.Debug(ctx, "Update S3 domain response", map[string]any{
		"response": string(respBody),
	})

	var result client.AuditResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse update S3 domain response: %w", err)
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("S3 domain update failed: %s (code: %d)", result.Message, result.ResponseCode)
	}

	tflog.Info(ctx, "S3 domain update initiated", map[string]any{
		"audit_id":  result.Data.Audit.AuditID,
		"domain_id": domainID,
		"quota":     quota,
	})

	return &result, nil
}

// UpdateS3DomainAndWait updates domain quota and blocks until audit completes and
// action-state update confirms the new quota.
func UpdateS3DomainAndWait(c *client.Client, ctx context.Context, domainID string, quota float64) (*client.AuditLogResponse, error) {
	tflog.Info(ctx, "Updating S3 domain and waiting for completion", map[string]any{
		"domain_id": domainID,
		"quota":     quota,
	})

	updateResp, err := UpdateS3Domain(c, ctx, domainID, quota)
	if err != nil {
		return nil, err
	}

	actionStateBody := map[string]any{
		"quota":      quota,
		"resourceId": domainID,
	}

	auditLog, err := c.WaitForAuditCompletion(ctx, updateResp.Data.Audit.AuditID, "update", actionStateModule, actionStateBody)
	if err != nil {
		return nil, fmt.Errorf("S3 domain update failed: %w", err)
	}

	tflog.Info(ctx, "S3 domain updated successfully", map[string]any{
		"audit_id":  updateResp.Data.Audit.AuditID,
		"domain_id": domainID,
		"quota":     quota,
	})

	return auditLog, nil
}

// DeleteS3Domain sends DELETE /ics-operations/domain/{domain_id}.
func DeleteS3Domain(c *client.Client, ctx context.Context, domainID string) (*client.AuditResponse, error) {
	tflog.Debug(ctx, "Deleting S3 domain", map[string]any{
		"domain_id": domainID,
	})

	path := fmt.Sprintf("%s/domain/%s", common.ICSOperationsPath, domainID)
	respBody, err := c.DoRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to delete S3 domain: %w", err)
	}

	tflog.Debug(ctx, "Delete S3 domain response", map[string]any{
		"response": string(respBody),
	})

	var result client.AuditResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse delete S3 domain response: %w", err)
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("S3 domain deletion failed: %s (code: %d)", result.Message, result.ResponseCode)
	}

	tflog.Info(ctx, "S3 domain deletion initiated", map[string]any{
		"audit_id":  result.Data.Audit.AuditID,
		"domain_id": domainID,
	})

	return &result, nil
}

// ConfirmS3DomainDeleted calls action-state module=domain, action=delete to verify
// the domain is removed. Idempotent when the domain is already gone.
func ConfirmS3DomainDeleted(c *client.Client, ctx context.Context, domainID string) error {
	tflog.Debug(ctx, "Confirming S3 domain deleted via action-state", map[string]any{
		"domain_id": domainID,
	})

	body := map[string]any{
		"resourceId": domainID,
	}

	_, err := common.UpdateActionState(ctx, c, actionStateModule, "delete", body)
	if err != nil {
		return fmt.Errorf("S3 domain delete confirmation failed: %w", err)
	}

	tflog.Info(ctx, "S3 domain delete confirmed", map[string]any{
		"domain_id": domainID,
	})

	return nil
}

// DeleteS3DomainAndWait deletes a domain, polls audit until complete, and confirms
// via action-state delete. If the domain is already gone (HTTP 404 on DELETE),
// action-state delete alone is used (idempotent destroy).
func DeleteS3DomainAndWait(c *client.Client, ctx context.Context, domainID string) error {
	tflog.Info(ctx, "Deleting S3 domain and waiting for completion", map[string]any{
		"domain_id": domainID,
	})

	deleteResp, err := DeleteS3Domain(c, ctx, domainID)
	if err != nil {
		if isDomainNotFoundError(err) {
			tflog.Warn(ctx, "S3 domain not found on DELETE; confirming via action-state delete", map[string]any{
				"domain_id": domainID,
			})
			return ConfirmS3DomainDeleted(c, ctx, domainID)
		}
		return err
	}

	_, err = c.WaitForAuditCompletion(ctx, deleteResp.Data.Audit.AuditID, "delete", actionStateModule, map[string]any{
		"resourceId": domainID,
	})
	if err != nil {
		return fmt.Errorf("S3 domain deletion failed: %w", err)
	}

	tflog.Info(ctx, "S3 domain deleted successfully", map[string]any{
		"audit_id":  deleteResp.Data.Audit.AuditID,
		"domain_id": domainID,
	})

	return nil
}
