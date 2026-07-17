// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package network_public_ip

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/client"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/common"
)

const (
	actionStateModulePublicIP = "publicIp"
	actionStateActionCreate   = "create"
	actionStateActionRead     = "read"
	actionStateActionDissociate = "delete"
)

// publicIPReadActionData is the `data` object for action-state read (module=publicIp).
type publicIPReadActionData struct {
	PublicIPDetail []struct {
		PrivateIP string `json:"privateIp"`
		PublicIP  string `json:"publicIp"`
	} `json:"publicIpDetail"`
}

// associatePublicIPRequest is the JSON body for the associate API.
type associatePublicIPRequest struct {
	PrivateIP            string `json:"privateIp"`
	PublicIPPricingModel string `json:"publicIpPricingModel"`
	RetainOnDissociate   bool   `json:"retainOnDissociate"`
	AllowFetchOrProvisionPublicIp bool `json:"allowFetchOrProvisionPublicIp"`
}

// dissociatePublicIPRequest is the JSON body for the dissociate API.
type dissociatePublicIPRequest struct {
	PrivateIP          string `json:"privateIp"`
	PublicIP           string `json:"publicIp"`
	RetainOnDissociate bool   `json:"retainOnDissociate"`
}

// AssociatePublicIP calls PUT .../publicIp/{resourceType}/{resourceId}/associate.
func AssociatePublicIP(c *client.Client, ctx context.Context, resourceType, resourceID string, body *associatePublicIPRequest) (*client.AuditResponse, error) {
	tflog.Debug(ctx, "Associating public IP", map[string]any{
		"resource_type": resourceType,
		"resource_id":   resourceID,
	})

	path := fmt.Sprintf("%s/publicIp/%s/%s/associate", common.NetworkOperationsPath, resourceType, url.PathEscape(resourceID))
	respBody, err := c.DoRequest(ctx, http.MethodPut, path, body)
	if err != nil {
		return nil, fmt.Errorf("failed to associate public IP: %w", err)
	}

	var result client.AuditResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse associate public IP response: %w", err)
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("associate public IP failed: %s (code: %d)", result.Message, result.ResponseCode)
	}

	tflog.Info(ctx, "Associate public IP initiated", map[string]any{
		"audit_id":      result.Data.Audit.AuditID,
		"resource_type": resourceType,
		"resource_id":   resourceID,
	})

	return &result, nil
}

// DissociatePublicIP calls PUT .../publicIp/{resourceType}/{resourceId}/dissociate.
func DissociatePublicIP(c *client.Client, ctx context.Context, resourceType, resourceID string, body *dissociatePublicIPRequest) (*client.AuditResponse, error) {
	tflog.Debug(ctx, "Dissociating public IP", map[string]any{
		"resource_type": resourceType,
		"resource_id":   resourceID,
		"private_ip":    body.PrivateIP,
		"public_ip":     body.PublicIP,
	})

	path := fmt.Sprintf("%s/publicIp/%s/%s/dissociate", common.NetworkOperationsPath, resourceType, url.PathEscape(resourceID))
	respBody, err := c.DoRequest(ctx, http.MethodPut, path, body)
	if err != nil {
		return nil, fmt.Errorf("failed to dissociate public IP: %w", err)
	}

	var result client.AuditResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse dissociate public IP response: %w", err)
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("dissociate public IP failed: %s (code: %d)", result.Message, result.ResponseCode)
	}

	tflog.Info(ctx, "Dissociate public IP initiated", map[string]any{
		"audit_id":      result.Data.Audit.AuditID,
		"resource_type": resourceType,
		"resource_id":   resourceID,
	})

	return &result, nil
}

func actionStateRequestBody(resourceType, privateIP string) map[string]any {
	return map[string]any{
		"resourceType": resourceType,
		"privateIp":    privateIP,
	}
}

// ReadPublicIPAssociation calls POST …/action-state?module=publicIp&action=read with resourceId, privateIp, and resourceType.
// It returns the public IP from the matching publicIpDetail entry.
func ReadPublicIPAssociation(c *client.Client, ctx context.Context, resourceID int64, resourceType, privateIP string) (string, error) {
	tflog.Debug(ctx, "Reading public IP via action-state", map[string]any{
		"resource_id":    resourceID,
		"resource_type":  resourceType,
		"private_ip":     privateIP,
	})

	body := map[string]any{
		"resourceId":   resourceID,
		"privateIp":    privateIP,
		"resourceType": resourceType,
	}

	actionResp, err := common.UpdateActionState(ctx, c, actionStateModulePublicIP, actionStateActionRead, body)
	if err != nil {
		return "", fmt.Errorf("public IP read (action-state): %w", err)
	}

	if !strings.EqualFold(actionResp.Status, "success") {
		return "", fmt.Errorf("public IP read failed: %s (code: %d)", actionResp.Message, actionResp.ResponseCode)
	}

	var data publicIPReadActionData
	if len(actionResp.Data) > 0 {
		if err := json.Unmarshal(actionResp.Data, &data); err != nil {
			return "", fmt.Errorf("parse public IP read data: %w", err)
		}
	}

	for _, d := range data.PublicIPDetail {
		if d.PrivateIP == privateIP && d.PublicIP != "" {
			return d.PublicIP, nil
		}
	}

	if len(data.PublicIPDetail) == 1 && data.PublicIPDetail[0].PublicIP != "" {
		return data.PublicIPDetail[0].PublicIP, nil
	}

	return "", fmt.Errorf("no publicIpDetail entry for private_ip %q", privateIP)
}

