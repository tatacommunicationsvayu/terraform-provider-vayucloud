// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package virtualmachine_image

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

// Ensure provider-defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &VirtualMachineImageDataSource{}

// NewVirtualMachineImageDataSource creates a new data source for reading VM images.
func NewVirtualMachineImageDataSource() datasource.DataSource {
	return &VirtualMachineImageDataSource{}
}

// VirtualMachineImageDataSource defines the data source implementation.
type VirtualMachineImageDataSource struct {
	client *client.Client
}

// VirtualMachineImageModel represents a single image in the data source model.
type VirtualMachineImageModel struct {
	ID                  types.Int64  `tfsdk:"id"`
	OsType              types.String `tfsdk:"os_type"`
	Name                types.String `tfsdk:"name"`
	OsMake              types.String `tfsdk:"os_make"`
	OsModel             types.String `tfsdk:"os_model"`
	OsVersion           types.String `tfsdk:"os_version"`
	OsServicePack       types.String `tfsdk:"os_service_pack"`
}

// VirtualMachineImageDataSourceModel describes the data source data model.
type VirtualMachineImageDataSourceModel struct {
	// Placeholder identifier for Terraform
	ID types.String `tfsdk:"id"`

	// Required inputs
	ZoneID    types.String `tfsdk:"zone_id"`
	ImageType types.String `tfsdk:"type"`
	OsMake    types.String `tfsdk:"os_make"`

	// Filters
	Filters []client.FilterModel `tfsdk:"filter"`

	// Computed list of images
	Images []VirtualMachineImageModel `tfsdk:"images"`

	// API response metadata
	Status       types.String `tfsdk:"status"`
	Message      types.String `tfsdk:"message"`
	ResponseCode types.Int64  `tfsdk:"response_code"`
}

// Metadata returns the data source type name.
func (d *VirtualMachineImageDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_virtualmachine_image"
}

