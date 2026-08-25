// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package s3_token

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

var _ datasource.DataSource = &S3TokenDataSource{}

func NewS3TokenDataSource() datasource.DataSource {
	return &S3TokenDataSource{}
}

type S3TokenDataSource struct {
	client *client.Client
}

type S3TokenDataSourceModel struct {
	ID types.String `tfsdk:"id"`

	DomainID         types.String `tfsdk:"domain_id"`
	UserName         types.String `tfsdk:"user_name"`
	AccessKeyID      types.String `tfsdk:"access_key_id"`
	TokenExpiryDate  types.String `tfsdk:"token_expiry_date"`
	TokenDescription types.String `tfsdk:"token_description"`

	Status       types.String `tfsdk:"status"`
	Message      types.String `tfsdk:"message"`
	ResponseCode types.Int64  `tfsdk:"response_code"`
}

func (d *S3TokenDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_s3_token"
}

func (d *S3TokenDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves metadata of an S3 access token from the VayuCloud ICS operations API.",
		MarkdownDescription: "Retrieves metadata of an S3 access token from the VayuCloud ICS operations API.\n\n" +
			"Calls `GET /ics-operations/domain/{domain_id}/users/{username}/tokens/{access_token}`.\n\n" +
			"The secret key is **never** returned by the API.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Composite id `{domain_id}:{user_name}:{access_key_id}`.",
				Computed:    true,
			},
			"domain_id": schema.StringAttribute{
				Description: "Parent S3 domain id.",
				Required:    true,
			},
			"user_name": schema.StringAttribute{
				Description: "S3 username the token belongs to (3–128 chars; alphanumeric start).",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(3, 128),
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._@-]*$`),
						"user_name must start with alphanumeric and may contain letters, numbers, dots, underscores, at-signs, and hyphens",
					),
				},
			},
			"access_key_id": schema.StringAttribute{
				Description: "Access key / token id to look up (1–256 chars).",
				Required:    true,
				Sensitive:   true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 256),
				},
			},
			"token_expiry_date": schema.StringAttribute{
				Description: "Token expiry date when returned by the API.",
				Computed:    true,
			},
			"token_description": schema.StringAttribute{
				Description: "Token description when returned by the API.",
				Computed:    true,
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

func (d *S3TokenDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *S3TokenDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data S3TokenDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domainID := data.DomainID.ValueString()
	userName := data.UserName.ValueString()
	accessKey := data.AccessKeyID.ValueString()

	tflog.Debug(ctx, "Reading S3 token data source", map[string]any{
		"domain_id":     domainID,
		"user_name":     userName,
		"access_key_id": accessKey,
	})

	payload, envelope, err := GetToken(d.client, ctx, domainID, userName, accessKey)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading S3 Token", err.Error())
		return
	}

	data.ID = types.StringValue(payload.ID)
	if payload.ID == "" {
		data.ID = types.StringValue(FormatTokenID(domainID, userName, accessKey))
	}
	data.TokenExpiryDate = stringAttrOrNull(NormalizeTokenExpiryDate(payload.ExpiryDate))
	data.TokenDescription = stringAttrOrNull(payload.TokenDescription)
	data.Status = types.StringValue(envelope.Status)
	data.Message = types.StringValue(envelope.Message)
	data.ResponseCode = types.Int64Value(int64(envelope.ResponseCode))

	tflog.Info(ctx, "S3 token data source read successfully", map[string]any{
		"id": data.ID.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func stringAttrOrNull(s string) types.String {
	if s == "" {
		return types.StringNull()
	}
	return types.StringValue(s)
}
