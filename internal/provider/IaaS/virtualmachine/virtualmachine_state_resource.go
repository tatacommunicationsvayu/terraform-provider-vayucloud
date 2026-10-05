// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package virtualmachine

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/client"
)

var _ resource.Resource = &VirtualMachineStateResource{}
var _ resource.ResourceWithModifyPlan = &VirtualMachineStateResource{}

func NewVirtualMachineStateResource() resource.Resource {
	return &VirtualMachineStateResource{}
}

type VirtualMachineStateResource struct {
	client *client.Client
}

type VirtualMachineStateResourceModel struct {
	ID           types.String `tfsdk:"id"`
	InstanceID   types.Int64  `tfsdk:"instance_id"`
	Action       types.String `tfsdk:"action"`
	PowerStatus  types.String `tfsdk:"power_status"`
	AuditID      types.String `tfsdk:"audit_id"`
	Status       types.String `tfsdk:"status"`
}

// actionToAPIPath maps HCL action values to API endpoint path segments.
var actionToAPIPath = map[string]string{
	"power_off":   "power-off",
	"power_on":    "power-on",
	"suspend":     "suspend",
	"hard_reboot": "hard-reboot",
	"soft_reboot": "soft-reboot",
	"resume":      "resume",
}

// actionToDesiredPowerStatus maps each action to the power status the API returns when that state is achieved (Active, stopped, suspended).
var actionToDesiredPowerStatus = map[string]string{
	"power_off":   "stopped",
	"power_on":    "Active",
	"suspend":     "suspended",
	"resume":      "Active",
	"hard_reboot": "Active",
	"soft_reboot": "Active",
}

var powerStatusToAction = map[string]string{
	"stopped":   "power_off",
	"Active":    "power_on",
	"suspended": "suspend",
	"resume":      "power_on",
	"hard_reboot": "power_on",
	"soft_reboot": "power_on",
}

// actionToAllowedSourcePowerStatuses maps each action to power statuses from which the API permits that operation.
var actionToAllowedSourcePowerStatuses = map[string][]string{
	"power_off":   {"ACTIVE"},
	"power_on":    {"STOPPED", "SHUTOFF"},
	"suspend":     {"ACTIVE"},
	"resume":      {"Suspended", "SUSPENDED"},
	"soft_reboot": {"ACTIVE"},
	"hard_reboot": {"ACTIVE", "STOPPED", "SHUTOFF", "ERROR"},
}

func powerStatusAllowsAction(action string, powerStatus string) bool {
	allowed, ok := actionToAllowedSourcePowerStatuses[action]
	if !ok {
		return false
	}
	current := strings.TrimSpace(powerStatus)
	for _, permitted := range allowed {
		if strings.EqualFold(current, permitted) {
			return true
		}
	}
	return false
}

func validateActionForPowerStatus(action string, powerStatus string) error {
	if powerStatusAllowsAction(action, powerStatus) {
		return nil
	}
	allowed := actionToAllowedSourcePowerStatuses[action]
	return fmt.Errorf(
		"action %q is not allowed when power_status is %q; allowed power_status values: %s",
		action, strings.TrimSpace(powerStatus), strings.Join(allowed, ", "),
	)
}

func (r *VirtualMachineStateResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_virtualmachine_state"
}

