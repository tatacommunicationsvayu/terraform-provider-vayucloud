// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package network_zone

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/client"
)

// Ensure provider-defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &NetworkZoneDataSource{}

// NewNetworkZoneDataSource creates a new data source for reading a network zone.
func NewNetworkZoneDataSource() datasource.DataSource {
	return &NetworkZoneDataSource{}
}

// NetworkZoneDataSource defines the data source implementation.
type NetworkZoneDataSource struct {
	client *client.Client
}

// NetworkZoneDataSourceModel describes the data source data model.
type NetworkZoneDataSourceModel struct {
	// ID is a placeholder for Terraform (required for data sources)
	ID types.String `tfsdk:"id"`

	// NetworkZoneID is the zone resource ID to look up
	NetworkZoneID types.String `tfsdk:"network_zone_id"`

	// Name is the zone name
	Name types.String `tfsdk:"name"`

	// EnvironmentID is the environment ID
	EnvironmentID types.Int64 `tfsdk:"environment_id"`

	// FirewallID is the firewall ID
	FirewallID types.Int64 `tfsdk:"firewall_id"`

	// NoOfIPs is the number of IPs
	NoOfIPs types.Int64 `tfsdk:"no_of_ips"`

	// Purpose is the purpose of the zone
	Purpose types.String `tfsdk:"purpose"`

	// DataPlane is the data plane type
	DataPlane types.String `tfsdk:"data_plane"`

	// ZoneType is the zone type (e.g., "overlay", "vlan")
	ZoneType types.String `tfsdk:"zone_type"`

	// NoOfV6IPs is the number of IPv6 IPs
	NoOfV6IPs types.Int64 `tfsdk:"no_of_v6_ips"`

	// IPv6CIDR is the IPv6 CIDR value
	IPv6CIDR types.String `tfsdk:"ipv6_cidr"`

	// Status is the API response status (e.g., "success")
	Status types.String `tfsdk:"status"`

	// Message contains additional information
	Message types.String `tfsdk:"message"`

	// ResponseCode is the API response code (0 = success)
	ResponseCode types.Int64 `tfsdk:"response_code"`

	// RawResponse contains the raw JSON response (useful for debugging)
	RawResponse types.String `tfsdk:"raw_response"`
}

// Metadata returns the data source type name.
func (d *NetworkZoneDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network_zone"
}

// Schema defines the schema for the data source.
func (d *NetworkZoneDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Retrieves details of a network zone from the VayuCloud API.",
		MarkdownDescription: "Retrieves details of a network zone from the VayuCloud API.\n\nThis data source calls `GET /network/zone-state/{zoneId}` to return the current state of a network zone.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:         "Placeholder identifier for Terraform.",
				MarkdownDescription: "Placeholder identifier for Terraform.",
				Computed:            true,
			},
			"network_zone_id": schema.StringAttribute{
				Description:         "The resource ID of the network zone to look up.",
				MarkdownDescription: "The resource ID of the network zone to look up.",
				Required:            true,
			},
			"name": schema.StringAttribute{
				Description:         "The name of the network zone.",
				MarkdownDescription: "The name of the network zone.",
				Computed:            true,
			},
			"environment_id": schema.Int64Attribute{
				Description:         "The environment ID.",
				MarkdownDescription: "The environment ID.",
				Computed:            true,
			},
			"firewall_id": schema.Int64Attribute{
				Description:         "The firewall ID.",
				MarkdownDescription: "The firewall ID.",
				Computed:            true,
			},
			"no_of_ips": schema.Int64Attribute{
				Description:         "The number of IPs.",
				MarkdownDescription: "The number of IPs.",
				Computed:            true,
			},
			"purpose": schema.StringAttribute{
				Description:         "The purpose of the network zone.",
				MarkdownDescription: "The purpose of the network zone.",
				Computed:            true,
			},
			"data_plane": schema.StringAttribute{
				Description:         "The data plane type (e.g., 'Auto IPAM', 'Data Plane CIDR').",
				MarkdownDescription: "The data plane type (e.g., `Auto IPAM`, `Data Plane CIDR`).",
				Computed:            true,
			},
			"zone_type": schema.StringAttribute{
				Description:         "The network zone type (e.g., 'overlay', 'vlan').",
				MarkdownDescription: "The network zone type (e.g., `overlay`, `vlan`).",
				Computed:            true,
			},
			"no_of_v6_ips": schema.Int64Attribute{
				Description:         "The number of IPv6 IPs.",
				MarkdownDescription: "The number of IPv6 IPs.",
				Computed:            true,
			},
			"ipv6_cidr": schema.StringAttribute{
				Description:         "The IPv6 CIDR value.",
				MarkdownDescription: "The IPv6 CIDR value.",
				Computed:            true,
			},
			"status": schema.StringAttribute{
				Description:         "The API response status (e.g., 'success').",
				MarkdownDescription: "The API response status (e.g., `success`).",
				Computed:            true,
			},
			"message": schema.StringAttribute{
				Description:         "Additional information from the API response.",
				MarkdownDescription: "Additional information from the API response.",
				Computed:            true,
			},
			"response_code": schema.Int64Attribute{
				Description:         "The API response code (0 = success).",
				MarkdownDescription: "The API response code (`0` = success).",
				Computed:            true,
			},
			"raw_response": schema.StringAttribute{
				Description:         "The raw JSON response from the API (useful for debugging).",
				MarkdownDescription: "The raw JSON response from the API (useful for debugging).",
				Computed:            true,
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *NetworkZoneDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = client
}

