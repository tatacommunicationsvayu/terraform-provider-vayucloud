// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package virtualmachine

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
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
	PublicIPPricingModel        types.String `tfsdk:"public_ip_pricing_model"`
}

type CustomCredentialsModel struct {
	Username types.String `tfsdk:"username"`
	Password types.String `tfsdk:"password"`
}

var (
	customCredentialsUsernameRegex = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9]*$`)
	customCredentialsPasswordUpper = regexp.MustCompile(`[A-Z]`)
	customCredentialsPasswordLower = regexp.MustCompile(`[a-z]`)
	customCredentialsPasswordDigit = regexp.MustCompile(`[0-9]`)
	// Special characters for the "at least one" requirement: ! @ # $ % ^ & * ( ) _ + - = ?
	customCredentialsPasswordSpecial = regexp.MustCompile(`[!@#$%^&*()_+\-=?]`)
)

// customCredentialsPasswordNoMonotonicSequence rejects passwords with a monotonic
// ascending or descending character run longer than 3 (e.g. "abcd", "1234", "dcba").
type customCredentialsPasswordNoMonotonicSequenceValidator struct{}

func customCredentialsPasswordNoMonotonicSequence() validator.String {
	return customCredentialsPasswordNoMonotonicSequenceValidator{}
}

func (v customCredentialsPasswordNoMonotonicSequenceValidator) Description(_ context.Context) string {
	return "password must not contain a monotonic character sequence longer than 3 characters"
}

func (v customCredentialsPasswordNoMonotonicSequenceValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v customCredentialsPasswordNoMonotonicSequenceValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	if hasMonotonicSequenceLongerThan(req.ConfigValue.ValueString(), 3) {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Attribute Value",
			"The password contains a monotonic sequence longer than 3 characters so it is invalid",
		)
	}
}

// hasMonotonicSequenceLongerThan reports whether s contains an ascending or
// descending run of consecutive characters (code-point step ±1) longer than maxLen.
func hasMonotonicSequenceLongerThan(s string, maxLen int) bool {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return false
	}

	seqLen := 1
	dir := 0
	for i := 1; i < len(runes); i++ {
		diff := int(runes[i]) - int(runes[i-1])
		if diff == 1 || diff == -1 {
			if seqLen == 1 {
				dir = diff
				seqLen = 2
			} else if diff == dir {
				seqLen++
				if seqLen > maxLen {
					return true
				}
			} else {
				dir = diff
				seqLen = 2
			}
			continue
		}
		seqLen = 1
		dir = 0
	}
	return false
}