// Schema defines the schema for the data source.
func (d *VirtualMachineImageDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Retrieves the list of virtual machine images/templates for a given zone from the VayuCloud API.",
		MarkdownDescription: "Retrieves the list of virtual machine images/templates for a given zone from the VayuCloud API.\n\nThis data source calls the `vm-instances/{zone_id}/templates` API to return all available VM images for the specified zone, type, and OS make.\n\nOptionally, use `filter` blocks to narrow down results by field values.",

		Blocks: map[string]schema.Block{
			"filter": client.FilterBlockSchema(),
		},

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:         "Placeholder identifier for Terraform.",
				MarkdownDescription: "Placeholder identifier for Terraform.",
				Computed:            true,
			},
			"zone_id": schema.StringAttribute{
				Description:         "The zone ID to retrieve images for.",
				MarkdownDescription: "The zone ID to retrieve images for.",
				Required:            true,
			},
			"type": schema.StringAttribute{
				Description:         "The image type filter (e.g., 'KVM', 'VCD_ESXI').",
				MarkdownDescription: "The image type filter (e.g., `KVM`, `VCD_ESXI`).",
				Optional:            true,
			},
			"os_make": schema.StringAttribute{
				Description:         "The OS make filter (e.g., 'Ubuntu', 'CentOS', 'Windows').",
				MarkdownDescription: "The OS make filter (e.g., `Ubuntu`, `CentOS`, `Windows`).",
				Optional:            true,
			},
			"images": schema.ListNestedAttribute{
				Description:         "The list of virtual machine images.",
				MarkdownDescription: "The list of virtual machine images.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.Int64Attribute{
							Description:         "The image ID.",
							MarkdownDescription: "The image ID.",
							Computed:            true,
						},
						"os_type": schema.StringAttribute{
							Description:         "The OS type (e.g., 'Linux', 'Windows').",
							MarkdownDescription: "The OS type (e.g., `Linux`, `Windows`).",
							Computed:            true,
						},
						"name": schema.StringAttribute{
							Description:         "The image name.",
							MarkdownDescription: "The image name.",
							Computed:            true,
						},
						"os_make": schema.StringAttribute{
							Description:         "The OS make (e.g., 'Ubuntu', 'CentOS').",
							MarkdownDescription: "The OS make (e.g., `Ubuntu`, `CentOS`).",
							Computed:            true,
						},
						"os_model": schema.StringAttribute{
							Description:         "The OS model (e.g., 'Ubuntu Linux').",
							MarkdownDescription: "The OS model (e.g., `Ubuntu Linux`).",
							Computed:            true,
						},
						"os_version": schema.StringAttribute{
							Description:         "The OS version (e.g., '22.04 LTS').",
							MarkdownDescription: "The OS version (e.g., `22.04 LTS`).",
							Computed:            true,
						},
						"os_service_pack": schema.StringAttribute{
							Description:         "The OS service pack, if any.",
							MarkdownDescription: "The OS service pack, if any.",
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

// Configure adds the provider configured client to the data source.
func (d *VirtualMachineImageDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

// Read refreshes the Terraform state with the latest data from the API.
func (d *VirtualMachineImageDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data VirtualMachineImageDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	zoneID := data.ZoneID.ValueString()

	// Validate that the network zone ID exists
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
	var imageType, osMake string
	if !data.ImageType.IsNull() && !data.ImageType.IsUnknown() {
		imageType = data.ImageType.ValueString()
	}
	if !data.OsMake.IsNull() && !data.OsMake.IsUnknown() {
		osMake = data.OsMake.ValueString()
	}

	tflog.Debug(ctx, "Reading virtual machine image data source", map[string]any{
		"zone_id":    zoneID,
		"image_type": imageType,
		"os_make":    osMake,
	})

	imageResponse, err := GetVirtualMachineImages(d.client, ctx, zoneID, imageType, osMake)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Virtual Machine Images",
			fmt.Sprintf("Could not read virtual machine images for zone %s: %s", zoneID, err.Error()),
		)
		return
	}

	// Map top-level API response fields
	data.ID = types.StringValue(fmt.Sprintf("vm-images-%s-%s-%s", zoneID, imageType, osMake))
	data.Status = types.StringValue(imageResponse.Status)
	data.Message = types.StringValue(imageResponse.Message)
	data.ResponseCode = types.Int64Value(int64(imageResponse.ResponseCode))

	if len(imageResponse.Data.Image) == 0 {
		resp.Diagnostics.AddWarning(
			"Empty Response Data",
			fmt.Sprintf("No images found for zone %s with type %s and osMake %s.", zoneID, imageType, osMake),
		)
		data.Images = []VirtualMachineImageModel{}
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
		return
	}

	// Map image data
	images := make([]VirtualMachineImageModel, len(imageResponse.Data.Image))
	for i, img := range imageResponse.Data.Image {
		osServicePack := types.StringNull()
		if img.OsServicePack != nil {
			osServicePack = types.StringValue(*img.OsServicePack)
		}

		images[i] = VirtualMachineImageModel{
			ID:                  types.Int64Value(img.ID),
			OsType:              types.StringValue(img.OsType),
			Name:         		 types.StringValue(img.DisplayName),
			OsMake:              types.StringValue(img.OsMake),
			OsModel:             types.StringValue(img.OsModel),
			OsVersion:           types.StringValue(img.OsVersion),
			OsServicePack:       osServicePack,
		}
	}

	// Apply filters if provided
	if len(data.Filters) > 0 {
		tflog.Debug(ctx, "Applying filters to virtual machine images", map[string]any{
			"filter_count":     len(data.Filters),
			"pre_filter_count": len(images),
		})

		filtered, err := client.ApplyFilters(images, data.Filters)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Filtering Virtual Machine Images",
				"Could not apply filters: "+err.Error(),
			)
			return
		}
		images = filtered

		tflog.Debug(ctx, "Filters applied to virtual machine images", map[string]any{
			"post_filter_count": len(images),
		})
	}

	data.Images = images

	tflog.Info(ctx, "Virtual machine images read successfully", map[string]any{
		"zone_id": zoneID,
		"count":   len(images),
		"status":  data.Status.ValueString(),
	})

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
