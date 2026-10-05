// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package network_firewall

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
var _ datasource.DataSource = &NetworkFirewallDataSource{}

// NewNetworkFirewallDataSource creates a new data source for reading a network firewall.
func NewNetworkFirewallDataSource() datasource.DataSource {
	return &NetworkFirewallDataSource{}
}

// NetworkFirewallDataSource defines the data source implementation.
type NetworkFirewallDataSource struct {
	client *client.Client
}

// NetworkFirewallDataSourceModel describes the data source data model.
type NetworkFirewallDataSourceModel struct {
	// ID is a placeholder for Terraform (required for data sources)
	ID types.String `tfsdk:"id"`

	// ResourceID is the firewall resource ID to look up
	NetworkFirewallID types.String `tfsdk:"network_firewall_id"`

	// FirewallDisplayName is the display name for the firewall
	FirewallDisplayName types.String `tfsdk:"firewall_display_name"`

	// FirewallThroughput is the throughput setting (e.g., "2Mbps")
	FirewallThroughput types.String `tfsdk:"firewall_throughput"`

	// InternetBandwidth is the bandwidth setting (e.g., "2Mbps")
	InternetBandwidth types.String `tfsdk:"internet_bandwidth"`

	AccessType types.String `tfsdk:"access_type"`

	MinimumCommitment types.String `tfsdk:"minimum_commitment"`

	// FirewallPricingModel is the pricing model (e.g., "daily", "monthly")
	FirewallPricingModel types.String `tfsdk:"firewall_pricing_model"`

	// InternetPricingModel is the internet pricing model (e.g., "daily", "monthly")
	InternetPricingModel types.String `tfsdk:"internet_pricing_model"`

	// EngagementID is the engagement ID
	EngagementID types.Int64 `tfsdk:"engagement_id"`
	// EndpointID is the endpoint ID
	EndpointID types.Int64 `tfsdk:"endpoint_id"`

	// Hypervisor is the hypervisor type (e.g., "KVM", "VCD_ESXI")
	Hypervisor types.String `tfsdk:"hypervisor"`

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
func (d *NetworkFirewallDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network_firewall"
}

// Schema defines the schema for the data source.
func (d *NetworkFirewallDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Retrieves details of a network firewall from the VayuCloud API.",
		MarkdownDescription: "Retrieves details of a network firewall from the VayuCloud API.\n\nThis data source calls `GET /network_operations/firewall-state/{firewallId}` to return the current state of a network firewall.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:         "Placeholder identifier for Terraform.",
				MarkdownDescription: "Placeholder identifier for Terraform.",
				Computed:            true,
			},
			"network_firewall_id": schema.StringAttribute{
				Description:         "The network firewall ID of the network firewall to look up.",
				MarkdownDescription: "The network firewall ID of the network firewall to look up.",
				Required:            true,
			},
			"firewall_display_name": schema.StringAttribute{
				Description:         "The display name of the firewall.",
				MarkdownDescription: "The display name of the firewall.",
				Computed:            true,
			},
			"firewall_throughput": schema.StringAttribute{
				Description:         "The throughput setting (e.g., '2Mbps').",
				MarkdownDescription: "The throughput setting (e.g., `2Mbps`).",
				Computed:            true,
			},
			"internet_bandwidth": schema.StringAttribute{
				Description:         "The internet bandwidth setting (e.g., '2Mbps').",
				MarkdownDescription: "The internet bandwidth setting (e.g., `2Mbps`).",
				Computed:            true,
			},
			"access_type": schema.StringAttribute{
				Description:         "The access type (e.g., 'Bandwidth', 'DataTransfer').",
				MarkdownDescription: "The access type (e.g., `Bandwidth`, `DataTransfer`).",
				Computed:            true,
			},
			"minimum_commitment": schema.StringAttribute{
				Description:         "The minimum commitment (e.g., '500GB').",
				MarkdownDescription: "The minimum commitment (e.g., `500GB`).",
				Computed:            true,
			},
			"firewall_pricing_model": schema.StringAttribute{
				Description:         "The firewall pricing model (e.g., 'daily', 'monthly').",
				MarkdownDescription: "The firewall pricing model (e.g., `daily`, `monthly`).",
				Computed:            true,
			},
			"internet_pricing_model": schema.StringAttribute{
				Description:         "The internet pricing model (e.g., 'daily', 'monthly').",
				MarkdownDescription: "The internet pricing model (e.g., `daily`, `monthly`).",
				Computed:            true,
			},
			"engagement_id": schema.Int64Attribute{
				Description:         "The engagement ID.",
				MarkdownDescription: "The engagement ID.",
				Computed:            true,
			},
			"endpoint_id": schema.Int64Attribute{
				Description:         "The endpoint ID.",
				MarkdownDescription: "The endpoint ID.",
				Computed:            true,
			},
			"hypervisor": schema.StringAttribute{
				Description:         "The hypervisor type (e.g., 'KVM', 'VCD_ESXI').",
				MarkdownDescription: "The hypervisor type (e.g., `KVM`, `VCD_ESXI`).",
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
func (d *NetworkFirewallDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
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
func (d *NetworkFirewallDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data NetworkFirewallDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Reading network firewall data source", map[string]any{
		"network_firewall_id": data.NetworkFirewallID.ValueString(),
	})

	stateResponse, err := ReadNetworkFirewallState(d.client, ctx, data.NetworkFirewallID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Network Firewall",
			"Could not read network firewall: "+err.Error(),
		)
		return
	}

	// Map top-level API response fields
	data.ID = types.StringValue(data.NetworkFirewallID.ValueString())
	data.Status = types.StringValue(stateResponse.Status)
	data.Message = types.StringValue(stateResponse.Message)
	data.ResponseCode = types.Int64Value(int64(stateResponse.ResponseCode))
	// Check if response data is empty
	if len(stateResponse.Data) == 0 {
		resp.Diagnostics.AddWarning(
			"Empty Response Data",
			"Could not read network firewall: response data is empty, keeping existing state",
		)
		// Keep existing state and return
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
		return
	}

	// Unmarshal the JSON response data into a map
	// The API returns standard JSON types, not Terraform framework types
	var responseMap map[string]interface{}
	if err := json.Unmarshal(stateResponse.Data, &responseMap); err != nil {
		resp.Diagnostics.AddError(
			"Error Unmarshalling Response Data",
			fmt.Sprintf("Could not unmarshal response data: %s. Response: %s", err.Error(), string(stateResponse.Data)),
		)
		return
	}

	data.EngagementID = jsonMapInt64(responseMap, "engagement_id")
	data.EndpointID = jsonMapInt64(responseMap, "endpoint_id")
	data.Hypervisor = jsonMapString(responseMap, "hypervisor")
	data.FirewallDisplayName = jsonMapString(responseMap, "firewall_display_name")
	data.FirewallThroughput = jsonMapString(responseMap, "firewall_throughput")
	data.InternetBandwidth = jsonMapString(responseMap, "bandwidth")
	data.FirewallPricingModel = jsonMapString(responseMap, "firewall_pricing_model")
	data.InternetPricingModel = jsonMapString(responseMap, "internet_pricing_model")
	data.AccessType = jsonMapString(responseMap, "access_type")
	data.MinimumCommitment = jsonMapString(responseMap, "minimum_commitment")

	tflog.Info(ctx, "Network firewall read successfully", map[string]any{
		"id":     data.ID.ValueString(),
		"status": data.Status.ValueString(),
	})

	tflog.Info(ctx, "Read network firewall data source", map[string]any{
		"network_firewall_id":   data.NetworkFirewallID.ValueString(),
		"firewall_display_name": data.FirewallDisplayName.ValueString(),
		"firewall_throughput":   data.FirewallThroughput.ValueString(),
		"internet_bandwidth":    data.InternetBandwidth.ValueString(),
	})

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// jsonMapInt64 reads key from a JSON object decoded into map[string]interface{}.
// Returns null if the key is missing, the value is nil, or the value is not a JSON number (float64).
func jsonMapInt64(m map[string]interface{}, key string) types.Int64 {
	v, ok := m[key]
	if !ok || v == nil {
		return types.Int64Null()
	}
	f, ok := v.(float64)
	if !ok {
		return types.Int64Null()
	}
	return types.Int64Value(int64(f))
}

// jsonMapString reads key from a JSON object decoded into map[string]interface{}.
// Returns null if the key is missing, the value is nil, or the value is not a string.
func jsonMapString(m map[string]interface{}, key string) types.String {
	v, ok := m[key]
	if !ok || v == nil {
		return types.StringNull()
	}
	s, ok := v.(string)
	if !ok {
		return types.StringNull()
	}
	return types.StringValue(s)
}
