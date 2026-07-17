// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package resource_group_business_unit

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
)

// Ensure provider-defined types fully satisfy framework interfaces.
var _ resource.Resource = &ResourceGroupBusinessUnitResource{}
var _ resource.ResourceWithImportState = &ResourceGroupBusinessUnitResource{}
var _ resource.ResourceWithModifyPlan = &ResourceGroupBusinessUnitResource{}

// NewResourceGroupBusinessUnitResource creates a new resource group business unit resource.
func NewResourceGroupBusinessUnitResource() resource.Resource {
	return &ResourceGroupBusinessUnitResource{}
}

// ResourceGroupBusinessUnitResource defines the resource implementation.
type ResourceGroupBusinessUnitResource struct {
	client *client.Client
}

// ResourceGroupBusinessUnitResourceModel describes the resource data model.
type ResourceGroupBusinessUnitResourceModel struct {
	// ID is the unique identifier (resource_id from audit after creation)
	ID types.String `tfsdk:"id"`

	// FirewallID is the firewall ID to associate the business unit with
	FirewallID types.Int64 `tfsdk:"firewall_id"`

	// BusinessUnit is the name of the business unit
	BusinessUnit types.String `tfsdk:"business_unit"`

	// Users is the list of user emails to associate with the business unit
	Users types.List `tfsdk:"users"`

	// Computed attributes
	// AuditID is the audit ID from the creation response
	AuditID types.String `tfsdk:"audit_id"`

	// Status is the final status from the audit log
	Status types.String `tfsdk:"status"`
}

// Metadata returns the resource type name.
func (r *ResourceGroupBusinessUnitResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_resource_group_business_unit"
}

