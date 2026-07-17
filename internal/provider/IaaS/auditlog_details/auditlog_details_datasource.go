// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package auditlog_details

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/client"
)

var _ datasource.DataSource = &AuditLogDetailsDataSource{}

func NewAuditLogDetailsDataSource() datasource.DataSource {
	return &AuditLogDetailsDataSource{}
}

type AuditLogDetailsDataSource struct {
	client *client.Client
}

type AuditLogCommentModel struct {
	UpdatedTime types.String `tfsdk:"updated_time"`
	CommentType types.String `tfsdk:"comment_type"`
	UpdatedBy   types.String `tfsdk:"updated_by"`
	CommentID   types.String `tfsdk:"comment_id"`
	Comment     types.String `tfsdk:"comment"`
}

type AuditLogDetailsDataSourceModel struct {
	ID types.String `tfsdk:"id"`

	AuditID          types.String           `tfsdk:"audit_id"`
	UpdatedTime      types.String           `tfsdk:"updated_time"`
	ResourceID       types.String           `tfsdk:"resource_id"`
	UpdatedBy        types.String           `tfsdk:"updated_by"`
	Comments         []AuditLogCommentModel `tfsdk:"comments"`
	ResponseStatus   types.String           `tfsdk:"response_status"`
	ResourceCategory types.String           `tfsdk:"resource_category"`
	Output           types.String           `tfsdk:"output"`
	Input            types.String           `tfsdk:"input"`
	Environment      types.String           `tfsdk:"environment"`
	CreatedBy        types.String           `tfsdk:"created_by"`
	Action           types.String           `tfsdk:"action"`
	CreatedTime      types.String           `tfsdk:"created_time"`
	EngagementID     types.String           `tfsdk:"engagement_id"`
	ResourceType     types.String           `tfsdk:"resource_type"`
	Status           types.String           `tfsdk:"status"`
}

func (d *AuditLogDetailsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_auditlog_details"
}

func (d *AuditLogDetailsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Retrieves detailed information for a specific audit log entry from the VayuCloud API.",
		MarkdownDescription: "Retrieves detailed information for a specific audit log entry from the VayuCloud API.\n\nThis data source calls the `auditlogservice/auditlog/info/{auditId}` API to return the full audit log detail including comments, input/output, and status.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:         "Placeholder identifier for Terraform.",
				MarkdownDescription: "Placeholder identifier for Terraform.",
				Computed:            true,
			},
			"audit_id": schema.StringAttribute{
				Description:         "The audit log ID to fetch details for.",
				MarkdownDescription: "The audit log ID to fetch details for (e.g., `IPC-a48668e7-235a-49f5-90b6-64a1d852436d`).",
				Required:            true,
			},
			"updated_time": schema.StringAttribute{
				Description:         "The timestamp when the audit entry was last updated.",
				MarkdownDescription: "The timestamp when the audit entry was last updated.",
				Computed:            true,
			},
			"resource_id": schema.StringAttribute{
				Description:         "The resource ID associated with this audit log.",
				MarkdownDescription: "The resource ID associated with this audit log.",
				Computed:            true,
			},
			"updated_by": schema.StringAttribute{
				Description:         "The user who last updated the audit entry.",
				MarkdownDescription: "The user who last updated the audit entry.",
				Computed:            true,
			},
			"comments": schema.ListNestedAttribute{
				Description:         "The list of comments associated with this audit log entry.",
				MarkdownDescription: "The list of comments associated with this audit log entry.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"updated_time": schema.StringAttribute{
							Description:         "The timestamp when the comment was last updated.",
							MarkdownDescription: "The timestamp when the comment was last updated.",
							Computed:            true,
						},
						"comment_type": schema.StringAttribute{
							Description:         "The type of the comment (e.g., 'EXT').",
							MarkdownDescription: "The type of the comment (e.g., `EXT`).",
							Computed:            true,
						},
						"updated_by": schema.StringAttribute{
							Description:         "The user who last updated the comment.",
							MarkdownDescription: "The user who last updated the comment.",
							Computed:            true,
						},
						"comment_id": schema.StringAttribute{
							Description:         "The unique identifier of the comment.",
							MarkdownDescription: "The unique identifier of the comment.",
							Computed:            true,
						},
						"comment": schema.StringAttribute{
							Description:         "The comment text.",
							MarkdownDescription: "The comment text.",
							Computed:            true,
						},
					},
				},
			},
			"response_status": schema.StringAttribute{
				Description:         "The HTTP response status code from the API.",
				MarkdownDescription: "The HTTP response status code from the API (e.g., `200`).",
				Computed:            true,
			},
			"resource_category": schema.StringAttribute{
				Description:         "The category of the resource.",
				MarkdownDescription: "The category of the resource (e.g., `ENG`).",
				Computed:            true,
			},
			"output": schema.StringAttribute{
				Description:         "The output of the audit action (JSON string).",
				MarkdownDescription: "The output of the audit action (JSON string).",
				Computed:            true,
			},
			"input": schema.StringAttribute{
				Description:         "The input of the audit action.",
				MarkdownDescription: "The input of the audit action.",
				Computed:            true,
			},
			"environment": schema.StringAttribute{
				Description:         "The environment where the action was performed.",
				MarkdownDescription: "The environment where the action was performed (e.g., `UAT`).",
				Computed:            true,
			},
			"created_by": schema.StringAttribute{
				Description:         "The user who created the audit entry.",
				MarkdownDescription: "The user who created the audit entry.",
				Computed:            true,
			},
			"action": schema.StringAttribute{
				Description:         "The action performed.",
				MarkdownDescription: "The action performed (e.g., `deleteFirewall`).",
				Computed:            true,
			},
			"created_time": schema.StringAttribute{
				Description:         "The timestamp when the audit entry was created.",
				MarkdownDescription: "The timestamp when the audit entry was created.",
				Computed:            true,
			},
			"engagement_id": schema.StringAttribute{
				Description:         "The engagement ID associated with this audit log.",
				MarkdownDescription: "The engagement ID associated with this audit log.",
				Computed:            true,
			},
			"resource_type": schema.StringAttribute{
				Description:         "The type of resource.",
				MarkdownDescription: "The type of resource (e.g., `ENG`).",
				Computed:            true,
			},
			"status": schema.StringAttribute{
				Description:         "The status of the audit entry.",
				MarkdownDescription: "The status of the audit entry (e.g., `Completed`).",
				Computed:            true,
			},
		},
	}
}

