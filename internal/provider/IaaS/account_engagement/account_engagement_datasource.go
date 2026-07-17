// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package account_engagement

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
var _ datasource.DataSource = &AccountEngagementDataSource{}

// NewAccountEngagementDataSource creates a new data source for reading account engagements.
func NewAccountEngagementDataSource() datasource.DataSource {
	return &AccountEngagementDataSource{}
}

// AccountEngagementDataSource defines the data source implementation.
type AccountEngagementDataSource struct {
	client *client.Client
}

// AccountEngagementModel represents a single engagement in the data source model.
type AccountEngagementModel struct {
	// EngagementName is the name of the engagement
	EngagementName types.String `tfsdk:"engagement_name"`

	// ID is the engagement ID
	ID types.Int64 `tfsdk:"id"`

	// EngagementType is the engagement type (e.g., "EPC")
	EngagementType types.String `tfsdk:"engagement_type"`

	// CustomerName is the customer name
	CustomerName types.String `tfsdk:"customer_name"`
}

// AccountEngagementDataSourceModel describes the data source data model.
type AccountEngagementDataSourceModel struct {
	// ID is a placeholder for Terraform (required for data sources)
	ID types.String `tfsdk:"id"`

	// Filters is the optional list of filter blocks
	Filters []client.FilterModel `tfsdk:"filter"`

	// Engagements is the list of engagements returned by the API
	Engagements []AccountEngagementModel `tfsdk:"engagements"`

	// Status is the API response status (e.g., "success")
	Status types.String `tfsdk:"status"`

	// Message contains additional information
	Message types.String `tfsdk:"message"`

	// ResponseCode is the API response code (200 = success)
	ResponseCode types.Int64 `tfsdk:"response_code"`
}

// Metadata returns the data source type name.
func (d *AccountEngagementDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_account_engagement"
}

var engagementTypeMap = map[string]string{
	"EPC": "Vayucloud",
	"IPC - Migration": "Vayucloud",
	"OS":"Vayucloud Storage",
	"PaaS":"Vayucloud Platform as a Service",
	"AI":"Vayucloud AI",
	"AIStudio":"Vayucloud AI Studio",
}

// Schema defines the schema for the data source.
func (d *AccountEngagementDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Retrieves the list of account engagements from the VayuCloud API.",
		MarkdownDescription: "Retrieves the list of account engagements from the VayuCloud API.\n\nThis data source calls the `getuserengagements` API to return all engagements associated with the authenticated user.\n\nOptionally, use `filter` blocks to narrow down results by field values.",

		Blocks: map[string]schema.Block{
			"filter": client.FilterBlockSchema(),
		},

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:         "Placeholder identifier for Terraform.",
				MarkdownDescription: "Placeholder identifier for Terraform.",
				Computed:            true,
			},
			"engagements": schema.ListNestedAttribute{
				Description:         "The list of account engagements.",
				MarkdownDescription: "The list of account engagements.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"engagement_name": schema.StringAttribute{
							Description:         "The name of the engagement.",
							MarkdownDescription: "The name of the engagement.",
							Computed:            true,
						},
						"id": schema.Int64Attribute{
							Description:         "The engagement ID.",
							MarkdownDescription: "The engagement ID.",
							Computed:            true,
						},
						"engagement_type": schema.StringAttribute{
							Description:         "The engagement type (e.g., 'EPC').",
							MarkdownDescription: "The engagement type (e.g., 'Vayucloud', 'Vayucloud Storage', 'Vayucloud Platform as a Service', 'Vayucloud AI', 'Vayucloud AI Studio').",
							Computed:            true,
						},
						"customer_name": schema.StringAttribute{
							Description:         "The customer name.",
							MarkdownDescription: "The customer name.",
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
				Description:         "The API response code (200 = success).",
				MarkdownDescription: "The API response code (`200` = success).",
				Computed:            true,
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *AccountEngagementDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *AccountEngagementDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data AccountEngagementDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Reading account engagement data source")

	// Call the API to get account engagements
	engagementResponse, err := GetAccountEngagements(d.client, ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Account Engagements",
			"Could not read account engagements: "+err.Error(),
		)
		return
	}

	// Map top-level API response fields
	data.ID = types.StringValue("account-engagements")
	data.Status = types.StringValue(engagementResponse.Status)
	data.Message = types.StringValue(engagementResponse.Message)
	data.ResponseCode = types.Int64Value(int64(engagementResponse.ResponseCode))

	// Check if response data is empty
	if len(engagementResponse.Data) == 0 {
		resp.Diagnostics.AddWarning(
			"Empty Response Data",
			"No engagements found in the API response.",
		)
		data.Engagements = []AccountEngagementModel{}
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
		return
	}

	// Map engagement data
	engagements := make([]AccountEngagementModel, len(engagementResponse.Data))
	for i, engagement := range engagementResponse.Data {
		engagementType := engagementTypeMap[engagement.EngagementType]
		if engagementType == "" {
			engagementType = "Unknown"
		}
		engagements[i] = AccountEngagementModel{
			EngagementName: types.StringValue(engagement.EngagementName),
			ID:             types.Int64Value(engagement.ID),
			EngagementType: types.StringValue(engagementType),
			CustomerName:   types.StringValue(engagement.CustomerName),
		}
	}

	// Apply filters if provided
	if len(data.Filters) > 0 {
		tflog.Debug(ctx, "Applying filters to account engagements", map[string]any{
			"filter_count":     len(data.Filters),
			"pre_filter_count": len(engagements),
		})

		filtered, err := client.ApplyFilters(engagements, data.Filters)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Filtering Account Engagements",
				"Could not apply filters: "+err.Error(),
			)
			return
		}
		engagements = filtered

		tflog.Debug(ctx, "Filters applied to account engagements", map[string]any{
			"post_filter_count": len(engagements),
		})
	}

	data.Engagements = engagements

	tflog.Info(ctx, "Account engagements read successfully", map[string]any{
		"count":  len(engagements),
		"status": data.Status.ValueString(),
	})

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
