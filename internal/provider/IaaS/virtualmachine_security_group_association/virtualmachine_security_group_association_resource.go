// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package virtualmachine_security_group_association

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/client"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/virtualmachine"
)

var _ resource.Resource = &VirtualMachineSecurityGroupAssociationResource{}
var _ resource.ResourceWithModifyPlan = &VirtualMachineSecurityGroupAssociationResource{}

// NewVirtualMachineSecurityGroupAssociationResource creates the VM security group association resource.
func NewVirtualMachineSecurityGroupAssociationResource() resource.Resource {
	return &VirtualMachineSecurityGroupAssociationResource{}
}

// VirtualMachineSecurityGroupAssociationResource associates security groups with a virtual machine.
type VirtualMachineSecurityGroupAssociationResource struct {
	client *client.Client
}

// VirtualMachineSecurityGroupAssociationResourceModel is the Terraform model.
type VirtualMachineSecurityGroupAssociationResourceModel struct {
	ID               types.String `tfsdk:"id"`
	InstanceID       types.Int64  `tfsdk:"instance_id"`
	SecurityGroupIDs types.List   `tfsdk:"security_group_ids"`
	AuditID          types.String `tfsdk:"audit_id"`
	Status           types.String `tfsdk:"status"`
}

func (r *VirtualMachineSecurityGroupAssociationResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_virtualmachine_security_group_association"
}

func (r *VirtualMachineSecurityGroupAssociationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Associates one or more security groups with a virtual machine.",
		MarkdownDescription: "Attaches security groups to a VM via " +
			"`POST .../security-group/vm/{instance_id}` and waits for audit completion (no action-state). " +
			"Destroy detaches the same set via `DELETE .../security-group/vm/{instance_id}`. " +
			"Changing `security_group_ids` detaches removed IDs and attaches newly added IDs.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Stable identifier (same as `instance_id`).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"instance_id": schema.Int64Attribute{
				Description: "Virtual machine instance ID to attach security groups to.",
				Required:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"security_group_ids": schema.ListAttribute{
				Description: "List of security group UUIDs to associate with the instance.",
				Required:    true,
				ElementType: types.StringType,
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
					listvalidator.UniqueValues(),
					listvalidator.ValueStringsAre(stringvalidator.LengthAtLeast(1)),
				},
			},
			"audit_id": schema.StringAttribute{
				Description: "Audit ID from the last completed async operation.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"status": schema.StringAttribute{
				Description: "Audit status from the last completed operation.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *VirtualMachineSecurityGroupAssociationResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// ModifyPlan validates that the instance exists during terraform plan.
// When security_group_ids change, audit_id/status are marked unknown so Update
// can store the new audit without "inconsistent result after apply".
func (r *VirtualMachineSecurityGroupAssociationResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if !req.Plan.Raw.IsNull() && !req.State.Raw.IsNull() {
		var plan, state VirtualMachineSecurityGroupAssociationResourceModel
		resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
		resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
		if resp.Diagnostics.HasError() {
			return
		}
		if securityGroupIDsDiffer(ctx, plan.SecurityGroupIDs, state.SecurityGroupIDs) {
			plan.AuditID = types.StringUnknown()
			plan.Status = types.StringUnknown()
			resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
			if resp.Diagnostics.HasError() {
				return
			}
		}
	}

	if r.client == nil {
		return
	}

	var plan VirtualMachineSecurityGroupAssociationResourceModel
	if req.Plan.Raw.IsNull() {
		resp.Diagnostics.Append(req.State.Get(ctx, &plan)...)
	} else {
		resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
		var instanceID types.Int64
		resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("instance_id"), &instanceID)...)
		if resp.Diagnostics.HasError() {
			return
		}
		if instanceID.IsUnknown() {
			return
		}
	}
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.InstanceID.IsUnknown() || plan.InstanceID.IsNull() {
		return
	}

	tflog.Debug(ctx, "Planning VM security group association", map[string]any{
		"instance_id": plan.InstanceID.ValueInt64(),
	})

	if err := virtualmachine.ValidateInstanceExists(r.client, ctx, plan.InstanceID.ValueInt64()); err != nil {
		resp.Diagnostics.AddError("Invalid Instance ID", err.Error())
		return
	}
}

