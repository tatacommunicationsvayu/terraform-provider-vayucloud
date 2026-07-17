// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package resource_group_environment

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/client"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/resource_group_business_unit"
)

var _ datasource.DataSource = &ResourceGroupEnvironmentListDataSource{}

func NewResourceGroupEnvironmentListDataSource() datasource.DataSource {
	return &ResourceGroupEnvironmentListDataSource{}
}

type ResourceGroupEnvironmentListDataSource struct {
	client *client.Client
}

// EnvironmentListItemModel represents a single environment in the list output.
type EnvironmentListItemModel struct {
	ResourceGroupEnvironmentID types.Int64  `tfsdk:"resource_group_environment_id"`
	Environment                types.String `tfsdk:"environment"`
	BusinessUnitID             types.Int64  `tfsdk:"business_unit_id"`
	Status                     types.String `tfsdk:"status"`
}

// ResourceGroupEnvironmentListDataSourceModel describes the data source data model.
type ResourceGroupEnvironmentListDataSourceModel struct {
	ID types.Int64 `tfsdk:"id"`

	// Required input to scope the listing
	BusinessUnitID types.Int64 `tfsdk:"business_unit_id"`

	// Optional filters
	Filters []client.FilterModel `tfsdk:"filter"`

	// Output list
	Environments []EnvironmentListItemModel `tfsdk:"environments"`

	// API response metadata
	Status       types.String `tfsdk:"status"`
	Message      types.String `tfsdk:"message"`
	ResponseCode types.Int64  `tfsdk:"response_code"`
}

func (d *ResourceGroupEnvironmentListDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_resource_group_environment_list"
}

func (d *ResourceGroupEnvironmentListDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Retrieves the list of resource group environments from the VayuCloud API for a given business unit.",
		MarkdownDescription: "Retrieves the list of resource group environments from the VayuCloud API for a given business unit.\n\nThis data source calls the action-state API with `module=engagementComponents`, `action=list`, and `resourceType=ENV`.\n\nOptionally, use `filter` blocks to narrow down results by field values.",

		Blocks: map[string]schema.Block{
			"filter": client.FilterBlockSchema(),
		},

		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "Placeholder identifier for Terraform.",
				Computed:    true,
			},
			"business_unit_id": schema.Int64Attribute{
				Description: "The business unit ID to list environments for.",
				Required:    true,
			},
			"environments": schema.ListNestedAttribute{
				Description: "The list of resource group environments.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"resource_group_environment_id": schema.Int64Attribute{
							Description: "The environment resource ID.",
							Computed:    true,
						},
						"environment": schema.StringAttribute{
							Description: "The name of the environment.",
							Computed:    true,
						},
						"business_unit_id": schema.Int64Attribute{
							Description: "The business unit ID associated with the environment.",
							Computed:    true,
						},
						"status": schema.StringAttribute{
							Description: "The status of the environment (e.g., 'ACTIVE').",
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

func (d *ResourceGroupEnvironmentListDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ResourceGroupEnvironmentListDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ResourceGroupEnvironmentListDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	businessUnitID := data.BusinessUnitID.ValueInt64()

	err := resource_group_business_unit.ValidateBusinessUnitExists(d.client, ctx, businessUnitID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Validating Business Unit",
			err.Error(),
		)
		return
	}

	tflog.Debug(ctx, "Reading resource group environment list data source", map[string]any{
		"business_unit_id": businessUnitID,
	})

	items, actionStateResp, err := ListEnvironments(d.client, ctx, businessUnitID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Listing Environments",
			fmt.Sprintf("Could not list environments for business unit %d: %s",
				businessUnitID, err.Error()),
		)
		return
	}

	data.ID = types.Int64Value(businessUnitID)
	data.Status = types.StringValue(actionStateResp.Status)
	data.Message = types.StringValue(actionStateResp.Message)
	data.ResponseCode = types.Int64Value(int64(actionStateResp.ResponseCode))

	if len(items) == 0 {
		resp.Diagnostics.AddWarning(
			"Empty Response Data",
			"No environments found for the given business unit.",
		)
		data.Environments = []EnvironmentListItemModel{}
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
		return
	}

	environments := make([]EnvironmentListItemModel, len(items))
	for i, env := range items {
		status := env.Status
		if status == "" {
			status = "ACTIVE"
		}

		environments[i] = EnvironmentListItemModel{
			ResourceGroupEnvironmentID: types.Int64Value(env.ID),
			Environment:                types.StringValue(env.Environment),
			BusinessUnitID:             types.Int64Value(int64(env.BusinessUnitID)),
			Status:                     types.StringValue(status),
		}
	}

	if len(data.Filters) > 0 {
		tflog.Debug(ctx, "Applying filters to environments", map[string]any{
			"filter_count":     len(data.Filters),
			"pre_filter_count": len(environments),
		})

		filtered, err := client.ApplyFilters(environments, data.Filters)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Filtering Environments",
				"Could not apply filters: "+err.Error(),
			)
			return
		}
		environments = filtered

		tflog.Debug(ctx, "Filters applied to environments", map[string]any{
			"post_filter_count": len(environments),
		})
	}

	data.Environments = environments

	tflog.Info(ctx, "Environment list read successfully", map[string]any{
		"count":            len(environments),
		"business_unit_id": businessUnitID,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