// AssociatePublicIPAndWait runs associate, waits for audit completion, then action-state (module=publicIp, action=create).
func AssociatePublicIPAndWait(c *client.Client, ctx context.Context, resourceType, resourceID string, body *associatePublicIPRequest) (*client.AuditLogResponse, error) {
	resp, err := AssociatePublicIP(c, ctx, resourceType, resourceID, body)
	if err != nil {
		return nil, err
	}

	reqBody := actionStateRequestBody(resourceType, body.PrivateIP)
	return c.WaitForAuditCompletion(ctx, resp.Data.Audit.AuditID, actionStateActionCreate, actionStateModulePublicIP, reqBody)
}

// DissociatePublicIPAndWait runs dissociate, waits for audit, then the same action-state call as associate (per platform contract).
func DissociatePublicIPAndWait(c *client.Client, ctx context.Context, resourceType, resourceID, privateIP, publicIP string, retainOnDissociate bool) (*client.AuditLogResponse, error) {
	body := &dissociatePublicIPRequest{
		PrivateIP:          privateIP,
		PublicIP:           publicIP,
		RetainOnDissociate: retainOnDissociate,
	}
	resp, err := DissociatePublicIP(c, ctx, resourceType, resourceID, body)
	if err != nil {
		return nil, err
	}

	reqBody := actionStateRequestBody(resourceType, privateIP)
	return c.WaitForAuditCompletion(ctx, resp.Data.Audit.AuditID, actionStateActionDissociate, actionStateModulePublicIP, reqBody)
}

// MapPricingModelToAPI converts Terraform pricing_model (case-insensitive) to API casing (e.g. Daily).
func MapPricingModelToAPI(pricingModel string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(pricingModel)) {
	case "daily":
		return "Daily", nil
	case "monthly":
		return "Monthly", nil
	case "reserved_1":
		return "Reserved_1", nil
	case "reserved_2":
		return "Reserved_2", nil
	case "reserved_3":
		return "Reserved_3", nil
	default:
		return "", fmt.Errorf("unsupported public_ip_pricing_model %q (expected daily or monthly)", pricingModel)
	}
}

// ListPublicIPsResponse is the API envelope for GET .../network/public-ips.
type ListPublicIPsResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Data    struct {
		Content []ListPublicIPItem `json:"content"`
	} `json:"data"`
}

// ListPublicIPItem is one element of data.content.
type ListPublicIPItem struct {
	PublicIPSegment string `json:"publicIpSegment"`
	IsUsed          bool   `json:"isUsed"`
	Location        string `json:"location"`
	Purpose         string `json:"purpose"`
	Description     string `json:"description"`
}

// ListPublicIPsByFirewall calls GET .../network/public-ips?firewall-ci={firewallID}.
func ListPublicIPsByFirewall(c *client.Client, ctx context.Context, firewallID int64) ([]ListPublicIPItem, error) {
	q := url.Values{}
	q.Set("firewall-ci", fmt.Sprintf("%d", firewallID))
	path := fmt.Sprintf("%s/public-ips?%s", common.NetworkServicePath, q.Encode())
	return listPublicIPs(c, ctx, path)
}

// ListPublicIPsByEngagement calls GET .../network/public-ips?engagement={engagementID}.
func ListPublicIPsByEngagement(c *client.Client, ctx context.Context, engagementID int64) ([]ListPublicIPItem, error) {
	q := url.Values{}
	q.Set("engagement", fmt.Sprintf("%d", engagementID))
	path := fmt.Sprintf("%s/public-ips?%s", common.NetworkServicePath, q.Encode())
	return listPublicIPs(c, ctx, path)
}

func listPublicIPs(c *client.Client, ctx context.Context, path string) ([]ListPublicIPItem, error) {
	tflog.Debug(ctx, "Listing public IPs", map[string]any{"path": path})

	respBody, err := c.DoRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var parsed ListPublicIPsResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse public IPs list: %w", err)
	}

	if strings.ToLower(parsed.Status) != "success" {
		return nil, fmt.Errorf("list public IPs failed: %s", parsed.Message)
	}

	return parsed.Data.Content, nil
}
