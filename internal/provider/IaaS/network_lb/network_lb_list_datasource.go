package network_lb

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/client"
)

var (
	_ datasource.DataSource              = &NetworkLBListDataSource{}
	_ datasource.DataSourceWithConfigure = &NetworkLBListDataSource{}
)

func NewNetworkLBListDataSource() datasource.DataSource {
	return &NetworkLBListDataSource{}
}

type NetworkLBListDataSource struct {
	client *client.Client
}

type NetworkLBListDataSourceModel struct {
	EngagementID  types.Int64 `tfsdk:"engagement_id"`
	EndpointID    types.Int64 `tfsdk:"endpoint_id"`
	LoadBalancers []LBItem    `tfsdk:"load_balancers"`
}

type LBItem struct {
	ID           types.String `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	DisplayName  types.String `tfsdk:"display_name"`
	LBType       types.String `tfsdk:"lb_type"`
	Bandwidth    types.String `tfsdk:"bandwidth"`
	FirewallID   types.String `tfsdk:"firewall_id"`
	CIStatus     types.String `tfsdk:"ci_status"`
	EngagementID types.Int64  `tfsdk:"engagement_id"`
	EndpointID   types.Int64  `tfsdk:"endpoint_id"`
}

func (d *NetworkLBListDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network_lb_list"
}

func (d *NetworkLBListDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists all Load Balancers for an engagement and endpoint.",

		Attributes: map[string]schema.Attribute{
			"engagement_id": schema.Int64Attribute{
				MarkdownDescription: "Engagement ID.",
				Required:            true,
			},
			"endpoint_id": schema.Int64Attribute{
				MarkdownDescription: "Endpoint ID.",
				Required:            true,
			},
			"load_balancers": schema.ListNestedAttribute{
				MarkdownDescription: "List of Load Balancers.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "LB CI Master ID.",
							Computed:            true,
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "LB CI name.",
							Computed:            true,
						},
						"display_name": schema.StringAttribute{
							MarkdownDescription: "Display name.",
							Computed:            true,
						},
						"lb_type": schema.StringAttribute{
							MarkdownDescription: "LB type (F5, HAProxy, AVI).",
							Computed:            true,
						},
						"bandwidth": schema.StringAttribute{
							MarkdownDescription: "Configured bandwidth in Mbps (e.g. 100Mbps). Plain numbers from the API (e.g. 100) are normalized to Mbps.",
							Computed:            true,
						},
						"firewall_id": schema.StringAttribute{
							MarkdownDescription: "Parent firewall CI Master ID.",
							Computed:            true,
						},
						"ci_status": schema.StringAttribute{
							MarkdownDescription: "CI status.",
							Computed:            true,
						},
						"engagement_id": schema.Int64Attribute{
							MarkdownDescription: "Engagement ID.",
							Computed:            true,
						},
						"endpoint_id": schema.Int64Attribute{
							MarkdownDescription: "Endpoint ID.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (d *NetworkLBListDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected DataSource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T", req.ProviderData),
		)
		return
	}

	d.client = client
}

func (d *NetworkLBListDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data NetworkLBListDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Listing Load Balancers", map[string]any{
		"engagement_id": data.EngagementID.ValueInt64(),
		"endpoint_id":   data.EndpointID.ValueInt64(),
	})

	items, _, err := ListLoadBalancers(d.client, ctx, data.EngagementID.ValueInt64(), data.EndpointID.ValueInt64())
	if err != nil {
		resp.Diagnostics.AddError("Failed to List Load Balancers", err.Error())
		return
	}

	lbItems := make([]LBItem, 0, len(items))
	for _, item := range items {
		lbItem := LBItem{
			EngagementID: data.EngagementID,
			EndpointID:   data.EndpointID,
		}
		if id, ok := int64FromActionStateMap(item, "id"); ok {
			lbItem.ID = types.StringValue(strconv.FormatInt(id, 10))
		} else if v := stringFromActionStateMap(item, "id"); v != "" {
			lbItem.ID = types.StringValue(v)
		}
		if v := stringFromActionStateMap(item, "name", "ci_name"); v != "" {
			lbItem.Name = types.StringValue(v)
		}
		if v := stringFromActionStateMap(item, "display_name", "lbDisplayName"); v != "" {
			lbItem.DisplayName = types.StringValue(v)
		}
		if v := stringFromActionStateMap(item, "lb_type"); v != "" {
			lbItem.LBType = types.StringValue(v)
		}
		if v := stringFromActionStateMap(item, "bandwidth", "lb_bandwidth"); v != "" {
			lbItem.Bandwidth = types.StringValue(normalizeLBBandwidth(v))
		}
		if firewallCI, ok := int64FromActionStateMap(item, "firewall_ci", "firewallCI"); ok {
			lbItem.FirewallID = types.StringValue(strconv.FormatInt(firewallCI, 10))
		} else if v := stringFromActionStateMap(item, "firewall_ci", "firewallCI"); v != "" {
			lbItem.FirewallID = types.StringValue(v)
		}
		if v := stringFromActionStateMap(item, "ci_status"); v != "" {
			lbItem.CIStatus = types.StringValue(v)
		}
		if engagementID, ok := int64FromActionStateMap(item, "engagement_id", "engagementId"); ok {
			lbItem.EngagementID = types.Int64Value(engagementID)
		}
		if endpointID, ok := int64FromActionStateMap(item, "endpoint_id", "endpointId"); ok {
			lbItem.EndpointID = types.Int64Value(endpointID)
		}
		lbItems = append(lbItems, lbItem)
	}

	data.LoadBalancers = lbItems

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
