// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package virtualmachine

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/client"
)

var _ resource.Resource = &VirtualMachineResource{}
var _ resource.ResourceWithImportState = &VirtualMachineResource{}
var _ resource.ResourceWithModifyPlan = &VirtualMachineResource{}

func NewVirtualMachineResource() resource.Resource {
	return &VirtualMachineResource{}
}

type VirtualMachineResource struct {
	client *client.Client
}

type DiskPartitionModel struct {
	Partition types.String `tfsdk:"partition"`
	Size      types.Int64  `tfsdk:"size"`
}

type AdditionalDiskModel struct {
	ID          types.Int64  `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Size        types.Int64  `tfsdk:"size"`
	IOPS        types.Int64  `tfsdk:"iops"`
	DiskType    types.String `tfsdk:"disk_type"` // HDD, SSD
	CreatedDate types.String `tfsdk:"created_date"`
}

type PublicIPModel struct {
	IP                          types.String `tfsdk:"ip"`
	AssignPublicIP              types.String `tfsdk:"assign_public_ip"`
	RetainPublicIPOnTermination types.String `tfsdk:"retain_public_ip_on_termination"`
	PublicIPPricingModel         types.String `tfsdk:"public_ip_pricing_model"`
}

type VirtualMachineResourceModel struct {
	ID types.String `tfsdk:"id"`

	Name                 types.String          `tfsdk:"name"`
	VMPurpose            types.String          `tfsdk:"vm_purpose"`
	ImageID              types.Int64           `tfsdk:"image_id"`
	FlavorID             types.Int64           `tfsdk:"flavor_id"`
	ZoneID               types.Int64           `tfsdk:"zone_id"`
	IOPS                 types.Int64           `tfsdk:"iops"`
	IsKdumpOrPageEnabled types.String          `tfsdk:"is_kdump_or_page_enabled"`
	UsageType            types.String          `tfsdk:"usage_type"`
	PricingModel         types.String          `tfsdk:"pricing_model"`
	RootDiskSize         types.Int64           `tfsdk:"root_disk_size"`
	RootDiskId           types.Int64           `tfsdk:"root_disk_id"`
	DiskPartitions       []DiskPartitionModel  `tfsdk:"root_disk_partitions"`
	// Pointer so Terraform null / omitted optional block decodes correctly (non-pointer struct cannot represent null).
	PublicIP             *PublicIPModel        `tfsdk:"public_ip"`
	AdditionalDisk       []AdditionalDiskModel `tfsdk:"additional_disk"`
	PowerStatus          types.String          `tfsdk:"power_status"`

	AuditID types.String `tfsdk:"audit_id"`
	Status  types.String `tfsdk:"status"`
}

func (r *VirtualMachineResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_virtualmachine"
}

func (r *VirtualMachineResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Manages a virtual machine resource in VayuCloud.",
		MarkdownDescription: "Manages a virtual machine resource in VayuCloud.\n\nThis resource creates a virtual machine through an asynchronous provisioning process. The resource will poll the audit log until the VM creation is complete, then verify via the action state API.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier of the virtual machine (resource ID).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the virtual machine.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"vm_purpose": schema.StringAttribute{
				Description: "The VM purpose (e.g., 'WEB', 'DB', 'OTHERS').",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"image_id": schema.Int64Attribute{
				Description: "The image ID for the virtual machine.",
				Required:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"flavor_id": schema.Int64Attribute{
				Description: "The flavor ID for the virtual machine.",
				Required:    true,
			},
			"zone_id": schema.Int64Attribute{
				Description: "The zone ID where the virtual machine will be created.",
				Required:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"iops": schema.Int64Attribute{
				Description: "The IOPS value for the root disk.",
				Required:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"is_kdump_or_page_enabled": schema.StringAttribute{
				Description: "Whether kdump/page is enabled. 'Yes' for Windows/RHEL/SUSE, 'No' for Ubuntu/Rocky.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("yes", "no", "Yes", "No"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"usage_type": schema.StringAttribute{
				Description: "The usage type (e.g., 'ppu', 'reserved').",
				Optional:    true,
				Computed: 	 true,
				Default:     stringdefault.StaticString("ppu"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"pricing_model": schema.StringAttribute{
				Description: "The pricing model (e.g., 'hourly', 'daily', 'monthly', 'reserved_1').",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("daily"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"root_disk_size": schema.Int64Attribute{
				Description: "The root disk size in GB.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"root_disk_id": schema.Int64Attribute{
				Description: "The root disk ID.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"root_disk_partitions": schema.ListNestedAttribute{
				Description: "List of disk partitions for the root disk.",
				Optional:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"partition": schema.StringAttribute{
							Description: "The partition mount point (e.g., '/root', '/boot').",
							Required:    true,
							Validators: []validator.String{
								stringvalidator.OneOf("/root", "/boot", "/usr", "/swap", "/cDrive", "/page"),
							},
						},
						"size": schema.Int64Attribute{
							Description: "The partition size in GB.",
							Required:    true,
						},
					},
				},
			},
			"additional_disk": schema.ListNestedAttribute{
				Description: "List of additional disks to attach to the virtual machine.",
				Optional:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"size": schema.Int64Attribute{
							Description: "The additional disk size in GB.",
							Required:    true,
						},
						"iops": schema.Int64Attribute{
							Description: "The IOPS value for the additional disk.",
							Required:    true,
						},
						"disk_type": schema.StringAttribute{
							Description: "The type of the additional disk (HDD, SSD).",
							Optional:    true,
							Computed:    true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"created_date": schema.StringAttribute{
							Description: "The created date of the additional disk.",
							Optional:    true,
							Computed:    true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"name": schema.StringAttribute{
							Description: "The name of the additional disk.",
							Optional:    true,
							Computed:    true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"id": schema.Int64Attribute{
							Description: "The ID of the additional disk.",
							Optional:    true,
							Computed:    true,
							PlanModifiers: []planmodifier.Int64{
								int64planmodifier.UseStateForUnknown(),
							},
						},
					},
				},
			},
			"public_ip": schema.SingleNestedAttribute{
				Description: "The public IP configuration.",
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"ip": schema.StringAttribute{
						Description: "The IP address of the virtual machine.",
						Optional:    true,
						Computed:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"assign_public_ip": schema.StringAttribute{
						Description: "Whether to assign a public IP to the virtual machine.",
						Optional:    true,
						Computed:    true,
						Default:     stringdefault.StaticString("no"),
						Validators: []validator.String{
							stringvalidator.OneOf("yes", "no", "Yes", "No"),
						},
					},
					"retain_public_ip_on_termination": schema.StringAttribute{
						Description: "Whether to retain the public IP on termination.",
						Optional:    true,
						Computed:    true,
						Default:     stringdefault.StaticString("yes"),
						Validators: []validator.String{
							stringvalidator.OneOf("yes", "no", "Yes", "No"),
						},
					},
					"public_ip_pricing_model": schema.StringAttribute{
						Description: "The pricing model for the public IP.",
						Optional:    true,
						Computed:    true,
						Default:     stringdefault.StaticString("daily"),
						Validators: []validator.String{
							stringvalidator.OneOf("daily", "monthly", "reserved_1", "reserved_2", "reserved_3"),
						},
					},
				},
			},
			"power_status": schema.StringAttribute{
				Description: "The power status of the virtual machine.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"audit_id": schema.StringAttribute{
				Description: "The audit ID from the creation response.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"status": schema.StringAttribute{
				Description: "The final status from the audit log.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *VirtualMachineResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = c
}

// vmCreateRequestFromPlanModel mirrors the VirtualMachineCreateRequest built in Create (used only by ModifyPlan).
func vmCreateRequestFromPlanModel(data *VirtualMachineResourceModel) *VirtualMachineCreateRequest {
	createReq := &VirtualMachineCreateRequest{
		Name:                 data.Name.ValueString(),
		VMPurpose:            data.VMPurpose.ValueString(),
		ImageID:              data.ImageID.ValueInt64(),
		FlavorID:             data.FlavorID.ValueInt64(),
		ZoneID:               data.ZoneID.ValueInt64(),
		IOPS:                 data.IOPS.ValueInt64(),
		IsKdumpOrPageEnabled: data.IsKdumpOrPageEnabled.ValueString(),
	}

	if !data.UsageType.IsNull() && !data.UsageType.IsUnknown() {
		createReq.UsageType = data.UsageType.ValueString()
	}
	if !data.PricingModel.IsNull() && !data.PricingModel.IsUnknown() {
		createReq.PricingModel = data.PricingModel.ValueString()
	}
	if !data.RootDiskSize.IsNull() && !data.RootDiskSize.IsUnknown() {
		createReq.RootDiskSize = data.RootDiskSize.ValueInt64()
	}
	if data.PublicIP != nil {
		if !data.PublicIP.AssignPublicIP.IsNull() && !data.PublicIP.AssignPublicIP.IsUnknown() && strings.ToLower(data.PublicIP.AssignPublicIP.ValueString()) == "yes" {
			createReq.AssignPublicIp = "yes"
			if !data.PublicIP.RetainPublicIPOnTermination.IsNull() && !data.PublicIP.RetainPublicIPOnTermination.IsUnknown() && strings.ToLower(data.PublicIP.RetainPublicIPOnTermination.ValueString()) == "yes" {
				createReq.RetainPublicIPOnTermination = "yes"
			} else {
				createReq.RetainPublicIPOnTermination = "no"
			}
			if !data.PublicIP.PublicIPPricingModel.IsNull() && !data.PublicIP.PublicIPPricingModel.IsUnknown() {
				createReq.PublicIpPricingModel = data.PublicIP.PublicIPPricingModel.ValueString()
			} else {
				createReq.PublicIpPricingModel = data.PricingModel.ValueString()
			}
		}
	}

	if len(data.DiskPartitions) > 0 {
		partitions := make([]DiskPartition, len(data.DiskPartitions))
		for i, dp := range data.DiskPartitions {
			partitions[i] = DiskPartition{
				Partition: dp.Partition.ValueString(),
				Size:      dp.Size.ValueInt64(),
			}
		}
		createReq.DiskPartitions = partitions
	}

	if len(data.AdditionalDisk) > 0 {
		disks := make([]AdditionalDisk, len(data.AdditionalDisk))
		for i, ad := range data.AdditionalDisk {
			disks[i] = AdditionalDisk{
				Size: ad.Size.ValueInt64(),
				IOPS: ad.IOPS.ValueInt64(),
			}
		}
		createReq.AdditionalDisk = disks
	}

	return createReq
}

// ModifyPlan runs pre-create API validation during plan for new resources only (same payload shape as Create).
func (r *VirtualMachineResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if r.client == nil {
		return
	}
	if req.Plan.Raw.IsNull() {
		return
	}

	var plan VirtualMachineResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Pre-create validate API applies only to new instances (id unknown until apply).
	if !plan.ID.IsUnknown() {
		return
	}

	if plan.Name.IsUnknown() || plan.VMPurpose.IsUnknown() || plan.ImageID.IsUnknown() || plan.FlavorID.IsUnknown() ||
		plan.ZoneID.IsUnknown() || plan.IOPS.IsUnknown() || plan.IsKdumpOrPageEnabled.IsUnknown() {
		return
	}

	createReq := vmCreateRequestFromPlanModel(&plan)
	if err := PreCreateValidateInstance(r.client, ctx, createReq); err != nil {
		resp.Diagnostics.AddError(
			"Virtual Machine Pre-Create Validation Failed",
			err.Error(),
		)
		return
	}
}

func (r *VirtualMachineResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data VirtualMachineResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating virtual machine", map[string]any{
		"name":       data.Name.ValueString(),
		"zone_id":    data.ZoneID.ValueInt64(),
		"image_id":   data.ImageID.ValueInt64(),
		"flavor_id":  data.FlavorID.ValueInt64(),
		"vm_purpose": data.VMPurpose.ValueString(),
	})

	// Build the create request directly from the model
	createReq := &VirtualMachineCreateRequest{
		Name:                 data.Name.ValueString(),
		VMPurpose:            data.VMPurpose.ValueString(),
		ImageID:              data.ImageID.ValueInt64(),
		FlavorID:             data.FlavorID.ValueInt64(),
		ZoneID:               data.ZoneID.ValueInt64(),
		IOPS:                 data.IOPS.ValueInt64(),
		IsKdumpOrPageEnabled: data.IsKdumpOrPageEnabled.ValueString(),
	}

	if !data.UsageType.IsNull() && !data.UsageType.IsUnknown() {
		createReq.UsageType = data.UsageType.ValueString()
	}
	if !data.PricingModel.IsNull() && !data.PricingModel.IsUnknown() {
		createReq.PricingModel = data.PricingModel.ValueString()
	}
	if !data.RootDiskSize.IsNull() && !data.RootDiskSize.IsUnknown() {
		createReq.RootDiskSize = data.RootDiskSize.ValueInt64()
	}
	if data.PublicIP != nil {
		if !data.PublicIP.AssignPublicIP.IsNull() && !data.PublicIP.AssignPublicIP.IsUnknown() && strings.ToLower(data.PublicIP.AssignPublicIP.ValueString()) == "yes" {
			createReq.AssignPublicIp = "yes"
			if !data.PublicIP.RetainPublicIPOnTermination.IsNull() && !data.PublicIP.RetainPublicIPOnTermination.IsUnknown() && strings.ToLower(data.PublicIP.RetainPublicIPOnTermination.ValueString()) == "yes" {
				createReq.RetainPublicIPOnTermination = "yes"
			} else {
				createReq.RetainPublicIPOnTermination = "no"
			}
			if !data.PublicIP.PublicIPPricingModel.IsNull() && !data.PublicIP.PublicIPPricingModel.IsUnknown() {
				createReq.PublicIpPricingModel = data.PublicIP.PublicIPPricingModel.ValueString()
			} else {
				createReq.PublicIpPricingModel = data.PricingModel.ValueString()
			}
		}
	}

	// Map disk partitions
	if len(data.DiskPartitions) > 0 {
		partitions := make([]DiskPartition, len(data.DiskPartitions))
		for i, dp := range data.DiskPartitions {
			partitions[i] = DiskPartition{
				Partition: dp.Partition.ValueString(),
				Size:      dp.Size.ValueInt64(),
			}
		}
		createReq.DiskPartitions = partitions
	}

	// Map additional disks
	if len(data.AdditionalDisk) > 0 {
		disks := make([]AdditionalDisk, len(data.AdditionalDisk))
		for i, ad := range data.AdditionalDisk {
			disks[i] = AdditionalDisk{
				Size: ad.Size.ValueInt64(),
				IOPS: ad.IOPS.ValueInt64(),
			}
		}
		createReq.AdditionalDisk = disks
	}

	// Run pre-create validation
	if err := PreCreateValidateInstance(r.client, ctx, createReq); err != nil {
		resp.Diagnostics.AddError(
			"Virtual Machine Pre-Create Validation Failed",
			err.Error(),
		)
		return
	}

	// Create virtual machine and wait for completion
	auditLog, err := CreateVirtualMachineAndWait(r.client, ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Virtual Machine",
			"Could not create virtual machine: "+err.Error(),
		)
		return
	}

	// Set the ID and audit ID from the audit log
	if auditLog.ResourceID.String() != "" {
		data.ID = types.StringValue(auditLog.ResourceID.String())
	} else {
		resp.Diagnostics.AddError(
			"Error Creating Virtual Machine",
			"Could not create virtual machine: ResourceID is empty",
		)
		return
	}
	data.AuditID = types.StringValue(auditLog.AuditID)
	data.Status = types.StringValue(auditLog.Status)
	data.PowerStatus = types.StringValue("active")

	if data.RootDiskId.IsUnknown() || data.RootDiskId.IsNull() {
		data.RootDiskId = types.Int64Null()
	}
	if data.RootDiskSize.IsUnknown() || data.RootDiskSize.IsNull() {
		data.RootDiskSize = types.Int64Null()
	}
	// If public_ip was omitted in config, keep PublicIP nil so state stays null for the whole block.
	// Do not allocate an empty PublicIPModel{} here — that would turn null into an object with all-null
	// attributes and trigger "was null, but now ..." apply errors.
	if data.PublicIP != nil {
		if data.PublicIP.IP.IsUnknown() || data.PublicIP.IP.IsNull() {
			data.PublicIP.IP = types.StringNull()
		}
		if data.PublicIP.AssignPublicIP.IsUnknown() || data.PublicIP.AssignPublicIP.IsNull() {
			data.PublicIP.AssignPublicIP = types.StringNull()
		}
		if data.PublicIP.RetainPublicIPOnTermination.IsUnknown() || data.PublicIP.RetainPublicIPOnTermination.IsNull() {
			data.PublicIP.RetainPublicIPOnTermination = types.StringNull()
		}
		if data.PublicIP.PublicIPPricingModel.IsUnknown() || data.PublicIP.PublicIPPricingModel.IsNull() {
			data.PublicIP.PublicIPPricingModel = types.StringNull()
		}
	}
	if data.UsageType.IsUnknown() || data.UsageType.IsNull() {
		data.UsageType = types.StringNull()
	}
	if data.PricingModel.IsUnknown() || data.PricingModel.IsNull() {
		data.PricingModel = types.StringNull()
	}
	if data.AdditionalDisk != nil {
		for i := range data.AdditionalDisk {
			if data.AdditionalDisk[i].ID.IsUnknown() || data.AdditionalDisk[i].ID.IsNull() {
				data.AdditionalDisk[i].ID = types.Int64Null()
			}
			if data.AdditionalDisk[i].Name.IsUnknown() || data.AdditionalDisk[i].Name.IsNull() {
				data.AdditionalDisk[i].Name = types.StringNull()
			}
			if data.AdditionalDisk[i].DiskType.IsUnknown() || data.AdditionalDisk[i].DiskType.IsNull() {
				data.AdditionalDisk[i].DiskType = types.StringNull()
			}
			if data.AdditionalDisk[i].CreatedDate.IsUnknown() || data.AdditionalDisk[i].CreatedDate.IsNull() {
				data.AdditionalDisk[i].CreatedDate = types.StringNull()
			}
		}
	}

	tflog.Info(ctx, "Virtual machine created successfully", map[string]any{
		"id":       data.ID.ValueString(),
		"audit_id": data.AuditID.ValueString(),
		"status":   data.Status.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *VirtualMachineResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data VirtualMachineResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	instanceID := data.ID.ValueString()

	tflog.Debug(ctx, "Reading virtual machine", map[string]any{
		"id":       instanceID,
		"audit_id": data.AuditID.ValueString(),
	})

	vmResp, err := GetVirtualMachineDetail(r.client, ctx, instanceID)
	if err != nil {
		resp.Diagnostics.AddWarning(
			"Virtual Machine Not Found",
			fmt.Sprintf("Could not read virtual machine %s, it may have been deleted: %s", instanceID, err.Error()),
		)
		resp.State.RemoveResource(ctx)
		return
	}

	vm := vmResp.Data

	// Update fields available from the API response
	data.Name = types.StringValue(vm.Name)
	data.ImageID = types.Int64Value(vm.ImageID)
	data.FlavorID = types.Int64Value(vm.FlavorID)
	data.ZoneID = types.Int64Value(vm.ZoneID)
	if vm.PricingModel != "" {
		data.PricingModel = types.StringValue(vm.PricingModel)
	}
	// data.PowerStatus = types.StringValue(vm.PowerStatus)
	// API may return root disk size as "rootDisk" or not at all; derive from root volume if zero
	rootSize := vm.RootDiskSize
	data.AdditionalDisk = nil // replace from API; do not append to existing state
	for _, disk := range vm.Volumes {
		if strings.ToLower(disk.DiskType) == "root" {
			data.RootDiskId = types.Int64Value(int64(disk.ID))
			if rootSize == 0 {
				rootSize = disk.Size
			}
		} else {
			data.AdditionalDisk = append(data.AdditionalDisk, AdditionalDiskModel{
				ID:          types.Int64Value(disk.ID),
				Name:        types.StringValue(disk.Name),
				Size:        types.Int64Value(disk.Size),
				IOPS:        types.Int64Value(disk.IOPS),
				DiskType:    types.StringValue(disk.DiskType),
				CreatedDate: types.StringValue(disk.CreatedDate),
			})
		}
	}
	data.RootDiskSize = types.Int64Value(rootSize)

	if vm.PublicIP != "" {
		if data.PublicIP == nil {
			data.PublicIP = &PublicIPModel{}
		}
		data.PublicIP.IP = types.StringValue(vm.PublicIP)
		data.PublicIP.AssignPublicIP = types.StringValue("yes")
		if data.PublicIP.RetainPublicIPOnTermination.IsUnknown() || data.PublicIP.RetainPublicIPOnTermination.IsNull() {
			data.PublicIP.RetainPublicIPOnTermination = types.StringValue("no")
		}
		if !data.PublicIP.PublicIPPricingModel.IsNull() && !data.PublicIP.PublicIPPricingModel.IsUnknown() {
			data.PublicIP.PublicIPPricingModel = types.StringValue(data.PricingModel.ValueString())
		}
	}

	tflog.Info(ctx, "Virtual machine read successfully", map[string]any{
		"id":             instanceID,
		"name":           vm.Name,
		"power_status":   vm.PowerStatus,
		"zone_id":        vm.ZoneID,
		"root_disk_size": vm.RootDiskSize,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *VirtualMachineResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan VirtualMachineResourceModel
	var state VirtualMachineResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Read current state
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Updating virtual machine", map[string]any{
		"id":              plan.ID.ValueString(),
		"root_disk_id":    plan.RootDiskId.ValueInt64(),
		"additional_disk": plan.AdditionalDisk,
	})
	if plan.FlavorID.ValueInt64() != state.FlavorID.ValueInt64() {
		//update flavor
		if err := ValidateFlavor(r.client, ctx, plan.ID.ValueString(), plan.FlavorID.ValueInt64()); err != nil {
			resp.Diagnostics.AddError(
				"Error Updating Flavor",
				"Could not update flavor: "+err.Error(),
			)
			return
		}
		_, err := UpdateFlavorAndWait(r.client, ctx, plan.ID.ValueString(), plan.FlavorID.ValueInt64())
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Updating Flavor",
				"Could not update flavor: "+err.Error(),
			)
			return
		}
		tflog.Info(ctx, "Flavor updated successfully", map[string]any{
			"id":        plan.ID.ValueString(),
			"flavor_id": plan.FlavorID.ValueInt64(),
		})
	}
	if plan.RootDiskSize.ValueInt64() != state.RootDiskSize.ValueInt64() {
		//update root disk size
		// Run pre-create validation
		if err := ValidateVolume(r.client, ctx, plan.ID.ValueString(), plan.RootDiskId.ValueInt64(), plan.RootDiskSize.ValueInt64()); err != nil {
			resp.Diagnostics.AddError(
				"Error Validating Root Disk Size",
				"Could not validate root disk size: "+err.Error(),
			)
			return
		}
		_, err := UpdateVolumeSizeAndWait(r.client, ctx, plan.ID.ValueString(), plan.RootDiskId.ValueInt64(), plan.RootDiskSize.ValueInt64())
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Updating Root Disk Size",
				"Could not update root disk size: "+err.Error(),
			)
			return
		}
		tflog.Info(ctx, "Root disk size updated successfully", map[string]any{
			"id":             plan.ID.ValueString(),
			"root_disk_size": plan.RootDiskSize.ValueInt64(),
		})
	}
	if len(plan.AdditionalDisk) > 0 && len(plan.AdditionalDisk) == len(state.AdditionalDisk) {
		for i := range plan.AdditionalDisk {
			if plan.AdditionalDisk[i].Size.ValueInt64() != state.AdditionalDisk[i].Size.ValueInt64() {
				//update additional disk
				if err := ValidateVolume(r.client, ctx, plan.ID.ValueString(), plan.AdditionalDisk[i].ID.ValueInt64(), plan.AdditionalDisk[i].Size.ValueInt64()); err != nil {
					resp.Diagnostics.AddError(
						"Error Validating Additional Disk Size",
						"Could not validate additional disk size: "+err.Error(),
					)
					return
				}
				_, err := UpdateVolumeSizeAndWait(r.client, ctx, plan.ID.ValueString(), plan.AdditionalDisk[i].ID.ValueInt64(), plan.AdditionalDisk[i].Size.ValueInt64())
				if err != nil {
					resp.Diagnostics.AddError(
						"Error Updating Additional Disk Size",
						"Could not update additional disk size: "+err.Error(),
					)
					return
				}
				tflog.Info(ctx, "Additional disk size updated successfully", map[string]any{
					"id":                   plan.ID.ValueString(),
					"additional_disk_size": plan.AdditionalDisk[i].Size.ValueInt64(),
				})
			}
		}
	}

	// Ensure computed attributes are known after apply (plan may have unknown values on update)
	if plan.AuditID.IsUnknown() || plan.AuditID.IsNull() {
		plan.AuditID = state.AuditID
	}
	if plan.Status.IsUnknown() || plan.Status.IsNull() {
		plan.Status = state.Status
	}

	tflog.Info(ctx, "Virtual machine update completed", map[string]any{
		"id": plan.ID.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// publicIPRetentionPolicyQuery maps retain_public_ip_on_termination to the DELETE API query value.
// yes → retain, no → release. Null, unknown, or unrecognized values default to retain.
func publicIPRetentionPolicyQuery(retain types.String) string {
	if retain.IsNull() || retain.IsUnknown() {
		return "Retain"
	}
	switch strings.ToLower(strings.TrimSpace(retain.ValueString())) {
	case "yes":
		return "Retain"
	case "no":
		return "Release"
	default:
		return "Retain"
	}
}

func (r *VirtualMachineResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data VirtualMachineResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	retain := types.StringNull()
	if data.PublicIP != nil {
		retain = data.PublicIP.RetainPublicIPOnTermination
	}
	policy := publicIPRetentionPolicyQuery(retain)

	tflog.Debug(ctx, "Deleting virtual machine", map[string]any{
		"id":                         data.ID.ValueString(),
		"audit_id":                   data.AuditID.ValueString(),
		"public_ip_retention_policy": policy,
	})

	_, err := DeleteVirtualMachineAndWait(r.client, ctx, data.ID.ValueString(), policy)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Virtual Machine",
			"Could not delete virtual machine: "+err.Error(),
		)
		return
	}

	tflog.Info(ctx, "Virtual machine deleted successfully", map[string]any{
		"id": data.ID.ValueString(),
	})
}

func (r *VirtualMachineResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
