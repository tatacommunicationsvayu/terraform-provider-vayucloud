// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package network_public_ip

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/client"
)

var _ datasource.DataSource = &NetworkPublicIPsDataSource{}
var _ datasource.DataSourceWithConfigValidators = &NetworkPublicIPsDataSource{}

// NewNetworkPublicIPsDataSource lists public IPs for a firewall or engagement.
func NewNetworkPublicIPsDataSource() datasource.DataSource {
	return &NetworkPublicIPsDataSource{}
}

// NetworkPublicIPsDataSource reads GET .../network/public-ips with firewall-ci or engagement filter.
type NetworkPublicIPsDataSource struct {
	client *client.Client
}

// NetworkPublicIPListItemModel is one row from data.content.
type NetworkPublicIPListItemModel struct {
	PublicIPSegment types.String `tfsdk:"public_ip_segment"`
	IsUsed          types.Bool   `tfsdk:"is_used"`
	Location        types.String `tfsdk:"location"`
	Purpose         types.String `tfsdk:"purpose"`
	Description     types.String `tfsdk:"description"`
}

// NetworkPublicIPsDataSourceModel is the data source root model.
type NetworkPublicIPsDataSourceModel struct {
	ID            types.String                   `tfsdk:"id"`
	FirewallID    types.Int64                    `tfsdk:"firewall_id"`
	EngagementID  types.Int64                    `tfsdk:"engagement_id"`
	Filters       []client.FilterModel           `tfsdk:"filter"`
	PublicIPItems []NetworkPublicIPListItemModel `tfsdk:"public_ips"`
}

type exactlyOneFirewallOrEngagement struct{}

func (v exactlyOneFirewallOrEngagement) Description(_ context.Context) string {
	return "Exactly one of firewall_id or engagement_id must be set."
}

func (v exactlyOneFirewallOrEngagement) MarkdownDescription(_ context.Context) string {
	return "Exactly one of `firewall_id` or `engagement_id` must be set."
}

func (v exactlyOneFirewallOrEngagement) ValidateDataSource(ctx context.Context, req datasource.ValidateConfigRequest, resp *datasource.ValidateConfigResponse) {
	var firewallID, engagementID types.Int64
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("firewall_id"), &firewallID)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("engagement_id"), &engagementID)...)
	if resp.Diagnostics.HasError() {
		return
	}

	fwSet := !firewallID.IsNull() && !firewallID.IsUnknown()
	engSet := !engagementID.IsNull() && !engagementID.IsUnknown()

	switch {
	case fwSet && engSet:
		resp.Diagnostics.AddError(
			"Invalid configuration",
			"Specify only one of firewall_id or engagement_id, not both.",
		)
	case !fwSet && !engSet:
		resp.Diagnostics.AddError(
			"Invalid configuration",
			"Either firewall_id or engagement_id must be set.",
		)
	}
}

func (d *NetworkPublicIPsDataSource) ConfigValidators(_ context.Context) []datasource.ConfigValidator {
	return []datasource.ConfigValidator{
		exactlyOneFirewallOrEngagement{},
	}
}

func (d *NetworkPublicIPsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network_public_ips"
}

func (d *NetworkPublicIPsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Lists public IP inventory returned by the network public-ips API for a firewall or engagement.",
		MarkdownDescription: "Calls `GET .../network/public-ips?firewall-ci={id}` or `?engagement={id}` and exposes each `content` entry (subset of fields).\n\nOptionally, use `filter` blocks to narrow results client-side by `public_ip_segment`, `is_used`, `location`, `purpose`, or `description`.",

		Blocks: map[string]schema.Block{
			"filter": client.FilterBlockSchema(),
		},
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Same as the firewall_id or engagement_id used in the request, as a string.",
				Computed:    true,
			},
			"firewall_id": schema.Int64Attribute{
				Description: "When set, query uses `firewall-ci` (mutually exclusive with engagement_id).",
				Optional:    true,
			},
			"engagement_id": schema.Int64Attribute{
				Description: "When set, query uses `engagement` (mutually exclusive with firewall_id).",
				Optional:    true,
			},
			"public_ips": schema.ListNestedAttribute{
				Description: "Public IP rows from the API `content` array.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"public_ip_segment": schema.StringAttribute{
							Description: "publicIpSegment from the API.",
							Computed:    true,
						},
						"is_used": schema.BoolAttribute{
							Description: "isUsed from the API.",
							Computed:    true,
						},
						"location": schema.StringAttribute{
							Description: "location from the API.",
							Computed:    true,
						},
						"purpose": schema.StringAttribute{
							Description: "purpose from the API.",
							Computed:    true,
						},
						"description": schema.StringAttribute{
							Description: "description from the API.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *NetworkPublicIPsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *NetworkPublicIPsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data NetworkPublicIPsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if d.client == nil {
		resp.Diagnostics.AddError("Client not configured", "Expected configured API client")
		return
	}

	var items []ListPublicIPItem
	var err error

	if !data.FirewallID.IsNull() && !data.FirewallID.IsUnknown() {
		fw := data.FirewallID.ValueInt64()
		data.ID = types.StringValue(fmt.Sprintf("firewall-%d", fw))
		items, err = ListPublicIPsByFirewall(d.client, ctx, fw)
	} else {
		eng := data.EngagementID.ValueInt64()
		data.ID = types.StringValue(fmt.Sprintf("engagement-%d", eng))
		items, err = ListPublicIPsByEngagement(d.client, ctx, eng)
	}

	if err != nil {
		resp.Diagnostics.AddError("Error listing public IPs", err.Error())
		return
	}

	out := make([]NetworkPublicIPListItemModel, 0, len(items))
	for _, it := range items {
		out = append(out, NetworkPublicIPListItemModel{
			PublicIPSegment: types.StringValue(it.PublicIPSegment),
			IsUsed:          types.BoolValue(it.IsUsed),
			Location:        types.StringValue(it.Location),
			Purpose:         types.StringValue(it.Purpose),
			Description:     types.StringValue(it.Description),
		})
	}

	if len(data.Filters) > 0 {
		tflog.Debug(ctx, "Applying filters to public IPs", map[string]any{
			"filter_count":     len(data.Filters),
			"pre_filter_count": len(out),
		})
		filtered, filterErr := client.ApplyFilters(out, data.Filters)
		if filterErr != nil {
			resp.Diagnostics.AddError("Error Filtering Public IPs", filterErr.Error())
			return
		}
		out = filtered

		tflog.Debug(ctx, "Filters applied to public IPs", map[string]any{
			"post_filter_count": len(out),
		})
	}

	data.PublicIPItems = out

	tflog.Info(ctx, "Public IPs data source read", map[string]any{"count": len(out)})
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
