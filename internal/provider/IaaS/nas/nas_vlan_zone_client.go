// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package nas

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

const nasVlanZoneCheckPath = "/isNasVlanZone"

// NasVlanZoneCheck holds the portal payload under CatalystResponse.data.
type NasVlanZoneCheck struct {
	IsNasVlanZone  bool    `json:"isNasVlanZone"`
	NasVlanZoneIDs []int64 `json:"nasVlanZoneIds"`
	RawBody        string
}

type catalystNasVlanEnvelope struct {
	Status       string          `json:"status"`
	Message      string          `json:"message"`
	ResponseCode int             `json:"responseCode"`
	Data         json.RawMessage `json:"data"`
}

// GetNasVlanZoneCheck calls GET .../nas/isNasVlanZone?engagementId=&endpointId= and parses CatalystResponse.data.
func GetNasVlanZoneCheck(c *client.Client, ctx context.Context, engagementID, endpointID int64) (*NasVlanZoneCheck, error) {
	path := fmt.Sprintf("%s%s?engagementId=%d&endpointId=%d",
		common.FileStorageServicePath, nasVlanZoneCheckPath, engagementID, endpointID)
	respBody, err := c.DoRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("NAS VLAN zone check: %w", err)
	}
	raw := string(respBody)
	var env catalystNasVlanEnvelope
	if err := json.Unmarshal(respBody, &env); err != nil {
		return nil, fmt.Errorf("parse NAS VLAN zone check envelope: %w", err)
	}
	st := strings.TrimSpace(strings.ToLower(env.Status))
	if st != "" && st != "success" {
		msg := strings.TrimSpace(env.Message)
		if msg == "" {
			msg = "unexpected status"
		}
		return nil, fmt.Errorf("NAS VLAN zone check failed: %s (responseCode=%d)", msg, env.ResponseCode)
	}

	out := &NasVlanZoneCheck{RawBody: raw}
	if len(env.Data) == 0 || string(env.Data) == "null" {
		out.IsNasVlanZone = false
		out.NasVlanZoneIDs = []int64{}
		return out, nil
	}
	if err := json.Unmarshal(env.Data, out); err != nil {
		return nil, fmt.Errorf("parse NAS VLAN zone check data: %w", err)
	}
	if out.NasVlanZoneIDs == nil {
		out.NasVlanZoneIDs = []int64{}
	}
	tflog.Info(ctx, "NAS VLAN zone check", map[string]any{
		"engagement_id": engagementID, "endpoint_id": endpointID,
		"is_nas_vlan_zone": out.IsNasVlanZone, "zone_count": len(out.NasVlanZoneIDs),
	})
	return out, nil
}
