// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package virtualmachine_flavor

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/client"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/network_zone"
)

var _ datasource.DataSource = &VirtualMachineFlavorDataSource{}

func NewVirtualMachineFlavorDataSource() datasource.DataSource {
	return &VirtualMachineFlavorDataSource{}
}

type VirtualMachineFlavorDataSource struct {
	client *client.Client
}

type PricingModelModel struct {
	PPU      types.String `tfsdk:"ppu"`
	Reserved types.String `tfsdk:"reserved"`
}

type LinuxPartitionModel struct {
	Usr   types.Int64 `tfsdk:"usr"`
	Swap  types.Int64 `tfsdk:"swap"`
	Root  types.Int64 `tfsdk:"root"`
	Boot  types.Int64 `tfsdk:"boot"`
	Kdump types.Int64 `tfsdk:"kdump"`
}

type WindowsPartitionModel struct {
	CDrive types.Int64 `tfsdk:"c_drive"`
	Page   types.Int64 `tfsdk:"page"`
}

type VirtualMachineFlavorModel struct {
	ID                          types.Int64           `tfsdk:"id"`
	Name                        types.String          `tfsdk:"name"`
	ArtifactType                types.String          `tfsdk:"artifact_type"`
	OsModel                     types.String          `tfsdk:"os_model"`
	P2RPricingModel             PricingModelModel     `tfsdk:"p2r_pricing_model"`
	VCpu                        types.Int64           `tfsdk:"vcpu"`
	VRam                        types.Int64           `tfsdk:"vram"`
	VGPU                        types.Int64           `tfsdk:"vgpu"`
	VDiskL                      types.Int64           `tfsdk:"vdisk_l"`
	VDiskW                      types.Int64           `tfsdk:"vdisk_w"`
	RootStoragePartitionLinux   LinuxPartitionModel   `tfsdk:"root_storage_partition_linux"`
	RootStoragePartitionWindows WindowsPartitionModel `tfsdk:"root_storage_partition_windows"`
}

type VirtualMachineFlavorDataSourceModel struct {
	ID types.String `tfsdk:"id"`

	ZoneID types.String `tfsdk:"zone_id"`

	Filters []client.FilterModel `tfsdk:"filter"`

	Flavors []VirtualMachineFlavorModel `tfsdk:"flavors"`

	Status       types.String `tfsdk:"status"`
	Message      types.String `tfsdk:"message"`
	ResponseCode types.Int64  `tfsdk:"response_code"`
}

func (d *VirtualMachineFlavorDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_virtualmachine_flavor"
}

func (d *VirtualMachineFlavorDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Retrieves the list of virtual machine flavors for a given zone from the VayuCloud API.",
		MarkdownDescription: "Retrieves the list of virtual machine flavors for a given zone from the VayuCloud API.\n\nThis data source calls the `vm-instances/{zone_id}/flavors` API to return all available VM flavors for the specified zone.\n\nOptionally, use `filter` blocks to narrow down results by field values.",

		Blocks: map[string]schema.Block{
			"filter": client.FilterBlockSchema(),
		},

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Placeholder identifier for Terraform.",
				Computed:    true,
			},
			"zone_id": schema.StringAttribute{
				Description: "The zone ID to retrieve flavors for.",
				Required:    true,
			},
			"flavors": schema.ListNestedAttribute{
				Description: "The list of virtual machine flavors.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.Int64Attribute{
							Description: "The flavor ID.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "The flavor name (e.g., 'B8', 'N1').",
							Computed:    true,
						},
						"artifact_type": schema.StringAttribute{
							Description: "The artifact type (e.g., 'ipc_standard').",
							Computed:    true,
						},
						"os_model": schema.StringAttribute{
							Description: "The OS model (e.g., 'ubuntu', 'rhel').",
							Computed:    true,
						},
						"p2r_pricing_model": schema.SingleNestedAttribute{
							Description: "The pricing model options for this flavor.",
							Computed:    true,
							Attributes: map[string]schema.Attribute{
								"ppu": schema.StringAttribute{
									Description: "Comma-separated list of pay-per-use pricing options (e.g., 'hourly,daily,monthly').",
									Computed:    true,
								},
								"reserved": schema.StringAttribute{
									Description: "Comma-separated list of reserved pricing options (e.g., 'reserved_1').",
									Computed:    true,
								},
							},
						},
						"vcpu": schema.Int64Attribute{
							Description: "The number of virtual CPUs.",
							Computed:    true,
						},
						"vram": schema.Int64Attribute{
							Description: "The amount of virtual RAM in MB.",
							Computed:    true,
						},
						"vgpu": schema.Int64Attribute{
							Description: "The number of virtual GPUs.",
							Computed:    true,
						},
						"vdisk_l": schema.Int64Attribute{
							Description: "The Linux root disk size in GB.",
							Computed:    true,
						},
						"vdisk_w": schema.Int64Attribute{
							Description: "The Windows root disk size in GB.",
							Computed:    true,
						},
						"root_storage_partition_linux": schema.SingleNestedAttribute{
							Description: "Linux root storage partition sizes in GB.",
							Computed:    true,
							Attributes: map[string]schema.Attribute{
								"usr": schema.Int64Attribute{
									Description: "The /usr partition size in GB.",
									Computed:    true,
								},
								"swap": schema.Int64Attribute{
									Description: "The swap partition size in GB.",
									Computed:    true,
								},
								"root": schema.Int64Attribute{
									Description: "The /root partition size in GB.",
									Computed:    true,
								},
								"boot": schema.Int64Attribute{
									Description: "The /boot partition size in GB.",
									Computed:    true,
								},
								"kdump": schema.Int64Attribute{
									Description: "The kdump partition size in GB.",
									Computed:    true,
								},
							},
						},
						"root_storage_partition_windows": schema.SingleNestedAttribute{
							Description: "Windows root storage partition sizes in GB.",
							Computed:    true,
							Attributes: map[string]schema.Attribute{
								"c_drive": schema.Int64Attribute{
									Description: "The C: drive size in GB.",
									Computed:    true,
								},
								"page": schema.Int64Attribute{
									Description: "The page file size in GB.",
									Computed:    true,
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
				Description: "The API response code (200 = success).",
				Computed:    true,
			},
		},
	}
}