func (d *AuditLogDetailsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *AuditLogDetailsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data AuditLogDetailsDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	auditID := data.AuditID.ValueString()

	tflog.Debug(ctx, "Reading audit log details data source", map[string]any{
		"audit_id": auditID,
	})

	detailsResponse, err := GetAuditLogDetails(d.client, ctx, auditID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Audit Log Details",
			"Could not read audit log details for audit ID "+auditID+": "+err.Error(),
		)
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("auditlog-details-%s", auditID))
	data.UpdatedTime = types.StringValue(detailsResponse.UpdatedTime)
	data.ResourceID = types.StringValue(detailsResponse.ResourceID.String())
	data.UpdatedBy = types.StringValue(detailsResponse.UpdatedBy)
	data.ResponseStatus = types.StringValue(detailsResponse.ResponseStatus)
	data.ResourceCategory = types.StringValue(detailsResponse.ResourceCategory)
	data.Output = types.StringValue(detailsResponse.Output)
	data.Input = types.StringValue(detailsResponse.Input)
	data.Environment = types.StringValue(detailsResponse.Environment)
	data.CreatedBy = types.StringValue(detailsResponse.CreatedBy)
	data.Action = types.StringValue(detailsResponse.Action)
	data.CreatedTime = types.StringValue(detailsResponse.CreatedTime)
	data.EngagementID = types.StringValue(detailsResponse.EngagementID.String())
	data.ResourceType = types.StringValue(detailsResponse.ResourceType)
	data.Status = types.StringValue(detailsResponse.Status)

	comments := make([]AuditLogCommentModel, len(detailsResponse.Comments))
	for i, c := range detailsResponse.Comments {
		comments[i] = AuditLogCommentModel{
			UpdatedTime: types.StringValue(c.UpdatedTime),
			CommentType: types.StringValue(c.CommentType),
			UpdatedBy:   types.StringValue(c.UpdatedBy),
			CommentID:   types.StringValue(c.CommentID.String()),
			Comment:     types.StringValue(c.Comment),
		}
	}
	data.Comments = comments

	tflog.Info(ctx, "Audit log details read successfully", map[string]any{
		"audit_id":      auditID,
		"status":        data.Status.ValueString(),
		"action":        data.Action.ValueString(),
		"comment_count": len(comments),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
