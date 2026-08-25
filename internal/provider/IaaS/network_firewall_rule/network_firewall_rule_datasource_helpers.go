// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package network_firewall_rule

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// NetworkFirewallRuleListItemModel is one firewall rule in list data source output.
type NetworkFirewallRuleListItemModel struct {
	ID                   types.String `tfsdk:"id"`
	RuleName             types.String `tfsdk:"rule_name"`
	FirewallID           types.Int64  `tfsdk:"firewall_id"`
	Source               types.String `tfsdk:"source"`
	Action               types.String `tfsdk:"action"`
	SourceZoneID         types.Int64  `tfsdk:"source_zone_id"`
	SourceAddresses      types.List   `tfsdk:"source_addresses"`
	Destination          types.String `tfsdk:"destination"`
	DestinationZoneID    types.Int64  `tfsdk:"destination_zone_id"`
	DestinationAddresses types.List   `tfsdk:"destination_addresses"`
	ScheduleStartDate    types.String `tfsdk:"schedule_start_date"`
	ScheduleEndDate      types.String `tfsdk:"schedule_end_date"`
	Services             types.List   `tfsdk:"services"`
	Status               types.String `tfsdk:"status"`
}

func mapActionStateToRuleItemModel(ctx context.Context, state FirewallRuleActionState, configFirewallID int64) (NetworkFirewallRuleListItemModel, diag.Diagnostics) {
	var d diag.Diagnostics

	firewallID := state.FirewallID
	if firewallID == 0 {
		firewallID = configFirewallID
	}

	item := NetworkFirewallRuleListItemModel{
		ID:         types.StringValue(state.ID),
		RuleName:   types.StringValue(state.RuleName),
		FirewallID: types.Int64Value(firewallID),
		Source:     types.StringValue(state.Source),
		Action:     types.StringValue(state.Action),
		Destination: types.StringValue(state.Destination),
		Status:     types.StringValue(state.Status),
	}

	if state.SourceZoneID != nil {
		item.SourceZoneID = types.Int64Value(*state.SourceZoneID)
	} else {
		item.SourceZoneID = types.Int64Null()
	}
	if state.DestinationZoneID != nil {
		item.DestinationZoneID = types.Int64Value(*state.DestinationZoneID)
	} else {
		item.DestinationZoneID = types.Int64Null()
	}

	if state.ScheduleStartDate != "" {
		item.ScheduleStartDate = types.StringValue(state.ScheduleStartDate)
	} else {
		item.ScheduleStartDate = types.StringNull()
	}
	if state.ScheduleEndDate != "" {
		item.ScheduleEndDate = types.StringValue(state.ScheduleEndDate)
	} else {
		item.ScheduleEndDate = types.StringNull()
	}

	setStringListAttr(ctx, &d, &item.SourceAddresses, state.SourceAddresses)
	setStringListAttr(ctx, &d, &item.DestinationAddresses, state.DestinationAddresses)
	setStringListAttr(ctx, &d, &item.Services, state.Services)

	source := strings.ToLower(state.Source)
	destination := strings.ToLower(state.Destination)
	if source != "zone" && source != "nas" {
		item.SourceZoneID = types.Int64Null()
	}
	if destination != "zone" && destination != "nas" {
		item.DestinationZoneID = types.Int64Null()
	}

	return item, d
}
