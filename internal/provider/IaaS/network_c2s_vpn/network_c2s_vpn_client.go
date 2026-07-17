// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package network_c2s_vpn

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/client"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/common"
)

// CreateC2SVPNRequest is the body for initial VPN creation (single user).
type CreateC2SVPNRequest struct {
	PricingModel string          `json:"pricingModel"`
	User         c2svpnUserCreds `json:"user"`
}

type c2svpnUserCreds struct {
	Passwd string `json:"passwd"`
	Name   string `json:"name"`
}

// VPNUserEntry is a username/password pair for create/add/reset operations.
type VPNUserEntry struct {
	Passwd string `json:"passwd"`
	Name   string `json:"name"`
}

// VPNUsersCreateRequest is the body for adding VPN users.
type VPNUsersCreateRequest struct {
	Operation string         `json:"operation"`
	Users     []VPNUserEntry `json:"users"`
}

// VPNUsersResetRequest is the body for resetting VPN user passwords.
type VPNUsersResetRequest struct {
	Operation string         `json:"operation"`
	Users     []VPNUserEntry `json:"users"`
}

// VPNUserNameOnly identifies a user by name (delete user).
type VPNUserNameOnly struct {
	Name string `json:"name"`
}

// VPNUsersDeleteRequest is the body for removing VPN users.
type VPNUsersDeleteRequest struct {
	Users []VPNUserNameOnly `json:"users"`
}

// C2SVPNReadData is the `data` object from action-state read (module=c2svpn).
type C2SVPNReadData struct {
	PreSharedKey string `json:"pre_shared_key"`
	VPNName      string `json:"vpn_name"`
	ResourceID   int64  `json:"resourceId"`
	VPNStatus    string `json:"vpn_status"`
	VPNIP        string `json:"vpn_ip"`
	PeerID       string `json:"peer_id"`
	VPNNoOfUsers int    `json:"vpn_no_of_users"`
	Users        []struct {
		Name string `json:"name"`
	} `json:"users"`
}

// EffectiveVPNUserCount returns the user count from read data (vpn_no_of_users), or len(users) if the count is absent/zero.
func (d *C2SVPNReadData) EffectiveVPNUserCount() int {
	if d.VPNNoOfUsers > 0 {
		return d.VPNNoOfUsers
	}
	return len(d.Users)
}

// CreateC2SVPN starts VPN provisioning for a firewall.
//
// POST {APIURL}/uat-networkservice/network_operations/vpn/{firewallId}
func CreateC2SVPN(c *client.Client, ctx context.Context, firewallID int64, req *CreateC2SVPNRequest) (*client.AuditResponse, error) {
	tflog.Debug(ctx, "Creating C2S VPN", map[string]any{
		"firewall_id": firewallID,
	})

	path := fmt.Sprintf("%s/vpn/%d", common.NetworkOperationsPath, firewallID)
	respBody, err := c.DoRequest(ctx, http.MethodPost, path, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create C2S VPN: %w", err)
	}

	var result client.AuditResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse create C2S VPN response: %w", err)
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("C2S VPN creation failed: %s (code: %d)", result.Message, result.ResponseCode)
	}

	tflog.Info(ctx, "C2S VPN creation initiated", map[string]any{
		"audit_id":    result.Data.Audit.AuditID,
		"firewall_id": firewallID,
	})

	return &result, nil
}

// CreateC2SVPNAndWait creates VPN and waits for audit + action-state (module=c2svpn, action=create).
func CreateC2SVPNAndWait(c *client.Client, ctx context.Context, firewallID int64, req *CreateC2SVPNRequest) (*client.AuditLogResponse, error) {
	createResp, err := CreateC2SVPN(c, ctx, firewallID, req)
	if err != nil {
		return nil, err
	}

	requestBody := map[string]any{
		"vpnNoOfUsers": 1,
	}

	auditLog, err := c.WaitForAuditCompletion(ctx, createResp.Data.Audit.AuditID, "create", "c2svpn", requestBody)
	if err != nil {
		return nil, fmt.Errorf("C2S VPN creation failed: %w", err)
	}

	tflog.Info(ctx, "C2S VPN created successfully", map[string]any{
		"audit_id":    createResp.Data.Audit.AuditID,
		"firewall_id": firewallID,
	})

	return auditLog, nil
}