func (r *VirtualMachineStateResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Manages the power state of a VayuCloud virtual machine.",
		MarkdownDescription: "Manages the power state of a VayuCloud virtual machine.\n\nThis resource performs power operations (`power_off`, `power_on`, `suspend`, `hard_reboot`, `soft_reboot`, `resume`) on an existing virtual machine. The operation is executed on create and whenever the `action` attribute changes.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier (same as instance_id).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"instance_id": schema.Int64Attribute{
				Description: "The ID of the virtual machine to perform the action on.",
				Required:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"action": schema.StringAttribute{
				Description: "The power action to perform: power_off, power_on, suspend, hard_reboot, soft_reboot, or resume.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("power_off", "power_on", "suspend", "hard_reboot", "soft_reboot", "resume"),
				},
			},
			"power_status": schema.StringAttribute{
				Description: "The current power status of the virtual machine after the action.",
				Computed:    true,
			},
			"audit_id": schema.StringAttribute{
				Description: "The audit ID from the last power action.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"status": schema.StringAttribute{
				Description: "The status of the last power action.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *VirtualMachineStateResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// ModifyPlan validates that the instance exists during terraform plan.
// Destroy plans use prior state when the plan value is null.
func (r *VirtualMachineStateResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if r.client == nil {
		return
	}

	var plan VirtualMachineStateResourceModel
	if req.Plan.Raw.IsNull() {
		resp.Diagnostics.Append(req.State.Get(ctx, &plan)...)
	} else {
		resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.InstanceID.IsUnknown() {
		return
	}

	if err := ValidateInstanceExists(r.client, ctx, plan.InstanceID.ValueInt64()); err != nil {
		resp.Diagnostics.AddError("Invalid Instance ID", err.Error())
		return
	}

	if plan.Action.IsUnknown() || req.Plan.Raw.IsNull() {
		return
	}

	instanceIDStr := fmt.Sprintf("%d", plan.InstanceID.ValueInt64())
	vmResp, err := GetVirtualMachineDetail(r.client, ctx, instanceIDStr)
	if err != nil {
		resp.Diagnostics.AddError(
			"Virtual Machine Read Failed",
			fmt.Sprintf("Could not read virtual machine %s to validate action: %s", instanceIDStr, err.Error()),
		)
		return
	}

	if err := validateActionForPowerStatus(plan.Action.ValueString(), vmResp.Data.PowerStatus); err != nil {
		resp.Diagnostics.AddError("Invalid Action for VM Power Status", err.Error())
	}
}

func (r *VirtualMachineStateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data VirtualMachineStateResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	instanceID := data.InstanceID.ValueInt64()
	actionName := data.Action.ValueString()

	resp.Diagnostics.AddWarning(
		fmt.Sprintf("Performing '%s' on instance %d", actionName, instanceID),
		"Terraform shows 'Creating...' but the provider is executing a power state change. Please wait for the operation to complete.",
	)

	if err := ValidateInstanceExists(r.client, ctx, instanceID); err != nil {
		resp.Diagnostics.AddError("Invalid Instance ID", err.Error())
		return
	}

	apiPath, ok := actionToAPIPath[actionName]
	if !ok {
		resp.Diagnostics.AddError(
			"Invalid Action",
			fmt.Sprintf("Unknown action %q.", actionName),
		)
		return
	}

	instanceIDStr := fmt.Sprintf("%d", instanceID)
	vmBefore, err := GetVirtualMachineDetail(r.client, ctx, instanceIDStr)
	if err != nil {
		resp.Diagnostics.AddError(
			"Virtual Machine Read Failed",
			fmt.Sprintf("Could not read virtual machine %s before power action: %s", instanceIDStr, err.Error()),
		)
		return
	}
	if err := validateActionForPowerStatus(actionName, vmBefore.Data.PowerStatus); err != nil {
		resp.Diagnostics.AddError("Invalid Action for VM Power Status", err.Error())
		return
	}

	tflog.Info(ctx, "Performing VM power action", map[string]any{
		"instance_id": instanceID,
		"action":      actionName,
	})

	auditLog, err := PerformVMPowerActionAndWait(r.client, ctx, instanceIDStr, apiPath)
	if err != nil {
		resp.Diagnostics.AddError(
			"VM Power Action Failed",
			fmt.Sprintf("Could not perform %s on instance %d: %s", actionName, instanceID, err.Error()),
		)
		return
	}

	data.ID = types.StringValue(instanceIDStr)
	data.AuditID = types.StringValue(auditLog.AuditID)
	data.Status = types.StringValue(auditLog.Status)

	vmResp, err := GetVirtualMachineDetail(r.client, ctx, instanceIDStr)
	if err != nil {
		data.PowerStatus = types.StringValue("unknown")
	} else {
		data.PowerStatus = types.StringValue(vmResp.Data.PowerStatus)
	}

	tflog.Info(ctx, "VM power action completed", map[string]any{
		"instance_id":  instanceID,
		"action":       actionName,
		"audit_id":     auditLog.AuditID,
		"power_status": data.PowerStatus.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *VirtualMachineStateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data VirtualMachineStateResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	instanceIDStr := fmt.Sprintf("%d", data.InstanceID.ValueInt64())

	vmResp, err := GetVirtualMachineDetail(r.client, ctx, instanceIDStr)
	if err != nil {
		resp.Diagnostics.AddWarning(
			"Virtual Machine Not Found",
			fmt.Sprintf("Could not read virtual machine %s: %s", instanceIDStr, err.Error()),
		)
		resp.State.RemoveResource(ctx)
		return
	}

	data.PowerStatus = types.StringValue(vmResp.Data.PowerStatus)
	data.Action = types.StringValue(powerStatusToAction[vmResp.Data.PowerStatus])

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *VirtualMachineStateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan VirtualMachineStateResourceModel
	var state VirtualMachineStateResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	instanceID := plan.InstanceID.ValueInt64()
	actionName := plan.Action.ValueString()
	tflog.Info(ctx, "Updating VM power state", map[string]any{
		"instance_id": instanceID,
		"action":      actionName,
	})
	tflog.Info(ctx, "Current Power Status", map[string]any{
		"current_power_status": state.PowerStatus.ValueString(),
		"desired_status":       actionToDesiredPowerStatus[actionName],
	})
	instanceIDStr := fmt.Sprintf("%d", instanceID)

	apiPath, ok := actionToAPIPath[actionName]
	if !ok {
		resp.Diagnostics.AddError(
			"Invalid Action",
			fmt.Sprintf("Unknown action %q.", actionName),
		)
		return
	}

	currentPowerStatus := strings.TrimSpace(state.PowerStatus.ValueString())
	tflog.Info(ctx, "Current Power Status", map[string]any{
		"current_power_status": currentPowerStatus,
	})
	desiredStatus := actionToDesiredPowerStatus[actionName]
	tflog.Info(ctx, "Desired Power Status", map[string]any{
		"desired_status":       desiredStatus,
	})
	alreadyInDesiredState := strings.EqualFold(currentPowerStatus, desiredStatus)
	tflog.Info(ctx, "Already In Desired State", map[string]any{
		"already_in_desired_state": alreadyInDesiredState,
	})

	if alreadyInDesiredState {
		tflog.Info(ctx, "VM already in desired power state, skipping action", map[string]any{
			"instance_id":    instanceID,
			"action":         actionName,
			"power_status":   currentPowerStatus,
		})
		plan.ID = state.ID
		plan.PowerStatus = state.PowerStatus
		plan.AuditID = state.AuditID
		plan.Status = state.Status
	} else {
		vmBefore, err := GetVirtualMachineDetail(r.client, ctx, instanceIDStr)
		if err != nil {
			resp.Diagnostics.AddError(
				"Virtual Machine Read Failed",
				fmt.Sprintf("Could not read virtual machine %s before power action: %s", instanceIDStr, err.Error()),
			)
			return
		}
		if err := validateActionForPowerStatus(actionName, vmBefore.Data.PowerStatus); err != nil {
			resp.Diagnostics.AddError("Invalid Action for VM Power Status", err.Error())
			return
		}

		resp.Diagnostics.AddWarning(
			fmt.Sprintf("Performing '%s' on instance %d", actionName, instanceID),
			fmt.Sprintf("Terraform shows 'Modifying...' but the provider is changing power state from '%s' to '%s'. Please wait for the operation to complete.",
				state.Action.ValueString(), actionName),
		)
		tflog.Info(ctx, "Updating VM power state", map[string]any{
			"instance_id":     instanceID,
			"old_action":      state.Action.ValueString(),
			"new_action":      actionName,
			"current_status":  currentPowerStatus,
			"desired_status":  desiredStatus,
		})

		auditLog, err := PerformVMPowerActionAndWait(r.client, ctx, instanceIDStr, apiPath)
		if err != nil {
			resp.Diagnostics.AddError(
				"VM Power Action Failed",
				fmt.Sprintf("Could not perform %s on instance %d: %s", actionName, instanceID, err.Error()),
			)
			return
		}

		plan.ID = state.ID
		// plan.AuditID = types.StringValue(auditLog.AuditID)
		// plan.Status = types.StringValue(auditLog.Status)

		// Read VM again to get power_status after the action
		vmResp, err := GetVirtualMachineDetail(r.client, ctx, instanceIDStr)
		if err != nil {
			plan.PowerStatus = state.PowerStatus
		} else {
			plan.PowerStatus = types.StringValue(vmResp.Data.PowerStatus)
		}

		tflog.Info(ctx, "VM power state updated", map[string]any{
			"instance_id":  instanceID,
			"action":       actionName,
			"audit_id":     auditLog.AuditID,
			"power_status": plan.PowerStatus.ValueString(),
			"status":       auditLog.Status,
		})
	}

	// Ensure computed attributes are known after apply (fallback if any were not set)
	if plan.AuditID.IsUnknown() || plan.AuditID.IsNull() {
		plan.AuditID = state.AuditID
	}
	if plan.Status.IsUnknown() || plan.Status.IsNull() {
		plan.Status = state.Status
	}
	if plan.PowerStatus.IsUnknown() || plan.PowerStatus.IsNull() {
		plan.PowerStatus = state.PowerStatus
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *VirtualMachineStateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Info(ctx, "Removing virtualmachine_state from Terraform state (no power action performed)")
}
