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

var _ datasource.DataSource = &S3UserDataSource{}

func NewS3UserDataSource() datasource.DataSource {
	return &S3UserDataSource{}
}

type S3UserDataSource struct {
	client *client.Client
}

type S3UserDataSourceModel struct {
	ID types.String `tfsdk:"id"`

	DomainID types.String `tfsdk:"domain_id"`
	UserName types.String `tfsdk:"user_name"`

	Status       types.String `tfsdk:"status"`
	Message      types.String `tfsdk:"message"`
	ResponseCode types.Int64  `tfsdk:"response_code"`
}

func (d *S3UserDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_s3_user"
}

func (d *S3UserDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves details of an S3 user from the VayuCloud ICS operations API.",
		MarkdownDescription: "Retrieves details of an S3 user from the VayuCloud ICS operations API.\n\n" +
			"Calls `GET /ics-operations/domain/{domain_id}/users/{username}`.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Composite id `{domain_id}:{user_name}`.",
				Computed:    true,
			},
			"domain_id": schema.StringAttribute{
				Description: "Parent S3 domain id.",
				Required:    true,
			},
			"user_name": schema.StringAttribute{
				Description: "S3 username to look up (3–128 chars; alphanumeric start; letters, numbers, `.`, `_`, `@`, `-`).",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(3, 128),
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._@-]*$`),
						"user_name must start with alphanumeric and may contain letters, numbers, dots, underscores, at-signs, and hyphens",
					),
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

func (d *S3UserDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *S3UserDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data S3UserDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domainID := data.DomainID.ValueString()
	userName := data.UserName.ValueString()

	tflog.Debug(ctx, "Reading S3 user data source", map[string]any{
		"domain_id": domainID,
		"user_name": userName,
	})

	payload, envelope, err := GetUser(d.client, ctx, domainID, userName)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading S3 User", err.Error())
		return
	}

	data.ID = types.StringValue(payload.ID)
	if payload.ID == "" {
		data.ID = types.StringValue(FormatUserID(domainID, userName))
	}
	data.Status = types.StringValue(envelope.Status)
	data.Message = types.StringValue(envelope.Message)
	data.ResponseCode = types.Int64Value(int64(envelope.ResponseCode))

	tflog.Info(ctx, "S3 user data source read successfully", map[string]any{
		"id": data.ID.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
