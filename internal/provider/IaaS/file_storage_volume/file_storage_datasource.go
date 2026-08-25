// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package file_storage_volume

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/client"
)

var _ datasource.DataSource = &FileStorageDataSource{}

// NewFileStorageDataSource returns a data source that reads volume details via fetchVolumeDetails.
func NewFileStorageDataSource() datasource.DataSource {
	return &FileStorageDataSource{}
}

// FileStorageDataSource reads one file storage volume by engagement, file server id, and volume name.
type FileStorageDataSource struct {
	client *client.Client
}

type fileStorageDataSourceModel struct {
	ID                 types.String `tfsdk:"id"`
	EngagementID       types.Int64  `tfsdk:"engagement_id"`
	FileServerID       types.String `tfsdk:"file_server_id"`
	Name               types.String `tfsdk:"name"`
	VolumeID           types.String `tfsdk:"volume_id"`
	FileServerName     types.String `tfsdk:"file_server_name"`
	Clients            types.List   `tfsdk:"clients"`
	SizeGb             types.Int64  `tfsdk:"size_gb"`
	RawResponse        types.String `tfsdk:"raw_response"`
}

func (d *FileStorageDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_file_storage"
}

func (d *FileStorageDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads one NAS file storage volume using fetchVolumeDetails.",
		MarkdownDescription: "Calls `GET .../nas/fetchVolumeDetails/{engagement_id}/{file_server_id}/{name}` (portal `uat-portalservice/nas` prefix). Supply `engagement_id`, `file_server_id` (platform vserver id), and `name` (volume name). Returns volume id, file server display name, size, and attached client IPs.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Synthetic identifier for this data source instance.",
				Computed:    true,
			},
			"engagement_id": schema.Int64Attribute{
				Description: "Engagement id (first path segment after fetchVolumeDetails).",
				Required:    true,
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
				},
			},
			"file_server_id": schema.StringAttribute{
				Description: "Platform file server id (second path segment; same as `vayucloud_file_server.id`).",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 256),
				},
			},
			"name": schema.StringAttribute{
				Description: "File storage volume name (third path segment; file_storage_name in the API path).",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 256),
				},
			},
			"volume_id": schema.StringAttribute{
				Description: "Platform volume CI id (`volCi` from the API).",
				Computed:    true,
			},
			"file_server_name": schema.StringAttribute{
				Description: "Parent file server display name (`vserverDisplayName` from the API).",
				Computed:    true,
			},
			"clients": schema.ListAttribute{
				Description: "Client IPs attached to this volume.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"size_gb": schema.Int64Attribute{
				Description: "Volume size in GB when returned by the API.",
				Computed:    true,
			},
			"raw_response": schema.StringAttribute{
				Description: "Raw JSON response body from fetchVolumeDetails (for troubleshooting).",
				Computed:    true,
			},
		},
	}
}

func (d *FileStorageDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *FileStorageDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data fileStorageDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	engagementID := data.EngagementID.ValueInt64()
	fileServerID := data.FileServerID.ValueString()
	name := data.Name.ValueString()

	tflog.Debug(ctx, "Reading file storage volume details", map[string]any{
		"engagement_id":  engagementID,
		"file_server_id": fileServerID,
		"name":           name,
	})

	lookup, err := GetFileStorageVolumeDetails(d.client, ctx, engagementID, fileServerID, name)
	if err != nil {
		if errors.Is(err, ErrFileStorageVolumeNotFound) {
			resp.Diagnostics.AddError(
				"File Storage Volume Not Found",
				fmt.Sprintf("fetchVolumeDetails returned no volume for engagement_id=%d, file_server_id=%s, name=%q.", engagementID, fileServerID, name),
			)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading File Storage Volume",
			err.Error(),
		)
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("file-storage-%d-%s-%s", engagementID, fileServerID, name))
	if lookup.VolumeID != "" {
		data.VolumeID = types.StringValue(lookup.VolumeID)
	} else {
		data.VolumeID = types.StringNull()
	}
	if lookup.VserverDisplayName != "" {
		data.FileServerName = types.StringValue(lookup.VserverDisplayName)
	} else {
		data.FileServerName = types.StringNull()
	}
	clientElems := make([]basetypes.StringValue, 0, len(lookup.Clients))
	for _, ip := range lookup.Clients {
		clientElems = append(clientElems, types.StringValue(ip))
	}
	clientsList, diags := types.ListValueFrom(ctx, types.StringType, clientElems)
	resp.Diagnostics.Append(diags...)
	if !resp.Diagnostics.HasError() {
		data.Clients = clientsList
	}
	if lookup.SizeGb > 0 {
		data.SizeGb = types.Int64Value(lookup.SizeGb)
	} else {
		data.SizeGb = types.Int64Null()
	}
	data.RawResponse = types.StringValue(lookup.RawBody)

	tflog.Info(ctx, "File storage volume details read", map[string]any{
		"engagement_id": engagementID, "file_server_id": fileServerID, "volume_id": lookup.VolumeID,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
