// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package s3_bucket

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

var _ datasource.DataSource = &S3BucketDataSource{}

func NewS3BucketDataSource() datasource.DataSource {
	return &S3BucketDataSource{}
}

type S3BucketDataSource struct {
	client *client.Client
}

type S3BucketDataSourceModel struct {
	ID types.String `tfsdk:"id"`

	DomainID   types.String `tfsdk:"domain_id"`
	BucketName types.String `tfsdk:"bucket_name"`

	VersioningStatus types.String `tfsdk:"versioning_status"`
	CreatedAt        types.String `tfsdk:"created_at"`
	StorageClass     types.String `tfsdk:"storage_class"`

	Status       types.String `tfsdk:"status"`
	Message      types.String `tfsdk:"message"`
	ResponseCode types.Int64  `tfsdk:"response_code"`
}

func (d *S3BucketDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_s3_bucket"
}

func (d *S3BucketDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves details of an S3 bucket from the VayuCloud ICS operations API.",
		MarkdownDescription: "Retrieves details of an S3 bucket from the VayuCloud ICS operations API.\n\n" +
			"Calls `GET /ics-operations/domain/{domain_id}/buckets/{bucket_name}`.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Composite id `{domain_id}:{bucket_name}`.",
				Computed:    true,
			},
			"domain_id": schema.StringAttribute{
				Description: "Parent S3 domain id.",
				Required:    true,
			},
			"bucket_name": schema.StringAttribute{
				Description: "Bucket name to look up (3–63 chars; letters, numbers, `.`, `-`; alphanumeric start/end).",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(3, 63),
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9.-]*[a-zA-Z0-9])?$`),
						"bucket_name must start and end with alphanumeric characters and may contain letters, numbers, dots, and hyphens",
					),
				},
			},
			"versioning_status": schema.StringAttribute{
				Description: "Versioning state: Disabled, Enabled, or Suspended.",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "Creation timestamp when returned by the API.",
				Computed:    true,
			},
			"storage_class": schema.StringAttribute{
				Description: "Domain-derived storage class when returned by the API.",
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

func (d *S3BucketDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *S3BucketDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data S3BucketDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domainID := data.DomainID.ValueString()
	bucketName := data.BucketName.ValueString()

	tflog.Debug(ctx, "Reading S3 bucket data source", map[string]any{
		"domain_id":   domainID,
		"bucket_name": bucketName,
	})

	payload, envelope, err := GetBucket(d.client, ctx, domainID, bucketName)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading S3 Bucket", err.Error())
		return
	}

	data.ID = types.StringValue(payload.ID)
	if payload.ID == "" {
		data.ID = types.StringValue(FormatBucketID(domainID, bucketName))
	}
	data.VersioningStatus = types.StringValue(payload.VersioningStatus)
	data.CreatedAt = stringAttrOrNull(payload.CreatedAt)
	data.StorageClass = stringAttrOrNull(payload.StorageClass)
	data.Status = types.StringValue(envelope.Status)
	data.Message = types.StringValue(envelope.Message)
	data.ResponseCode = types.Int64Value(int64(envelope.ResponseCode))

	tflog.Info(ctx, "S3 bucket data source read successfully", map[string]any{
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
