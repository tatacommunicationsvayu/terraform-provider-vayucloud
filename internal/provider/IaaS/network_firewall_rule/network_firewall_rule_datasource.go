// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package network_firewall_rule

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/client"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/network_firewall"
)

var _ datasource.DataSource = &NetworkFirewallRuleDataSource{}

// NewNetworkFirewallRuleDataSource reads one firewall rule via action-state (module=firewallRule, action=read).
func NewNetworkFirewallRuleDataSource() datasource.DataSource {
	return &NetworkFirewallRuleDataSource{}
}

// NetworkFirewallRuleDataSource reads rule state for a firewall and rule ID.
type NetworkFirewallRuleDataSource struct {
	client *client.Client
}

// NetworkFirewallRuleDataSourceModel is the data source root model.
type NetworkFirewallRuleDataSourceModel struct {
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

	Message      types.String `tfsdk:"message"`
	ResponseCode types.Int64  `tfsdk:"response_code"`
	RawResponse  types.String `tfsdk:"raw_response"`
}

func (d *NetworkFirewallRuleDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network_firewall_rule"
}

func ruleListAttributeSchema() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Description: "Rule identifier on the firewall.",
			Computed:    true,
		},
		"rule_name": schema.StringAttribute{
			Description: "Rule name from the platform.",
			Computed:    true,
		},
		"firewall_id": schema.Int64Attribute{
			Description: "Firewall resource ID.",
			Computed:    true,
		},
		"source": schema.StringAttribute{
			Description: "Source type (`internet`, `zone`, or `nas`).",
			Computed:    true,
		},
		"action": schema.StringAttribute{
			Description: "Rule action (`allow` or `deny`).",
			Computed:    true,
		},
		"source_zone_id": schema.Int64Attribute{
			Description: "Source zone ID when applicable.",
			Computed:    true,
		},
		"source_addresses": schema.ListAttribute{
			Description: "Source addresses in CIDR notation.",
			Computed:    true,
			ElementType: types.StringType,
		},
		"destination": schema.StringAttribute{
			Description: "Destination type (`internet`, `zone`, `nas`, `vcs`, or `load_balancer`).",
			Computed:    true,
		},
		"destination_zone_id": schema.Int64Attribute{
			Description: "Destination zone ID when applicable.",
			Computed:    true,
		},
		"destination_addresses": schema.ListAttribute{
			Description: "Destination addresses in CIDR notation.",
			Computed:    true,
			ElementType: types.StringType,
		},
		"schedule_start_date": schema.StringAttribute{
			Description: "Schedule start date (`YYYY-MM-DD`) when set.",
			Computed:    true,
		},
		"schedule_end_date": schema.StringAttribute{
			Description: "Schedule end date (`YYYY-MM-DD`) when set.",
			Computed:    true,
		},
		"services": schema.ListAttribute{
			Description: "Service names for the rule.",
			Computed:    true,
			ElementType: types.StringType,
		},
		"status": schema.StringAttribute{
			Description: "Rule status from the platform.",
			Computed:    true,
		},
	}
}

func (d *NetworkFirewallRuleDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	attrs := ruleListAttributeSchema()
	attrs["firewall_id"] = schema.Int64Attribute{
		Description: "Firewall resource ID the rule belongs to.",
		Required:    true,
	}
	attrs["id"] = schema.StringAttribute{
		Description: "Rule identifier on the firewall.",
		Required:    true,
	}
	attrs["message"] = schema.StringAttribute{
		Description: "API message text.",
		Computed:    true,
	}
	attrs["response_code"] = schema.Int64Attribute{
		Description: "API response code (`0` indicates success).",
		Computed:    true,
	}
	attrs["raw_response"] = schema.StringAttribute{
		Description: "Raw JSON `data` payload from the action-state response (for debugging).",
		Computed:    true,
	}

	resp.Schema = schema.Schema{
		Description:         "Reads a network firewall rule from the VayuCloud action-state API.",
		MarkdownDescription: "Reads a network firewall rule for a firewall and rule ID.\n\nThis data source calls the action-state API with `module=firewallRule` and `action=read` — the same flow as the [`vayucloud_network_firewall_rule`](../resources/network_firewall_rule.md) resource read operation.",
		Attributes:          attrs,
	}
}

func (d *NetworkFirewallRuleDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	cl, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T.", req.ProviderData),
		)
		return
	}
	d.client = cl
}

func (d *NetworkFirewallRuleDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data NetworkFirewallRuleDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if d.client == nil {
		resp.Diagnostics.AddError("Client not configured", "Expected configured API client")
		return
	}

	firewallID := data.FirewallID.ValueInt64()
	ruleID := data.ID.ValueString()

	if err := network_firewall.ValidateFirewallExists(d.client, ctx, firewallID); err != nil {
		resp.Diagnostics.AddError("Invalid Firewall ID", err.Error())
		return
	}

	tflog.Debug(ctx, "Reading network firewall rule data source", map[string]any{
		"firewall_id": firewallID,
		"rule_id":     ruleID,
	})

	actionResp, err := ReadNetworkFirewallRule(d.client, ctx, firewallID, ruleID)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Network Firewall Rule", err.Error())
		return
	}

	data.Message = types.StringValue(actionResp.Message)
	data.ResponseCode = types.Int64Value(int64(actionResp.ResponseCode))

	raw, err := json.Marshal(actionResp.Data)
	if err != nil {
		resp.Diagnostics.AddError("Error Encoding Raw Response", err.Error())
		return
	}
	data.RawResponse = types.StringValue(string(raw))

	if len(actionResp.Data) == 0 {
		resp.Diagnostics.AddError("Error Reading Network Firewall Rule", "Response data is empty")
		return
	}

	var responseMap map[string]interface{}
	if err := json.Unmarshal(actionResp.Data, &responseMap); err != nil {
		var asString string
		if err2 := json.Unmarshal(actionResp.Data, &asString); err2 == nil && strings.TrimSpace(asString) != "" {
			resp.Diagnostics.AddError(
				"Error Reading Network Firewall Rule",
				fmt.Sprintf("Platform returned an error instead of rule state: %s", asString),
			)
			return
		}
		resp.Diagnostics.AddError(
			"Error Unmarshalling Response Data",
			fmt.Sprintf("Could not unmarshal response data: %s", err.Error()),
		)
		return
	}

	state := MapFirewallRuleFromResponseMap(responseMap)
	item, diags := mapActionStateToRuleItemModel(ctx, state, firewallID)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.ID = item.ID
	data.RuleName = item.RuleName
	data.FirewallID = item.FirewallID
	data.Source = item.Source
	data.Action = item.Action
	data.SourceZoneID = item.SourceZoneID
	data.SourceAddresses = item.SourceAddresses
	data.Destination = item.Destination
	data.DestinationZoneID = item.DestinationZoneID
	data.DestinationAddresses = item.DestinationAddresses
	data.ScheduleStartDate = item.ScheduleStartDate
	data.ScheduleEndDate = item.ScheduleEndDate
	data.Services = item.Services
	data.Status = item.Status

	tflog.Info(ctx, "Network firewall rule data source read successfully", map[string]any{
		"id":          data.ID.ValueString(),
		"firewall_id": firewallID,
		"rule_name":   data.RuleName.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
