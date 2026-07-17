// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package resource_group_business_unit

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/client"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/common"
)

// Ensure provider-defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &ResourceGroupBusinessUnitDataSource{}

// NewResourceGroupBusinessUnitDataSource creates a new data source for reading a resource group business unit.
func NewResourceGroupBusinessUnitDataSource() datasource.DataSource {
	return &ResourceGroupBusinessUnitDataSource{}
}

// ResourceGroupBusinessUnitDataSource defines the data source implementation.
type ResourceGroupBusinessUnitDataSource struct {
	client *client.Client
}

// ResourceGroupBusinessUnitDataSourceModel describes the data source data model.
type ResourceGroupBusinessUnitDataSourceModel struct {
	// ID is a placeholder for Terraform (required for data sources)
	ID types.String `tfsdk:"id"`

	// ResourceGroupBusinessUnitID is the business unit resource ID to look up
	ResourceGroupBusinessUnitID types.String `tfsdk:"resource_group_business_unit_id"`

	// BusinessUnit is the name of the business unit
	BusinessUnit types.String `tfsdk:"business_unit"`

	// Users is the list of user emails associated with the business unit
	Users types.List `tfsdk:"users"`

	// Status is the API response status (e.g., "success")
	Status types.String `tfsdk:"status"`

	// Message contains additional information
	Message types.String `tfsdk:"message"`

	// ResponseCode is the API response code (0 = success)
	ResponseCode types.Int64 `tfsdk:"response_code"`
}

// Metadata returns the data source type name.
func (d *ResourceGroupBusinessUnitDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_resource_group_business_unit"
}

// Schema defines the schema for the data source.
func (d *ResourceGroupBusinessUnitDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Retrieves details of a resource group business unit from the VayuCloud API.",
		MarkdownDescription: "Retrieves details of a resource group business unit from the VayuCloud API.\n\nThis data source calls the action-state API with `module=engagementComponents` and `action=read` to return the current state of a business unit.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:         "Placeholder identifier for Terraform.",
				MarkdownDescription: "Placeholder identifier for Terraform.",
				Computed:            true,
			},
			"resource_group_business_unit_id": schema.StringAttribute{
				Description:         "The resource group business unit ID to look up.",
				MarkdownDescription: "The resource group business unit ID to look up.",
				Required:            true,
			},
			"business_unit": schema.StringAttribute{
				Description:         "The name of the business unit.",
				MarkdownDescription: "The name of the business unit.",
				Computed:            true,
			},
			"users": schema.ListAttribute{
				Description:         "The list of user emails associated with the business unit.",
				MarkdownDescription: "The list of user emails associated with the business unit.",
				Computed:            true,
				ElementType:         types.StringType,
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
func (d *ResourceGroupBusinessUnitDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *ResourceGroupBusinessUnitDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ResourceGroupBusinessUnitDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Reading resource group business unit data source", map[string]any{
		"resource_group_business_unit_id": data.ResourceGroupBusinessUnitID.ValueString(),
	})

	// Build the request body (same as resource Read)
	actionStateBody := map[string]any{
		"resourceId":   data.ResourceGroupBusinessUnitID.ValueString(),
		"resourceType": "BU",
	}

	// Call action-state API to get the latest business unit data
	actionStateResponse, err := common.UpdateActionState(ctx, d.client, "engagementComponents", "read", actionStateBody)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Resource Group Business Unit",
			"Could not read resource group business unit: "+err.Error(),
		)
		return
	}

	// Map top-level API response fields
	data.ID = types.StringValue(data.ResourceGroupBusinessUnitID.ValueString())
	data.Status = types.StringValue(actionStateResponse.Status)
	data.Message = types.StringValue(actionStateResponse.Message)
	data.ResponseCode = types.Int64Value(int64(actionStateResponse.ResponseCode))

	// Check if response data is empty
	if len(actionStateResponse.Data) == 0 {
		resp.Diagnostics.AddWarning(
			"Empty Response Data",
			"Could not read resource group business unit: response data is empty",
		)
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
		return
	}

	// Unmarshal the JSON response data into a map
	var responseMap map[string]interface{}
	if err := json.Unmarshal(actionStateResponse.Data, &responseMap); err != nil {
		resp.Diagnostics.AddError(
			"Error Unmarshalling Response Data",
			fmt.Sprintf("Could not unmarshal response data: %s. Response: %s", err.Error(), string(actionStateResponse.Data)),
		)
		return
	}

	data.BusinessUnit = types.StringValue(responseMap["business_unit"].(string))

	usersList, diags := types.ListValueFrom(ctx, types.StringType, responseMap["users"])
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Users = usersList

	tflog.Info(ctx, "Resource group business unit read successfully", map[string]any{
		"id":            data.ID.ValueString(),
		"business_unit": data.BusinessUnit.ValueString(),
		"status":        data.Status.ValueString(),
	})

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
