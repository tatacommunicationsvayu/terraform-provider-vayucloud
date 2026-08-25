package network_lb

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/client"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/network_firewall"
)

var (
	_ resource.Resource                = &NetworkLBResource{}
	_ resource.ResourceWithImportState = &NetworkLBResource{}
	_ resource.ResourceWithModifyPlan  = &NetworkLBResource{}
)

func NewNetworkLBResource() resource.Resource {
	return &NetworkLBResource{}
}

type NetworkLBResource struct {
	client *client.Client
}

type NetworkLBResourceModel struct {
	ID            types.String `tfsdk:"id"`
	EngagementID  types.Int64  `tfsdk:"engagement_id"`
	EndpointID    types.Int64  `tfsdk:"endpoint_id"`
	FirewallID    types.String `tfsdk:"firewall_id"`
	ZoneID        types.Int64  `tfsdk:"zone_id"`
	Name          types.String `tfsdk:"name"`
	Type          types.String `tfsdk:"type"`
	DisplayName   types.String `tfsdk:"display_name"`
	Bandwidth     types.String `tfsdk:"bandwidth"`
	PricingModel  types.String `tfsdk:"pricing_model"`
	CIStatus      types.String `tfsdk:"ci_status"`
	AuditID       types.String `tfsdk:"audit_id"`
	Status        types.String `tfsdk:"status"`
	LastUpdated   types.String `tfsdk:"last_updated"`
}

func (r *NetworkLBResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network_lb"
}

