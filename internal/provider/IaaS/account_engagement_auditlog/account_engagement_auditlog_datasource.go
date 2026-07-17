// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package account_engagement_auditlog

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/client"
)

var _ datasource.DataSource = &AccountEngagementAuditLogDataSource{}

func NewAccountEngagementAuditLogDataSource() datasource.DataSource {
	return &AccountEngagementAuditLogDataSource{}
}

type AccountEngagementAuditLogDataSource struct {
	client *client.Client
}

type AccountEngagementAuditLogModel struct {
	AuditID          types.String `tfsdk:"audit_id"`
	EngagementID     types.Int64  `tfsdk:"engagement_id"`
	ResourceType     types.String `tfsdk:"resource_type"`
	ResourceID       types.String `tfsdk:"resource_id"`
	ResourceCategory types.String `tfsdk:"resource_category"`
	Action           types.String `tfsdk:"action"`
	Description      types.String `tfsdk:"description"`
	CreatedBy        types.String `tfsdk:"created_by"`
	CreatedTime      types.String `tfsdk:"created_time"`
	UpdatedBy        types.String `tfsdk:"updated_by"`
	UpdatedTime      types.String `tfsdk:"updated_time"`
	Status           types.String `tfsdk:"status"`
	SnowTicketID     types.String `tfsdk:"snow_ticket_id"`
	ClusterName      types.String `tfsdk:"cluster_name"`
	ClusterStatus    types.String `tfsdk:"cluster_status"`
}

type AccountEngagementAuditLogDataSourceModel struct {
	ID types.String `tfsdk:"id"`

	EngagementID     types.Int64 `tfsdk:"engagement_id"`
	StartDate        types.String   `tfsdk:"start_date"`
	EndDate          types.String   `tfsdk:"end_date"`
	Action           []types.String `tfsdk:"action"`
	CreatedBy        types.String   `tfsdk:"created_by"`
	RequestStatus    types.String  `tfsdk:"request_status"`
	AuditID          types.String  `tfsdk:"audit_id"`
	ResourceCategory types.String  `tfsdk:"resource_category"`
	ResourceID       types.String  `tfsdk:"resource_id"`
	Filters          []client.FilterModel `tfsdk:"filter"`
	AuditLogs        []AccountEngagementAuditLogModel `tfsdk:"audit_logs"`
	TotalPages       types.Int64  `tfsdk:"total_pages"`
	TotalElements    types.Int64  `tfsdk:"total_elements"`
	APIStatus        types.String `tfsdk:"api_status"`
}

func (d *AccountEngagementAuditLogDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_account_engagement_auditlog"
}

func (d *AccountEngagementAuditLogDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Retrieves audit logs for a specific account engagement from the VayuCloud API.",
		MarkdownDescription: "Retrieves audit logs for a specific account engagement from the VayuCloud API.\n\nThis data source calls the `auditlog/audits/{engagementId}` API to return audit log entries for the specified engagement.\n\nOptionally, use request body filters (`start_date`, `end_date`, `action`, etc.) to narrow server-side results, and `filter` blocks for additional client-side filtering.",

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
				Description:         "The engagement ID to fetch audit logs for.",
				MarkdownDescription: "The engagement ID to fetch audit logs for.",
				Required:            true,
			},
			"start_date": schema.StringAttribute{
				Description:         "Optional start date filter for the audit log query (server-side).",
				MarkdownDescription: "Optional start date filter for the audit log query (server-side).",
				Optional:            true,
			},
			"end_date": schema.StringAttribute{
				Description:         "Optional end date filter for the audit log query (server-side).",
				MarkdownDescription: "Optional end date filter for the audit log query (server-side).",
				Optional:            true,
			},
			"action": schema.ListAttribute{
				Description:         "Optional list of actions to filter the audit log query (server-side).",
				MarkdownDescription: "Optional list of actions to filter the audit log query (server-side).",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"created_by": schema.StringAttribute{
				Description:         "Optional created_by filter for the audit log query (server-side).",
				MarkdownDescription: "Optional `created_by` filter for the audit log query (server-side).",
				Optional:            true,
			},
			"request_status": schema.StringAttribute{
				Description:         "Optional status filter for the audit log query (server-side).",
				MarkdownDescription: "Optional status filter for the audit log query (server-side).",
				Optional:            true,
			},
			"audit_id": schema.StringAttribute{
				Description:         "Optional audit_id filter for the audit log query (server-side).",
				MarkdownDescription: "Optional `audit_id` filter for the audit log query (server-side).",
				Optional:            true,
			},
			"resource_category": schema.StringAttribute{
				Description:         "Optional resource_category filter for the audit log query (server-side).",
				MarkdownDescription: "Optional `resource_category` filter for the audit log query (server-side).",
				Optional:            true,
			},
			"resource_id": schema.StringAttribute{
				Description:         "Optional resource_id filter for the audit log query (server-side).",
				MarkdownDescription: "Optional `resource_id` filter for the audit log query (server-side).",
				Optional:            true,
			},
			"audit_logs": schema.ListNestedAttribute{
				Description:         "The list of audit log entries.",
				MarkdownDescription: "The list of audit log entries.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"audit_id":          schema.StringAttribute{Description: "The unique audit log identifier.", Computed: true},
						"engagement_id":     schema.Int64Attribute{Description: "The engagement ID.", Computed: true},
						"resource_type":     schema.StringAttribute{Description: "The type of resource (e.g., 'CIM').", Computed: true},
						"resource_id":       schema.StringAttribute{Description: "The resource ID.", Computed: true},
						"resource_category": schema.StringAttribute{Description: "The category of the resource.", Computed: true},
						"action":            schema.StringAttribute{Description: "The action performed.", Computed: true},
						"description":       schema.StringAttribute{Description: "A description of the audit action.", Computed: true},
						"created_by":        schema.StringAttribute{Description: "The user who created the audit entry.", Computed: true},
						"created_time":      schema.StringAttribute{Description: "The timestamp when created.", Computed: true},
						"updated_by":        schema.StringAttribute{Description: "The user who last updated.", Computed: true},
						"updated_time":      schema.StringAttribute{Description: "The timestamp when last updated.", Computed: true},
						"status":            schema.StringAttribute{Description: "The status of the audit entry.", Computed: true},
						"snow_ticket_id":    schema.StringAttribute{Description: "The ServiceNow ticket ID, if applicable.", Computed: true},
						"cluster_name":      schema.StringAttribute{Description: "The cluster name, if applicable.", Computed: true},
						"cluster_status":    schema.StringAttribute{Description: "The cluster status, if applicable.", Computed: true},
					},
				},
			},
			"total_pages": schema.Int64Attribute{
				Description:         "The total number of pages in the paginated response.",
				MarkdownDescription: "The total number of pages in the paginated response.",
				Computed:            true,
			},
			"total_elements": schema.Int64Attribute{
				Description:         "The total number of audit log entries.",
				MarkdownDescription: "The total number of audit log entries.",
				Computed:            true,
			},
			"api_status": schema.StringAttribute{
				Description:         "The API response status (e.g., 'success').",
				MarkdownDescription: "The API response status (e.g., `success`).",
				Computed:            true,
			},
		},
	}
}

func (d *AccountEngagementAuditLogDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T", req.ProviderData))
		return
	}
	d.client = c
}

func (d *AccountEngagementAuditLogDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data AccountEngagementAuditLogDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	engagementID := data.EngagementID.ValueInt64()
	tflog.Debug(ctx, "Reading account engagement audit log data source", map[string]any{"engagement_id": engagementID})

	var actions []string
	if len(data.Action) > 0 {
		actions = make([]string, len(data.Action))
		for i, a := range data.Action {
			actions[i] = a.ValueString()
		}
	}

	auditRequest := &AuditLogRequest{
		StartDate:        optionalStringPtr(data.StartDate),
		EndDate:          optionalStringPtr(data.EndDate),
		Action:           actions,
		CreatedBy:        optionalStringPtr(data.CreatedBy),
		Status:           optionalStringPtr(data.RequestStatus),
		AuditID:          optionalStringPtr(data.AuditID),
		ResourceCategory: optionalStringPtr(data.ResourceCategory),
		ResourceID:       optionalStringPtr(data.ResourceID),
	}

	auditResponse, err := GetAccountEngagementAuditLogs(d.client, ctx, engagementID, auditRequest)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Engagement Audit Logs",
			"Could not read engagement audit logs: "+err.Error())
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("engagement-auditlog-%d", engagementID))
	data.APIStatus = types.StringValue(auditResponse.Status)
	data.TotalPages = types.Int64Value(int64(auditResponse.Data.TotalPages))
	data.TotalElements = types.Int64Value(int64(auditResponse.Data.TotalElements))

	if len(auditResponse.Data.Content) == 0 {
		resp.Diagnostics.AddWarning("Empty Response Data", "No audit logs found.")
		data.AuditLogs = []AccountEngagementAuditLogModel{}
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
		return
	}

	auditLogs := make([]AccountEngagementAuditLogModel, len(auditResponse.Data.Content))
	for i, entry := range auditResponse.Data.Content {
		auditLogs[i] = AccountEngagementAuditLogModel{
			AuditID:          types.StringValue(entry.AuditID),
			EngagementID:     types.Int64Value(entry.EngagementID),
			ResourceType:     types.StringValue(entry.ResourceType),
			ResourceID:       types.StringValue(entry.ResourceID.String()),
			ResourceCategory: types.StringValue(entry.ResourceCategory),
			Action:           types.StringValue(entry.Action),
			Description:      types.StringValue(entry.Description),
			CreatedBy:        types.StringValue(entry.CreatedBy),
			CreatedTime:      types.StringValue(entry.CreatedTime),
			UpdatedBy:        types.StringValue(entry.UpdatedBy),
			UpdatedTime:      types.StringValue(entry.UpdatedTime),
			Status:           types.StringValue(entry.Status),
			SnowTicketID:     nullableStringValue(entry.SnowTicketID),
			ClusterName:      nullableStringValue(entry.ClusterName),
			ClusterStatus:    nullableStringValue(entry.ClusterStatus),
		}
	}

	if len(data.Filters) > 0 {
		filtered, err := client.ApplyFilters(auditLogs, data.Filters)
		if err != nil {
			resp.Diagnostics.AddError("Error Filtering Audit Logs", "Could not apply filters: "+err.Error())
			return
		}
		auditLogs = filtered
	}

	data.AuditLogs = auditLogs
	tflog.Info(ctx, "Engagement audit logs read successfully", map[string]any{"count": len(auditLogs)})
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func optionalStringPtr(v types.String) *string {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	s := v.ValueString()
	return &s
}

func nullableStringValue(s *string) types.String {
	if s == nil {
		return types.StringNull()
	}
	return types.StringValue(*s)
}
