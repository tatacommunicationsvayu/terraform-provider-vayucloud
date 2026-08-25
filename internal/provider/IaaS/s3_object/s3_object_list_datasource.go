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

var _ datasource.DataSource = &S3ObjectListDataSource{}

func NewS3ObjectListDataSource() datasource.DataSource {
	return &S3ObjectListDataSource{}
}

type S3ObjectListDataSource struct {
	client *client.Client
}

type S3ObjectListItemModel struct {
	ID           types.String `tfsdk:"id"`
	DomainID     types.String `tfsdk:"domain_id"`
	BucketName   types.String `tfsdk:"bucket_name"`
	Key          types.String `tfsdk:"key"`
	Prefix       types.String `tfsdk:"prefix"`
	FileName     types.String `tfsdk:"file_name"`
	ETag         types.String `tfsdk:"etag"`
	Size         types.Int64  `tfsdk:"size"`
	LastModified types.String `tfsdk:"last_modified"`
}

type S3ObjectListDataSourceModel struct {
	ID types.String `tfsdk:"id"`

	DomainID   types.String `tfsdk:"domain_id"`
	BucketName types.String `tfsdk:"bucket_name"`
	Prefix     types.String `tfsdk:"prefix"`

	Filters []client.FilterModel `tfsdk:"filter"`

	Objects []S3ObjectListItemModel `tfsdk:"objects"`

	Status       types.String `tfsdk:"status"`
	Message      types.String `tfsdk:"message"`
	ResponseCode types.Int64  `tfsdk:"response_code"`
}

func (d *S3ObjectListDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_s3_object_list"
}

func (d *S3ObjectListDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists S3 objects within a VayuCloud S3 bucket.",
		MarkdownDescription: "Lists S3 objects within a VayuCloud S3 bucket.\n\n" +
			"Calls `GET /ics-operations/domain/{domain_id}/buckets/{bucket_name}/objects` " +
			"and follows `continuation_token` until all pages are retrieved (1000 keys per page).\n\n" +
			"Use `filter` blocks to narrow results client-side.",

		Blocks: map[string]schema.Block{
			"filter": client.FilterBlockSchema(),
		},

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Placeholder identifier derived from domain_id, bucket_name, and prefix.",
				Computed:    true,
			},
			"domain_id": schema.StringAttribute{
				Description: "Parent S3 domain id.",
				Required:    true,
			},
			"bucket_name": schema.StringAttribute{
				Description: "Bucket name to list objects from (3–63 chars; letters, numbers, `.`, `-`; alphanumeric start/end).",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(3, 63),
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9.-]*[a-zA-Z0-9])?$`),
						"bucket_name must start and end with alphanumeric characters and may contain letters, numbers, dots, and hyphens",
					),
				},
			},
			"prefix": schema.StringAttribute{
				Description: "Optional key prefix filter (maps to `?prefix=`; up to 1024 chars; empty lists from root).",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtMost(1024),
				},
			},
			"objects": schema.ListNestedAttribute{
				Description: "Objects in the bucket matching the prefix.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "Composite id `{domain_id}:{bucket_name}:{object_key}`.",
							Computed:    true,
						},
						"domain_id": schema.StringAttribute{
							Description: "Parent domain id.",
							Computed:    true,
						},
						"bucket_name": schema.StringAttribute{
							Description: "Bucket name.",
							Computed:    true,
						},
						"key": schema.StringAttribute{
							Description: "S3 object key (`object_key` in the API).",
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
						"etag": schema.StringAttribute{
							Description: "Entity tag from the API.",
							Computed:    true,
						},
						"size": schema.Int64Attribute{
							Description: "Object size in bytes.",
							Computed:    true,
						},
						"last_modified": schema.StringAttribute{
							Description: "Last modification timestamp (ISO-8601 UTC).",
							Computed:    true,
						},
					},
				},
			},
			"status": schema.StringAttribute{
				Description: "API response status from the last list page.",
				Computed:    true,
			},
			"message": schema.StringAttribute{
				Description: "API response message from the last list page.",
				Computed:    true,
			},
			"response_code": schema.Int64Attribute{
				Description: "API response code from the last list page.",
				Computed:    true,
			},
		},
	}
}

func (d *S3ObjectListDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *S3ObjectListDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data S3ObjectListDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domainID := data.DomainID.ValueString()
	bucketName := data.BucketName.ValueString()
	prefix := ""
	if !data.Prefix.IsNull() && !data.Prefix.IsUnknown() {
		prefix = data.Prefix.ValueString()
	}

	tflog.Debug(ctx, "Reading S3 object list data source", map[string]any{
		"domain_id":   domainID,
		"bucket_name": bucketName,
		"prefix":      prefix,
	})

	items, envelope, err := ListObjects(d.client, ctx, domainID, bucketName, prefix)
	if err != nil {
		resp.Diagnostics.AddError("Error Listing S3 Objects", err.Error())
		return
	}

	listID := domainID + ":" + bucketName
	if prefix != "" {
		listID += ":" + prefix
	}
	data.ID = types.StringValue(listID)
	data.Status = types.StringValue(envelope.Status)
	data.Message = types.StringValue(envelope.Message)
	data.ResponseCode = types.Int64Value(int64(envelope.ResponseCode))

	objects := make([]S3ObjectListItemModel, len(items))
	for i, item := range items {
		objects[i] = payloadToListItemModel(&item)
	}

	if len(data.Filters) > 0 {
		filtered, filterErr := client.ApplyFilters(objects, data.Filters)
		if filterErr != nil {
			resp.Diagnostics.AddError("Error Filtering S3 Objects", filterErr.Error())
			return
		}
		objects = filtered
	}

	if len(objects) == 0 {
		resp.Diagnostics.AddWarning(
			"Empty Response Data",
			fmt.Sprintf("No S3 objects found in bucket %q for domain %s.", bucketName, domainID),
		)
	}

	data.Objects = objects

	tflog.Info(ctx, "S3 object list read successfully", map[string]any{
		"domain_id":   domainID,
		"bucket_name": bucketName,
		"count":       len(objects),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func payloadToListItemModel(payload *ObjectPayload) S3ObjectListItemModel {
	id := payload.ID
	if id == "" {
		id = FormatObjectID(payload.DomainID, payload.BucketName, payload.ObjectKey)
	}
	return S3ObjectListItemModel{
		ID:           types.StringValue(id),
		DomainID:     types.StringValue(payload.DomainID),
		BucketName:   types.StringValue(payload.BucketName),
		Key:          types.StringValue(payload.ObjectKey),
		Prefix:       stringAttrOrNull(payload.Prefix),
		FileName:     stringAttrOrNull(payload.FileName),
		ETag:         stringAttrOrNull(payload.ETag),
		Size:         types.Int64Value(payload.Size),
		LastModified: stringAttrOrNull(payload.LastModified),
	}
}
