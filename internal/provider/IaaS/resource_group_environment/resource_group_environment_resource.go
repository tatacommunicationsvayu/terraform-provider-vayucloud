// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package resource_group_environment

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
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
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/common"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/network_firewall"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/resource_group_business_unit"
)

// Ensure provider-defined types fully satisfy framework interfaces.
var _ resource.Resource = &ResourceGroupEnvironmentResource{}
var _ resource.ResourceWithImportState = &ResourceGroupEnvironmentResource{}
var _ resource.ResourceWithModifyPlan = &ResourceGroupEnvironmentResource{}

// NewResourceGroupEnvironmentResource creates a new resource group environment resource.
func NewResourceGroupEnvironmentResource() resource.Resource {
	return &ResourceGroupEnvironmentResource{}
}

// ResourceGroupEnvironmentResource defines the resource implementation.
type ResourceGroupEnvironmentResource struct {
	client *client.Client
}

// ResourceGroupEnvironmentResourceModel describes the resource data model.
type ResourceGroupEnvironmentResourceModel struct {
	// ID is the unique identifier (resource_id from audit after creation)
	ID types.String `tfsdk:"id"`

	// FirewallID is the firewall ID to associate the environment with
	FirewallID types.Int64 `tfsdk:"firewall_id"`

	// Environment is the name of the environment to create
	Environment types.String `tfsdk:"environment"`

	// BusinessUnitID is the business unit ID to associate with the environment
	BusinessUnitID types.Int64 `tfsdk:"business_unit_id"`

	// Computed attributes
	// AuditID is the audit ID from the creation response
	AuditID types.String `tfsdk:"audit_id"`

	// Status is the final status from the audit log
	Status types.String `tfsdk:"status"`
}

// Metadata returns the resource type name.
func (r *ResourceGroupEnvironmentResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_resource_group_environment"
}

