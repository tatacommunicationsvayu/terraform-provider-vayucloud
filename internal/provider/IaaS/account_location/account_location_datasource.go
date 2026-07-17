// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package account_location

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/client"
)

// Ensure provider-defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &AccountLocationDataSource{}

// NewAccountLocationDataSource creates a new data source for reading account locations.
func NewAccountLocationDataSource() datasource.DataSource {
	return &AccountLocationDataSource{}
}

// AccountLocationDataSource defines the data source implementation.
type AccountLocationDataSource struct {
	client *client.Client
}

// AccountLocationModel represents a single location in the data source model.
type AccountLocationModel struct {
	// EndpointID is the endpoint ID
	EndpointID types.Int64 `tfsdk:"endpoint_id"`

	// EndpointDisplayName is the display name of the endpoint/location
	EndpointDisplayName types.String `tfsdk:"endpoint_display_name"`
}

// AccountLocationDataSourceModel describes the data source data model.
type AccountLocationDataSourceModel struct {
	// ID is a placeholder for Terraform (required for data sources)
	ID types.String `tfsdk:"id"`

	// EngagementID is the engagement ID to look up locations for (required input)
	EngagementID types.Int64 `tfsdk:"engagement_id"`

	// Filters is the optional list of filter blocks
	Filters []client.FilterModel `tfsdk:"filter"`

	// Locations is the list of locations returned by the API
	Locations []AccountLocationModel `tfsdk:"locations"`

	// Status is the API response status (e.g., "success")
	Status types.String `tfsdk:"status"`

	// Message contains additional information
	Message types.String `tfsdk:"message"`

	// ResponseCode is the API response code (0 = success)
	ResponseCode types.Int64 `tfsdk:"response_code"`
}

// Metadata returns the data source type name.
func (d *AccountLocationDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_account_location"
}

// Schema defines the schema for the data source.
func (d *AccountLocationDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Retrieves the list of locations (endpoints) for a given engagement from the VayuCloud API.",
		MarkdownDescription: "Retrieves the list of locations (endpoints) for a given engagement from the VayuCloud API.\n\nThis data source calls the `getEndpointsByEngagement` API to return all locations associated with the specified engagement.\n\nOptionally, use `filter` blocks to narrow down results by field values.",

		Blocks: map[string]schema.Block{
			"filter": client.FilterBlockSchema(),
		},

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:         "Placeholder identifier for Terraform.",
				MarkdownDescription: "Placeholder identifier for Terraform.",
				Computed:            true,
			},
			"engagement_id": schema.Int64Attribute{
				Description:         "The engagement ID to retrieve locations for.",
				MarkdownDescription: "The engagement ID to retrieve locations for.",
				Required:            true,
			},
			"locations": schema.ListNestedAttribute{
				Description:         "The list of locations (endpoints) for the engagement.",
				MarkdownDescription: "The list of locations (endpoints) for the engagement.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"endpoint_id": schema.Int64Attribute{
							Description:         "The endpoint ID.",
							MarkdownDescription: "The endpoint ID.",
							Computed:            true,
						},
						"endpoint_display_name": schema.StringAttribute{
							Description:         "The display name of the endpoint/location.",
							MarkdownDescription: "The display name of the endpoint/location.",
							Computed:            true,
						},
					},
				},
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
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *AccountLocationDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
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

// Read refreshes the Terraform state with the latest data from the API.
func (d *AccountLocationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data AccountLocationDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	engagementID := data.EngagementID.ValueInt64()

	tflog.Debug(ctx, "Reading account location data source", map[string]any{
		"engagement_id": engagementID,
	})

	// Call the API to get account locations
	locationResponse, err := GetAccountLocations(d.client, ctx, engagementID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Account Locations",
			fmt.Sprintf("Could not read account locations for engagement %d: %s", engagementID, err.Error()),
		)
		return
	}

	// Map top-level API response fields
	data.ID = types.StringValue(fmt.Sprintf("account-locations-%d", engagementID))
	data.Status = types.StringValue(locationResponse.Status)
	if locationResponse.Message != nil {
		data.Message = types.StringValue(*locationResponse.Message)
	} else {
		data.Message = types.StringValue("")
	}
	data.ResponseCode = types.Int64Value(int64(locationResponse.ResponseCode))

	// Check if response data is empty
	if len(locationResponse.Data) == 0 {
		resp.Diagnostics.AddWarning(
			"Empty Response Data",
			fmt.Sprintf("No locations found for engagement %d.", engagementID),
		)
		data.Locations = []AccountLocationModel{}
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
		return
	}

	// Map location data
	locations := make([]AccountLocationModel, len(locationResponse.Data))
	for i, location := range locationResponse.Data {
		locations[i] = AccountLocationModel{
			EndpointID:          types.Int64Value(location.EndpointID),
			EndpointDisplayName: types.StringValue(location.EndpointDisplayName),
		}
	}

	// Apply filters if provided
	if len(data.Filters) > 0 {
		tflog.Debug(ctx, "Applying filters to account locations", map[string]any{
			"filter_count":     len(data.Filters),
			"pre_filter_count": len(locations),
		})

		filtered, err := client.ApplyFilters(locations, data.Filters)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Filtering Account Locations",
				"Could not apply filters: "+err.Error(),
			)
			return
		}
		locations = filtered

		tflog.Debug(ctx, "Filters applied to account locations", map[string]any{
			"post_filter_count": len(locations),
		})
	}

	data.Locations = locations

	tflog.Info(ctx, "Account locations read successfully", map[string]any{
		"engagement_id": engagementID,
		"count":         len(locations),
		"status":        data.Status.ValueString(),
	})

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