// Schema defines the schema for the business unit resource.
func (r *ResourceGroupBusinessUnitResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Manages a business unit component resource in VayuCloud.",
		MarkdownDescription: "Manages a business unit component resource in VayuCloud.\n\nThis resource creates a business unit component through an asynchronous provisioning process. The resource will poll the audit log until the business unit creation is complete.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:         "The unique identifier of the business unit.",
				MarkdownDescription: "The unique identifier of the business unit.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"firewall_id": schema.Int64Attribute{
				Description:         "The firewall ID to associate the business unit with.",
				MarkdownDescription: "The firewall ID to associate the business unit with.",
				Required:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"business_unit": schema.StringAttribute{
				Description:         "The name of the business unit. Can be updated in-place.",
				MarkdownDescription: "The name of the business unit. Can be updated in-place.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(5, 45),
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[a-zA-Z0-9 _-]+$`),
						"Value may only contain alphanumeric characters, underscores (_), and hyphens (-)",
					),
				},
			},
			"users": schema.ListAttribute{
				Description:         "The list of user emails to associate with the business unit.",
				MarkdownDescription: "The list of user emails to associate with the business unit.",
				Optional:            true,
				Computed:            true,
				ElementType:         types.StringType,
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
func (r *ResourceGroupBusinessUnitResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// ModifyPlan validates the firewall ID during terraform plan.
// Destroy plans use prior state when the plan value is null.
func (r *ResourceGroupBusinessUnitResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if r.client == nil {
		return
	}

	var plan ResourceGroupBusinessUnitResourceModel
	if req.Plan.Raw.IsNull() {
		resp.Diagnostics.Append(req.State.Get(ctx, &plan)...)
	} else {
		resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.FirewallID.IsUnknown() {
		return
	}

	if err := network_firewall.ValidateFirewallExists(r.client, ctx, plan.FirewallID.ValueInt64()); err != nil {
		resp.Diagnostics.AddError(
			"Invalid Firewall ID",
			err.Error(),
		)
		return
	}
}

// Create creates a new business unit resource.
func (r *ResourceGroupBusinessUnitResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ResourceGroupBusinessUnitResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating resource group business unit", map[string]any{
		"business_unit": data.BusinessUnit.ValueString(),
		"firewall_id":   data.FirewallID.ValueInt64(),
	})

	// Validate that the firewall ID exists
	if err := network_firewall.ValidateFirewallExists(r.client, ctx, data.FirewallID.ValueInt64()); err != nil {
		resp.Diagnostics.AddError(
			"Invalid Firewall ID",
			err.Error(),
		)
		return
	}

	// Extract users from the list
	var users []string
	if !data.Users.IsNull() && !data.Users.IsUnknown() {
		resp.Diagnostics.Append(data.Users.ElementsAs(ctx, &users, false)...)
	}
	// Always include the provider username in the users list
	providerUsername := r.client.GetUsername()
	found := false
	for _, u := range users {
		if u == providerUsername {
			found = true
			break
		}
	}
	if !found {
		users = append(users, providerUsername)
	}

	// Build the create request
	createReq := &ResourceGroupBusinessUnitCreateRequest{
		FirewallID:   data.FirewallID.ValueInt64(),
		BusinessUnit: data.BusinessUnit.ValueString(),
		Users:        users,
	}
	tflog.Debug(ctx, "Create request", map[string]any{
		"create_req": createReq,
	})

	// Create resource group business unit and wait for completion
	auditLog, err := CreateResourceGroupBusinessUnitAndWait(r.client, ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Resource Group Business Unit",
			"Could not create resource group business unit: "+err.Error(),
		)
		return
	}

	// Set the ID and audit ID
	if auditLog.ResourceID.String() != "" {
		data.ID = types.StringValue(auditLog.ResourceID.String())
	} else {
		resp.Diagnostics.AddError(
			"Error Creating Resource Group Business Unit",
			"Could not create resource group business unit: ResourceID is empty",
		)
		return
	}
	data.AuditID = types.StringValue(auditLog.AuditID)
	data.Status = types.StringValue(auditLog.Status)

	usersList, diags := types.ListValueFrom(ctx, types.StringType, users)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Users = usersList

	tflog.Info(ctx, "Resource group business unit created successfully", map[string]any{
		"id":       data.ID.ValueString(),
		"audit_id": data.AuditID.ValueString(),
		"status":   data.Status.ValueString(),
	})

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read refreshes the Terraform state with the latest data.
// This is a dummy implementation that preserves existing state.
func (r *ResourceGroupBusinessUnitResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ResourceGroupBusinessUnitResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Reading resource group business unit", map[string]any{
		"id":       data.ID.ValueString(),
		"audit_id": data.AuditID.ValueString(),
	})

	actionStateBody := map[string]any{
		"resourceId":   data.ID.ValueString(),
		"resourceType": "BU",
	}

	// Call action-state API to get the latest firewall data
	actionStateResponse, err := common.UpdateActionState(ctx, r.client, "engagementComponents", "read", actionStateBody)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Engagement Components",
			"Could not read engagement components: "+err.Error(),
		)
		return
	}

	// Check if response data is empty
	if len(actionStateResponse.Data) == 0 {
		resp.Diagnostics.AddWarning(
			"Empty Response Data",
			"Could not read engagement components: response data is empty, keeping existing state",
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
	data.BusinessUnit = types.StringValue(responseMap["business_unit"].(string))
	data.Status = types.StringValue(responseMap["status"].(string))

	usersList, diags := types.ListValueFrom(ctx, types.StringType, responseMap["users"])
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Users = usersList

	tflog.Info(ctx, "Business unit read successfully", map[string]any{
		"id":       data.ID.ValueString(),
		"audit_id": data.AuditID.ValueString(),
		"status":   data.Status.ValueString(),
	})

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update updates the business unit resource.
func (r *ResourceGroupBusinessUnitResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ResourceGroupBusinessUnitResourceModel
	var state ResourceGroupBusinessUnitResourceModel

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

	tflog.Debug(ctx, "Updating business unit", map[string]any{
		"id":            plan.ID.ValueString(),
		"business_unit": plan.BusinessUnit.ValueString(),
	})

	// Validate that the firewall ID exists
	if err := network_firewall.ValidateFirewallExists(r.client, ctx, plan.FirewallID.ValueInt64()); err != nil {
		resp.Diagnostics.AddError(
			"Invalid Firewall ID",
			err.Error(),
		)
		return
	}

	// Check if business_unit name changed
	if plan.BusinessUnit.ValueString() != state.BusinessUnit.ValueString() {
		tflog.Debug(ctx, "Business unit name changed", map[string]any{
			"old_name": state.BusinessUnit.ValueString(),
			"new_name": plan.BusinessUnit.ValueString(),
		})

		updateReq := &ResourceGroupBusinessUnitCreateRequest{
			BusinessUnit: plan.BusinessUnit.ValueString(),
		}

		// Update business unit and wait for completion
		auditLog, err := UpdateResourceGroupBusinessUnitAndWait(r.client, ctx, plan.ID.ValueString(), updateReq)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Updating Resource Group Business Unit",
				"Could not update resource group business unit: "+err.Error(),
			)
			return
		}

		// Update status from audit log
		plan.Status = types.StringValue(auditLog.Status)

		tflog.Info(ctx, "Resource group business unit updated successfully", map[string]any{
			"id":            plan.ID.ValueString(),
			"business_unit": plan.BusinessUnit.ValueString(),
		})
	} else {
		tflog.Debug(ctx, "No changes detected, skipping update", map[string]any{
			"id": plan.ID.ValueString(),
		})
	}

	// Resolve users: if plan has unknown users, carry forward from state
	if plan.Users.IsUnknown() {
		plan.Users = state.Users
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

// Delete deletes the business unit resource.
func (r *ResourceGroupBusinessUnitResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ResourceGroupBusinessUnitResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting resource group business unit", map[string]any{
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

	// Delete business unit and wait for completion
	_, err := DeleteResourceGroupBusinessUnitAndWait(r.client, ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Resource Group Business Unit",
			"Could not delete resource group business unit: "+err.Error(),
		)
		return
	}

	tflog.Info(ctx, "Resource group business unit deleted successfully", map[string]any{
		"id": data.ID.ValueString(),
	})
}

// ImportState imports an existing resource group business unit into Terraform state.
func (r *ResourceGroupBusinessUnitResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