// Schema defines the schema for the environment resource.
func (r *ResourceGroupEnvironmentResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Manages a resource group environment component resource in VayuCloud.",
		MarkdownDescription: "Manages a resource group environment component resource in VayuCloud.\n\nThis resource creates a resource group environment component through an asynchronous provisioning process. The resource will poll the audit log until the resource group environment creation is complete.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:         "The unique identifier of the resource group environment.",
				MarkdownDescription: "The unique identifier of the resource group environment.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"firewall_id": schema.Int64Attribute{
				Description:         "The firewall ID to associate the resource group environment with.",
				MarkdownDescription: "The firewall ID to associate the resource group environment with.",
				Required:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"environment": schema.StringAttribute{
				Description:         "The name of the resource group environment (e.g., \"Production\", \"Development\"). Can be updated in-place.",
				MarkdownDescription: "The name of the resource group environment (e.g., `Production`, `Development`). Can be updated in-place.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(5, 45),
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[a-zA-Z0-9 _-]+$`),
						"Value may only contain alphanumeric characters, underscores (_), and hyphens (-)",
					),
				},
			},
			"business_unit_id": schema.Int64Attribute{
				Description:         "The business unit ID to associate with the resource group environment. Can be updated in-place.",
				MarkdownDescription: "The business unit ID to associate with the resource group environment. Can be updated in-place.",
				Required:            true,
			},
			// Computed attributes
			"audit_id": schema.StringAttribute{
				Description:         "The audit ID from the creation response.",
				MarkdownDescription: "The audit ID from the creation response.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"status": schema.StringAttribute{
				Description:         "The final status from the audit log.",
				MarkdownDescription: "The final status from the audit log.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *ResourceGroupEnvironmentResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

// ModifyPlan validates firewall and business unit IDs during terraform plan.
// Destroy plans use prior state when the plan value is null.
func (r *ResourceGroupEnvironmentResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if r.client == nil {
		return
	}

	var plan ResourceGroupEnvironmentResourceModel
	if req.Plan.Raw.IsNull() {
		resp.Diagnostics.Append(req.State.Get(ctx, &plan)...)
	} else {
		resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.FirewallID.IsUnknown() || plan.BusinessUnitID.IsUnknown() {
		return
	}

	if err := network_firewall.ValidateFirewallExists(r.client, ctx, plan.FirewallID.ValueInt64()); err != nil {
		resp.Diagnostics.AddError(
			"Invalid Firewall ID",
			err.Error(),
		)
		return
	}

	if err := resource_group_business_unit.ValidateBusinessUnitExists(r.client, ctx, plan.BusinessUnitID.ValueInt64()); err != nil {
		resp.Diagnostics.AddError(
			"Invalid Business Unit ID",
			err.Error(),
		)
		return
	}

	if err := resource_group_business_unit.ValidateBusinessUnitExistsForFirewall(r.client, ctx, plan.BusinessUnitID.ValueInt64(), plan.FirewallID.ValueInt64()); err != nil {
		resp.Diagnostics.AddError(
			"Invalid Business Unit ID",
			err.Error(),
		)
		return
	}
}

// Create creates a new environment resource.
func (r *ResourceGroupEnvironmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ResourceGroupEnvironmentResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating environment", map[string]any{
		"firewall_id":      data.FirewallID.ValueInt64(),
		"environment":      data.Environment.ValueString(),
		"business_unit_id": data.BusinessUnitID.ValueInt64(),
	})

	// Validate that the firewall ID exists
	if err := network_firewall.ValidateFirewallExists(r.client, ctx, data.FirewallID.ValueInt64()); err != nil {
		resp.Diagnostics.AddError(
			"Invalid Firewall ID",
			err.Error(),
		)
		return
	}

	// Validate that the business unit ID exists
	if err := resource_group_business_unit.ValidateBusinessUnitExists(r.client, ctx, data.BusinessUnitID.ValueInt64()); err != nil {
		resp.Diagnostics.AddError(
			"Invalid Business Unit ID",
			err.Error(),
		)
		return
	}
	if err := resource_group_business_unit.ValidateBusinessUnitExistsForFirewall(r.client, ctx, data.BusinessUnitID.ValueInt64(), data.FirewallID.ValueInt64()); err != nil {
		resp.Diagnostics.AddError(
			"Invalid Business Unit ID",
			err.Error(),
		)
		return
	}

	// Build the create request
	createReq := &ResourceGroupEnvironmentCreateRequest{
		FirewallID:     data.FirewallID.ValueInt64(),
		Environment:    data.Environment.ValueString(),
		BusinessUnitID: data.BusinessUnitID.ValueInt64(),
	}

	// Create environment and wait for completion
	auditLog, err := CreateResourceGroupEnvironmentAndWait(r.client, ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Resource Group Environment",
			"Could not create resource group environment: "+err.Error(),
		)
		return
	}

	// Set the ID and audit ID
	if auditLog.ResourceID.String() != "" {
		data.ID = types.StringValue(auditLog.ResourceID.String())
	} else {
		resp.Diagnostics.AddError(
			"Error Creating Environment",
			"Could not create environment: ResourceID is empty",
		)
		return
	}
	data.AuditID = types.StringValue(auditLog.AuditID)
	data.Status = types.StringValue(auditLog.Status)

	tflog.Info(ctx, "Environment created successfully", map[string]any{
		"id":       data.ID.ValueString(),
		"audit_id": data.AuditID.ValueString(),
		"status":   data.Status.ValueString(),
	})

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read refreshes the Terraform state with the latest data.
// This is a dummy implementation that preserves existing state.
func (r *ResourceGroupEnvironmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ResourceGroupEnvironmentResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Reading resource group environment", map[string]any{
		"id":       data.ID.ValueString(),
		"audit_id": data.AuditID.ValueString(),
	})

	actionStateBody := map[string]any{
		"resourceId":   data.ID.ValueString(),
		"resourceType": "ENV",
	}

	// Call action-state API to get the latest firewall data
	actionStateResponse, err := common.UpdateActionState(ctx, r.client, "engagementComponents", "read", actionStateBody)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Resource Group Environment",
			"Could not read engagement components: "+err.Error(),
		)
		return
	}

	// Check if response data is empty
	if len(actionStateResponse.Data) == 0 {
		resp.Diagnostics.AddWarning(
			"Empty Response Data",
			"Could not read resource group environment: response data is empty, keeping existing state",
		)
		// Keep existing state and return
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
		return
	}

	// Unmarshal the JSON response data into a map
	// The API returns standard JSON types, not Terraform framework types
	var responseMap map[string]interface{}
	if err := json.Unmarshal(actionStateResponse.Data, &responseMap); err != nil {
		resp.Diagnostics.AddError(
			"Error Unmarshalling Response Data",
			fmt.Sprintf("Could not unmarshal response data: %s. Response: %s", err.Error(), string(actionStateResponse.Data)),
		)
		return
	}
	data.Environment = types.StringValue(responseMap["environment"].(string))
	// JSON numbers are decoded as float64 in map[string]interface{}
	data.BusinessUnitID = types.Int64Value(int64(responseMap["business_unit_id"].(float64)))
	statusValue, ok := responseMap["status"]
	if !ok || statusValue == nil {
		data.Status = types.StringValue("ACTIVE")
	} else {
		data.Status = types.StringValue(statusValue.(string))
	}

	tflog.Info(ctx, "Resource group environment read successfully", map[string]any{
		"id":       data.ID.ValueString(),
		"audit_id": data.AuditID.ValueString(),
		"status":   data.Status.ValueString(),
	})

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update updates the environment resource.
func (r *ResourceGroupEnvironmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ResourceGroupEnvironmentResourceModel
	var state ResourceGroupEnvironmentResourceModel

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

	tflog.Debug(ctx, "Updating resource group environment", map[string]any{
		"id":               plan.ID.ValueString(),
		"environment":      plan.Environment.ValueString(),
		"business_unit_id": plan.BusinessUnitID.ValueInt64(),
	})

	// Validate that the firewall ID exists
	if err := network_firewall.ValidateFirewallExists(r.client, ctx, plan.FirewallID.ValueInt64()); err != nil {
		resp.Diagnostics.AddError(
			"Invalid Firewall ID",
			err.Error(),
		)
		return
	}

	// Validate that the business unit ID exists
	if err := resource_group_business_unit.ValidateBusinessUnitExists(r.client, ctx, plan.BusinessUnitID.ValueInt64()); err != nil {
		resp.Diagnostics.AddError(
			"Invalid Business Unit ID",
			err.Error(),
		)
		return
	}

	if err := resource_group_business_unit.ValidateBusinessUnitExistsForFirewall(r.client, ctx, plan.BusinessUnitID.ValueInt64(), plan.FirewallID.ValueInt64()); err != nil {
		resp.Diagnostics.AddError(
			"Invalid Business Unit ID",
			err.Error(),
		)
		return
	}

	// Check if environment or business_unit_id changed
	environmentChanged := plan.Environment.ValueString() != state.Environment.ValueString()

	if environmentChanged {
		tflog.Debug(ctx, "Environment attributes changed", map[string]any{
			"environment_changed": environmentChanged,
		})

		updateReq := &ResourceGroupEnvironmentCreateRequest{
			Environment:    plan.Environment.ValueString(),
			BusinessUnitID: plan.BusinessUnitID.ValueInt64(),
		}

		// Update environment and wait for completion
		_, err := UpdateResourceGroupEnvironmentAndWait(r.client, ctx, plan.ID.ValueString(), updateReq)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Updating Resource Group Environment",
				"Could not update resource group environment: "+err.Error(),
			)
			return
		}

		// Update status from audit log
		// plan.Status = types.StringValue(auditLog.Status)
		// plan.AuditID = types.StringValue(auditLog.AuditID)

		tflog.Info(ctx, "Resource group environment updated successfully", map[string]any{
			"id":          plan.ID.ValueString(),
			"environment": plan.Environment.ValueString(),
			"audit_id":    plan.AuditID.ValueString(),
			"status":      plan.Status.ValueString(),
		})
	} else {
		tflog.Debug(ctx, "No changes detected, skipping update", map[string]any{
			"id": plan.ID.ValueString(),
		})
	}

	// Ensure computed attributes are known after apply (plan may have unknown values on update)
	if plan.AuditID.IsUnknown() || plan.AuditID.IsNull() {
		plan.AuditID = state.AuditID
	}
	if plan.Status.IsUnknown() || plan.Status.IsNull() {
		plan.Status = state.Status
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete deletes the environment resource.
func (r *ResourceGroupEnvironmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ResourceGroupEnvironmentResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting resource group environment", map[string]any{
		"id":       data.ID.ValueString(),
		"audit_id": data.AuditID.ValueString(),
	})

	// Validate that the firewall ID exists
	if err := network_firewall.ValidateFirewallExists(r.client, ctx, data.FirewallID.ValueInt64()); err != nil {
		resp.Diagnostics.AddError(
			"Invalid Firewall ID",
			err.Error(),
		)
		return
	}

	// Validate that the business unit ID exists
	if err := resource_group_business_unit.ValidateBusinessUnitExists(r.client, ctx, data.BusinessUnitID.ValueInt64()); err != nil {
		resp.Diagnostics.AddError(
			"Invalid Business Unit ID",
			err.Error(),
		)
		return
	}
	if err := resource_group_business_unit.ValidateBusinessUnitExistsForFirewall(r.client, ctx, data.BusinessUnitID.ValueInt64(), data.FirewallID.ValueInt64()); err != nil {
		resp.Diagnostics.AddError(
			"Invalid Business Unit ID",
			err.Error(),
		)
		return
	}

	// Delete environment and wait for completion
	_, err := DeleteResourceGroupEnvironmentAndWait(r.client, ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Resource Group Environment",
			"Could not delete resource group environment: "+err.Error(),
		)
		return
	}

	tflog.Info(ctx, "Resource group environment deleted successfully", map[string]any{
		"id": data.ID.ValueString(),
	})
}

// ImportState imports an existing environment into Terraform state.
func (r *ResourceGroupEnvironmentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
