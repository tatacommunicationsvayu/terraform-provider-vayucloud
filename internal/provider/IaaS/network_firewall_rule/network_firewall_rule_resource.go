// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package network_firewall_rule

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/client"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/network_firewall"
)

var (
	cidrRegex = regexp.MustCompile(`^(\d{1,3}\.){3}\d{1,3}/\d{1,2}$`)
	dateRegex = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
)

// Ensure provider-defined types fully satisfy framework interfaces.
var _ resource.Resource = &NetworkFirewallRuleResource{}
var _ resource.ResourceWithImportState = &NetworkFirewallRuleResource{}
var _ resource.ResourceWithModifyPlan = &NetworkFirewallRuleResource{}

// NewNetworkFirewallRuleResource creates a new network firewall rules resource.
func NewNetworkFirewallRuleResource() resource.Resource {
	return &NetworkFirewallRuleResource{}
}

// NetworkFirewallRuleResource defines the resource implementation.
type NetworkFirewallRuleResource struct {
	client *client.Client
}

// NetworkFirewallRuleResourceModel describes the resource data model.
type NetworkFirewallRuleResourceModel struct {
	ID                   types.String `tfsdk:"id"`
	RuleName             types.String `tfsdk:"rule_name"`
	FirewallID           types.Int64  `tfsdk:"firewall_id"`
	Source               types.String `tfsdk:"source"`
	Action               types.String `tfsdk:"action"`
	SourceZoneID         types.Int64  `tfsdk:"source_zone_id"`
	SourceAddresses      types.List   `tfsdk:"source_addresses"`
	Destination          types.String `tfsdk:"destination"`
	DestinationZoneID    types.Int64  `tfsdk:"destination_zone_id"`
	DestinationAddresses types.List   `tfsdk:"destination_addresses"`
	ScheduleStartDate    types.String `tfsdk:"schedule_start_date"`
	ScheduleEndDate      types.String `tfsdk:"schedule_end_date"`
	Services             types.List   `tfsdk:"services"`
	AuditID              types.String `tfsdk:"audit_id"`
	Status               types.String `tfsdk:"status"`
}

// Metadata returns the resource type name.
func (r *NetworkFirewallRuleResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network_firewall_rule"
}

