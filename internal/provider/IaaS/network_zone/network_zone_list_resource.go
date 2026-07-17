// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package network_zone

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/client"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/resource_group_environment"
)

var _ datasource.DataSource = &NetworkZoneListDataSource{}

func NewNetworkZoneListDataSource() datasource.DataSource {
	return &NetworkZoneListDataSource{}
}

type NetworkZoneListDataSource struct {
	client *client.Client
}

// NetworkZoneListItemModel represents a single network zone in the list output.
type NetworkZoneListItemModel struct {
	NetworkZoneID types.Int64  `tfsdk:"network_zone_id"`
	Name          types.String `tfsdk:"name"`
	EnvironmentID types.Int64  `tfsdk:"environment_id"`
	FirewallID    types.Int64  `tfsdk:"firewall_id"`
	NoOfIPs       types.Int64  `tfsdk:"no_of_ips"`
	Purpose       types.String `tfsdk:"purpose"`
	ZoneType      types.String `tfsdk:"zone_type"`
	NoOfV6IPs     types.Int64  `tfsdk:"no_of_v6_ips"`
	IPv6CIDR      types.String `tfsdk:"ipv6_cidr"`
}

// NetworkZoneListDataSourceModel describes the data source data model.
type NetworkZoneListDataSourceModel struct {
	ID types.Int64 `tfsdk:"id"`

	// Required input to scope the listing
	EnvironmentID types.Int64 `tfsdk:"environment_id"`

	// Optional filters
	Filters []client.FilterModel `tfsdk:"filter"`

	// Output list
	NetworkZones []NetworkZoneListItemModel `tfsdk:"network_zones"`

	// API response metadata
	Status       types.String `tfsdk:"status"`
	Message      types.String `tfsdk:"message"`
	ResponseCode types.Int64  `tfsdk:"response_code"`
}

func (d *NetworkZoneListDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network_zone_list"
}

func (d *NetworkZoneListDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Retrieves the list of network zones from the VayuCloud API for a given environment.",
		MarkdownDescription: "Retrieves the list of network zones from the VayuCloud API for a given environment.\n\nThis data source calls the action-state API with `module=zone` and `action=list`.\n\nOptionally, use `filter` blocks to narrow down results by field values.",

		Blocks: map[string]schema.Block{
			"filter": client.FilterBlockSchema(),
		},

		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "Placeholder identifier for Terraform.",
				Computed:    true,
			},
			"environment_id": schema.Int64Attribute{
				Description: "The environment ID to list network zones for.",
				Required:    true,
			},
			"network_zones": schema.ListNestedAttribute{
				Description: "The list of network zones.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"network_zone_id": schema.Int64Attribute{
							Description: "The network zone resource ID.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "The name of the network zone.",
							Computed:    true,
						},
						"environment_id": schema.Int64Attribute{
							Description: "The environment ID.",
							Computed:    true,
						},
						"firewall_id": schema.Int64Attribute{
							Description: "The firewall ID.",
							Computed:    true,
						},
						"no_of_ips": schema.Int64Attribute{
							Description: "The number of IPs.",
							Computed:    true,
						},
						"purpose": schema.StringAttribute{
							Description: "The purpose of the network zone.",
							Computed:    true,
						},
						"zone_type": schema.StringAttribute{
							Description: "The network zone type (e.g., 'overlay', 'vlan').",
							Computed:    true,
						},
						"no_of_v6_ips": schema.Int64Attribute{
							Description: "The number of IPv6 IPs.",
							Computed:    true,
						},
						"ipv6_cidr": schema.StringAttribute{
							Description: "The IPv6 CIDR value.",
							Computed:    true,
						},
					},
				},
			},
			"status": schema.StringAttribute{
				Description: "The API response status (e.g., 'success').",
				Computed:    true,
			},
			"message": schema.StringAttribute{
				Description: "Additional information from the API response.",
				Computed:    true,
			},
			"response_code": schema.Int64Attribute{
				Description: "The API response code (0 = success).",
				Computed:    true,
			},
		},
	}
}

func (d *NetworkZoneListDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = c
}

func (d *NetworkZoneListDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data NetworkZoneListDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	environmentID := data.EnvironmentID.ValueInt64()

	err := resource_group_environment.ValidateEnvironmentExists(d.client, ctx, environmentID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Validating Environment",
			err.Error(),
		)
		return
	}

	tflog.Debug(ctx, "Reading network zone list data source", map[string]any{
		"environment_id": environmentID,
	})

	items, actionStateResp, err := ListNetworkZones(d.client, ctx, environmentID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Listing Network Zones",
			fmt.Sprintf("Could not list network zones for environment %d: %s",
				environmentID, err.Error()),
		)
		return
	}

	data.ID = types.Int64Value(environmentID)
	data.Status = types.StringValue(actionStateResp.Status)
	data.Message = types.StringValue(actionStateResp.Message)
	data.ResponseCode = types.Int64Value(int64(actionStateResp.ResponseCode))

	if len(items) == 0 {
		resp.Diagnostics.AddWarning(
			"Empty Response Data",
			"No network zones found for the given environment.",
		)
		data.NetworkZones = []NetworkZoneListItemModel{}
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
		return
	}

	zones := make([]NetworkZoneListItemModel, len(items))
	for i, z := range items {
		zones[i] = NetworkZoneListItemModel{
			NetworkZoneID: types.Int64Value(z.ID),
			Name:          types.StringValue(z.Name),
			EnvironmentID: types.Int64Value(int64(z.EnvironmentID)),
			FirewallID:    types.Int64Value(int64(z.FirewallID)),
			NoOfIPs:       types.Int64Value(int64(z.NoOfIPs)),
			Purpose:       types.StringValue(z.Purpose),
			ZoneType:      types.StringValue(z.ZoneType),
			NoOfV6IPs:     types.Int64Value(int64(z.NoOfV6IPs)),
			IPv6CIDR:      types.StringValue(z.IPv6CIDR),
		}
	}

	if len(data.Filters) > 0 {
		tflog.Debug(ctx, "Applying filters to network zones", map[string]any{
			"filter_count":     len(data.Filters),
			"pre_filter_count": len(zones),
		})

		filtered, err := client.ApplyFilters(zones, data.Filters)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Filtering Network Zones",
				"Could not apply filters: "+err.Error(),
			)
			return
		}
		zones = filtered

		tflog.Debug(ctx, "Filters applied to network zones", map[string]any{
			"post_filter_count": len(zones),
		})
	}

	data.NetworkZones = zones

	tflog.Info(ctx, "Network zone list read successfully", map[string]any{
		"count":          len(zones),
		"environment_id": environmentID,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