func (r *VirtualMachineSecurityGroupAssociationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data VirtualMachineSecurityGroupAssociationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	instanceID := data.InstanceID.ValueInt64()
	if err := virtualmachine.ValidateInstanceExists(r.client, ctx, instanceID); err != nil {
		resp.Diagnostics.AddError("Invalid Instance ID", err.Error())
		return
	}

	sgIDs, diags := listToStrings(ctx, data.SecurityGroupIDs)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	auditLog, err := AttachVMSecurityGroupsAndWait(r.client, ctx, instanceID, sgIDs)
	if err != nil {
		resp.Diagnostics.AddError("Attach Security Groups Failed", err.Error())
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("%d", instanceID))
	data.AuditID = types.StringValue(auditLog.AuditID)
	data.Status = types.StringValue(auditLog.Status)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *VirtualMachineSecurityGroupAssociationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data VirtualMachineSecurityGroupAssociationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	instanceID := data.InstanceID.ValueInt64()
	if err := virtualmachine.ValidateInstanceExists(r.client, ctx, instanceID); err != nil {
		tflog.Warn(ctx, "Instance no longer exists; removing association from state", map[string]any{
			"instance_id": instanceID,
			"error":       err.Error(),
		})
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *VirtualMachineSecurityGroupAssociationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state VirtualMachineSecurityGroupAssociationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	instanceID := state.InstanceID.ValueInt64()
	planIDs, diags := listToStrings(ctx, plan.SecurityGroupIDs)
	resp.Diagnostics.Append(diags...)
	stateIDs, diags := listToStrings(ctx, state.SecurityGroupIDs)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	toDetach, toAttach := diffStringSets(stateIDs, planIDs)
	var lastAudit, lastStatus string

	if len(toDetach) > 0 {
		auditLog, err := DetachVMSecurityGroupsAndWait(r.client, ctx, instanceID, toDetach)
		if err != nil {
			resp.Diagnostics.AddError("Detach Security Groups Failed", err.Error())
			return
		}
		lastAudit = auditLog.AuditID
		lastStatus = auditLog.Status
	}

	if len(toAttach) > 0 {
		auditLog, err := AttachVMSecurityGroupsAndWait(r.client, ctx, instanceID, toAttach)
		if err != nil {
			resp.Diagnostics.AddError("Attach Security Groups Failed", err.Error())
			return
		}
		lastAudit = auditLog.AuditID
		lastStatus = auditLog.Status
	}

	plan.ID = state.ID
	if lastAudit != "" {
		plan.AuditID = types.StringValue(lastAudit)
		plan.Status = types.StringValue(lastStatus)
	} else {
		plan.AuditID = state.AuditID
		plan.Status = state.Status
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *VirtualMachineSecurityGroupAssociationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data VirtualMachineSecurityGroupAssociationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	instanceID := data.InstanceID.ValueInt64()
	sgIDs, diags := listToStrings(ctx, data.SecurityGroupIDs)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if len(sgIDs) == 0 {
		return
	}

	if _, err := DetachVMSecurityGroupsAndWait(r.client, ctx, instanceID, sgIDs); err != nil {
		resp.Diagnostics.AddError("Detach Security Groups Failed", err.Error())
		return
	}
}

func listToStrings(ctx context.Context, list types.List) ([]string, diag.Diagnostics) {
	var out []string
	if list.IsNull() || list.IsUnknown() {
		return out, nil
	}
	var diags diag.Diagnostics
	diags.Append(list.ElementsAs(ctx, &out, false)...)
	return out, diags
}

// securityGroupIDsDiffer reports whether the planned and state SG ID sets differ.
// Unknown/null lists are treated as "may change" so audit fields stay unknown.
func securityGroupIDsDiffer(ctx context.Context, planList, stateList types.List) bool {
	if planList.IsUnknown() || stateList.IsUnknown() {
		return true
	}
	planIDs, _ := listToStrings(ctx, planList)
	stateIDs, _ := listToStrings(ctx, stateList)
	toDetach, toAttach := diffStringSets(stateIDs, planIDs)
	return len(toDetach) > 0 || len(toAttach) > 0
}

func diffStringSets(stateIDs, planIDs []string) (toDetach, toAttach []string) {
	planSet := make(map[string]struct{}, len(planIDs))
	for _, id := range planIDs {
		planSet[id] = struct{}{}
	}
	stateSet := make(map[string]struct{}, len(stateIDs))
	for _, id := range stateIDs {
		stateSet[id] = struct{}{}
		if _, ok := planSet[id]; !ok {
			toDetach = append(toDetach, id)
		}
	}
	for _, id := range planIDs {
		if _, ok := stateSet[id]; !ok {
			toAttach = append(toAttach, id)
		}
	}
	return toDetach, toAttach
}
