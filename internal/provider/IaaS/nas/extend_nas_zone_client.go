// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package nas

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

const extendNASZonePath = "/extendNASZone/%d/%d"
const deconfigureNASZonePath = "/deconfigureNASZone/%d"

// ticketIDQueryParam matches portal Constants.TICKET_ID ("ticketId").
const ticketIDQueryParam = "ticketId"

type extendNASZoneCatalyst struct {
	Status       string          `json:"status"`
	Message      string          `json:"message"`
	ResponseCode int             `json:"responseCode"`
	Data         json.RawMessage `json:"data"`
}

// ExtendNASZone calls POST .../nas/extendNASZone/{zoneId}/{vserverId} with JSON body
// {"isFirewallConfigured": <bool>} and optional ticketId query parameter for audit correlation.
func ExtendNASZone(c *client.Client, ctx context.Context, zoneID, vserverID int64, isFirewallConfigured bool, ticketID string) (rawBody string, err error) {
	rel := fmt.Sprintf(common.FileStorageServicePath+extendNASZonePath, zoneID, vserverID)
	if strings.TrimSpace(ticketID) != "" {
		rel = rel + "?" + ticketIDQueryParam + "=" + url.QueryEscape(strings.TrimSpace(ticketID))
	}
	payload := map[string]bool{
		"isFirewallConfigured": isFirewallConfigured,
	}
	respBody, err := c.DoRequest(ctx, http.MethodPost, rel, payload)
	if err != nil {
		return "", fmt.Errorf("extend NAS zone: %w", err)
	}
	raw := string(respBody)
	var env extendNASZoneCatalyst
	if jErr := json.Unmarshal(respBody, &env); jErr != nil {
		return raw, fmt.Errorf("parse extend NAS zone response: %w", jErr)
	}
	st := strings.TrimSpace(strings.ToLower(env.Status))
	if st != "" && st != "success" {
		msg := strings.TrimSpace(env.Message)
		if msg == "" {
			msg = "extend NAS zone failed"
		}
		return raw, fmt.Errorf("%s (status=%q)", msg, env.Status)
	}
	if err := extendNASZoneCheckInnerData(env.Data); err != nil {
		return raw, err
	}
	tflog.Info(ctx, "extend NAS zone accepted", map[string]any{"zone_id": zoneID, "vserver_id": vserverID})
	return raw, nil
}

// extendNASZoneCheckInnerData inspects portal JSONObject payload for known failure strings.
func extendNASZoneCheckInnerData(data json.RawMessage) error {
	if len(data) == 0 || string(data) == "null" {
		return nil
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return nil
	}
	v, ok := m["data"]
	if !ok {
		return nil
	}
	s, ok := v.(string)
	if !ok {
		return nil
	}
	low := strings.ToLower(strings.TrimSpace(s))
	if low == "" {
		return nil
	}
	if strings.Contains(low, "error in extend nas zone") || strings.HasPrefix(low, "error") {
		return fmt.Errorf("extend NAS zone: %s", strings.TrimSpace(s))
	}
	if strings.Contains(low, "no freeips") {
		return fmt.Errorf("extend NAS zone: %s", strings.TrimSpace(s))
	}
	return nil
}

// DeconfigureNASZone calls DELETE .../nas/deconfigureNASZone/{zoneId} with optional ticketId query parameter.
// This is deconfigure-only (portal NAS teardown workflow); it does not delete the zone record.
// Call before deleting the NAS vserver when the zone was extended via extendNASZone.
func DeconfigureNASZone(c *client.Client, ctx context.Context, zoneID int64, ticketID string) (rawBody string, err error) {
	rel := fmt.Sprintf(common.FileStorageServicePath+deconfigureNASZonePath, zoneID)
	if strings.TrimSpace(ticketID) != "" {
		rel = rel + "?" + ticketIDQueryParam + "=" + url.QueryEscape(strings.TrimSpace(ticketID))
	}
	respBody, err := c.DoRequest(ctx, http.MethodDelete, rel, nil)
	if err != nil {
		return "", fmt.Errorf("deconfigure NAS zone: %w", err)
	}
	raw := string(respBody)
	var env extendNASZoneCatalyst
	if jErr := json.Unmarshal(respBody, &env); jErr != nil {
		return raw, fmt.Errorf("parse deconfigure NAS zone response: %w", jErr)
	}
	st := strings.TrimSpace(strings.ToLower(env.Status))
	if st != "" && st != "success" {
		msg := strings.TrimSpace(env.Message)
		if msg == "" {
			msg = "deconfigure NAS zone failed"
		}
		return raw, fmt.Errorf("%s (status=%q)", msg, env.Status)
	}
	if err := deconfigureNASZoneCheckInnerData(env.Data); err != nil {
		return raw, err
	}
	tflog.Info(ctx, "deconfigure NAS zone accepted", map[string]any{"zone_id": zoneID})
	return raw, nil
}

// deconfigureNASZoneCheckInnerData inspects portal JSONObject payload for failure text.
func deconfigureNASZoneCheckInnerData(data json.RawMessage) error {
	if len(data) == 0 || string(data) == "null" {
		return nil
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return nil
	}
	v, ok := m["data"]
	if !ok {
		return nil
	}
	s, ok := v.(string)
	if !ok {
		return nil
	}
	low := strings.ToLower(strings.TrimSpace(s))
	if low == "" {
		return nil
	}
	if strings.Contains(low, "error") || strings.Contains(low, "failed") {
		return fmt.Errorf("deconfigure NAS zone: %s", strings.TrimSpace(s))
	}
	return nil
}
