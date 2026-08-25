// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package security_group

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/client"
)

var _ datasource.DataSource = &ListSecurityGroupRulesDataSource{}

// NewListSecurityGroupRulesDataSource lists rules for a security group.
func NewListSecurityGroupRulesDataSource() datasource.DataSource {
	return &ListSecurityGroupRulesDataSource{}
}

// ListSecurityGroupRulesDataSource implements vayucloud_list_security_group_rules.
type ListSecurityGroupRulesDataSource struct {
	client *client.Client
}

// ListSecurityGroupRulesDataSourceModel is the Terraform model.
type ListSecurityGroupRulesDataSourceModel struct {
	ID              types.String                     `tfsdk:"id"`
	FirewallID      types.Int64                      `tfsdk:"firewall_id"`
	SecurityGroupID types.String                     `tfsdk:"security_group_id"`
	Filters         []client.FilterModel             `tfsdk:"filter"`
	Rules           []ListSecurityGroupRuleItemModel `tfsdk:"rules"`
}

// ListSecurityGroupRuleItemModel is one rule from the rules list API.
type ListSecurityGroupRuleItemModel struct {
	ID              types.String `tfsdk:"id"`
	SecurityGroupID types.String `tfsdk:"security_group_id"`
	Protocol        types.String `tfsdk:"protocol"`
	EtherType       types.String `tfsdk:"ether_type"`
	Direction       types.String `tfsdk:"direction"`
	PortRangeMin    types.Int64  `tfsdk:"port_range_min"`
	PortRangeMax    types.Int64  `tfsdk:"port_range_max"`
	RemoteIPPrefix  types.String `tfsdk:"remote_ip_prefix"`
	RemoteGroupID   types.String `tfsdk:"remote_group_id"`
	Description     types.String `tfsdk:"description"`
}

func (d *ListSecurityGroupRulesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_list_security_group_rules"
}

func (d *ListSecurityGroupRulesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists security group rules for a security group under a firewall.",
		MarkdownDescription: "Lists rules via " +
			"`GET .../security-group/rules/{firewall_id}/{security_group_id}`.\n\n" +
			"Optionally, use `filter` blocks to narrow results client-side " +
			"(e.g. `protocol`, `direction`, `id`).",
		Blocks: map[string]schema.Block{
			"filter": client.FilterBlockSchema(),
		},
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Placeholder identifier (`firewall_id/security_group_id`).",
				Computed:    true,
			},
			"firewall_id": schema.Int64Attribute{
				Description: "Firewall ID that owns the security group.",
				Required:    true,
			},
			"security_group_id": schema.StringAttribute{
				Description: "Security group UUID whose rules should be listed.",
				Required:    true,
			},
			"rules": schema.ListNestedAttribute{
				Description: "Rules returned by the API.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "Rule UUID.",
							Computed:    true,
						},
						"security_group_id": schema.StringAttribute{
							Description: "Parent security group UUID.",
							Computed:    true,
						},
						"protocol": schema.StringAttribute{
							Description: "Protocol (tcp, udp, icmp, …).",
							Computed:    true,
						},
						"ether_type": schema.StringAttribute{
							Description: "Ether type (IPv4 / IPv6).",
							Computed:    true,
						},
						"direction": schema.StringAttribute{
							Description: "Direction (ingress / egress).",
							Computed:    true,
						},
						"port_range_min": schema.Int64Attribute{
							Description: "Minimum port (null when not set).",
							Computed:    true,
						},
						"port_range_max": schema.Int64Attribute{
							Description: "Maximum port (null when not set).",
							Computed:    true,
						},
						"remote_ip_prefix": schema.StringAttribute{
							Description: "Remote CIDR (null when using a remote group).",
							Computed:    true,
						},
						"remote_group_id": schema.StringAttribute{
							Description: "Remote security group UUID (null when using a CIDR).",
							Computed:    true,
						},
						"description": schema.StringAttribute{
							Description: "Rule description.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *ListSecurityGroupRulesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T.", req.ProviderData),
		)
		return
	}
	d.client = c
}

func (d *ListSecurityGroupRulesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ListSecurityGroupRulesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	firewallID := data.FirewallID.ValueInt64()
	sgID := data.SecurityGroupID.ValueString()
	tflog.Debug(ctx, "Reading list_security_group_rules data source", map[string]any{
		"firewall_id":       firewallID,
		"security_group_id": sgID,
	})

	items, err := ListSecurityGroupRules(d.client, ctx, firewallID, sgID)
	if err != nil {
		resp.Diagnostics.AddError("Error Listing Security Group Rules", err.Error())
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("%d/%s", firewallID, sgID))
	rules := make([]ListSecurityGroupRuleItemModel, 0, len(items))
	for _, rule := range items {
		rules = append(rules, ListSecurityGroupRuleItemModel{
			ID:              types.StringValue(rule.ID),
			SecurityGroupID: types.StringValue(rule.SecurityGroupID),
			Protocol:        types.StringValue(rule.Protocol),
			EtherType:       types.StringValue(rule.EtherType),
			Direction:       types.StringValue(rule.Direction),
			PortRangeMin:    optionalInt64(rule.PortRangeMin),
			PortRangeMax:    optionalInt64(rule.PortRangeMax),
			RemoteIPPrefix:  optionalString(rule.RemoteIPPrefix),
			RemoteGroupID:   optionalString(rule.RemoteGroupID),
			Description:     types.StringValue(rule.Description),
		})
	}

	if len(data.Filters) > 0 {
		tflog.Debug(ctx, "Applying filters to security group rules", map[string]any{
			"filter_count":     len(data.Filters),
			"pre_filter_count": len(rules),
		})
		filtered, filterErr := client.ApplyFilters(rules, data.Filters)
		if filterErr != nil {
			resp.Diagnostics.AddError("Error Filtering Security Group Rules", filterErr.Error())
			return
		}
		rules = filtered
		tflog.Debug(ctx, "Filters applied to security group rules", map[string]any{
			"post_filter_count": len(rules),
		})
	}

	data.Rules = rules

	tflog.Info(ctx, "Read list_security_group_rules data source", map[string]any{
		"firewall_id":       firewallID,
		"security_group_id": sgID,
		"count":             len(rules),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