// ReadC2SVPN reads VPN state via common.UpdateActionState (POST …/configservice/action-state?module=c2svpn&action=read).
func ReadC2SVPN(c *client.Client, ctx context.Context, resourceID int64) (*common.ActionStateResponse, *C2SVPNReadData, error) {
	tflog.Debug(ctx, "Reading C2S VPN action state", map[string]any{
		"resource_id": resourceID,
	})

	body := map[string]any{
		"resourceId": resourceID,
	}

	actionResp, err := common.UpdateActionState(ctx, c, "c2svpn", "read", body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read C2S VPN: %w", err)
	}

	if actionResp.Status != "success" {
		return actionResp, nil, fmt.Errorf("C2S VPN read failed: %s (code: %d)", actionResp.Message, actionResp.ResponseCode)
	}

	var data C2SVPNReadData
	if len(actionResp.Data) > 0 {
		if err := json.Unmarshal(actionResp.Data, &data); err != nil {
			return actionResp, nil, fmt.Errorf("failed to parse C2S VPN read data: %w", err)
		}
	}

	tflog.Info(ctx, "C2S VPN read successfully", map[string]any{
		"resource_id": resourceID,
	})

	return actionResp, &data, nil
}

// AddVPNUsers adds users (operation=create).
//
// POST {APIURL}/uat-networkservice/network_operations/vpn/users/{firewallId}
func AddVPNUsers(c *client.Client, ctx context.Context, firewallID int64, users []VPNUserEntry) (*client.AuditResponse, error) {
	req := VPNUsersCreateRequest{
		Operation: "create",
		Users:     users,
	}
	return postVPNUsers(c, ctx, firewallID, req)
}

// ResetVPNUserPasswords resets passwords (operation=reset).
func ResetVPNUserPasswords(c *client.Client, ctx context.Context, firewallID int64, users []VPNUserEntry) (*client.AuditResponse, error) {
	req := VPNUsersResetRequest{
		Operation: "reset",
		Users:     users,
	}
	return postVPNUsers(c, ctx, firewallID, req)
}

func postVPNUsers(c *client.Client, ctx context.Context, firewallID int64, body any) (*client.AuditResponse, error) {
	path := fmt.Sprintf("%s/vpn/users/%d", common.NetworkOperationsPath, firewallID)
	respBody, err := c.DoRequest(ctx, http.MethodPost, path, body)
	if err != nil {
		return nil, fmt.Errorf("failed to call VPN users API: %w", err)
	}

	var result client.AuditResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse VPN users response: %w", err)
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("VPN users operation failed: %s (code: %d)", result.Message, result.ResponseCode)
	}

	tflog.Info(ctx, "VPN users operation initiated", map[string]any{
		"audit_id":    result.Data.Audit.AuditID,
		"firewall_id": firewallID,
	})

	return &result, nil
}

// RemoveVPNUsers removes users by name.
//
// POST {APIURL}/uat-networkservice/network_operations/vpn/users/{firewallId}
func RemoveVPNUsers(c *client.Client, ctx context.Context, firewallID int64, names []string) (*client.AuditResponse, error) {
	users := make([]VPNUserNameOnly, 0, len(names))
	for _, n := range names {
		users = append(users, VPNUserNameOnly{Name: n})
	}
	body := VPNUsersDeleteRequest{Users: users}
	path := fmt.Sprintf("%s/vpn/users/%d", common.NetworkOperationsPath, firewallID)

	respBody, err := c.DoRequest(ctx, http.MethodDelete, path, body)
	if err != nil {
		return nil, fmt.Errorf("failed to remove VPN users: %w", err)
	}

	var result client.AuditResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse remove VPN users response: %w", err)
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("remove VPN users failed: %s (code: %d)", result.Message, result.ResponseCode)
	}

	tflog.Info(ctx, "VPN user removal initiated", map[string]any{
		"audit_id":    result.Data.Audit.AuditID,
		"firewall_id": firewallID,
	})

	return &result, nil
}