func (r *NetworkLBResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a VayuCloud Load Balancer. Create/delete operations are asynchronous. " +
			"`zone_id` is required. The zone must exist on the firewall and is used for virtual services and pool member discovery.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Load Balancer CI Master ID (returned from audit after create).",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"engagement_id": schema.Int64Attribute{
				MarkdownDescription: "Engagement ID (populated from the platform after create/read).",
				Computed:            true,
				Optional:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"endpoint_id": schema.Int64Attribute{
				MarkdownDescription: "Endpoint ID (populated from the platform after create/read).",
				Computed:            true,
				Optional:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"firewall_id": schema.StringAttribute{
				MarkdownDescription: "Firewall CI Master ID on which to enable LB. Changing an existing non-empty value forces resource replacement.",
				Required:            true,
			},
			"zone_id": schema.Int64Attribute{
				MarkdownDescription: "Network zone ID on the firewall for virtual service placement and pool member VM discovery. Required. Validated against `firewall_id` before load balancer enable.",
				Required:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Load Balancer name (populated from the platform after create/read).",
				Computed:            true,
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "Load Balancer type (F5, HAProxy, AVI); determined by firewall hypervisor and populated after create/read.",
				Computed:            true,
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"display_name": schema.StringAttribute{
				MarkdownDescription: "Load Balancer display name (populated from the platform after create/read).",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"bandwidth": schema.StringAttribute{
				MarkdownDescription: "Load balancer bandwidth in **Mbps (megabits per second)**. The numeric value is always treated as Mbps — e.g. `100` and `100Mbps` both mean 100 Mbps. Gbps is not supported; use the Mbps equivalent (`1000` or `1000Mbps` for 1 Gbps). Maximum 1000 Mbps per platform limits. The API receives values like `100Mbps`; read/refresh may return a plain number (`100`) without causing plan drift.",
				Required:            true,
			},
			"pricing_model": schema.StringAttribute{
				MarkdownDescription: "LB pricing model sent on enable/modify (e.g. daily, monthly). Defaults to daily when omitted.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("daily"),
			},
			"ci_status": schema.StringAttribute{
				MarkdownDescription: "CI status (ACTIVE, INACTIVE, etc.).",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"audit_id": schema.StringAttribute{
				MarkdownDescription: "Audit ID from the latest operation.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "Audit status.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"last_updated": schema.StringAttribute{
				MarkdownDescription: "Timestamp of last update.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *NetworkLBResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *NetworkLBResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data NetworkLBResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	planBandwidth := data.Bandwidth.ValueString()
	bandwidth := normalizeLBBandwidth(planBandwidth)
	if bandwidth == "" {
		resp.Diagnostics.AddError(
			"Missing Bandwidth",
			"bandwidth is required to enable a load balancer (e.g. 100 or 100Mbps).",
		)
		return
	}
	if err := validateLBBandwidthLimit(planBandwidth); err != nil {
		resp.Diagnostics.AddError("Invalid Bandwidth", err.Error())
		return
	}

	pricingModel := resolveLBPricingModel(data.PricingModel)

	tflog.Info(ctx, "Creating Load Balancer", map[string]any{
		"firewall_id":   data.FirewallID.ValueString(),
		"zone_id":       data.ZoneID.ValueInt64(),
		"lb_bandwidth":  bandwidth,
		"pricing_model": pricingModel,
	})

	firewallID, err := strconv.ParseInt(data.FirewallID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Firewall ID", fmt.Sprintf("Could not parse firewall ID: %s", err.Error()))
		return
	}

	if err := validateLBZoneForFirewall(r.client, ctx, data.ZoneID.ValueInt64(), firewallID); err != nil {
		resp.Diagnostics.AddError("Invalid Zone ID", err.Error())
		return
	}

	enableReq := &LoadBalancerEnablementRequest{
		LBBandwidth:  bandwidth,
		PricingModel: pricingModel,
	}

	auditResp, err := EnableLoadBalancer(r.client, ctx, firewallID, enableReq)
	if err != nil {
		resp.Diagnostics.AddError("Failed to Enable Load Balancer", err.Error())
		return
	}

	auditResourceID, diags := r.pollAuditLog(ctx, auditResp.Data.Audit.AuditID, "create", "loadbalancer")
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	lbCiID, resolveDiags := r.resolveLbCiIDAfterAudit(ctx, auditResourceID, firewallID)
	resp.Diagnostics.Append(resolveDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.ID = types.StringValue(strconv.FormatInt(lbCiID, 10))
	data.AuditID = types.StringValue(auditResp.Data.Audit.AuditID)
	data.Status = types.StringValue(auditResp.Status)
	data.LastUpdated = types.StringValue(time.Now().Format(time.RFC3339))

	diags = r.refreshStateFromActionState(ctx, &data)
	resp.Diagnostics.Append(diags...)

	// Keep config format in state so plan matches .tf (e.g. "22" not "22Mbps").
	data.Bandwidth = types.StringValue(planBandwidth)
	data.PricingModel = types.StringValue(pricingModel)

	normalizeUnknownValues(&data)

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *NetworkLBResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data NetworkLBResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Reading Load Balancer", map[string]any{
		"lb_id": data.ID.ValueString(),
	})

	diags := r.refreshStateFromActionState(ctx, &data)
	resp.Diagnostics.Append(diags...)
	normalizeUnknownValues(&data)

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *NetworkLBResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state NetworkLBResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !plan.FirewallID.Equal(state.FirewallID) {
		resp.Diagnostics.AddError(
			"Update Not Supported",
			"Only bandwidth can be updated in place. To change firewall_id, remove and recreate the resource.",
		)
		return
	}

	configBandwidth := plan.Bandwidth.ValueString()

	bandwidthChanged := normalizeLBBandwidth(configBandwidth) != normalizeLBBandwidth(state.Bandwidth.ValueString())
	if !bandwidthChanged {
		diags := r.refreshStateFromActionState(ctx, &plan)
		resp.Diagnostics.Append(diags...)
		plan.Bandwidth = types.StringValue(configBandwidth)
		plan.PricingModel = types.StringValue(resolveLBPricingModel(plan.PricingModel))
		normalizeUnknownValues(&plan)
		resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
		return
	}

	planBandwidth := normalizeLBBandwidth(plan.Bandwidth.ValueString())
	if planBandwidth == "" {
		resp.Diagnostics.AddError(
			"Invalid Bandwidth",
			"bandwidth must be set when changing LB bandwidth (e.g. 100Mbps).",
		)
		return
	}
	if err := validateLBBandwidthLimit(plan.Bandwidth.ValueString()); err != nil {
		resp.Diagnostics.AddError("Invalid Bandwidth", err.Error())
		return
	}

	firewallID, err := strconv.ParseInt(plan.FirewallID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Firewall ID", fmt.Sprintf("Could not parse firewall ID: %s", err.Error()))
		return
	}

	pricingModel := resolveLBPricingModel(plan.PricingModel)

	tflog.Info(ctx, "Modifying Load Balancer bandwidth", map[string]any{
		"lb_id":         plan.ID.ValueString(),
		"firewall_id":   plan.FirewallID.ValueString(),
		"old_bandwidth": state.Bandwidth.ValueString(),
		"new_bandwidth": planBandwidth,
		"pricing_model": pricingModel,
	})

	modifyReq := &LoadBalancerEnablementRequest{
		LBBandwidth:  planBandwidth,
		PricingModel: pricingModel,
	}

	auditResp, err := ModifyLoadBalancer(r.client, ctx, firewallID, modifyReq)
	if err != nil {
		resp.Diagnostics.AddError("Failed to Modify Load Balancer Bandwidth", err.Error())
		return
	}

	_, diags := r.pollAuditLog(ctx, auditResp.Data.Audit.AuditID, "update", "loadbalancer")
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.AuditID = types.StringValue(auditResp.Data.Audit.AuditID)
	plan.Status = types.StringValue(auditResp.Status)
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC3339))

	resp.Diagnostics.Append(r.refreshStateFromActionState(ctx, &plan)...)
	plan.Bandwidth = types.StringValue(configBandwidth)
	plan.PricingModel = types.StringValue(pricingModel)
	normalizeUnknownValues(&plan)

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *NetworkLBResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data NetworkLBResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Deleting Load Balancer", map[string]any{
		"lb_id":        data.ID.ValueString(),
		"firewall_id":  data.FirewallID.ValueString(),
	})

	// Use firewall_id for the disable call, not the load balancer ID
	if data.FirewallID.IsNull() || data.FirewallID.ValueString() == "" {
		resp.Diagnostics.AddError(
			"Missing Firewall ID",
			"Cannot delete load balancer without firewall_id. Re-import after backend returns firewall_ci, or set firewall_id in resource.tf and run terraform apply.",
		)
		return
	}

	firewallCiID, err := strconv.ParseInt(data.FirewallID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Firewall ID", fmt.Sprintf("Could not parse firewall ID: %s", err.Error()))
		return
	}

	auditResp, err := DisableLoadBalancer(r.client, ctx, firewallCiID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to Disable Load Balancer", err.Error())
		return
	}

	_, diags := r.pollAuditLog(ctx, auditResp.Data.Audit.AuditID, "delete", "loadbalancer")
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Load Balancer deleted successfully", map[string]any{
		"lb_id": data.ID.ValueString(),
	})
}

