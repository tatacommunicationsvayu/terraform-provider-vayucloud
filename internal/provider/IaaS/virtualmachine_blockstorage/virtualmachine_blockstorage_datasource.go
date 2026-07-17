// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package virtualmachine_blockstorage

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/client"
)

var _ datasource.DataSource = &VirtualMachineBlockStorageDataSource{}

// NewVirtualMachineBlockStorageDataSource creates the data source.
func NewVirtualMachineBlockStorageDataSource() datasource.DataSource {
	return &VirtualMachineBlockStorageDataSource{}
}

// VirtualMachineBlockStorageDataSource reads a single attached volume on a VM.
type VirtualMachineBlockStorageDataSource struct {
	client *client.Client
}

// VirtualMachineBlockStorageDataSourceModel is the Terraform model for the data source.
type VirtualMachineBlockStorageDataSourceModel struct {
	ID types.String `tfsdk:"id"`

	InstanceID types.Int64 `tfsdk:"instance_id"`
	VolumeID   types.Int64  `tfsdk:"volume_id"`

	Name        types.String `tfsdk:"name"`
	Size        types.Int64  `tfsdk:"size"`
	IOPS        types.Int64  `tfsdk:"iops"`
	DiskType    types.String `tfsdk:"disk_type"`
	CreatedDate types.String `tfsdk:"created_date"`
}

func (d *VirtualMachineBlockStorageDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_virtualmachine_blockstorage"
}

func (d *VirtualMachineBlockStorageDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Reads an attached block storage volume on a virtual machine by instance and volume ID.",
		MarkdownDescription: "Reads an attached block storage volume on a virtual machine by instance and volume ID, using the instance detail API.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Synthetic identifier for this data source (instance_id and volume_id).",
				Computed:    true,
			},
			"instance_id": schema.Int64Attribute{
				Description: "The virtual machine instance ID.",
				Required:    true,
			},
			"volume_id": schema.Int64Attribute{
				Description: "The volume (disk) ID.",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "The volume name.",
				Computed:    true,
			},
			"size": schema.Int64Attribute{
				Description: "The volume size in GB.",
				Computed:    true,
			},
			"iops": schema.Int64Attribute{
				Description: "The IOPS value for the volume.",
				Computed:    true,
			},
			"disk_type": schema.StringAttribute{
				Description: "The disk type (e.g. HDD, SSD, root).",
				Computed:    true,
			},
			"created_date": schema.StringAttribute{
				Description: "The volume creation date from the API.",
				Computed:    true,
			},
		},
	}
}

func (d *VirtualMachineBlockStorageDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *VirtualMachineBlockStorageDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data VirtualMachineBlockStorageDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	instanceID := data.InstanceID.ValueInt64()
	volumeID := data.VolumeID.ValueInt64()

	tflog.Debug(ctx, "Reading virtual machine block storage", map[string]any{
		"instance_id": instanceID,
		"volume_id":   volumeID,
	})

	vol, err := GetVolumeByID(d.client, ctx, instanceID, volumeID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Volume",
			err.Error(),
		)
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("%s-%d", instanceID, volumeID))
	data.Name = types.StringValue(vol.Name)
	data.Size = types.Int64Value(vol.Size)
	data.IOPS = types.Int64Value(vol.IOPS)
	data.DiskType = types.StringValue(vol.DiskType)
	data.CreatedDate = types.StringValue(vol.CreatedDate)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
