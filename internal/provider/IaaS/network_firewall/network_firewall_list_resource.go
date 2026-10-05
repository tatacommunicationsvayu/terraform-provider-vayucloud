// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package network_firewall

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/client"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/account_engagement"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/account_location"
)

var _ datasource.DataSource = &NetworkFirewallListDataSource{}

func NewNetworkFirewallListDataSource() datasource.DataSource {
	return &NetworkFirewallListDataSource{}
}

type NetworkFirewallListDataSource struct {
	client *client.Client
}

// NetworkFirewallListItemModel represents a single firewall in the list output.
type NetworkFirewallListItemModel struct {
	NetworkFirewallID    types.Int64  `tfsdk:"network_firewall_id"`
	FirewallDisplayName  types.String `tfsdk:"firewall_display_name"`
	FirewallThroughput   types.String `tfsdk:"firewall_throughput"`
	InternetBandwidth    types.String `tfsdk:"internet_bandwidth"`
	FirewallPricingModel types.String `tfsdk:"firewall_pricing_model"`
	InternetPricingModel types.String `tfsdk:"internet_pricing_model"`
	EngagementID         types.Int64  `tfsdk:"engagement_id"`
	EndpointID           types.Int64  `tfsdk:"endpoint_id"`
	Hypervisor           types.String `tfsdk:"hypervisor"`
}

// NetworkFirewallListDataSourceModel describes the data source data model.
type NetworkFirewallListDataSourceModel struct {
	ID types.Int64 `tfsdk:"id"`

	// Required inputs to scope the listing
	EngagementID types.Int64 `tfsdk:"engagement_id"`
	EndpointID   types.Int64 `tfsdk:"endpoint_id"`

	// Optional filters
	Filters []client.FilterModel `tfsdk:"filter"`

	// Output list
	Firewalls []NetworkFirewallListItemModel `tfsdk:"firewalls"`

	// API response metadata
	Status       types.String `tfsdk:"status"`
	Message      types.String `tfsdk:"message"`
	ResponseCode types.Int64  `tfsdk:"response_code"`
}

func (d *NetworkFirewallListDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network_firewall_list"
}

func (d *NetworkFirewallListDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Retrieves the list of network firewalls from the VayuCloud API for a given engagement and endpoint.",
		MarkdownDescription: "Retrieves the list of network firewalls from the VayuCloud API for a given engagement and endpoint.\n\nThis data source calls `GET /network_operations/list-firewall-state/{engagementId}/{endpointId}`.\n\nOptionally, use `filter` blocks to narrow down results by field values.",

		Blocks: map[string]schema.Block{
			"filter": client.FilterBlockSchema(),
		},

		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "Placeholder identifier for Terraform.",
				Computed:    true,
			},
			"engagement_id": schema.Int64Attribute{
				Description: "The engagement ID to list firewalls for.",
				Required:    true,
			},
			"endpoint_id": schema.Int64Attribute{
				Description: "The endpoint ID to list firewalls for.",
				Required:    true,
			},
			"firewalls": schema.ListNestedAttribute{
				Description: "The list of network firewalls.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
					"network_firewall_id": schema.Int64Attribute{
						Description: "The firewall resource ID.",
						Computed:    true,
					},
						"firewall_display_name": schema.StringAttribute{
							Description: "The display name of the firewall.",
							Computed:    true,
						},
						"firewall_throughput": schema.StringAttribute{
							Description: "The throughput setting (e.g., '2Mbps').",
							Computed:    true,
						},
						"internet_bandwidth": schema.StringAttribute{
							Description: "The internet bandwidth setting (e.g., '2Mbps').",
							Computed:    true,
						},
						"firewall_pricing_model": schema.StringAttribute{
							Description: "The firewall pricing model (e.g., 'daily', 'monthly').",
							Computed:    true,
						},
						"internet_pricing_model": schema.StringAttribute{
							Description: "The internet pricing model (e.g., 'daily', 'monthly').",
							Computed:    true,
						},
						"engagement_id": schema.Int64Attribute{
							Description: "The engagement ID.",
							Computed:    true,
						},
						"endpoint_id": schema.Int64Attribute{
							Description: "The endpoint ID.",
							Computed:    true,
						},
						"hypervisor": schema.StringAttribute{
							Description: "The hypervisor type (e.g., 'KVM', 'VCD_ESXI').",
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

func (d *NetworkFirewallListDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *NetworkFirewallListDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data NetworkFirewallListDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	engagementID := data.EngagementID.ValueInt64()
	endpointID := data.EndpointID.ValueInt64()

	err := account_engagement.ValidateEngagementExists(d.client, ctx, engagementID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Validating Engagement",
			err.Error(),
		)
		return
	}

	err = account_location.ValidateEndpointExists(d.client, ctx, engagementID, endpointID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Validating Endpoint",
			err.Error(),
		)
		return
	}	
	tflog.Debug(ctx, "Reading network firewall list data source", map[string]any{
		"engagement_id": engagementID,
		"endpoint_id":   endpointID,
	})

	items, actionStateResp, err := ListNetworkFirewalls(d.client, ctx, engagementID, endpointID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Listing Network Firewalls",
			fmt.Sprintf("Could not list firewalls for engagement %d / endpoint %d: %s",
				engagementID, endpointID, err.Error()),
		)
		return
	}

	data.ID = types.Int64Value(engagementID)
	data.Status = types.StringValue(actionStateResp.Status)
	data.Message = types.StringValue(actionStateResp.Message)
	data.ResponseCode = types.Int64Value(int64(actionStateResp.ResponseCode))

	if len(items) == 0 {
		resp.Diagnostics.AddWarning(
			"Empty Response Data",
			"No firewalls found for the given engagement and endpoint.",
		)
		data.Firewalls = []NetworkFirewallListItemModel{}
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
		return
	}

	firewalls := make([]NetworkFirewallListItemModel, len(items))
	for i, fw := range items {
		firewalls[i] = NetworkFirewallListItemModel{
			NetworkFirewallID:    types.Int64Value(fw.ID),
			FirewallDisplayName:  types.StringValue(fw.FirewallDisplayName),
			FirewallThroughput:   types.StringValue(fw.FirewallThroughput),
			InternetBandwidth:    types.StringValue(fw.Bandwidth),
			FirewallPricingModel: types.StringValue(fw.FirewallPricingModel),
			InternetPricingModel: types.StringValue(fw.InternetPricingModel),
			EngagementID:         types.Int64Value(fw.EngagementID),
			EndpointID:           types.Int64Value(fw.EndpointID),
			Hypervisor:           types.StringValue(fw.Hypervisor),
		}
	}

	if len(data.Filters) > 0 {
		tflog.Debug(ctx, "Applying filters to network firewalls", map[string]any{
			"filter_count":     len(data.Filters),
			"pre_filter_count": len(firewalls),
		})

		filtered, err := client.ApplyFilters(firewalls, data.Filters)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Filtering Network Firewalls",
				"Could not apply filters: "+err.Error(),
			)
			return
		}
		firewalls = filtered

		tflog.Debug(ctx, "Filters applied to network firewalls", map[string]any{
			"post_filter_count": len(firewalls),
		})
	}

	data.Firewalls = firewalls

	tflog.Info(ctx, "Network firewall list read successfully", map[string]any{
		"count":         len(firewalls),
		"engagement_id": engagementID,
		"endpoint_id":   endpointID,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