func (d *VirtualMachineFlavorDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *VirtualMachineFlavorDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data VirtualMachineFlavorDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	zoneID := data.ZoneID.ValueString()

	zoneIDInt, err := client.StringToInt64(zoneID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Network Zone ID",
			fmt.Sprintf("Could not parse zone_id %q as a number: %s", zoneID, err.Error()),
		)
		return
	}
	if err := network_zone.ValidateNetworkZoneExists(d.client, ctx, zoneIDInt); err != nil {
		resp.Diagnostics.AddError(
			"Invalid Network Zone ID",
			err.Error(),
		)
		return
	}

	tflog.Debug(ctx, "Reading virtual machine flavor data source", map[string]any{
		"zone_id": zoneID,
	})

	flavorResponse, err := GetVirtualMachineFlavors(d.client, ctx, zoneID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Virtual Machine Flavors",
			fmt.Sprintf("Could not read virtual machine flavors for zone %s: %s", zoneID, err.Error()),
		)
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("vm-flavors-%s", zoneID))
	data.Status = types.StringValue(flavorResponse.Status)
	data.Message = types.StringValue(flavorResponse.Message)
	data.ResponseCode = types.Int64Value(int64(flavorResponse.ResponseCode))

	if len(flavorResponse.Data.Flavors) == 0 {
		resp.Diagnostics.AddWarning(
			"Empty Response Data",
			fmt.Sprintf("No flavors found for zone %s.", zoneID),
		)
		data.Flavors = []VirtualMachineFlavorModel{}
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
		return
	}

	flavors := make([]VirtualMachineFlavorModel, len(flavorResponse.Data.Flavors))
	for i, f := range flavorResponse.Data.Flavors {
		flavors[i] = VirtualMachineFlavorModel{
			ID:           types.Int64Value(f.ID),
			Name:         types.StringValue(f.Name),
			ArtifactType: types.StringValue(f.ArtifactType),
			OsModel:      types.StringValue(f.OsModel),
			P2RPricingModel: PricingModelModel{
				PPU:      types.StringValue(strings.Join(f.P2RPricingModel.PPU, ",")),
				Reserved: types.StringValue(strings.Join(f.P2RPricingModel.Reserved, ",")),
			},
			VCpu:   types.Int64Value(f.VCpu),
			VRam:   types.Int64Value(f.VRam),
			VGPU:   types.Int64Value(f.VGPU),
			VDiskL: types.Int64Value(f.VDiskL),
			VDiskW: types.Int64Value(f.VDiskW),
			RootStoragePartitionLinux: LinuxPartitionModel{
				Usr:   types.Int64Value(f.RootStoragePartitionLinux.Usr),
				Swap:  types.Int64Value(f.RootStoragePartitionLinux.Swap),
				Root:  types.Int64Value(f.RootStoragePartitionLinux.Root),
				Boot:  types.Int64Value(f.RootStoragePartitionLinux.Boot),
				Kdump: types.Int64Value(f.RootStoragePartitionLinux.Kdump),
			},
			RootStoragePartitionWindows: WindowsPartitionModel{
				CDrive: types.Int64Value(f.RootStoragePartitionWindows.CDrive),
				Page:   types.Int64Value(f.RootStoragePartitionWindows.Page),
			},
		}
	}

	if len(data.Filters) > 0 {
		tflog.Debug(ctx, "Applying filters to virtual machine flavors", map[string]any{
			"filter_count":     len(data.Filters),
			"pre_filter_count": len(flavors),
		})

		filtered, err := client.ApplyFilters(flavors, data.Filters)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Filtering Virtual Machine Flavors",
				"Could not apply filters: "+err.Error(),
			)
			return
		}
		flavors = filtered

		tflog.Debug(ctx, "Filters applied to virtual machine flavors", map[string]any{
			"post_filter_count": len(flavors),
		})
	}

	data.Flavors = flavors

	tflog.Info(ctx, "Virtual machine flavors read successfully", map[string]any{
		"zone_id": zoneID,
		"count":   len(flavors),
		"status":  data.Status.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
