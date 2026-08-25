// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package s3_user

import (
	"context"
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/client"
)

var _ datasource.DataSource = &S3UserListDataSource{}

func NewS3UserListDataSource() datasource.DataSource {
	return &S3UserListDataSource{}
}

type S3UserListDataSource struct {
	client *client.Client
}

type S3UserListItemModel struct {
	ID       types.String `tfsdk:"id"`
	DomainID types.String `tfsdk:"domain_id"`
	UserName types.String `tfsdk:"user_name"`
}

type S3UserListDataSourceModel struct {
	ID types.String `tfsdk:"id"`

	DomainID types.String `tfsdk:"domain_id"`
	UserName types.String `tfsdk:"user_name"`

	Filters []client.FilterModel `tfsdk:"filter"`

	Users []S3UserListItemModel `tfsdk:"users"`

	Status       types.String `tfsdk:"status"`
	Message      types.String `tfsdk:"message"`
	ResponseCode types.Int64  `tfsdk:"response_code"`
}

func (d *S3UserListDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_s3_user_list"
}

func (d *S3UserListDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists S3 users within a VayuCloud S3 domain.",
		MarkdownDescription: "Lists S3 users within a VayuCloud S3 domain.\n\n" +
			"Calls `GET /ics-operations/domain/{domain_id}/users` with optional `user_name` query filter. " +
			"Use `filter` blocks to narrow results client-side.",

		Blocks: map[string]schema.Block{
			"filter": client.FilterBlockSchema(),
		},

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Placeholder identifier (same as domain_id).",
				Computed:    true,
			},
			"domain_id": schema.StringAttribute{
				Description: "Parent S3 domain id to list users for.",
				Required:    true,
			},
			"user_name": schema.StringAttribute{
				Description: "Optional server-side filter — returns users matching this username (3–128 chars when set).",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(3, 128),
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._@-]*$`),
						"user_name must start with alphanumeric and may contain letters, numbers, dots, underscores, at-signs, and hyphens",
					),
				},
			},
			"users": schema.ListNestedAttribute{
				Description: "S3 users in the domain.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "Composite id `{domain_id}:{user_name}`.",
							Computed:    true,
						},
						"domain_id": schema.StringAttribute{
							Description: "Parent domain id.",
							Computed:    true,
						},
						"user_name": schema.StringAttribute{
							Description: "S3 username.",
							Computed:    true,
						},
					},
				},
			},
			"status": schema.StringAttribute{
				Description: "API response status.",
				Computed:    true,
			},
			"message": schema.StringAttribute{
				Description: "API response message.",
				Computed:    true,
			},
			"response_code": schema.Int64Attribute{
				Description: "API response code.",
				Computed:    true,
			},
		},
	}
}

func (d *S3UserListDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *S3UserListDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data S3UserListDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domainID := data.DomainID.ValueString()
	usernameFilter := ""
	if !data.UserName.IsNull() && !data.UserName.IsUnknown() {
		usernameFilter = data.UserName.ValueString()
	}

	tflog.Debug(ctx, "Reading S3 user list data source", map[string]any{
		"domain_id": domainID,
		"user_name": usernameFilter,
	})

	items, envelope, err := ListUsers(d.client, ctx, domainID, usernameFilter)
	if err != nil {
		resp.Diagnostics.AddError("Error Listing S3 Users", err.Error())
		return
	}

	data.ID = types.StringValue(domainID)
	data.Status = types.StringValue(envelope.Status)
	data.Message = types.StringValue(envelope.Message)
	data.ResponseCode = types.Int64Value(int64(envelope.ResponseCode))

	if len(items) == 0 {
		resp.Diagnostics.AddWarning(
			"Empty Response Data",
			fmt.Sprintf("No S3 users found for domain %s.", domainID),
		)
		data.Users = []S3UserListItemModel{}
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
		return
	}

	users := make([]S3UserListItemModel, len(items))
	for i, item := range items {
		users[i] = payloadToListItemModel(&item)
	}

	if len(data.Filters) > 0 {
		filtered, filterErr := client.ApplyFilters(users, data.Filters)
		if filterErr != nil {
			resp.Diagnostics.AddError("Error Filtering S3 Users", filterErr.Error())
			return
		}
		users = filtered
	}

	data.Users = users

	tflog.Info(ctx, "S3 user list read successfully", map[string]any{
		"domain_id": domainID,
		"count":     len(users),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func payloadToListItemModel(payload *UserPayload) S3UserListItemModel {
	id := payload.ID
	if id == "" && payload.DomainID != "" && payload.Username != "" {
		id = FormatUserID(payload.DomainID, payload.Username)
	}
	return S3UserListItemModel{
		ID:       types.StringValue(id),
		DomainID: types.StringValue(payload.DomainID),
		UserName: types.StringValue(payload.Username),
	}
}
