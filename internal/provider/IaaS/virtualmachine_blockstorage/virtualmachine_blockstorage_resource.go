// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package virtualmachine_blockstorage

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/client"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/virtualmachine"
)

// int64MultipleOf validates that the configured int64 is divisible by divisor (e.g. 10 for GB steps).
type int64MultipleOfValidator struct {
	divisor int64
}

func int64MultipleOf(divisor int64) validator.Int64 {
	return int64MultipleOfValidator{divisor: divisor}
}

func (v int64MultipleOfValidator) Description(_ context.Context) string {
	return fmt.Sprintf("value must be a multiple of %d", v.divisor)
}

func (v int64MultipleOfValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v int64MultipleOfValidator) ValidateInt64(_ context.Context, req validator.Int64Request, resp *validator.Int64Response) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	if v.divisor == 0 {
		return
	}
	val := req.ConfigValue.ValueInt64()
	if val%v.divisor != 0 {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Attribute Value",
			fmt.Sprintf("Expected a multiple of %d, got %d.", v.divisor, val),
		)
	}
}

var (
	_ resource.Resource                = &VirtualMachineBlockStorageResource{}
	_ resource.ResourceWithImportState = &VirtualMachineBlockStorageResource{}
	_ resource.ResourceWithModifyPlan  = &VirtualMachineBlockStorageResource{}
)

// NewVirtualMachineBlockStorageResource creates the resource.
func NewVirtualMachineBlockStorageResource() resource.Resource {
	return &VirtualMachineBlockStorageResource{}
}

// VirtualMachineBlockStorageResource manages attach, resize, and delete of a VM-attached volume.
type VirtualMachineBlockStorageResource struct {
	client *client.Client
}

// VirtualMachineBlockStorageResourceModel maps Terraform state to the API.
type VirtualMachineBlockStorageResourceModel struct {
	ID types.Int64 `tfsdk:"id"`

	InstanceID types.Int64 `tfsdk:"instance_id"`
	Name       types.String `tfsdk:"name"`
	Size       types.Int64  `tfsdk:"size"`
	IOPS       types.Int64  `tfsdk:"iops"`

	DiskType    types.String `tfsdk:"disk_type"`
	CreatedDate types.String `tfsdk:"created_date"`

	AuditID types.String `tfsdk:"audit_id"`
	Status  types.String `tfsdk:"status"`
}

func (r *VirtualMachineBlockStorageResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_virtualmachine_blockstorage"
}

func (r *VirtualMachineBlockStorageResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Manages a block storage volume attached to a virtual machine (attach, resize, delete).",
		MarkdownDescription: "Manages a block storage volume attached to a virtual machine. Create runs attach-volume validation, then attach-volume with audit polling. Updates resize the volume using the same validate/resize flow as `vayucloud_virtualmachine`. Delete removes the volume from the instance.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "The volume (disk) ID after attach.",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"instance_id": schema.Int64Attribute{
				Description: "The virtual machine instance ID.",
				Required:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the volume to attach.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthBetween(5, 45),
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[a-zA-Z0-9 _]+$`),
						"Value may only contain alphanumeric characters and underscores (_)",
					),
				},
			},
			"size": schema.Int64Attribute{
				Description: "The volume size in GB. Value must be a multiple of 10 and between 10 and 10000.",
				Required:    true,
				Validators: []validator.Int64{
					int64validator.Between(10, 5000),
					int64MultipleOf(10),
				},
			},
			"iops": schema.Int64Attribute{
				Description: "The IOPS tier for the volume. Allowed values: 1, 3, or 5.",
				Required:    true,
				Validators: []validator.Int64{
					int64validator.OneOf(1, 3, 5),
				},
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"disk_type": schema.StringAttribute{
				Description: "The disk type from the API (e.g. HDD, SSD).",
				Validators: []validator.String{
					stringvalidator.OneOf("SSD"),
				},
				Optional: true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"created_date": schema.StringAttribute{
				Description: "The volume creation date from the API.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"audit_id": schema.StringAttribute{
				Description: "The audit ID from the last async operation.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"status": schema.StringAttribute{
				Description: "Status from the last async operation.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *VirtualMachineBlockStorageResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T.", req.ProviderData),
		)
		return
	}
	r.client = c
}

// attachVolumeRequestFromPlan mirrors the AttachVolumeRequest built in Create (used only by ModifyPlan).
func attachVolumeRequestFromPlan(plan *VirtualMachineBlockStorageResourceModel) *AttachVolumeRequest {
	return &AttachVolumeRequest{
		Name: plan.Name.ValueString(),
		Size: plan.Size.ValueInt64(),
		IOPS: plan.IOPS.ValueInt64(),
	}
}

// ModifyPlan runs attach-volume validation during plan for new attachments only (same as Create).
func (r *VirtualMachineBlockStorageResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if r.client == nil {
		return
	}
	if req.Plan.Raw.IsNull() {
		return
	}

	var plan VirtualMachineBlockStorageResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// ValidateAttachVolume applies only when attaching a new volume (id unknown until apply).
	if !plan.ID.IsUnknown() {
		return
	}

	if plan.InstanceID.IsUnknown() || plan.Name.IsUnknown() || plan.Size.IsUnknown() || plan.IOPS.IsUnknown() {
		return
	}

	attachReq := attachVolumeRequestFromPlan(&plan)
	if err := ValidateAttachVolume(r.client, ctx, plan.InstanceID.ValueInt64(), attachReq); err != nil {
		resp.Diagnostics.AddError("Attach Volume Validation Failed", err.Error())
		return
	}
}

func (r *VirtualMachineBlockStorageResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan VirtualMachineBlockStorageResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	instanceID := plan.InstanceID.ValueInt64()
	attachReq := &AttachVolumeRequest{
		Name: plan.Name.ValueString(),
		Size: plan.Size.ValueInt64(),
		IOPS: plan.IOPS.ValueInt64(),
	}

	if err := ValidateAttachVolume(r.client, ctx, instanceID, attachReq); err != nil {
		resp.Diagnostics.AddError("Attach Volume Validation Failed", err.Error())
		return
	}

	auditLog, err := AttachVolumeAndWait(r.client, ctx, instanceID, attachReq)
	if err != nil {
		resp.Diagnostics.AddError("Error Attaching Volume", err.Error())
		return
	}

	if auditLog.ResourceID.String() != "" {
		volumeID, err := strconv.ParseInt(strings.TrimSpace(auditLog.ResourceID.String()), 10, 64)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Parsing Volume ID",
				"Could not parse volume ID: "+err.Error(),
			)
			return
		}
		plan.ID = types.Int64Value(volumeID)
	} else {
		resp.Diagnostics.AddError(
			"Error Creating Virtual Machine",
			"Could not create virtual machine: ResourceID is empty",
		)
		return
	}

	plan.AuditID = types.StringValue(auditLog.AuditID)
	plan.Status = types.StringValue(auditLog.Status)
	if plan.Size.IsUnknown() || plan.Size.IsNull() {
		plan.Size = types.Int64Null()
	}
	if plan.IOPS.IsUnknown() || plan.IOPS.IsNull() {
		plan.IOPS = types.Int64Null()
	}
	if plan.DiskType.IsUnknown() || plan.DiskType.IsNull() {
		plan.DiskType = types.StringNull()
	}
	if plan.CreatedDate.IsUnknown() || plan.CreatedDate.IsNull() {
		plan.CreatedDate = types.StringNull()
	}

	tflog.Info(ctx, "Volume attached successfully", map[string]any{
		"instance_id": instanceID,
		"volume_id":   plan.ID.ValueInt64(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *VirtualMachineBlockStorageResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state VirtualMachineBlockStorageResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	instanceID := state.InstanceID.ValueInt64()
	volumeID := state.ID.ValueInt64()

	vol, err := GetVolumeByID(r.client, ctx, instanceID, volumeID)
	if err != nil {
		resp.Diagnostics.AddWarning(
			"Volume Not Found",
			fmt.Sprintf("Could not read volume %d on instance %s: %s", volumeID, instanceID, err.Error()),
		)
		resp.State.RemoveResource(ctx)
		return
	}

	state.Name = types.StringValue(vol.Name)
	state.Size = types.Int64Value(vol.Size)
	state.IOPS = types.Int64Value(vol.IOPS)
	state.DiskType = types.StringValue(vol.DiskType)
	state.CreatedDate = types.StringValue(vol.CreatedDate)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *VirtualMachineBlockStorageResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state VirtualMachineBlockStorageResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.Size.ValueInt64() != state.Size.ValueInt64() {
		instanceID := plan.InstanceID.ValueInt64()
		diskID := plan.ID.ValueInt64()

		if err := virtualmachine.ValidateVolume(r.client, ctx, fmt.Sprintf("%d", instanceID), diskID, plan.Size.ValueInt64()); err != nil {
			resp.Diagnostics.AddError("Error Validating Volume Resize", err.Error())
			return
		}
		if _, err := virtualmachine.UpdateVolumeSizeAndWait(r.client, ctx, fmt.Sprintf("%d", instanceID), diskID, plan.Size.ValueInt64()); err != nil {
			resp.Diagnostics.AddError("Error Resizing Volume", err.Error())
			return
		}
		tflog.Info(ctx, "Volume size updated", map[string]any{
			"instance_id": instanceID,
			"volume_id":   diskID,
			"size":        plan.Size.ValueInt64(),
		})
	}

	if plan.AuditID.IsUnknown() || plan.AuditID.IsNull() {
		plan.AuditID = state.AuditID
	}
	if plan.Status.IsUnknown() || plan.Status.IsNull() {
		plan.Status = state.Status
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *VirtualMachineBlockStorageResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state VirtualMachineBlockStorageResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	instanceID := state.InstanceID.ValueInt64()
	volumeID := state.ID.ValueInt64()

	if _, err := DeleteAttachedVolumeAndWait(r.client, ctx, instanceID, volumeID); err != nil {
		resp.Diagnostics.AddError("Error Deleting Volume", err.Error())
		return
	}

	tflog.Info(ctx, "Volume deleted", map[string]any{
		"instance_id": instanceID,
		"volume_id":   volumeID,
	})
}

// Import ID format: instance_id,volume_id
func (r *VirtualMachineBlockStorageResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, ",")
	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid Import Identifier",
			`Expected format "instance_id,volume_id" (e.g. "12345,67890").`,
		)
		return
	}
	instanceIDStr := strings.TrimSpace(parts[0])
	volumeIDStr := strings.TrimSpace(parts[1])
	instanceID, err := strconv.ParseInt(instanceIDStr, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Import Identifier", "instance_id must be an integer: "+err.Error())
		return
	}
	volumeID, err := strconv.ParseInt(volumeIDStr, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Import Identifier", "volume_id must be an integer: "+err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("instance_id"), types.Int64Value(instanceID))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), types.Int64Value(volumeID))...)
}
