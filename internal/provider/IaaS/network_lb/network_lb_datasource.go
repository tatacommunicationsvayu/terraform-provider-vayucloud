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
	_ datasource.DataSource              = &NetworkLBDataSource{}
	_ datasource.DataSourceWithConfigure = &NetworkLBDataSource{}
)

func NewNetworkLBDataSource() datasource.DataSource {
	return &NetworkLBDataSource{}
}

type NetworkLBDataSource struct {
	client *client.Client
}

type NetworkLBDataSourceModel struct {
	ID           types.String `tfsdk:"id"`
	EngagementID types.Int64  `tfsdk:"engagement_id"`
	EndpointID   types.Int64  `tfsdk:"endpoint_id"`
	FirewallID   types.String `tfsdk:"firewall_id"`
	Name         types.String `tfsdk:"name"`
	DisplayName  types.String `tfsdk:"display_name"`
	Type         types.String `tfsdk:"type"`
	Bandwidth    types.String `tfsdk:"bandwidth"`
	CIStatus     types.String `tfsdk:"ci_status"`
}

func (d *NetworkLBDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network_lb"
}

func (d *NetworkLBDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves a single Load Balancer by ID.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Load Balancer CI Master ID.",
				Required:            true,
			},
			"engagement_id": schema.Int64Attribute{
				MarkdownDescription: "Engagement ID (from platform read).",
				Computed:            true,
			},
			"endpoint_id": schema.Int64Attribute{
				MarkdownDescription: "Endpoint ID (from platform read).",
				Computed:            true,
			},
			"firewall_id": schema.StringAttribute{
				MarkdownDescription: "Parent firewall CI Master ID.",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Load Balancer CI name.",
				Computed:            true,
			},
			"display_name": schema.StringAttribute{
				MarkdownDescription: "Display name.",
				Computed:            true,
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "Load Balancer type.",
				Computed:            true,
			},
			"bandwidth": schema.StringAttribute{
				MarkdownDescription: "Configured bandwidth in Mbps (e.g. 100Mbps). Plain numbers from the API (e.g. 100) are normalized to Mbps.",
				Computed:            true,
			},
			"ci_status": schema.StringAttribute{
				MarkdownDescription: "CI status.",
				Computed:            true,
			},
		},
	}
}

func (d *NetworkLBDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *NetworkLBDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data NetworkLBDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Reading Load Balancer", map[string]any{
		"lb_id": data.ID.ValueString(),
	})

	lbCiID, err := strconv.ParseInt(data.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid LB ID", fmt.Sprintf("Could not parse LB ID: %s", err.Error()))
		return
	}

	details, err := GetLoadBalancerDetails(d.client, ctx, lbCiID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to Read Load Balancer", err.Error())
		return
	}

	if v := stringFromActionStateMap(details, "name", "ci_name"); v != "" {
		data.Name = types.StringValue(v)
	}
	if v := stringFromActionStateMap(details, "display_name", "lbDisplayName"); v != "" {
		data.DisplayName = types.StringValue(v)
	}
	if v := stringFromActionStateMap(details, "lb_type"); v != "" {
		data.Type = types.StringValue(v)
	}
	if v := stringFromActionStateMap(details, "bandwidth", "lb_bandwidth"); v != "" {
		data.Bandwidth = types.StringValue(normalizeLBBandwidth(v))
	}
	if firewallCI, ok := int64FromActionStateMap(details, "firewall_ci", "firewallCI"); ok {
		data.FirewallID = types.StringValue(strconv.FormatInt(firewallCI, 10))
	} else if v := stringFromActionStateMap(details, "firewall_ci", "firewallCI"); v != "" {
		data.FirewallID = types.StringValue(v)
	}
	if v := stringFromActionStateMap(details, "ci_status"); v != "" {
		data.CIStatus = types.StringValue(v)
	}
	if engagementID, ok := int64FromActionStateMap(details, "engagement_id", "engagementId"); ok {
		data.EngagementID = types.Int64Value(engagementID)
	}
	if endpointID, ok := int64FromActionStateMap(details, "endpoint_id", "endpointId"); ok {
		data.EndpointID = types.Int64Value(endpointID)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
