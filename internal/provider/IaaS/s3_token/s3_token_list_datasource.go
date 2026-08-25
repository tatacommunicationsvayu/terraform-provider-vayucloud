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

var _ datasource.DataSource = &S3TokenListDataSource{}

func NewS3TokenListDataSource() datasource.DataSource {
	return &S3TokenListDataSource{}
}

type S3TokenListDataSource struct {
	client *client.Client
}

type S3TokenListItemModel struct {
	ID               types.String `tfsdk:"id"`
	DomainID         types.String `tfsdk:"domain_id"`
	UserName         types.String `tfsdk:"user_name"`
	AccessKeyID      types.String `tfsdk:"access_key_id"`
	TokenExpiryDate  types.String `tfsdk:"token_expiry_date"`
	TokenDescription types.String `tfsdk:"token_description"`
}

type S3TokenListDataSourceModel struct {
	ID types.String `tfsdk:"id"`

	DomainID    types.String `tfsdk:"domain_id"`
	UserName    types.String `tfsdk:"user_name"`
	AccessKeyID types.String `tfsdk:"access_key_id"`

	Filters []client.FilterModel `tfsdk:"filter"`

	Tokens []S3TokenListItemModel `tfsdk:"tokens"`

	Status       types.String `tfsdk:"status"`
	Message      types.String `tfsdk:"message"`
	ResponseCode types.Int64  `tfsdk:"response_code"`
}

func (d *S3TokenListDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_s3_token_list"
}

func (d *S3TokenListDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists S3 access tokens within a VayuCloud S3 domain.",
		MarkdownDescription: "Lists S3 access tokens within a VayuCloud S3 domain.\n\n" +
			"When `user_name` is set, calls `GET .../users/{username}/tokens`. " +
			"Otherwise calls `GET .../domain/{domain_id}/tokens` with optional query filters. " +
			"Secrets are **never** included.",

		Blocks: map[string]schema.Block{
			"filter": client.FilterBlockSchema(),
		},

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Placeholder identifier (same as domain_id).",
				Computed:    true,
			},
			"domain_id": schema.StringAttribute{
				Description: "Parent S3 domain id to list tokens for.",
				Required:    true,
			},
			"user_name": schema.StringAttribute{
				Description: "When set, lists tokens for this user only (3–128 chars). When omitted, lists all tokens in the domain.",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(3, 128),
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._@-]*$`),
						"user_name must start with alphanumeric and may contain letters, numbers, dots, underscores, at-signs, and hyphens",
					),
				},
			},
			"access_key_id": schema.StringAttribute{
				Description: "Optional server-side filter by access key / token id (1–256 chars when set).",
				Optional:    true,
				Sensitive:   true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 256),
				},
			},
			"tokens": schema.ListNestedAttribute{
				Description: "S3 tokens (metadata only — no secrets).",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "Composite id `{domain_id}:{user_name}:{access_key_id}`.",
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
						"access_key_id": schema.StringAttribute{
							Description: "Access key / token id.",
							Computed:    true,
							Sensitive:   true,
						},
						"token_expiry_date": schema.StringAttribute{
							Description: "Expiry date when returned by the API.",
							Computed:    true,
						},
						"token_description": schema.StringAttribute{
							Description: "Description when returned by the API.",
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

func (d *S3TokenListDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *S3TokenListDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data S3TokenListDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domainID := data.DomainID.ValueString()
	userName := optionalStringValue(data.UserName)
	accessKey := optionalStringValue(data.AccessKeyID)

	tflog.Debug(ctx, "Reading S3 token list data source", map[string]any{
		"domain_id":     domainID,
		"user_name":     userName,
		"access_key_id": accessKey,
	})

	var (
		items    []TokenPayload
		envelope *CatalystResponse
		err      error
	)

	if userName != "" {
		items, envelope, err = ListUserTokens(d.client, ctx, domainID, userName, accessKey)
	} else {
		items, envelope, err = ListDomainTokens(d.client, ctx, domainID, userName, accessKey)
	}
	if err != nil {
		resp.Diagnostics.AddError("Error Listing S3 Tokens", err.Error())
		return
	}

	data.ID = types.StringValue(domainID)
	data.Status = types.StringValue(envelope.Status)
	data.Message = types.StringValue(envelope.Message)
	data.ResponseCode = types.Int64Value(int64(envelope.ResponseCode))

	if len(items) == 0 {
		resp.Diagnostics.AddWarning(
			"Empty Response Data",
			fmt.Sprintf("No S3 tokens found for domain %s.", domainID),
		)
		data.Tokens = []S3TokenListItemModel{}
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
		return
	}

	tokens := make([]S3TokenListItemModel, len(items))
	for i, item := range items {
		tokens[i] = payloadToListItemModel(&item)
	}

	if len(data.Filters) > 0 {
		filtered, filterErr := client.ApplyFilters(tokens, data.Filters)
		if filterErr != nil {
			resp.Diagnostics.AddError("Error Filtering S3 Tokens", filterErr.Error())
			return
		}
		tokens = filtered
	}

	data.Tokens = tokens

	tflog.Info(ctx, "S3 token list read successfully", map[string]any{
		"domain_id": domainID,
		"count":     len(tokens),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func payloadToListItemModel(payload *TokenPayload) S3TokenListItemModel {
	id := payload.ID
	if id == "" && payload.DomainID != "" && payload.Username != "" && payload.AccessToken != "" {
		id = FormatTokenID(payload.DomainID, payload.Username, payload.AccessToken)
	}
	return S3TokenListItemModel{
		ID:               types.StringValue(id),
		DomainID:         types.StringValue(payload.DomainID),
		UserName:         types.StringValue(payload.Username),
		AccessKeyID:      types.StringValue(payload.AccessToken),
		TokenExpiryDate:  stringAttrOrNull(NormalizeTokenExpiryDate(payload.ExpiryDate)),
		TokenDescription: stringAttrOrNull(payload.TokenDescription),
	}
}

func optionalStringValue(v types.String) string {
	if v.IsNull() || v.IsUnknown() {
		return ""
	}
	return v.ValueString()
}