func (r *NetworkLBResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Parse import ID format: "lbId" or "lbId:engagementId"
	// Examples:
	//   "395241" - only LB ID
	//   "395241:15770" - LB ID and engagement ID
	parts := strings.Split(req.ID, ":")
	
	lbId := parts[0]
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), lbId)...)
	
	logMap := map[string]any{
		"lb_id": lbId,
	}
	
	if len(parts) >= 2 {
		// Format: "395241:15770" - engagement ID provided
		engagementId, err := strconv.ParseInt(parts[1], 10, 64)
		if err == nil {
			resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("engagement_id"), types.Int64Value(engagementId))...)
			logMap["engagement_id"] = engagementId
		}
	}
	
	// Get current state to refresh
	var data NetworkLBResourceModel
	resp.Diagnostics.Append(resp.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Failed to get imported state", logMap)
		return
	}
	
	// Refresh state from backend to populate all fields including firewall_id
	diags := r.refreshStateFromActionState(ctx, &data)
	resp.Diagnostics.Append(diags...)
	
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
	tflog.Info(ctx, "Imported LB (firewall_id auto-populated from backend)", logMap)
}

func (r *NetworkLBResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if r.client == nil || req.Plan.Raw.IsNull() {
		return
	}

	var planData NetworkLBResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &planData)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !planData.Bandwidth.IsNull() && !planData.Bandwidth.IsUnknown() {
		if normalizeLBBandwidth(planData.Bandwidth.ValueString()) == "" {
			resp.Diagnostics.AddError(
				"Invalid Bandwidth",
				"bandwidth must be a positive value in Mbps (e.g. 100 or 100Mbps). Gbps is not supported — use the Mbps number instead (e.g. 1000 for 1 Gbps).",
			)
			return
		}
		if err := validateLBBandwidthLimit(planData.Bandwidth.ValueString()); err != nil {
			resp.Diagnostics.AddError("Invalid Bandwidth", err.Error())
			return
		}
	}

	var stateData NetworkLBResourceModel
	hasState := !req.State.Raw.IsNull()
	if hasState {
		resp.Diagnostics.Append(req.State.Get(ctx, &stateData)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	if planData.FirewallID.IsNull() || planData.FirewallID.IsUnknown() {
		if hasState && firewallIDRequiresReplace(planData.FirewallID, stateData.FirewallID) {
			resp.RequiresReplace.Append(path.Root("firewall_id"))
		}
		return
	}

	firewallID, err := strconv.ParseInt(strings.TrimSpace(planData.FirewallID.ValueString()), 10, 64)
	if err != nil || firewallID <= 0 {
		resp.Diagnostics.AddError(
			"Invalid Firewall ID",
			fmt.Sprintf("firewall_id must be a positive integer: %s", planData.FirewallID.ValueString()),
		)
		return
	}

	if err := network_firewall.ValidateFirewallExists(r.client, ctx, firewallID); err != nil {
		resp.Diagnostics.AddError("Invalid Firewall ID", err.Error())
		return
	}

	if !planData.ZoneID.IsNull() && !planData.ZoneID.IsUnknown() {
		if err := validateLBZoneForFirewall(r.client, ctx, planData.ZoneID.ValueInt64(), firewallID); err != nil {
			resp.Diagnostics.AddError("Invalid Zone ID", err.Error())
			return
		}
	} else if !hasState && !planData.ZoneID.IsUnknown() {
		resp.Diagnostics.AddError(
			"Missing Zone ID",
			"zone_id is required. Set the network zone ID on the firewall where virtual services and pool member VMs will be placed. "+
				"Resolve zone_id from vayucloud_network_zone_list for your firewall_id.",
		)
		return
	}

	// checkLBNotEnabled := !hasState
	// if hasState && firewallIDRequiresReplace(planData.FirewallID, stateData.FirewallID) {
	// 	checkLBNotEnabled = true
	// }
	// if checkLBNotEnabled {
	// 	lbCiID, err := FindLoadBalancerCIOnFirewall(r.client, ctx, firewallID)
	// 	if err != nil {
	// 		resp.Diagnostics.AddError("Load Balancer Plan Check Failed", err.Error())
	// 		return
	// 	}
	// 	if lbCiID > 0 {
	// 		resp.Diagnostics.AddError(
	// 			"Load Balancer Already Enabled",
	// 			fmt.Sprintf(
	// 				"A load balancer is already enabled on firewall %d (LB CI %d). Import the existing load balancer instead of creating a new one: terraform import vayucloud_network_lb.<name> %d",
	// 				firewallID,
	// 				lbCiID,
	// 				lbCiID,
	// 			),
	// 		)
	// 		return
	// 	}
	// }
	// }

	if !hasState {
		return
	}

	// Only replace when both plan and state have a firewall_id and they differ.
	// Empty state firewall_id means refresh has not populated it yet — not a replacement.
	if firewallIDRequiresReplace(planData.FirewallID, stateData.FirewallID) {
		resp.RequiresReplace.Append(path.Root("firewall_id"))
	}

	// Bandwidth modify issues a new audit; volatile computed fields must be unknown in plan.
	if !planData.Bandwidth.IsNull() && !planData.Bandwidth.IsUnknown() &&
		normalizeLBBandwidth(planData.Bandwidth.ValueString()) != normalizeLBBandwidth(stateData.Bandwidth.ValueString()) {
		resp.Plan.SetAttribute(ctx, path.Root("audit_id"), types.StringUnknown())
		resp.Plan.SetAttribute(ctx, path.Root("status"), types.StringUnknown())
		resp.Plan.SetAttribute(ctx, path.Root("last_updated"), types.StringUnknown())
		resp.Plan.SetAttribute(ctx, path.Root("ci_status"), types.StringUnknown())
	}
}

// Helper Methods

func (r *NetworkLBResource) pollAuditLog(ctx context.Context, auditID, action, module string) (string, diag.Diagnostics) {
	var diags diag.Diagnostics

	tflog.Debug(ctx, "Polling audit log", map[string]any{
		"audit_id": auditID,
		"action":   action,
		"module":   module,
	})

	auditLog, err := r.client.WaitForAuditCompletion(ctx, auditID, action, module, nil)
	if err != nil {
		diags.AddError(
			"Audit Polling Timeout",
			fmt.Sprintf("Load Balancer operation did not complete within timeout: %s", err.Error()),
		)
		return "", diags
	}

	resourceID := auditLog.ResourceID.String()
	tflog.Info(ctx, "Audit polling completed successfully", map[string]any{
		"audit_id":    auditID,
		"resourceId":  resourceID,
	})

	return resourceID, diags
}

// resolveLbCiIDAfterAudit returns the LB CI from audit resourceId when the backend
// has called updateAuditDetails. Falls back to firewall lookup for older backends.
func (r *NetworkLBResource) resolveLbCiIDAfterAudit(ctx context.Context, auditResourceID string, firewallID int64) (int64, diag.Diagnostics) {
	var diags diag.Diagnostics

	auditID, err := strconv.ParseInt(strings.TrimSpace(auditResourceID), 10, 64)
	if err != nil || auditID <= 0 {
		tflog.Info(ctx, "Audit resourceId missing or invalid; resolving LB CI from firewall", map[string]any{
			"audit_resourceId": auditResourceID,
			"firewall_ci_id":   firewallID,
		})
		lbCiID, resolveErr := ResolveLbCiIDFromFirewall(r.client, ctx, firewallID)
		if resolveErr != nil {
			diags.AddError(
				"Failed to Resolve Load Balancer CI",
				fmt.Sprintf("Could not determine LB CI from audit or firewall %d: %s", firewallID, resolveErr.Error()),
			)
		}
		return lbCiID, diags
	}

	if auditID == firewallID {
		tflog.Info(ctx, "Audit resourceId is firewall CI; resolving linked LB CI", map[string]any{
			"firewall_ci_id": firewallID,
		})
		lbCiID, resolveErr := ResolveLbCiIDFromFirewall(r.client, ctx, firewallID)
		if resolveErr != nil {
			diags.AddError(
				"Failed to Resolve Load Balancer CI",
				fmt.Sprintf("Audit returned firewall CI %d but linked LB CI was not found: %s", firewallID, resolveErr.Error()),
			)
		}
		return lbCiID, diags
	}

	tflog.Info(ctx, "Using Load Balancer CI from audit resourceId", map[string]any{
		"lb_ci_id": auditID,
	})
	return auditID, diags
}

func (r *NetworkLBResource) refreshStateFromActionState(ctx context.Context, data *NetworkLBResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	tflog.Debug(ctx, "Refreshing state from action-state API", map[string]any{
		"lb_id": data.ID.ValueString(),
	})

	lbCiID, err := strconv.ParseInt(data.ID.ValueString(), 10, 64)
	if err != nil {
		diags.AddWarning("Invalid LB ID", fmt.Sprintf("Could not parse LB ID for action-state call: %s", err.Error()))
		return diags
	}

	details, err := GetLoadBalancerDetails(r.client, ctx, lbCiID)
	if err != nil {
		diags.AddWarning("Failed to Refresh Load Balancer State", fmt.Sprintf("Could not read LB from platform: %s. Continuing with available state.", err.Error()))
		return diags
	}
	payload, err := json.MarshalIndent(details, "", "  ")
	if err != nil {
		diags.AddWarning("Failed to Marshal Load Balancer Details", fmt.Sprintf("Could not marshal LB details: %s", err.Error()))
		return diags
	}

    tflog.Info(ctx, "LB ACTION STATE RESPONSE", map[string]any{
    	"lb_id":   lbCiID,
    	"payload": string(payload),
    	"error":   err,
    })

	tflog.Debug(ctx, "Load Balancer action-state read payload", map[string]any{
		"details": details,
	})

	// Map fields returned by LoadBalancerStateServiceImpl.read (snake_case JSON keys).
	if v := stringFromActionStateMap(details, "ci_name","name"); v != "" {
		data.Name = types.StringValue(v)
	}
	if v := stringFromActionStateMap(details, "display_name", "lbDisplayName"); v != "" {
		data.DisplayName = types.StringValue(v)
	}
	if v := stringFromActionStateMap(details, "bandwidth", "lb_bandwidth"); v != "" {
		preferred := data.Bandwidth.ValueString()
		data.Bandwidth = types.StringValue(reconcileBandwidthState(preferred, v))
	}
	if v := stringFromActionStateMap(details, "lb_type"); v != "" {
		data.Type = types.StringValue(v)
	}
	if v := stringFromActionStateMap(details, "ci_status"); v != "" {
		data.CIStatus = types.StringValue(v)
	}
	if id, ok := int64FromActionStateMap(details, "id"); ok {
		data.ID = types.StringValue(strconv.FormatInt(id, 10))
	}
	if engagementID, ok := int64FromActionStateMap(details, "engagement_id", "engagementId"); ok {
		data.EngagementID = types.Int64Value(engagementID)
	}
	if endpointID, ok := int64FromActionStateMap(details, "endpoint_id", "endpointId"); ok {
		data.EndpointID = types.Int64Value(endpointID)
	}
	if firewallCI, ok := int64FromActionStateMap(details, "firewall_ci", "firewallCI"); ok {
		data.FirewallID = types.StringValue(strconv.FormatInt(firewallCI, 10))
	} else if v := stringFromActionStateMap(details, "firewall_ci", "firewallCI"); v != "" {
		data.FirewallID = types.StringValue(v)
	}

	data.LastUpdated = types.StringValue(time.Now().Format(time.RFC3339))

	tflog.Debug(ctx, "Load Balancer state refreshed successfully", map[string]any{
		"lb_id":         data.ID.ValueString(),
		"name":          data.Name.ValueString(),
		"display_name":  data.DisplayName.ValueString(),
		"bandwidth":     data.Bandwidth.ValueString(),
		"firewall_id":   data.FirewallID.ValueString(),
		"engagement_id": data.EngagementID.ValueInt64(),
		"endpoint_id":   data.EndpointID.ValueInt64(),
	})

	return diags
}

func normalizeUnknownValues(data *NetworkLBResourceModel) {
	if data.Name.IsUnknown() {
		data.Name = types.StringNull()
	}

	if data.Type.IsUnknown() {
		data.Type = types.StringNull()
	}

	if data.DisplayName.IsUnknown() {
		data.DisplayName = types.StringNull()
	}

	if data.CIStatus.IsUnknown() {
		data.CIStatus = types.StringNull()
	}

	if data.AuditID.IsUnknown() {
		data.AuditID = types.StringNull()
	}

	if data.Status.IsUnknown() {
		data.Status = types.StringNull()
	}

	if data.FirewallID.IsUnknown() {
		data.FirewallID = types.StringNull()
	}

	if data.Bandwidth.IsUnknown() {
		data.Bandwidth = types.StringNull()
	}

	if data.PricingModel.IsUnknown() || data.PricingModel.IsNull() {
		data.PricingModel = types.StringValue("daily")
	}

	if data.EngagementID.IsUnknown() {
		data.EngagementID = types.Int64Null()
	}

	if data.EndpointID.IsUnknown() {
		data.EndpointID = types.Int64Null()
	}

	if data.ZoneID.IsUnknown() {
		data.ZoneID = types.Int64Null()
	}
}
