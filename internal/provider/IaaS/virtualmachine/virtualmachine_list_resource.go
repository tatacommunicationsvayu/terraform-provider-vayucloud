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
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/network_zone"
)

var _ datasource.DataSource = &VirtualMachineListDataSource{}

func NewVirtualMachineListDataSource() datasource.DataSource {
	return &VirtualMachineListDataSource{}
}

type VirtualMachineListDataSource struct {
	client *client.Client
}

type VirtualMachineListVolumeModel struct {
	ID          types.Int64  `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Size        types.Int64  `tfsdk:"size"`
	DiskType    types.String `tfsdk:"disk_type"`
	CreatedDate types.String `tfsdk:"created_date"`
	IOPS        types.Int64  `tfsdk:"iops"`
}

type VirtualMachineListItemModel struct {
	InstanceID    types.Int64  `tfsdk:"instance_id"`
	Name          types.String `tfsdk:"name"`
	Hostname      types.String `tfsdk:"hostname"`
	ZoneID        types.Int64  `tfsdk:"zone_id"`
	IP            types.String `tfsdk:"ip"`
	RootDiskSize  types.Int64  `tfsdk:"root_disk"`
	PowerStatus   types.String `tfsdk:"power_status"`
	FlavorName    types.String `tfsdk:"flavor_name"`
	ImageName     types.String `tfsdk:"image_name"`
	OsType        types.String `tfsdk:"os_type"`
	OsVersion     types.String `tfsdk:"os_version"`
	OsModel       types.String `tfsdk:"os_model"`
	OsMake        types.String `tfsdk:"os_make"`
	OsServicePack types.String `tfsdk:"os_service_pack"`
	PricingModel  types.String `tfsdk:"pricing_model"`
	VCpu          types.Int64  `tfsdk:"vcpu"`
	VRam          types.Int64  `tfsdk:"vram"`

	Volumes []VirtualMachineListVolumeModel `tfsdk:"volumes"`
}

type VirtualMachineListDataSourceModel struct {
	ID types.Int64 `tfsdk:"id"`

	ZoneID types.Int64 `tfsdk:"zone_id"`

	Filters []client.FilterModel `tfsdk:"filter"`

	VirtualMachines []VirtualMachineListItemModel `tfsdk:"virtual_machines"`

	Status       types.String `tfsdk:"status"`
	Message      types.String `tfsdk:"message"`
	ResponseCode types.Int64  `tfsdk:"response_code"`
}

func (d *VirtualMachineListDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_virtualmachine_list"
}

func (d *VirtualMachineListDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Retrieves the list of virtual machines from the VayuCloud API for a given network zone.",
		MarkdownDescription: "Retrieves the list of virtual machines from the VayuCloud API for a given network zone.\n\nThis data source calls `GET /vm-instances/list-instance-state/{zoneId}`.\n\nOptionally, use `filter` blocks to narrow down results by field values.",

		Blocks: map[string]schema.Block{
			"filter": client.FilterBlockSchema(),
		},

		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "Placeholder identifier for Terraform.",
				Computed:    true,
			},
			"zone_id": schema.Int64Attribute{
				Description: "The network zone ID to list virtual machines for.",
				Required:    true,
			},
			"virtual_machines": schema.ListNestedAttribute{
				Description: "The list of virtual machines.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"instance_id": schema.Int64Attribute{
							Description: "The virtual machine instance ID.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "The name of the virtual machine.",
							Computed:    true,
						},
						"hostname": schema.StringAttribute{
							Description: "The hostname of the virtual machine.",
							Computed:    true,
						},
						"zone_id": schema.Int64Attribute{
							Description: "The zone ID where the virtual machine is deployed.",
							Computed:    true,
						},
						"ip": schema.StringAttribute{
							Description: "The IP address of the virtual machine.",
							Computed:    true,
						},
						"root_disk": schema.Int64Attribute{
							Description: "The root disk size in GB.",
							Computed:    true,
						},
						"power_status": schema.StringAttribute{
							Description: "The power status of the virtual machine (e.g., 'ACTIVE').",
							Computed:    true,
						},
						"flavor_name": schema.StringAttribute{
							Description: "The flavor name of the virtual machine.",
							Computed:    true,
						},
						"image_name": schema.StringAttribute{
							Description: "The image name of the virtual machine.",
							Computed:    true,
						},
						"os_type": schema.StringAttribute{
							Description: "The OS type (e.g., 'Linux', 'Windows').",
							Computed:    true,
						},
						"os_version": schema.StringAttribute{
							Description: "The OS version.",
							Computed:    true,
						},
						"os_model": schema.StringAttribute{
							Description: "The OS model (e.g., 'Ubuntu Linux').",
							Computed:    true,
						},
						"os_make": schema.StringAttribute{
							Description: "The OS make (e.g., 'Ubuntu').",
							Computed:    true,
						},
						"os_service_pack": schema.StringAttribute{
							Description: "The OS service pack.",
							Computed:    true,
						},
						"pricing_model": schema.StringAttribute{
							Description: "The pricing model of the virtual machine (e.g., 'hourly').",
							Computed:    true,
						},
						"vcpu": schema.Int64Attribute{
							Description: "The number of virtual CPUs allocated.",
							Computed:    true,
						},
						"vram": schema.Int64Attribute{
							Description: "The amount of virtual RAM in MB.",
							Computed:    true,
						},
						"volumes": schema.ListNestedAttribute{
							Description: "The list of volumes attached to the virtual machine.",
							Computed:    true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"id": schema.Int64Attribute{
										Description: "The volume ID.",
										Computed:    true,
									},
									"name": schema.StringAttribute{
										Description: "The volume name.",
										Computed:    true,
									},
									"size": schema.Int64Attribute{
										Description: "The volume size in GB.",
										Computed:    true,
									},
									"disk_type": schema.StringAttribute{
										Description: "The disk type (e.g., 'Root', 'Data').",
										Computed:    true,
									},
									"created_date": schema.StringAttribute{
										Description: "The volume creation date.",
										Computed:    true,
									},
									"iops": schema.Int64Attribute{
										Description: "The IOPS value for the volume.",
										Computed:    true,
									},
								},
							},
						},
					},
				},
			},
			"status": schema.StringAttribute{
				Description: "The API response status (e.g., 'success').",
				Computed:    true,
			},
			"message": schema.StringAttribute{
				Description: "Additional information from the API response.",
				Computed:    true,
			},
			"response_code": schema.Int64Attribute{
				Description: "The API response code (0 = success).",
				Computed:    true,
			},
		},
	}
}

func (d *VirtualMachineListDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *VirtualMachineListDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data VirtualMachineListDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	zoneID := data.ZoneID.ValueInt64()

	err := network_zone.ValidateNetworkZoneExists(d.client, ctx, zoneID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Validating Network Zone",
			err.Error(),
		)
		return
	}

	tflog.Debug(ctx, "Reading virtual machine list data source", map[string]any{
		"zone_id": zoneID,
	})

	items, actionStateResp, err := ListVirtualMachines(d.client, ctx, zoneID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Listing Virtual Machines",
			fmt.Sprintf("Could not list virtual machines for zone %d: %s",
				zoneID, err.Error()),
		)
		return
	}

	data.ID = types.Int64Value(zoneID)
	data.Status = types.StringValue(actionStateResp.Status)
	data.Message = types.StringValue(actionStateResp.Message)
	data.ResponseCode = types.Int64Value(int64(actionStateResp.ResponseCode))

	if len(items) == 0 {
		resp.Diagnostics.AddWarning(
			"Empty Response Data",
			"No virtual machines found for the given zone.",
		)
		data.VirtualMachines = []VirtualMachineListItemModel{}
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
		return
	}

	vms := make([]VirtualMachineListItemModel, len(items))
	for i, vm := range items {
		volumes := make([]VirtualMachineListVolumeModel, len(vm.Volumes))
		for j, v := range vm.Volumes {
			volumes[j] = VirtualMachineListVolumeModel{
				ID:          types.Int64Value(v.ID),
				Name:        types.StringValue(v.Name),
				Size:        types.Int64Value(v.Size),
				DiskType:    types.StringValue(v.DiskType),
				CreatedDate: types.StringValue(v.CreatedDate),
				IOPS:        types.Int64Value(v.IOPS),
			}
		}

		vms[i] = VirtualMachineListItemModel{
			InstanceID:    types.Int64Value(vm.ID),
			Name:          types.StringValue(vm.Name),
			Hostname:      types.StringValue(vm.Hostname),
			ZoneID:        types.Int64Value(vm.ZoneID),
			IP:            types.StringValue(vm.IP),
			RootDiskSize:  types.Int64Value(vm.RootDiskSize),
			PowerStatus:   types.StringValue(vm.PowerStatus),
			FlavorName:    types.StringValue(vm.FlavorName),
			ImageName:     types.StringValue(vm.ImageName),
			OsType:        types.StringValue(vm.OsType),
			OsVersion:     types.StringValue(vm.OsVersion),
			OsModel:       types.StringValue(vm.OsModel),
			OsMake:        types.StringValue(vm.OsMake),
			OsServicePack: types.StringValue(vm.OsServicePack),
			PricingModel:  types.StringValue(vm.PricingModel),
			VCpu:          types.Int64Value(vm.VCpu),
			VRam:          types.Int64Value(vm.VRam),
			Volumes:       volumes,
		}
	}

	if len(data.Filters) > 0 {
		tflog.Debug(ctx, "Applying filters to virtual machines", map[string]any{
			"filter_count":     len(data.Filters),
			"pre_filter_count": len(vms),
		})

		filtered, err := client.ApplyFilters(vms, data.Filters)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Filtering Virtual Machines",
				"Could not apply filters: "+err.Error(),
			)
			return
		}
		vms = filtered

		tflog.Debug(ctx, "Filters applied to virtual machines", map[string]any{
			"post_filter_count": len(vms),
		})
	}

	data.VirtualMachines = vms

	tflog.Info(ctx, "Virtual machine list read successfully", map[string]any{
		"count":   len(vms),
		"zone_id": zoneID,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
