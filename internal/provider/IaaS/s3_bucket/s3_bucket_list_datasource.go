// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package s3_bucket

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/client"
)

var _ datasource.DataSource = &S3BucketListDataSource{}

func NewS3BucketListDataSource() datasource.DataSource {
	return &S3BucketListDataSource{}
}

type S3BucketListDataSource struct {
	client *client.Client
}

type S3BucketListItemModel struct {
	ID               types.String `tfsdk:"id"`
	DomainID         types.String `tfsdk:"domain_id"`
	BucketName       types.String `tfsdk:"bucket_name"`
	VersioningStatus types.String `tfsdk:"versioning_status"`
	CreatedAt        types.String `tfsdk:"created_at"`
	StorageClass     types.String `tfsdk:"storage_class"`
}

type S3BucketListDataSourceModel struct {
	ID types.String `tfsdk:"id"`

	DomainID types.String `tfsdk:"domain_id"`

	Filters []client.FilterModel `tfsdk:"filter"`

	Buckets []S3BucketListItemModel `tfsdk:"buckets"`

	Status       types.String `tfsdk:"status"`
	Message      types.String `tfsdk:"message"`
	ResponseCode types.Int64  `tfsdk:"response_code"`
}

func (d *S3BucketListDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_s3_bucket_list"
}

func (d *S3BucketListDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists S3 buckets within a VayuCloud S3 domain.",
		MarkdownDescription: "Lists S3 buckets within a VayuCloud S3 domain.\n\n" +
			"Calls `GET /ics-operations/domain/{domain_id}/buckets`. " +
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
				Description: "Parent S3 domain id to list buckets for.",
				Required:    true,
			},
			"buckets": schema.ListNestedAttribute{
				Description: "Buckets in the domain.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "Composite id `{domain_id}:{bucket_name}`.",
							Computed:    true,
						},
						"domain_id": schema.StringAttribute{
							Description: "Parent domain id.",
							Computed:    true,
						},
						"bucket_name": schema.StringAttribute{
							Description: "Normalized bucket name.",
							Computed:    true,
						},
						"versioning_status": schema.StringAttribute{
							Description: "Versioning state.",
							Computed:    true,
						},
						"created_at": schema.StringAttribute{
							Description: "Creation timestamp when available.",
							Computed:    true,
						},
						"storage_class": schema.StringAttribute{
							Description: "Storage class when available.",
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

func (d *S3BucketListDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *S3BucketListDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data S3BucketListDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domainID := data.DomainID.ValueString()

	tflog.Debug(ctx, "Reading S3 bucket list data source", map[string]any{
		"domain_id": domainID,
	})

	items, envelope, err := ListBuckets(d.client, ctx, domainID)
	if err != nil {
		resp.Diagnostics.AddError("Error Listing S3 Buckets", err.Error())
		return
	}

	data.ID = types.StringValue(domainID)
	data.Status = types.StringValue(envelope.Status)
	data.Message = types.StringValue(envelope.Message)
	data.ResponseCode = types.Int64Value(int64(envelope.ResponseCode))

	if len(items) == 0 {
		resp.Diagnostics.AddWarning(
			"Empty Response Data",
			fmt.Sprintf("No S3 buckets found for domain %s.", domainID),
		)
		data.Buckets = []S3BucketListItemModel{}
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
		return
	}

	buckets := make([]S3BucketListItemModel, len(items))
	for i, item := range items {
		buckets[i] = payloadToListItemModel(&item)
	}

	if len(data.Filters) > 0 {
		filtered, filterErr := client.ApplyFilters(buckets, data.Filters)
		if filterErr != nil {
			resp.Diagnostics.AddError("Error Filtering S3 Buckets", filterErr.Error())
			return
		}
		buckets = filtered
	}

	data.Buckets = buckets

	tflog.Info(ctx, "S3 bucket list read successfully", map[string]any{
		"domain_id": domainID,
		"count":     len(buckets),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func payloadToListItemModel(payload *BucketPayload) S3BucketListItemModel {
	id := payload.ID
	if id == "" && payload.DomainID != "" && payload.BucketName != "" {
		id = FormatBucketID(payload.DomainID, payload.BucketName)
	}
	return S3BucketListItemModel{
		ID:               types.StringValue(id),
		DomainID:         types.StringValue(payload.DomainID),
		BucketName:       types.StringValue(payload.BucketName),
		VersioningStatus: types.StringValue(payload.VersioningStatus),
		CreatedAt:        stringAttrOrNull(payload.CreatedAt),
		StorageClass:     stringAttrOrNull(payload.StorageClass),
	}
}
