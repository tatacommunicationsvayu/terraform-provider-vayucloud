// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package network_firewall_rule

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/client"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/network_firewall"
)

var _ datasource.DataSource = &NetworkFirewallRuleListDataSource{}

// NewNetworkFirewallRuleListDataSource lists firewall rules via action-state (module=firewallRule, action=list).
func NewNetworkFirewallRuleListDataSource() datasource.DataSource {
	return &NetworkFirewallRuleListDataSource{}
}

// NetworkFirewallRuleListDataSource lists rules for a firewall.
type NetworkFirewallRuleListDataSource struct {
	client *client.Client
}

// NetworkFirewallRuleListDataSourceModel is the data source root model.
type NetworkFirewallRuleListDataSourceModel struct {
	ID types.String `tfsdk:"id"`

	FirewallID types.Int64          `tfsdk:"firewall_id"`
	Filters    []client.FilterModel `tfsdk:"filter"`

	Rules []NetworkFirewallRuleListItemModel `tfsdk:"rules"`

	Status       types.String `tfsdk:"status"`
	Message      types.String `tfsdk:"message"`
	ResponseCode types.Int64  `tfsdk:"response_code"`
}

func (d *NetworkFirewallRuleListDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network_firewall_rule_list"
}

func (d *NetworkFirewallRuleListDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Lists network firewall rules for a firewall from the VayuCloud action-state API.",
		MarkdownDescription: "Lists network firewall rules for a firewall.\n\nThis data source calls `GET /network_operations/list-firewallrule-state/{firewallId}`. Each rule in `rules` exposes the same fields as [`vayucloud_network_firewall_rule`](network_firewall_rule.md) read.\n\nOptionally, use `filter` blocks to narrow results client-side.",

		Blocks: map[string]schema.Block{
			"filter": client.FilterBlockSchema(),
		},

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Same as `firewall_id` as a string (for Terraform).",
				Computed:    true,
			},
			"firewall_id": schema.Int64Attribute{
				Description: "Firewall resource ID to list rules for.",
				Required:    true,
			},
			"rules": schema.ListNestedAttribute{
				Description: "Firewall rules returned by the list action.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: ruleListAttributeSchema(),
				},
			},
			"status": schema.StringAttribute{
				Description: "Top-level API response status (e.g. success).",
				Computed:    true,
			},
			"message": schema.StringAttribute{
				Description: "API message text.",
				Computed:    true,
			},
			"response_code": schema.Int64Attribute{
				Description: "API response code (`0` indicates success).",
				Computed:    true,
			},
		},
	}
}

func (d *NetworkFirewallRuleListDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *NetworkFirewallRuleListDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data NetworkFirewallRuleListDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if d.client == nil {
		resp.Diagnostics.AddError("Client not configured", "Expected configured API client")
		return
	}

	firewallID := data.FirewallID.ValueInt64()

	if err := network_firewall.ValidateFirewallExists(d.client, ctx, firewallID); err != nil {
		resp.Diagnostics.AddError("Invalid Firewall ID", err.Error())
		return
	}

	tflog.Debug(ctx, "Reading network firewall rule list data source", map[string]any{
		"firewall_id": firewallID,
	})

	items, actionResp, err := ListNetworkFirewallRules(d.client, ctx, firewallID)
	if err != nil {
		resp.Diagnostics.AddError("Error Listing Network Firewall Rules", err.Error())
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("%d", firewallID))
	data.Status = types.StringValue(actionResp.Status)
	data.Message = types.StringValue(actionResp.Message)
	data.ResponseCode = types.Int64Value(int64(actionResp.ResponseCode))

	rules := make([]NetworkFirewallRuleListItemModel, 0, len(items))
	for _, itemMap := range items {
		state := MapFirewallRuleFromResponseMap(itemMap)
		rule, diags := mapActionStateToRuleItemModel(ctx, state, firewallID)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		rules = append(rules, rule)
	}

	if len(data.Filters) > 0 {
		tflog.Debug(ctx, "Applying filters to network firewall rules", map[string]any{
			"filter_count":     len(data.Filters),
			"pre_filter_count": len(rules),
		})

		filtered, filterErr := client.ApplyFilters(rules, data.Filters)
		if filterErr != nil {
			resp.Diagnostics.AddError("Error Filtering Network Firewall Rules", filterErr.Error())
			return
		}
		rules = filtered

		tflog.Debug(ctx, "Filters applied to network firewall rules", map[string]any{
			"post_filter_count": len(rules),
		})
	}

	data.Rules = rules

	tflog.Info(ctx, "Network firewall rule list read successfully", map[string]any{
		"count":       len(rules),
		"firewall_id": firewallID,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
