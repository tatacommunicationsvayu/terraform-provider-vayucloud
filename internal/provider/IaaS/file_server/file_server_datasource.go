// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package file_server

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/client"
)

var _ datasource.DataSource = &FileServerDataSource{}

// NewFileServerDataSource creates a data source that reads one file server by id.
func NewFileServerDataSource() datasource.DataSource {
	return &FileServerDataSource{}
}

// FileServerDataSource reads file server details from the API.
type FileServerDataSource struct {
	client *client.Client
}

// FileServerDataSourceModel maps the data source schema to API results.
type FileServerDataSourceModel struct {
	ID types.String `tfsdk:"id"`

	// VserverID is the platform file server id (same as vayucloud_file_server.id after create).
	VserverID types.String `tfsdk:"vserver_id"`

	// FileServerName is the display name from the API (populated from FileServerDetail in Read).
	FileServerName  types.String `tfsdk:"file_server_name"`
	EngagementID    types.Int64  `tfsdk:"engagement_id"`
	EndpointID      types.Int64  `tfsdk:"endpoint_id"`
	FileStorageType types.String `tfsdk:"file_storage_type"`
	Description     types.String `tfsdk:"description"`
}

func (d *FileServerDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_file_server"
}

func (d *FileServerDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Reads one file server by platform id via config action-state (module FileServer, action read).",
		MarkdownDescription: "Reads **a single** file server; set the `vserver_id` argument to the platform file server id. The provider calls `POST .../configservice/action-state?module=FileServer&action=read` with `resourceId` set to that id (same pattern as other IaaS resources). Response fields are mapped to `file_server_name`, `engagement_id`, `endpoint_id`, and `file_storage_type`. This data source never lists multiple file servers.\n\nTo list file servers for an engagement and endpoint, use `data.vayucloud_file_server_list` with `engagement_id` and `endpoint_id` (`GET .../nas/getNASVservers/{engagement_id}/{endpoint_id}`).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:         "Synthetic identifier for this data source instance (fileserver-{id} where id is the configured file server id).",
				MarkdownDescription: "Synthetic identifier for this data source instance.",
				Computed:            true,
			},
			"vserver_id": schema.StringAttribute{
				Description:         "Platform file server id to look up (numeric string).",
				MarkdownDescription: "Platform file server id to look up; must match the id returned by the `vayucloud_file_server` resource.",
				Required:            true,
			},
			"file_server_name": schema.StringAttribute{
				Description: "File server display name from the API (`fileServerName`, falling back to legacy `vserverName` / `name` only if missing).",
				Computed:    true,
			},
			"engagement_id": schema.Int64Attribute{
				Description: "Engagement id when returned by the API.",
				Computed:    true,
			},
			"endpoint_id": schema.Int64Attribute{
				Description: "Endpoint id when returned by the API.",
				Computed:    true,
			},
			"file_storage_type": schema.StringAttribute{
				Description: "File storage or protocol type when returned by the API (for example NFS or NAS-NFS).",
				Computed:    true,
			},
			"description": schema.StringAttribute{
				Description: "Description when returned by the API.",
				Computed:    true,
			},
		},
	}
}

func (d *FileServerDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *FileServerDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data FileServerDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	vserverID := data.VserverID.ValueString()
	tflog.Debug(ctx, "Reading file server data source", map[string]any{"file_server_id": vserverID})

	_, detail, err := ReadFileServer(d.client, ctx, vserverID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading File Server",
			fmt.Sprintf("Could not read file server %s: %s", vserverID, err.Error()),
		)
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("fileserver-%s", vserverID))

	name := detail.FileServerName
	data.FileServerName = types.StringValue(name)

	if detail.EngagementID != 0 {
		data.EngagementID = types.Int64Value(detail.EngagementID)
	} else {
		data.EngagementID = types.Int64Null()
	}
	if detail.EndpointID != 0 {
		data.EndpointID = types.Int64Value(detail.EndpointID)
	} else {
		data.EndpointID = types.Int64Null()
	}
	if detail.FileStorageType != "" {
		data.FileStorageType = types.StringValue(detail.FileStorageType)
	} else {
		data.FileStorageType = types.StringNull()
	}
	if detail.Description != "" {
		data.Description = types.StringValue(detail.Description)
	} else {
		data.Description = types.StringNull()
	}

	tflog.Info(ctx, "File server data source read", map[string]any{"file_server_id": vserverID, "file_server_name": name})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
