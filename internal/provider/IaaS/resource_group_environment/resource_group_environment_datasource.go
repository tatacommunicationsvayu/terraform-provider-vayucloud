// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package resource_group_environment

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
var _ datasource.DataSource = &ResourceGroupEnvironmentDataSource{}

// NewResourceGroupEnvironmentDataSource creates a new data source for reading a resource group environment.
func NewResourceGroupEnvironmentDataSource() datasource.DataSource {
	return &ResourceGroupEnvironmentDataSource{}
}

// ResourceGroupEnvironmentDataSource defines the data source implementation.
type ResourceGroupEnvironmentDataSource struct {
	client *client.Client
}

// ResourceGroupEnvironmentDataSourceModel describes the data source data model.
type ResourceGroupEnvironmentDataSourceModel struct {
	// ID is a placeholder for Terraform (required for data sources)
	ID types.String `tfsdk:"id"`

	// ResourceGroupEnvironmentID is the environment resource ID to look up
	ResourceGroupEnvironmentID types.String `tfsdk:"resource_group_environment_id"`

	// Environment is the name of the environment
	Environment types.String `tfsdk:"environment"`

	// BusinessUnitID is the business unit ID associated with the environment
	BusinessUnitID types.Int64 `tfsdk:"business_unit_id"`

	// Status is the environment status from API response data (e.g., "ACTIVE")
	Status types.String `tfsdk:"status"`

	// Message contains additional information
	Message types.String `tfsdk:"message"`

	// ResponseCode is the API response code (0 = success)
	ResponseCode types.Int64 `tfsdk:"response_code"`
}

// Metadata returns the data source type name.
func (d *ResourceGroupEnvironmentDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_resource_group_environment"
}

// Schema defines the schema for the data source.
func (d *ResourceGroupEnvironmentDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Retrieves details of a resource group environment from the VayuCloud API.",
		MarkdownDescription: "Retrieves details of a resource group environment from the VayuCloud API.\n\nThis data source calls the action-state API with `module=engagementComponents`, `action=read`, and `resourceType=ENV` to return the current state of a resource group environment.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:         "Placeholder identifier for Terraform.",
				MarkdownDescription: "Placeholder identifier for Terraform.",
				Computed:            true,
			},
			"resource_group_environment_id": schema.StringAttribute{
				Description:         "The resource group environment ID to look up.",
				MarkdownDescription: "The resource group environment ID to look up.",
				Required:            true,
			},
			"environment": schema.StringAttribute{
				Description:         "The name of the environment.",
				MarkdownDescription: "The name of the environment.",
				Computed:            true,
			},
			"business_unit_id": schema.Int64Attribute{
				Description:         "The business unit ID associated with the environment.",
				MarkdownDescription: "The business unit ID associated with the environment.",
				Computed:            true,
			},
			"status": schema.StringAttribute{
				Description:         "The status of the resource group environment (e.g., 'ACTIVE').",
				MarkdownDescription: "The status of the resource group environment (e.g., `ACTIVE`).",
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
func (d *ResourceGroupEnvironmentDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *ResourceGroupEnvironmentDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ResourceGroupEnvironmentDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Reading resource group environment data source", map[string]any{
		"resource_group_environment_id": data.ResourceGroupEnvironmentID.ValueString(),
	})

	// Build the request body (same as resource Read)
	actionStateBody := map[string]any{
		"resourceId":   data.ResourceGroupEnvironmentID.ValueString(),
		"resourceType": "ENV",
	}

	// Call action-state API to get the latest environment data
	actionStateResponse, err := common.UpdateActionState(ctx, d.client, "engagementComponents", "read", actionStateBody)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Resource Group Environment",
			"Could not read resource group environment: "+err.Error(),
		)
		return
	}

	// Map top-level API response fields
	data.ID = types.StringValue(data.ResourceGroupEnvironmentID.ValueString())
	data.Message = types.StringValue(actionStateResponse.Message)
	data.ResponseCode = types.Int64Value(int64(actionStateResponse.ResponseCode))
	data.Status = types.StringValue(actionStateResponse.Status)

	// Check if response data is empty
	if len(actionStateResponse.Data) == 0 {
		resp.Diagnostics.AddWarning(
			"Empty Response Data",
			"Could not read resource group environment: response data is empty",
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

	environmentValue, ok := responseMap["environment"].(string)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Environment Data Type",
			fmt.Sprintf("Expected environment to be string, got: %T", responseMap["environment"]),
		)
		return
	}
	data.Environment = types.StringValue(environmentValue)

	switch v := responseMap["business_unit_id"].(type) {
	case float64:
		data.BusinessUnitID = types.Int64Value(int64(v))
	case int64:
		data.BusinessUnitID = types.Int64Value(v)
	default:
		resp.Diagnostics.AddError(
			"Unexpected Business Unit ID Data Type",
			fmt.Sprintf("Expected business_unit_id to be numeric, got: %T", responseMap["business_unit_id"]),
		)
		return
	}

	statusValue, ok := responseMap["status"]
	if !ok || statusValue == nil {
		data.Status = types.StringValue("ACTIVE")
	} else {
		if statusStr, ok := statusValue.(string); ok {
			data.Status = types.StringValue(statusStr)
		}
	}

	tflog.Info(ctx, "Resource group environment read successfully", map[string]any{
		"id":               data.ID.ValueString(),
		"environment":      data.Environment.ValueString(),
		"business_unit_id": data.BusinessUnitID.ValueInt64(),
		"status":           data.Status.ValueString(),
	})

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
