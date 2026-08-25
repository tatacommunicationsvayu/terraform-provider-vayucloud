// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package file_server

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/client"
)

var _ datasource.DataSource = &FileServerListDataSource{}

// NewFileServerListDataSource lists NAS vservers for an engagement and endpoint (getNASVservers).
func NewFileServerListDataSource() datasource.DataSource {
	return &FileServerListDataSource{}
}

// FileServerListDataSource returns NAS vservers the API exposes for one engagement and endpoint.
type FileServerListDataSource struct {
	client *client.Client
}

// FileServerListItemModel is one vserver in the list output.
type FileServerListItemModel struct {
	VserverID        types.String `tfsdk:"vserver_id"`
	VserverName      types.String `tfsdk:"vserver_name"`
	EngagementID     types.Int64  `tfsdk:"engagement_id"`
	EndpointID       types.Int64  `tfsdk:"endpoint_id"`
	FileStorageType  types.String `tfsdk:"file_storage_type"`
	Description      types.String `tfsdk:"description"`
}

// FileServerListDataSourceModel is the list data source state.
type FileServerListDataSourceModel struct {
	ID types.String `tfsdk:"id"`

	EngagementID types.Int64 `tfsdk:"engagement_id"`
	EndpointID   types.Int64 `tfsdk:"endpoint_id"`

	Vservers []FileServerListItemModel `tfsdk:"vservers"`

	Status       types.String `tfsdk:"status"`
	Message      types.String `tfsdk:"message"`
	ResponseCode types.Int64  `tfsdk:"response_code"`
	RawResponse  types.String `tfsdk:"raw_response"`
}

func (d *FileServerListDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_file_server_list"
}

func (d *FileServerListDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Lists NAS vservers (file servers) for the given engagement and endpoint.",
		MarkdownDescription: "Returns NAS vservers the portal API exposes for `engagement_id` and `endpoint_id`, using `GET .../nas/getNASVservers/{engagement_id}/{endpoint_id}`. Results appear in the computed `vservers` list. For a **single** vserver by id, use `data.vayucloud_file_server` with `vserver_id`.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Synthetic id (fileservers-{engagement_id}-{endpoint_id}).",
				Computed:    true,
			},
			"engagement_id": schema.Int64Attribute{
				Description:         "Engagement id for the getNASVservers path.",
				MarkdownDescription: "Engagement id in the path `getNASVservers/{engagement_id}/{endpoint_id}`.",
				Required:            true,
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
				},
			},
			"endpoint_id": schema.Int64Attribute{
				Description:         "Endpoint id for the getNASVservers path.",
				MarkdownDescription: "Endpoint id in the path `getNASVservers/{engagement_id}/{endpoint_id}`.",
				Required:            true,
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
				},
			},
			"vservers": schema.ListNestedAttribute{
				Description:         "NAS vservers returned for this engagement and endpoint (may be empty).",
				MarkdownDescription: "Every vserver object returned by `getNASVservers` for this engagement and endpoint.",
				Computed:              true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"vserver_id": schema.StringAttribute{
							Description: "Platform vserver id (use with `vayucloud_file_server` or single `data.vayucloud_file_server`).",
							Computed:    true,
						},
						"vserver_name": schema.StringAttribute{
							Description: "File Server name.",
							Computed:    true,
						},
						"engagement_id": schema.Int64Attribute{
							Description: "Engagement id when present in the row.",
							Computed:    true,
						},
						"endpoint_id": schema.Int64Attribute{
							Description: "Endpoint id when present in the row.",
							Computed:    true,
						},
						"file_storage_type": schema.StringAttribute{
							Description: "File storage type when present.",
							Computed:    true,
						},
						"description": schema.StringAttribute{
							Description: "Description when present.",
							Computed:    true,
						},
					},
				},
			},
			"status": schema.StringAttribute{
				Description: "API status when returned by the envelope.",
				Computed:    true,
			},
			"message": schema.StringAttribute{
				Description: "API message when returned.",
				Computed:    true,
			},
			"response_code": schema.Int64Attribute{
				Description: "API responseCode when returned.",
				Computed:    true,
			},
			"raw_response": schema.StringAttribute{
				Description: "Raw JSON response for debugging.",
				Computed:    true,
			},
		},
	}
}

func (d *FileServerListDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T.", req.ProviderData),
		)
		return
	}
	d.client = c
}

func (d *FileServerListDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data FileServerListDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	engID := data.EngagementID.ValueInt64()
	endpointID := data.EndpointID.ValueInt64()
	tflog.Debug(ctx, "Reading file_server_list data source", map[string]any{
		"engagement_id": engID,
		"endpoint_id":   endpointID,
	})

	listResp, err := ListNASVservers(d.client, ctx, engID, endpointID)
	if err != nil {
		resp.Diagnostics.AddError("Error Listing File Servers", err.Error())
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("fileservers-%d-%d", engID, endpointID))
	data.Status = types.StringValue(listResp.Status)
	data.Message = types.StringValue(listResp.Message)
	data.ResponseCode = types.Int64Value(int64(listResp.ResponseCode))
	data.RawResponse = types.StringValue(listResp.RawBody)

	if len(listResp.Items) == 0 {
		resp.Diagnostics.AddWarning(
			"Empty NAS vserver list",
			fmt.Sprintf("getNASVservers returned no parseable rows for engagement_id=%d endpoint_id=%d. Check raw_response if the API shape differs.", engID, endpointID),
		)
	}

	items := make([]FileServerListItemModel, 0, len(listResp.Items))
	for _, e := range listResp.Items {
		row := FileServerListItemModel{
			VserverID:   types.StringValue(strconv.FormatInt(e.VserverID, 10)),
			VserverName: types.StringValue(e.VserverName),
		}
		if e.EngagementID != 0 {
			row.EngagementID = types.Int64Value(e.EngagementID)
		} else {
			row.EngagementID = types.Int64Null()
		}
		if e.EndpointID != 0 {
			row.EndpointID = types.Int64Value(e.EndpointID)
		} else {
			row.EndpointID = types.Int64Null()
		}
		if e.FileStorageType != "" {
			row.FileStorageType = types.StringValue(e.FileStorageType)
		} else {
			row.FileStorageType = types.StringNull()
		}
		if e.Description != "" {
			row.Description = types.StringValue(e.Description)
		} else {
			row.Description = types.StringNull()
		}
		items = append(items, row)
	}
	data.Vservers = items

	tflog.Info(ctx, "File server list read", map[string]any{
		"engagement_id": engID,
		"endpoint_id":   endpointID,
		"count":         len(items),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
