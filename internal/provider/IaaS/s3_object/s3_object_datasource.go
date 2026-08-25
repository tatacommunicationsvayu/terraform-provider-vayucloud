// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package s3_object

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

var _ datasource.DataSource = &S3ObjectDataSource{}

func NewS3ObjectDataSource() datasource.DataSource {
	return &S3ObjectDataSource{}
}

type S3ObjectDataSource struct {
	client *client.Client
}

type S3ObjectDataSourceModel struct {
	ID types.String `tfsdk:"id"`

	DomainID   types.String `tfsdk:"domain_id"`
	BucketName types.String `tfsdk:"bucket_name"`
	Key        types.String `tfsdk:"key"`

	ETag         types.String `tfsdk:"etag"`
	Size         types.Int64  `tfsdk:"size"`
	ContentType  types.String `tfsdk:"content_type"`
	LastModified types.String `tfsdk:"last_modified"`
	Prefix       types.String `tfsdk:"prefix"`
	FileName     types.String `tfsdk:"file_name"`

	Status       types.String `tfsdk:"status"`
	Message      types.String `tfsdk:"message"`
	ResponseCode types.Int64  `tfsdk:"response_code"`
}

func (d *S3ObjectDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_s3_object"
}

func (d *S3ObjectDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves metadata for an S3 object from the VayuCloud ICS operations API.",
		MarkdownDescription: "Retrieves metadata for an S3 object from the VayuCloud ICS operations API.\n\n" +
			"Calls `GET /ics-operations/domain/{domain_id}/buckets/{bucket_name}/objects?objectKey=`.\n\n" +
			"Object **body download is not available** on the ops API v1 — only metadata is returned.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Composite id `{domain_id}:{bucket_name}:{object_key}`.",
				Computed:    true,
			},
			"domain_id": schema.StringAttribute{
				Description: "Parent S3 domain id.",
				Required:    true,
			},
			"bucket_name": schema.StringAttribute{
				Description: "Bucket name containing the object (3–63 chars; letters, numbers, `.`, `-`; alphanumeric start/end).",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(3, 63),
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9.-]*[a-zA-Z0-9])?$`),
						"bucket_name must start and end with alphanumeric characters and may contain letters, numbers, dots, and hyphens",
					),
				},
			},
			"key": schema.StringAttribute{
				Description: "S3 object key to look up (1–1024 chars; maps to API `object_key`).",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 1024),
				},
			},
			"etag": schema.StringAttribute{
				Description: "Entity tag from the API.",
				Computed:    true,
			},
			"size": schema.Int64Attribute{
				Description: "Object size in bytes.",
				Computed:    true,
			},
			"content_type": schema.StringAttribute{
				Description: "MIME type when returned by the API.",
				Computed:    true,
			},
			"last_modified": schema.StringAttribute{
				Description: "Last modification timestamp (ISO-8601 UTC).",
				Computed:    true,
			},
			"prefix": schema.StringAttribute{
				Description: "Parent path prefix derived from the key.",
				Computed:    true,
			},
			"file_name": schema.StringAttribute{
				Description: "Final path segment derived from the key.",
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

func (d *S3ObjectDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *S3ObjectDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data S3ObjectDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domainID := data.DomainID.ValueString()
	bucketName := data.BucketName.ValueString()
	objectKey := normalizeObjectKey(data.Key.ValueString())

	tflog.Debug(ctx, "Reading S3 object data source", map[string]any{
		"domain_id":   domainID,
		"bucket_name": bucketName,
		"object_key":  objectKey,
	})

	payload, envelope, err := GetObject(d.client, ctx, domainID, bucketName, objectKey)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading S3 Object", err.Error())
		return
	}

	applyPayloadToDataSourceModel(payload, &data)
	data.Status = types.StringValue(envelope.Status)
	data.Message = types.StringValue(envelope.Message)
	data.ResponseCode = types.Int64Value(int64(envelope.ResponseCode))

	tflog.Info(ctx, "S3 object data source read successfully", map[string]any{
		"id": data.ID.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func applyPayloadToDataSourceModel(payload *ObjectPayload, data *S3ObjectDataSourceModel) {
	if payload.ID != "" {
		data.ID = types.StringValue(payload.ID)
	} else {
		data.ID = types.StringValue(FormatObjectID(payload.DomainID, payload.BucketName, payload.ObjectKey))
	}
	data.ETag = stringAttrOrNull(payload.ETag)
	data.Size = types.Int64Value(payload.Size)
	data.ContentType = stringAttrOrNull(payload.ContentType)
	data.LastModified = stringAttrOrNull(payload.LastModified)
	data.Prefix = stringAttrOrNull(payload.Prefix)
	data.FileName = stringAttrOrNull(payload.FileName)
}

func stringAttrOrNull(value string) types.String {
	if value == "" {
		return types.StringNull()
	}
	return types.StringValue(value)
}
