// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package virtualmachine

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/client"
)

var _ datasource.DataSource = &VirtualMachineDataSource{}

func NewVirtualMachineDataSource() datasource.DataSource {
	return &VirtualMachineDataSource{}
}

type VirtualMachineDataSource struct {
	client *client.Client
}

type VirtualMachineVolumeModel struct {
	ID          types.Int64  `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Size        types.Int64 `tfsdk:"size"`
	DiskType    types.String `tfsdk:"disk_type"`
	CreatedDate types.String `tfsdk:"created_date"`
	IOPS        types.Int64  `tfsdk:"iops"`
}

type VirtualMachineDataSourceModel struct {
	ID types.String `tfsdk:"id"`

	// Required input
	InstanceID types.String `tfsdk:"instance_id"`

	// Computed virtual machine attributes
	Name          types.String `tfsdk:"name"`
	Hostname      types.String `tfsdk:"hostname"`
	ZoneID        types.Int64  `tfsdk:"zone_id"`
	IP            types.String `tfsdk:"ip"`
	RootDiskSize  types.Int64  `tfsdk:"root_disk"`
	PowerStatus   types.String `tfsdk:"power_status"`
	FlavorID      types.Int64  `tfsdk:"flavor_id"`
	ImageID       types.Int64  `tfsdk:"image_id"`
	OsType        types.String `tfsdk:"os_type"`
	OsVersion     types.String `tfsdk:"os_version"`
	OsModel       types.String `tfsdk:"os_model"`
	OsMake        types.String `tfsdk:"os_make"`
	OsServicePack types.String `tfsdk:"os_service_pack"`
	PricingModel  types.String `tfsdk:"pricing_model"`
	VCpu          types.Int64  `tfsdk:"vcpu"`
	VRam          types.Int64  `tfsdk:"vram"`

	// Computed volumes
	Volumes []VirtualMachineVolumeModel `tfsdk:"volumes"`

	// API response metadata
	Status       types.String `tfsdk:"status"`
	Message      types.String `tfsdk:"message"`
	ResponseCode types.Int64  `tfsdk:"response_code"`
}

func (d *VirtualMachineDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_virtualmachine"
}

func (d *VirtualMachineDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Retrieves the details of a virtual machine from the VayuCloud API.",
		MarkdownDescription: "Retrieves the details of a virtual machine from the VayuCloud API.\n\nThis data source calls the `vm-instances/{instanceId}` API to return the full details of a specific VM, including its volumes, OS information, and resource allocation.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:         "Placeholder identifier for Terraform.",
				MarkdownDescription: "Placeholder identifier for Terraform.",
				Computed:            true,
			},
			"instance_id": schema.StringAttribute{
				Description:         "The instance ID to retrieve details for.",
				MarkdownDescription: "The instance ID to retrieve details for.",
				Required:            true,
			},
			"name": schema.StringAttribute{
				Description:         "The name of the virtual machine.",
				MarkdownDescription: "The name of the virtual machine.",
				Computed:            true,
			},
			"hostname": schema.StringAttribute{
				Description:         "The hostname of the virtual machine.",
				MarkdownDescription: "The hostname of the virtual machine.",
				Computed:            true,
			},
			"zone_id": schema.Int64Attribute{
				Description:         "The zone ID where the virtual machine is deployed.",
				MarkdownDescription: "The zone ID where the virtual machine is deployed.",
				Computed:            true,
			},
			"ip": schema.StringAttribute{
				Description:         "The IP address of the virtual machine.",
				MarkdownDescription: "The IP address of the virtual machine.",
				Computed:            true,
			},
			"root_disk": schema.Int64Attribute{
				Description:         "The root disk size in GB.",
				MarkdownDescription: "The root disk size in GB.",
				Computed:            true,
			},
			"power_status": schema.StringAttribute{
				Description:         "The power status of the virtual machine (e.g., 'ACTIVE').",
				MarkdownDescription: "The power status of the virtual machine (e.g., `ACTIVE`).",
				Computed:            true,
			},
			"flavor_id": schema.Int64Attribute{
				Description:         "The flavor ID of the virtual machine.",
				MarkdownDescription: "The flavor ID of the virtual machine.",
				Computed:            true,
			},
			"image_id": schema.Int64Attribute{
				Description:         "The image ID of the virtual machine.",
				MarkdownDescription: "The image ID of the virtual machine.",
				Computed:            true,
			},
			"os_type": schema.StringAttribute{
				Description:         "The OS type (e.g., 'Linux', 'Windows').",
				MarkdownDescription: "The OS type (e.g., `Linux`, `Windows`).",
				Computed:            true,
			},
			"os_version": schema.StringAttribute{
				Description:         "The OS version (e.g., '24.04 LTS').",
				MarkdownDescription: "The OS version (e.g., `24.04 LTS`).",
				Computed:            true,
			},
			"os_model": schema.StringAttribute{
				Description:         "The OS model (e.g., 'Ubuntu Linux').",
				MarkdownDescription: "The OS model (e.g., `Ubuntu Linux`).",
				Computed:            true,
			},
			"os_make": schema.StringAttribute{
				Description:         "The OS make (e.g., 'Ubuntu').",
				MarkdownDescription: "The OS make (e.g., `Ubuntu`).",
				Computed:            true,
			},
			"os_service_pack": schema.StringAttribute{
				Description:         "The OS service pack (e.g., 'NA').",
				MarkdownDescription: "The OS service pack (e.g., `NA`).",
				Computed:            true,
			},
			"pricing_model": schema.StringAttribute{
				Description:         "The pricing model of the virtual machine (e.g., 'hourly').",
				MarkdownDescription: "The pricing model of the virtual machine (e.g., `hourly`).",
				Computed:            true,
			},
			"vcpu": schema.Int64Attribute{
				Description:         "The number of virtual CPUs allocated.",
				MarkdownDescription: "The number of virtual CPUs allocated.",
				Computed:            true,
			},
			"vram": schema.Int64Attribute{
				Description:         "The amount of virtual RAM in MB.",
				MarkdownDescription: "The amount of virtual RAM in MB.",
				Computed:            true,
			},
			"volumes": schema.ListNestedAttribute{
				Description:         "The list of volumes attached to the virtual machine.",
				MarkdownDescription: "The list of volumes attached to the virtual machine.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.Int64Attribute{
							Description:         "The volume ID.",
							MarkdownDescription: "The volume ID.",
							Computed:            true,
						},
						"name": schema.StringAttribute{
							Description:         "The volume name.",
							MarkdownDescription: "The volume name.",
							Computed:            true,
						},
						"size": schema.Int64Attribute{
							Description:         "The volume size in GB.",
							MarkdownDescription: "The volume size in GB.",
							Computed:            true,
						},
						"disk_type": schema.StringAttribute{
							Description:         "The disk type (e.g., 'Root', 'Data').",
							MarkdownDescription: "The disk type (e.g., `Root`, `Data`).",
							Computed:            true,
						},
						"created_date": schema.StringAttribute{
							Description:         "The volume creation date.",
							MarkdownDescription: "The volume creation date.",
							Computed:            true,
						},
						"iops": schema.Int64Attribute{
							Description:         "The IOPS value for the volume.",
							MarkdownDescription: "The IOPS value for the volume.",
							Computed:            true,
						},
					},
				},
			},
			"status": schema.StringAttribute{
				Description:         "The API response status (e.g., 'success').",
				MarkdownDescription: "The API response status (e.g., `success`).",
				Computed:            true,
			},
			"message": schema.StringAttribute{
				Description:         "Additional information from the API response.",
				MarkdownDescription: "Additional information from the API response.",
				Computed:            true,
			},
			"response_code": schema.Int64Attribute{
				Description:         "The API response code (200 = success).",
				MarkdownDescription: "The API response code (`200` = success).",
				Computed:            true,
			},
		},
	}
}

func (d *VirtualMachineDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *VirtualMachineDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data VirtualMachineDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	instanceID := data.InstanceID.ValueString()

	tflog.Debug(ctx, "Reading virtual machine data source", map[string]any{
		"instance_id": instanceID,
	})

	vmResp, err := GetVirtualMachineDetail(d.client, ctx, instanceID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Virtual Machine",
			fmt.Sprintf("Could not read virtual machine %s: %s", instanceID, err.Error()),
		)
		return
	}

	vm := vmResp.Data

	data.ID = types.StringValue(fmt.Sprintf("virtualmachine-%s", instanceID))
	data.Name = types.StringValue(vm.Name)
	data.Hostname = types.StringValue(vm.Hostname)
	data.ZoneID = types.Int64Value(vm.ZoneID)
	data.IP = types.StringValue(vm.IP)
	data.RootDiskSize = types.Int64Value(vm.RootDiskSize)
	data.PowerStatus = types.StringValue(vm.PowerStatus)
	data.FlavorID = types.Int64Value(vm.FlavorID)
	data.ImageID = types.Int64Value(vm.ImageID)
	data.OsType = types.StringValue(vm.OsType)
	data.OsVersion = types.StringValue(vm.OsVersion)
	data.OsModel = types.StringValue(vm.OsModel)
	data.OsMake = types.StringValue(vm.OsMake)
	data.OsServicePack = types.StringValue(vm.OsServicePack)
	data.PricingModel = types.StringValue(vm.PricingModel)
	data.VCpu = types.Int64Value(vm.VCpu)
	data.VRam = types.Int64Value(vm.VRam)

	data.Status = types.StringValue(vmResp.Status)
	data.Message = types.StringValue(vmResp.Message)
	data.ResponseCode = types.Int64Value(int64(vmResp.ResponseCode))

	volumes := make([]VirtualMachineVolumeModel, len(vm.Volumes))
	for i, v := range vm.Volumes {
		volumes[i] = VirtualMachineVolumeModel{
			ID:          types.Int64Value(v.ID),
			Name:        types.StringValue(v.Name),
			Size:        types.Int64Value(v.Size),
			DiskType:    types.StringValue(v.DiskType),
			CreatedDate: types.StringValue(v.CreatedDate),
			IOPS:        types.Int64Value(v.IOPS),
		}
	}
	data.Volumes = volumes

	tflog.Info(ctx, "Virtual machine read successfully", map[string]any{
		"instance_id":  instanceID,
		"name":         vm.Name,
		"power_status": vm.PowerStatus,
		"volume_count": len(volumes),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