type VirtualMachineResourceModel struct {
	ID types.String `tfsdk:"id"`

	Name                 types.String         `tfsdk:"name"`
	VMPurpose            types.String         `tfsdk:"vm_purpose"`
	ImageID              types.Int64          `tfsdk:"image_id"`
	FlavorID             types.Int64          `tfsdk:"flavor_id"`
	ZoneID               types.Int64          `tfsdk:"zone_id"`
	IOPS                 types.Int64          `tfsdk:"iops"`
	IsKdumpOrPageEnabled types.String         `tfsdk:"is_kdump_or_page_enabled"`
	UsageType            types.String         `tfsdk:"usage_type"`
	PricingModel         types.String         `tfsdk:"pricing_model"`
	RootDiskSize         types.Int64          `tfsdk:"root_disk_size"`
	RootDiskId           types.Int64          `tfsdk:"root_disk_id"`
	DiskPartitions       []DiskPartitionModel `tfsdk:"root_disk_partitions"`
	// Pointer so Terraform null / omitted optional block decodes correctly (non-pointer struct cannot represent null).
	PublicIP           *PublicIPModel           `tfsdk:"public_ip"`
	CustomCredentials  *CustomCredentialsModel  `tfsdk:"custom_credentials"`
	AdditionalDisk     []AdditionalDiskModel    `tfsdk:"additional_disk"`
	PowerStatus        types.String             `tfsdk:"power_status"`
	IP                 types.String             `tfsdk:"ip"`

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
				Computed:    true,
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
			"custom_credentials": schema.SingleNestedAttribute{
				Description: "Optional custom OS login credentials sent at VM create as customCredentials.",
				Optional:    true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"username": schema.StringAttribute{
						Description: "Login username. At least 4 characters; must start with a letter; alphanumeric only; cannot be root or administrator.",
						Required:    true,
						Validators: []validator.String{
							stringvalidator.LengthAtLeast(4),
							stringvalidator.RegexMatches(customCredentialsUsernameRegex, "Username must start with a letter (a-z or A-Z) and contain only alphanumeric characters"),
							stringvalidator.NoneOfCaseInsensitive("root", "administrator"),
						},
					},
					"password": schema.StringAttribute{
						Description: "Login password (14–30 chars; must include uppercase, lowercase, digit, and a special character from !@#$%^&*()_+-=?; must not contain a monotonic sequence longer than 3 characters).",
						Required:    true,
						Sensitive:   true,
						Validators: []validator.String{
							stringvalidator.LengthBetween(14, 30),
							stringvalidator.RegexMatches(customCredentialsPasswordUpper, "Password must contain at least one uppercase letter (A-Z)"),
							stringvalidator.RegexMatches(customCredentialsPasswordLower, "Password must contain at least one lowercase letter (a-z)"),
							stringvalidator.RegexMatches(customCredentialsPasswordDigit, "Password must contain at least one digit (0-9)"),
							stringvalidator.RegexMatches(customCredentialsPasswordSpecial, "Password must contain at least one special character: ! @ # $ % ^ & * ( ) _ + - = ?"),
							customCredentialsPasswordNoMonotonicSequence(),
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
			"ip": schema.StringAttribute{
				Description: "The IP address of the virtual machine.",
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

	if data.CustomCredentials != nil {
		createReq.CustomCredentials = &CustomCredentials{
			Username: data.CustomCredentials.Username.ValueString(),
			Password: data.CustomCredentials.Password.ValueString(),
		}
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

	// Pre-create validate API applies only to brand-new instances (no id in plan or state).
	if !plan.ID.IsUnknown() {
		return
	}
	if !req.State.Raw.IsNull() {
		var state VirtualMachineResourceModel
		resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
		if resp.Diagnostics.HasError() {
			return
		}
		// Replacement: old instance still exists until apply destroys it; skip hostname checks.
		if !state.ID.IsNull() && !state.ID.IsUnknown() {
			return
		}
	}

	if plan.Name.IsUnknown() || plan.VMPurpose.IsUnknown() || plan.ImageID.IsUnknown() || plan.FlavorID.IsUnknown() ||
		plan.ZoneID.IsUnknown() || plan.IOPS.IsUnknown() || plan.IsKdumpOrPageEnabled.IsUnknown() {
		return
	}

	// createReq := vmCreateRequestFromPlanModel(&plan)
	// if err := PreCreateValidateInstance(r.client, ctx, createReq); err != nil {
	// 	resp.Diagnostics.AddError(
	// 		"Virtual Machine Pre-Create Validation Failed",
	// 		err.Error(),
	// 	)
	// 	return
	// }
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
	createReq := vmCreateRequestFromPlanModel(&data)

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
	auditID := auditLog.AuditID
	data.AuditID = types.StringValue(auditID)
	data.Status = types.StringValue(auditLog.Status)

	vmResp, err := GetVirtualMachineDetail(r.client, ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Virtual Machine After Create",
			fmt.Sprintf("Virtual machine was created but could not read details for %s: %s", data.ID.ValueString(), err.Error()),
		)
		return
	}
	applyVirtualMachineDetail(&data, vmResp.Data, true)

	if strings.TrimSpace(vmResp.Data.IP) != "" {
		data.IP = types.StringValue(vmResp.Data.IP)
	} else {
		data.IP = types.StringUnknown()
	}

	// Detail API does not return audit metadata; keep values from the create audit.
	data.AuditID = types.StringValue(auditID)
	data.Status = types.StringValue(auditLog.Status)

	tflog.Info(ctx, "Virtual machine created successfully", map[string]any{
		"id":       data.ID.ValueString(),
		"audit_id": data.AuditID.ValueString(),
		"status":   data.Status.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// applyVirtualMachineDetail maps GET instance detail into computed Terraform state fields.
// Config-only attributes (vm_purpose, usage_type, iops, disk partitions, etc.) are left unchanged.
// When preservePlannedAdditionalDisks is true (create), planned additional_disk entries are kept
// if the detail API has not listed them yet — avoids inconsistent result after apply.
func applyVirtualMachineDetail(data *VirtualMachineResourceModel, vm VirtualMachineDetail, preservePlannedAdditionalDisks bool) {
	plannedAdditional := append([]AdditionalDiskModel(nil), data.AdditionalDisk...)

	data.Name = types.StringValue(vm.Name)
	data.ImageID = types.Int64Value(vm.ImageID)
	data.FlavorID = types.Int64Value(vm.FlavorID)
	data.ZoneID = types.Int64Value(vm.ZoneID)
	data.IOPS = types.Int64Value(vm.Volumes[0].IOPS)
	data.VMPurpose = types.StringValue(vm.VMPurpose)
	data.UsageType = types.StringValue(vm.UsageType)
	data.IsKdumpOrPageEnabled = types.StringValue(vm.IsKdumpOrPageEnabled)
	if vm.PricingModel != "" {
		data.PricingModel = types.StringValue(vm.PricingModel)
	}
	if vm.PowerStatus != "" {
		data.PowerStatus = types.StringValue(vm.PowerStatus)
	}

	rootSize := vm.RootDiskSize
	data.AdditionalDisk = nil
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
	}

	if preservePlannedAdditionalDisks && len(plannedAdditional) > 0 {
		switch {
		case len(data.AdditionalDisk) == 0:
			data.AdditionalDisk = plannedAdditional
		case len(data.AdditionalDisk) < len(plannedAdditional):
			merged := make([]AdditionalDiskModel, len(plannedAdditional))
			copy(merged, data.AdditionalDisk)
			for i := len(data.AdditionalDisk); i < len(plannedAdditional); i++ {
				merged[i] = plannedAdditional[i]
			}
			data.AdditionalDisk = merged
		}
	}
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

	applyVirtualMachineDetail(&data, vmResp.Data, false)
	if strings.TrimSpace(vmResp.Data.IP) != "" {
		data.IP = types.StringValue(vmResp.Data.IP)
	} else {
		data.IP = types.StringUnknown()
	}
	if data.PublicIP != nil {
		if data.PublicIP.PublicIPPricingModel.IsNull() || data.PublicIP.PublicIPPricingModel.IsUnknown() {
			if !data.PricingModel.IsNull() && !data.PricingModel.IsUnknown() {
				data.PublicIP.PublicIPPricingModel = types.StringValue(data.PricingModel.ValueString())
			}
		}
		// else: leave data.PublicIP.PublicIPPricingModel as already loaded from state
	}

	tflog.Info(ctx, "Virtual machine read successfully", map[string]any{
		"id":             instanceID,
		"name":           vmResp.Data.Name,
		"power_status":   vmResp.Data.PowerStatus,
		"zone_id":        vmResp.Data.ZoneID,
		"root_disk_size": vmResp.Data.RootDiskSize,
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