// Read refreshes the Terraform state with the latest data from the API.
func (d *NetworkZoneDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data NetworkZoneDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Reading network zone data source", map[string]any{
		"network_zone_id": data.NetworkZoneID.ValueString(),
	})

	actionStateResponse, err := ReadNetworkZoneState(d.client, ctx, data.NetworkZoneID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Network Zone",
			"Could not read network zone: "+err.Error(),
		)
		return
	}

	data.ID = types.StringValue(data.NetworkZoneID.ValueString())
	data.Status = types.StringValue(actionStateResponse.Status)
	data.Message = types.StringValue(actionStateResponse.Message)
	data.ResponseCode = types.Int64Value(int64(actionStateResponse.ResponseCode))

	if len(actionStateResponse.Data) == 0 {
		resp.Diagnostics.AddWarning(
			"Empty Response Data",
			"Could not read network zone: response data is empty, keeping existing state",
		)
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
		return
	}

	data.RawResponse = types.StringValue(string(actionStateResponse.Data))

	var responseMap map[string]interface{}
	if err := json.Unmarshal(actionStateResponse.Data, &responseMap); err != nil {
		resp.Diagnostics.AddError(
			"Error Unmarshalling Response Data",
			fmt.Sprintf("Could not unmarshal response data: %s. Response: %s", err.Error(), string(actionStateResponse.Data)),
		)
		return
	}

	data.Name = types.StringValue(responseMap["name"].(string))
	data.EnvironmentID = types.Int64Value(int64(responseMap["environment_id"].(float64)))
	data.FirewallID = types.Int64Value(int64(responseMap["firewall_id"].(float64)))
	data.NoOfIPs = types.Int64Value(int64(responseMap["no_of_ips"].(float64)))
	data.Purpose = types.StringValue(responseMap["purpose"].(string))
	data.ZoneType = types.StringValue(responseMap["zone_type"].(string))

	if responseMap["dual_stack_mode"] != nil {
		data.NoOfV6IPs = types.Int64Value(int64(responseMap["no_of_v6_ips"].(float64)))
		data.IPv6CIDR = types.StringValue(responseMap["ipv6_cidr"].(string))
	} else {
		data.NoOfV6IPs = types.Int64Value(0)
		data.IPv6CIDR = types.StringValue("")
	}

	tflog.Info(ctx, "Network zone read successfully", map[string]any{
		"id":     data.ID.ValueString(),
		"status": data.Status.ValueString(),
	})

	tflog.Info(ctx, "Read network zone data source", map[string]any{
		"network_zone_id": data.NetworkZoneID.ValueString(),
		"name":            data.Name.ValueString(),
		"purpose":         data.Purpose.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