func waitVPNAudit(c *client.Client, ctx context.Context, auditID string, action string, vpnNoOfUsers int) (*client.AuditLogResponse, error) {
	requestBody := map[string]any{
		"vpnNoOfUsers": vpnNoOfUsers,
	}
	return c.WaitForAuditCompletion(ctx, auditID, action, "c2svpn", requestBody)
}

// AddVPNUsersAndWait adds users and waits for audit + action-state (action=update).
func AddVPNUsersAndWait(c *client.Client, ctx context.Context, firewallID int64, users []VPNUserEntry, vpnNoOfUsers int) (*client.AuditLogResponse, error) {
	resp, err := AddVPNUsers(c, ctx, firewallID, users)
	if err != nil {
		return nil, err
	}
	auditLog, err := waitVPNAudit(c, ctx, resp.Data.Audit.AuditID, "create", vpnNoOfUsers)
	if err != nil {
		return nil, fmt.Errorf("add VPN users failed: %w", err)
	}
	return auditLog, nil
}

// ResetVPNUserPasswordsAndWait resets passwords and waits.
func ResetVPNUserPasswordsAndWait(c *client.Client, ctx context.Context, firewallID int64, users []VPNUserEntry, vpnNoOfUsers int) (*client.AuditLogResponse, error) {
	resp, err := ResetVPNUserPasswords(c, ctx, firewallID, users)
	if err != nil {
		return nil, err
	}
	auditLog, err := waitVPNAudit(c, ctx, resp.Data.Audit.AuditID, "create", vpnNoOfUsers)
	if err != nil {
		return nil, fmt.Errorf("reset VPN user passwords failed: %w", err)
	}
	return auditLog, nil
}

// RemoveVPNUsersAndWait removes users and waits.
func RemoveVPNUsersAndWait(c *client.Client, ctx context.Context, firewallID int64, names []string, vpnNoOfUsers int) (*client.AuditLogResponse, error) {
	resp, err := RemoveVPNUsers(c, ctx, firewallID, names)
	if err != nil {
		return nil, err
	}
	auditLog, err := waitVPNAudit(c, ctx, resp.Data.Audit.AuditID, "create", vpnNoOfUsers)
	if err != nil {
		return nil, fmt.Errorf("remove VPN users failed: %w", err)
	}
	return auditLog, nil
}

// DeleteC2SVPN deletes the VPN for a firewall.
//
// DELETE {APIURL}/uat-networkservice/network_operations/vpn/{firewallId}
func DeleteC2SVPN(c *client.Client, ctx context.Context, firewallID int64) (*client.AuditResponse, error) {
	path := fmt.Sprintf("%s/vpn/%d", common.NetworkOperationsPath, firewallID)
	respBody, err := c.DoRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to delete C2S VPN: %w", err)
	}

	var result client.AuditResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse delete C2S VPN response: %w", err)
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("C2S VPN deletion failed: %s (code: %d)", result.Message, result.ResponseCode)
	}

	tflog.Info(ctx, "C2S VPN deletion initiated", map[string]any{
		"audit_id":    result.Data.Audit.AuditID,
		"firewall_id": firewallID,
	})

	return &result, nil
}

// DeleteC2SVPNAndWait deletes VPN and waits for audit + action-state (action=delete).
func DeleteC2SVPNAndWait(c *client.Client, ctx context.Context, firewallID int64) (*client.AuditLogResponse, error) {
	delResp, err := DeleteC2SVPN(c, ctx, firewallID)
	if err != nil {
		return nil, err
	}

	var requestBody any = nil
	auditLog, err := c.WaitForAuditCompletion(ctx, delResp.Data.Audit.AuditID, "delete", "c2svpn", requestBody)
	if err != nil {
		return nil, fmt.Errorf("C2S VPN deletion failed: %w", err)
	}

	tflog.Info(ctx, "C2S VPN deleted successfully", map[string]any{
		"audit_id":    delResp.Data.Audit.AuditID,
		"firewall_id": firewallID,
	})

	return auditLog, nil
}
