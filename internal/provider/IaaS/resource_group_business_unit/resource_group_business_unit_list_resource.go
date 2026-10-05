// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package resource_group_business_unit

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/client"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/network_firewall"
)

var _ datasource.DataSource = &ResourceGroupBusinessUnitListDataSource{}

func NewResourceGroupBusinessUnitListDataSource() datasource.DataSource {
	return &ResourceGroupBusinessUnitListDataSource{}
}

type ResourceGroupBusinessUnitListDataSource struct {
	client *client.Client
}

// BusinessUnitListItemModel represents a single business unit in the list output.
type BusinessUnitListItemModel struct {
	ResourceGroupBusinessUnitID types.Int64  `tfsdk:"resource_group_business_unit_id"`
	BusinessUnit                types.String `tfsdk:"business_unit"`
	Users                       types.List   `tfsdk:"users"`
}

// ResourceGroupBusinessUnitListDataSourceModel describes the data source data model.
type ResourceGroupBusinessUnitListDataSourceModel struct {
	ID types.Int64 `tfsdk:"id"`

	// Required input to scope the listing
	FirewallID types.Int64 `tfsdk:"firewall_id"`

	// Optional filters
	Filters []client.FilterModel `tfsdk:"filter"`

	// Output list
	BusinessUnits []BusinessUnitListItemModel `tfsdk:"business_units"`

	// API response metadata
	Status       types.String `tfsdk:"status"`
	Message      types.String `tfsdk:"message"`
	ResponseCode types.Int64  `tfsdk:"response_code"`
}

func (d *ResourceGroupBusinessUnitListDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_resource_group_business_unit_list"
}

func (d *ResourceGroupBusinessUnitListDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Retrieves the list of resource group business units from the VayuCloud API for a given firewall.",
		MarkdownDescription: "Retrieves the list of resource group business units from the VayuCloud API for a given firewall.\n\nThis data source calls `GET /securityservice/list-businessunit-state/{firewallId}`.\n\nOptionally, use `filter` blocks to narrow down results by field values.",

		Blocks: map[string]schema.Block{
			"filter": client.FilterBlockSchema(),
		},

		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "Placeholder identifier for Terraform.",
				Computed:    true,
			},
			"firewall_id": schema.Int64Attribute{
				Description: "The firewall ID to list business units for.",
				Required:    true,
			},
			"business_units": schema.ListNestedAttribute{
				Description: "The list of resource group business units.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"resource_group_business_unit_id": schema.Int64Attribute{
							Description: "The business unit resource ID.",
							Computed:    true,
						},
						"business_unit": schema.StringAttribute{
							Description: "The name of the business unit.",
							Computed:    true,
						},
						"users": schema.ListAttribute{
							Description: "The list of user emails associated with the business unit.",
							Computed:    true,
							ElementType: types.StringType,
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

func (d *ResourceGroupBusinessUnitListDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ResourceGroupBusinessUnitListDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ResourceGroupBusinessUnitListDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	firewallID := data.FirewallID.ValueInt64()

	err := network_firewall.ValidateFirewallExists(d.client, ctx, firewallID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Validating Firewall",
			err.Error(),
		)
		return
	}

	tflog.Debug(ctx, "Reading resource group business unit list data source", map[string]any{
		"firewall_id": firewallID,
	})

	items, actionStateResp, err := ListBusinessUnits(d.client, ctx, firewallID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Listing Business Units",
			fmt.Sprintf("Could not list business units for firewall %d: %s",
				firewallID, err.Error()),
		)
		return
	}

	data.ID = types.Int64Value(firewallID)
	data.Status = types.StringValue(actionStateResp.Status)
	data.Message = types.StringValue(actionStateResp.Message)
	data.ResponseCode = types.Int64Value(int64(actionStateResp.ResponseCode))

	if len(items) == 0 {
		resp.Diagnostics.AddWarning(
			"Empty Response Data",
			"No business units found for the given firewall.",
		)
		data.BusinessUnits = []BusinessUnitListItemModel{}
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
		return
	}

	businessUnits := make([]BusinessUnitListItemModel, len(items))
	for i, bu := range items {
		var usersList types.List
		var diags diag.Diagnostics
		if bu.Users == nil {
			usersList = types.ListNull(types.StringType)
		} else if rawSlice, ok := bu.Users.([]interface{}); ok {
			strs := make([]string, 0, len(rawSlice))
			for _, v := range rawSlice {
				strs = append(strs, fmt.Sprintf("%v", v))
			}
			usersList, diags = types.ListValueFrom(ctx, types.StringType, strs)
		} else {
			usersList = types.ListNull(types.StringType)
		}
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		businessUnits[i] = BusinessUnitListItemModel{
			ResourceGroupBusinessUnitID: types.Int64Value(bu.ID),
			BusinessUnit:                types.StringValue(bu.BusinessUnit),
			Users:                       usersList,
		}
	}

	if len(data.Filters) > 0 {
		tflog.Debug(ctx, "Applying filters to business units", map[string]any{
			"filter_count":     len(data.Filters),
			"pre_filter_count": len(businessUnits),
		})

		filtered, err := client.ApplyFilters(businessUnits, data.Filters)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Filtering Business Units",
				"Could not apply filters: "+err.Error(),
			)
			return
		}
		businessUnits = filtered

		tflog.Debug(ctx, "Filters applied to business units", map[string]any{
			"post_filter_count": len(businessUnits),
		})
	}

	data.BusinessUnits = businessUnits

	tflog.Info(ctx, "Business unit list read successfully", map[string]any{
		"count":       len(businessUnits),
		"firewall_id": firewallID,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