// Schema defines the schema for the network firewall rules resource.
func (r *NetworkFirewallRuleResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Manages a network firewall rule in VayuCloud.",
		MarkdownDescription: "Manages a network firewall rule in VayuCloud.\n\nThis resource creates or updates a firewall rule through an asynchronous provisioning process. The resource validates the rule during plan and polls the audit log until the operation is complete.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:         "The unique identifier of the firewall rule.",
				MarkdownDescription: "The unique identifier of the firewall rule.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"rule_name": schema.StringAttribute{
				Description:         "The name of the firewall rule. Only alphanumeric characters, underscores (_), and hyphens (-) are allowed.",
				MarkdownDescription: "The name of the firewall rule. Only alphanumeric characters, underscores (`_`), and hyphens (`-`) are allowed.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[a-zA-Z0-9_-]+$`),
						"Value may only contain alphanumeric characters, underscores (_), and hyphens (-)",
					),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"firewall_id": schema.Int64Attribute{
				Description:         "The firewall ID to associate the rule with. Cannot be changed after creation.",
				MarkdownDescription: "The firewall ID to associate the rule with. Cannot be changed after creation.",
				Required:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"source": schema.StringAttribute{
				Description:         "The source type. Must be one of: internet, zone, nas. Cannot be changed after creation.",
				MarkdownDescription: "The source type. Must be one of: `internet`, `zone`, `nas`. Cannot be changed after creation.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOfCaseInsensitive("internet", "zone", "nas"),
				},
			},
			"action": schema.StringAttribute{
				Description:         "The rule action. Must be allow or deny.",
				MarkdownDescription: "The rule action. Must be `allow` or `deny`.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOfCaseInsensitive("allow", "deny"),
				},
			},
			"source_zone_id": schema.Int64Attribute{
				Description:         "The source zone ID. Required when source is zone or nas (except zone with public IP /32 source addresses). Omitted or null when not applicable.",
				MarkdownDescription: "The source zone ID. Required when `source` is `zone` or `nas` (except `zone` with public IP `/32` source addresses). Omitted or null when not applicable.",
				Optional:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"source_addresses": schema.ListAttribute{
				Description:         "The source addresses in CIDR notation (e.g., 0.0.0.0/0).",
				MarkdownDescription: "The source addresses in CIDR notation (e.g., `0.0.0.0/0`).",
				Required:            true,
				ElementType:         types.StringType,
				Validators: []validator.List{
					listvalidator.ValueStringsAre(
						stringvalidator.RegexMatches(cidrRegex, "Value must be a valid CIDR notation (e.g., 10.0.0.0/24)"),
					),
				},
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
					listplanmodifier.UseStateForUnknown(),
				},
			},
			"destination": schema.StringAttribute{
				Description:         "The destination type. Must be one of: internet, zone, nas, vcs, load_balancer. Cannot be changed after creation.",
				MarkdownDescription: "The destination type. Must be one of: `internet`, `zone`, `nas`, `vcs`, `load_balancer`. Cannot be changed after creation.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOfCaseInsensitive("internet", "zone", "nas", "vcs", "load_balancer"),
				},
			},
			"destination_zone_id": schema.Int64Attribute{
				Description:         "The destination zone ID. Required when destination is zone or nas (except zone with public IP /32 destination addresses). Omitted or null when not applicable.",
				MarkdownDescription: "The destination zone ID. Required when `destination` is `zone` or `nas` (except `zone` with public IP `/32` destination addresses). Omitted or null when not applicable.",
				Optional:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"destination_addresses": schema.ListAttribute{
				Description:         "The destination addresses in CIDR notation (e.g., 10.0.0.0/24).",
				MarkdownDescription: "The destination addresses in CIDR notation (e.g., `10.0.0.0/24`).",
				Required:            true,
				ElementType:         types.StringType,
				Validators: []validator.List{
					listvalidator.ValueStringsAre(
						stringvalidator.RegexMatches(cidrRegex, "Value must be a valid CIDR notation (e.g., 10.0.0.0/24)"),
					),
				},
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
					listplanmodifier.UseStateForUnknown(),
				},
			},
			"schedule_start_date": schema.StringAttribute{
				Description:         "The schedule start date in YYYY-MM-DD format. Optional; omit for no schedule.",
				MarkdownDescription: "The schedule start date in `YYYY-MM-DD` format. Optional; omit for no schedule.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(dateRegex, "Value must be a date in YYYY-MM-DD format"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},

			},
			"schedule_end_date": schema.StringAttribute{
				Description:         "The schedule end date in YYYY-MM-DD format. Optional; omit for no schedule.",
				MarkdownDescription: "The schedule end date in `YYYY-MM-DD` format. Optional; omit for no schedule.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(dateRegex, "Value must be a date in YYYY-MM-DD format"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"services": schema.ListAttribute{
				Description:         "The services for the rule (e.g., HTTP, HTTPS).",
				MarkdownDescription: "The services for the rule (e.g., `HTTP`, `HTTPS`).",
				Required:            true,
				ElementType:         types.StringType,
			},
			"audit_id": schema.StringAttribute{
				Description:         "The audit ID from the create response. This value is set once at creation and does not change on update.",
				MarkdownDescription: "The audit ID from the create response. This value is set once at creation and does not change on update.",
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
func (r *NetworkFirewallRuleResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	cl, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = cl
}

// ModifyPlan checks firewall rule configuration during terraform plan.
func (r *NetworkFirewallRuleResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if r.client == nil || req.Plan.Raw.IsNull() {
		return
	}

	var plan NetworkFirewallRuleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	normalizeTerraformEnumAttributes(&plan)

	if plan.FirewallID.IsUnknown() || plan.RuleName.IsUnknown() || plan.Source.IsUnknown() ||
		plan.Action.IsUnknown() || plan.Destination.IsUnknown() ||
		plan.SourceAddresses.IsUnknown() || plan.DestinationAddresses.IsUnknown() ||
		plan.Services.IsUnknown() ||
		listContainsUnknown(ctx, plan.SourceAddresses) ||
		listContainsUnknown(ctx, plan.DestinationAddresses) ||
		listContainsUnknown(ctx, plan.Services) {
		return
	}

	resp.Diagnostics.Append(validateZoneRequirements(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := network_firewall.ValidateFirewallExists(r.client, ctx, plan.FirewallID.ValueInt64()); err != nil {
		resp.Diagnostics.AddError("Invalid Firewall ID", err.Error())
		return
	}

	finalizeOptionalRuleState(&plan)
	resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
}

// Create creates a new firewall rule resource.
func (r *NetworkFirewallRuleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data NetworkFirewallRuleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	finalizeOptionalRuleState(&data)

	normalizeTerraformEnumAttributes(&data)

	tflog.Debug(ctx, "Creating network firewall rule", map[string]any{
		"rule_name":   data.RuleName.ValueString(),
		"firewall_id": data.FirewallID.ValueInt64(),
	})

	resp.Diagnostics.Append(validateZoneRequirements(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := network_firewall.ValidateFirewallExists(r.client, ctx, data.FirewallID.ValueInt64()); err != nil {
		resp.Diagnostics.AddError("Invalid Firewall ID", err.Error())
		return
	}

	apiReq, diags := buildAPIRequest(ctx, &data, nil)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	auditLog, messages, err := CreateNetworkFirewallRuleAndWait(r.client, ctx, apiReq)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Network Firewall Rule", "Could not create network firewall rule: "+err.Error())
		return
	}

	for _, msg := range messages {
		resp.Diagnostics.AddWarning("Firewall Rule Validation", msg)
	}

	if auditLog.ResourceID.String() == "" {
		resp.Diagnostics.AddError("Error Creating Network Firewall Rule", "Could not create network firewall rule: ResourceID is empty")
		return
	}

	auditID := auditLog.AuditID
	data.ID = types.StringValue(auditLog.ResourceID.String())
	data.AuditID = types.StringValue(auditID)
	data.Status = types.StringValue(auditLog.Status)

	refreshDiags := r.refreshStateFromActionState(ctx, &data, true, false)
	resp.Diagnostics.Append(refreshDiags...)
	if refreshDiags.HasError() {
		return
	}

	// action-state read does not return audit_id; keep the value from the create audit
	data.AuditID = types.StringValue(auditID)
	if data.ID.IsNull() || data.ID.IsUnknown() || data.ID.ValueString() == "" {
		data.ID = types.StringValue(auditLog.ResourceID.String())
	}

	normalizeTerraformEnumAttributes(&data)
	finalizeOptionalRuleState(&data)

	tflog.Info(ctx, "Network firewall rule created successfully", map[string]any{
		"id":       data.ID.ValueString(),
		"audit_id": data.AuditID.ValueString(),
		"status":   data.Status.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read refreshes the Terraform state with the latest data from the API.
func (r *NetworkFirewallRuleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data NetworkFirewallRuleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Reading network firewall rule", map[string]any{
		"id":          data.ID.ValueString(),
		"firewall_id": data.FirewallID.ValueInt64(),
	})

	refreshDiags := r.refreshStateFromActionState(ctx, &data, false, true)
	resp.Diagnostics.Append(refreshDiags...)
	if refreshDiags.HasError() {
		return
	}

	tflog.Info(ctx, "Network firewall rule read completed", map[string]any{
		"id":     data.ID.ValueString(),
		"status": data.Status.ValueString(),
	})

	finalizeOptionalRuleState(&data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update updates the firewall rule resource.
func (r *NetworkFirewallRuleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan NetworkFirewallRuleResourceModel
	var state NetworkFirewallRuleResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Updating network firewall rule", map[string]any{
		"id":          plan.ID.ValueString(),
		"rule_name":   plan.RuleName.ValueString(),
		"firewall_id": plan.FirewallID.ValueInt64(),
	})

	resp.Diagnostics.Append(validateZoneRequirements(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := network_firewall.ValidateFirewallExists(r.client, ctx, plan.FirewallID.ValueInt64()); err != nil {
		resp.Diagnostics.AddError("Invalid Firewall ID", err.Error())
		return
	}

	ruleID, err := strconv.ParseInt(plan.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating Network Firewall Rule", "Could not parse rule ID: "+err.Error())
		return
	}

	if !hasEditableRuleChanges(plan, state) {
		tflog.Debug(ctx, "No editable changes detected, skipping update", map[string]any{
			"id": plan.ID.ValueString(),
		})
		out := plan
		if out.AuditID.IsUnknown() || out.AuditID.IsNull() {
			out.AuditID = state.AuditID
		}
		if out.Status.IsUnknown() || out.Status.IsNull() {
			out.Status = state.Status
		}
		finalizeOptionalRuleState(&out)
		resp.Diagnostics.Append(resp.State.Set(ctx, &out)...)
		return
	}

	apiReq, diags := buildUpdateAPIRequest(ctx, &plan, &state, ruleID)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	auditLog, messages, err := UpdateNetworkFirewallRuleAndWait(r.client, ctx, apiReq)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating Network Firewall Rule", "Could not update network firewall rule: "+err.Error())
		return
	}

	for _, msg := range messages {
		resp.Diagnostics.AddWarning("Firewall Rule Validation", msg)
	}

	plan.Status = types.StringValue(auditLog.Status)

	refreshDiags := r.refreshStateFromActionState(ctx, &plan, true, false)
	resp.Diagnostics.Append(refreshDiags...)
	if refreshDiags.HasError() {
		return
	}

	if plan.Status.IsUnknown() || plan.Status.IsNull() {
		plan.Status = types.StringValue(auditLog.Status)
	}
	if plan.ID.IsNull() || plan.ID.IsUnknown() || plan.ID.ValueString() == "" {
		plan.ID = state.ID
	}
	// audit_id is immutable after create; UseStateForUnknown keeps the prior value in plan
	plan.AuditID = state.AuditID

	normalizeTerraformEnumAttributes(&plan)
	finalizeOptionalRuleState(&plan)

	tflog.Info(ctx, "Network firewall rule updated successfully", map[string]any{
		"id":       plan.ID.ValueString(),
		"audit_id": plan.AuditID.ValueString(),
		"status":   plan.Status.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete deletes the firewall rule resource.
func (r *NetworkFirewallRuleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data NetworkFirewallRuleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting network firewall rule", map[string]any{
		"id":          data.ID.ValueString(),
		"firewall_id": data.FirewallID.ValueInt64(),
	})

	if err := network_firewall.ValidateFirewallExists(r.client, ctx, data.FirewallID.ValueInt64()); err != nil {
		resp.Diagnostics.AddError("Invalid Firewall ID", err.Error())
		return
	}

	_, err := DeleteNetworkFirewallRuleAndWait(r.client, ctx, data.FirewallID.ValueInt64(), data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error Deleting Network Firewall Rule", "Could not delete network firewall rule: "+err.Error())
		return
	}

	tflog.Info(ctx, "Network firewall rule deleted successfully", map[string]any{
		"id": data.ID.ValueString(),
	})
}

// ImportState imports an existing network firewall rule into Terraform state.
// Import ID format: firewall_id,rule_id (e.g. "372123,45678").
func (r *NetworkFirewallRuleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, ",")
	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid Import Identifier",
			`Expected format "firewall_id,rule_id" (e.g. "372123,45678").`,
		)
		return
	}

	firewallIDStr := strings.TrimSpace(parts[0])
	ruleIDStr := strings.TrimSpace(parts[1])
	if firewallIDStr == "" || ruleIDStr == "" {
		resp.Diagnostics.AddError(
			"Invalid Import Identifier",
			`Expected format "firewall_id,rule_id" (e.g. "372123,45678").`,
		)
		return
	}

	firewallID, err := strconv.ParseInt(firewallIDStr, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Import Identifier", "firewall_id must be an integer: "+err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("firewall_id"), types.Int64Value(firewallID))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), types.StringValue(ruleIDStr))...)
}

func (r *NetworkFirewallRuleResource) refreshStateFromActionState(ctx context.Context, data *NetworkFirewallRuleResourceModel, requireData bool, syncFromAPI bool) diag.Diagnostics {
	var d diag.Diagnostics

	actionStateResponse, err := ReadNetworkFirewallRule(r.client, ctx, data.FirewallID.ValueInt64(), data.ID.ValueString())
	if err != nil {
		d.AddError("Error Reading Network Firewall Rule", "Could not read network firewall rule: "+err.Error())
		return d
	}

	if len(actionStateResponse.Data) == 0 {
		msg := "Could not read network firewall rule: response data is empty"
		if requireData {
			d.AddError("Error Reading Network Firewall Rule", msg)
		} else {
			d.AddWarning("Empty Response Data", msg+", keeping existing values")
		}
		return d
	}

	var responseMap map[string]interface{}
	if err := json.Unmarshal(actionStateResponse.Data, &responseMap); err != nil {
		var asString string
		if err2 := json.Unmarshal(actionStateResponse.Data, &asString); err2 == nil && strings.TrimSpace(asString) != "" {
			d.AddError(
				"Error Reading Network Firewall Rule",
				fmt.Sprintf("Platform returned an error instead of rule state: %s", asString),
			)
			return d
		}
		d.AddError(
			"Error Unmarshalling Response Data",
			fmt.Sprintf("Could not unmarshal response data: %s. Response: %s", err.Error(), string(actionStateResponse.Data)),
		)
		return d
	}

	applyFirewallRuleResponseMap(ctx, data, responseMap, syncFromAPI, &d)
	return d
}

func applyFirewallRuleResponseMap(ctx context.Context, data *NetworkFirewallRuleResourceModel, responseMap map[string]interface{}, syncFromAPI bool, d *diag.Diagnostics) {
	setRequiredInt64FromAPI := func(target *types.Int64, keys ...string) {
		id := optionalInt64FromMap(responseMap, keys...)
		if id == nil {
			if !target.IsNull() && !target.IsUnknown() {
				return
			}
			*target = types.Int64Null()
			return
		}
		if !target.IsNull() && !target.IsUnknown() {
			return
		}
		*target = types.Int64Value(*id)
	}

	// Optional zone IDs are config-authoritative: when omitted (null), never populate from API.
	setOptionalZoneIDFromAPI := func(target *types.Int64, keys ...string) {
		if target.IsUnknown() {
			*target = types.Int64Null()
			return
		}
		if target.IsNull() {
			return
		}
		if !syncFromAPI {
			return
		}
		if id := optionalInt64FromMap(responseMap, keys...); id != nil {
			*target = types.Int64Value(*id)
		}
	}

	setOptionalScheduleFromAPI := func(target *types.String, keys ...string) {
		if target.IsUnknown() {
			*target = types.StringNull()
			return
		}
		if target.IsNull() {
			return
		}
		if !syncFromAPI {
			return
		}
		if v := stringFromMap(responseMap, keys...); v != "" {
			if normalized := normalizeScheduleDateForState(v); normalized != "" {
				*target = types.StringValue(normalized)
			}
			return
		}
		*target = types.StringNull()
	}

	if v := stringFromMap(responseMap, "ruleId", "rule_id", "id", "ruleid"); v != "" {
		data.ID = types.StringValue(v)
	} else if ruleID := optionalInt64FromMap(responseMap, "ruleId", "rule_id", "id", "ruleid"); ruleID != nil {
		data.ID = types.StringValue(strconv.FormatInt(*ruleID, 10))
	}

	if v := stringFromMap(responseMap, "ruleName", "rule_name"); v != "" {
		data.RuleName = types.StringValue(v)
	}
	setRequiredInt64FromAPI(&data.FirewallID, "firewallId", "firewall_id")
	if v := stringFromMap(responseMap, "source"); v != "" {
		if data.Source.IsNull() || data.Source.IsUnknown() {
			data.Source = types.StringValue(canonicalTerraformSource(v))
		}
	}
	if v := stringFromMap(responseMap, "action"); v != "" {
		data.Action = types.StringValue(canonicalTerraformAction(v))
	}
	setOptionalZoneIDFromAPI(&data.SourceZoneID, "sourceZoneId", "source_zone_id")
	if v := stringFromMap(responseMap, "destination"); v != "" {
		if data.Destination.IsNull() || data.Destination.IsUnknown() {
			data.Destination = types.StringValue(canonicalTerraformDestination(v))
		}
	}
	setOptionalZoneIDFromAPI(&data.DestinationZoneID, "destinationZoneId", "destination_zone_id")

	setOptionalScheduleFromAPI(&data.ScheduleStartDate, "scheduleStartDate", "schedule_start_date")
	setOptionalScheduleFromAPI(&data.ScheduleEndDate, "scheduleEndDate", "schedule_end_date")

	if v := stringFromMap(responseMap, "status"); v != "" {
		data.Status = types.StringValue(v)
	}

	setStringListAttr(ctx, d, &data.SourceAddresses, stringSliceFromMap(responseMap, "sourceAddresses", "source_addresses"))
	setStringListAttr(ctx, d, &data.DestinationAddresses, stringSliceFromMap(responseMap, "destinationAddresses", "destination_addresses"))
	if syncFromAPI || data.Services.IsNull() || data.Services.IsUnknown() {
		services := stringSliceFromMap(responseMap, "services")
		if syncFromAPI {
			source := data.Source.ValueString()
			if v := stringFromMap(responseMap, "source"); v != "" && (source == "" || data.Source.IsNull() || data.Source.IsUnknown()) {
				source = canonicalTerraformSource(v)
			}
			destination := data.Destination.ValueString()
			if v := stringFromMap(responseMap, "destination"); v != "" && (destination == "" || data.Destination.IsNull() || data.Destination.IsUnknown()) {
				destination = canonicalTerraformDestination(v)
			}
			services = normalizeServicesForState(services, source, destination)
		}
		setStringListAttr(ctx, d, &data.Services, services)
	}

	normalizeTerraformEnumAttributes(data)
	finalizeOptionalRuleState(data)
}

// finalizeOptionalRuleState ensures optional attributes are known (null or set) before plan/state writes.
// Zone IDs are optional: if omitted they must be null after apply, never unknown.
func finalizeOptionalRuleState(data *NetworkFirewallRuleResourceModel) {
	if data.ScheduleStartDate.IsUnknown() {
		data.ScheduleStartDate = types.StringNull()
	}
	if data.ScheduleEndDate.IsUnknown() {
		data.ScheduleEndDate = types.StringNull()
	}
	if data.SourceZoneID.IsUnknown() {
		data.SourceZoneID = types.Int64Null()
	}
	if data.DestinationZoneID.IsUnknown() {
		data.DestinationZoneID = types.Int64Null()
	}

	source := strings.ToLower(data.Source.ValueString())
	destination := strings.ToLower(data.Destination.ValueString())
	if source != "zone" && source != "nas" {
		data.SourceZoneID = types.Int64Null()
	}
	if destination != "zone" && destination != "nas" {
		data.DestinationZoneID = types.Int64Null()
	}
}

func setStringListAttr(ctx context.Context, d *diag.Diagnostics, target *types.List, values []string) {
	if values == nil {
		values = []string{}
	}
	list, diags := types.ListValueFrom(ctx, types.StringType, values)
	d.Append(diags...)
	if !diags.HasError() {
		*target = list
	}
}

func normalizeTerraformEnumAttributes(data *NetworkFirewallRuleResourceModel) {
	if !data.Source.IsNull() && !data.Source.IsUnknown() {
		data.Source = types.StringValue(canonicalTerraformSource(data.Source.ValueString()))
	}
	if !data.Destination.IsNull() && !data.Destination.IsUnknown() {
		data.Destination = types.StringValue(canonicalTerraformDestination(data.Destination.ValueString()))
	}
	if !data.Action.IsNull() && !data.Action.IsUnknown() {
		data.Action = types.StringValue(canonicalTerraformAction(data.Action.ValueString()))
	}
}

func validateZoneRequirements(ctx context.Context, data *NetworkFirewallRuleResourceModel) diag.Diagnostics {
	var d diag.Diagnostics
	source := strings.ToLower(data.Source.ValueString())
	destination := strings.ToLower(data.Destination.ValueString())

	if source == "zone" || source == "nas" {
		if data.SourceZoneID.IsNull() || data.SourceZoneID.IsUnknown() {
			addrs, diags := listStringsFromTerraform(ctx, data.SourceAddresses)
			d.Append(diags...)
			if d.HasError() {
				return d
			}
			if source != "zone" || !addressesArePublicHostRoutes(addrs) {
				d.AddError("Missing source_zone_id", "source_zone_id is required when source is 'zone' or 'nas' (unless source is 'zone' with public IP /32 addresses)")
			}
		}
	}

	if destination == "zone" || destination == "nas" {
		if data.DestinationZoneID.IsNull() || data.DestinationZoneID.IsUnknown() {
			addrs, diags := listStringsFromTerraform(ctx, data.DestinationAddresses)
			d.Append(diags...)
			if d.HasError() {
				return d
			}
			if destination != "zone" || !addressesArePublicHostRoutes(addrs) {
				d.AddError("Missing destination_zone_id", "destination_zone_id is required when destination is 'zone' or 'nas' (unless destination is 'zone' with public IP /32 addresses)")
			}
		}
	}

	return d
}

func addressesArePublicHostRoutes(addrs []string) bool {
	if len(addrs) == 0 {
		return false
	}
	for _, addr := range addrs {
		ip, network, err := net.ParseCIDR(strings.TrimSpace(addr))
		if err != nil {
			return false
		}
		ones, bits := network.Mask.Size()
		if ones != bits {
			return false
		}
		if ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
			return false
		}
	}
	return true
}

func buildAPIRequest(ctx context.Context, data *NetworkFirewallRuleResourceModel, ruleID *int64) (*NetworkFirewallRuleRequest, diag.Diagnostics) {
	var d diag.Diagnostics

	sourceAddresses, diags := listStringsFromTerraform(ctx, data.SourceAddresses)
	d.Append(diags...)
	destinationAddresses, diags := listStringsFromTerraform(ctx, data.DestinationAddresses)
	d.Append(diags...)
	services, diags := listStringsFromTerraform(ctx, data.Services)
	d.Append(diags...)
	if d.HasError() {
		return nil, d
	}

	req := &NetworkFirewallRuleRequest{
		RuleName:             data.RuleName.ValueString(),
		FirewallID:           data.FirewallID.ValueInt64(),
		Source:               apiSource(data.Source.ValueString()),
		Action:               apiAction(data.Action.ValueString()),
		SourceAddresses:      sourceAddresses,
		Destination:          apiDestination(data.Destination.ValueString()),
		DestinationAddresses: destinationAddresses,
		Services:             services,
		RuleID:               ruleID,
	}

	if !data.ScheduleStartDate.IsNull() && !data.ScheduleStartDate.IsUnknown() {
		req.ScheduleStartDate = data.ScheduleStartDate.ValueString()
	}
	if !data.ScheduleEndDate.IsNull() && !data.ScheduleEndDate.IsUnknown() {
		req.ScheduleEndDate = data.ScheduleEndDate.ValueString()
	}

	if !data.SourceZoneID.IsNull() && !data.SourceZoneID.IsUnknown() {
		zoneID := data.SourceZoneID.ValueInt64()
		req.SourceZoneID = &zoneID
	}
	if !data.DestinationZoneID.IsNull() && !data.DestinationZoneID.IsUnknown() {
		zoneID := data.DestinationZoneID.ValueInt64()
		req.DestinationZoneID = &zoneID
	}

	return req, d
}

// buildUpdateAPIRequest builds an update payload using editable fields from plan and
// keeps immutable identity/routing fields from state.
func buildUpdateAPIRequest(ctx context.Context, plan, state *NetworkFirewallRuleResourceModel, ruleID int64) (*NetworkFirewallRuleRequest, diag.Diagnostics) {
	merged := *plan
	merged.FirewallID = state.FirewallID
	merged.Source = state.Source
	merged.SourceZoneID = state.SourceZoneID
	merged.Destination = state.Destination
	merged.DestinationZoneID = state.DestinationZoneID
	return buildAPIRequest(ctx, &merged, &ruleID)
}

// hasEditableRuleChanges checks whether any of the editable fields between the desired (plan)
// and current (state) resource models are different, indicating a change that requires an update.
// It returns true if any editable field has changed; otherwise, it returns false.
func hasEditableRuleChanges(plan, state NetworkFirewallRuleResourceModel) bool {
	return !plan.Services.Equal(state.Services) ||
		!plan.Action.Equal(state.Action)
}

func listContainsUnknown(ctx context.Context, list types.List) bool {
	if list.IsNull() || list.IsUnknown() {
		return true
	}
	var elems []types.String
	if diags := list.ElementsAs(ctx, &elems, false); diags.HasError() {
		return true
	}
	for _, elem := range elems {
		if elem.IsUnknown() {
			return true
		}
	}
	return false
}

func listStringsFromTerraform(ctx context.Context, list types.List) ([]string, diag.Diagnostics) {
	var d diag.Diagnostics
	if list.IsNull() || list.IsUnknown() {
		return nil, d
	}

	var elems []types.String
	d.Append(list.ElementsAs(ctx, &elems, false)...)
	if d.HasError() {
		return nil, d
	}

	vals := make([]string, 0, len(elems))
	for i, elem := range elems {
		if elem.IsUnknown() {
			d.AddError(
				"Invalid List Value",
				fmt.Sprintf("List element at index %d is unknown; ensure referenced computed values are available (use depends_on if needed).", i),
			)
			return nil, d
		}
		if elem.IsNull() {
			continue
		}
		vals = append(vals, elem.ValueString())
	}
	return vals, d
}

// ApplyFirewallRuleResponseMapForTest exposes response mapping for integration probes.
func ApplyFirewallRuleResponseMapForTest(ctx context.Context, data *NetworkFirewallRuleResourceModel, responseMap map[string]interface{}, syncFromAPI bool, d *diag.Diagnostics) {
	applyFirewallRuleResponseMap(ctx, data, responseMap, syncFromAPI, d)
}
