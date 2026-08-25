// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package network_zone

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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/client"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/common"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/network_firewall"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/resource_group_environment"
)

// Ensure provider-defined types fully satisfy framework interfaces.
var _ resource.Resource = &NetworkZoneResource{}
var _ resource.ResourceWithImportState = &NetworkZoneResource{}
var _ resource.ResourceWithModifyPlan = &NetworkZoneResource{}

// NewNetworkZoneResource creates a new network zone resource.
func NewNetworkZoneResource() resource.Resource {
	return &NetworkZoneResource{}
}

// NetworkZoneResource defines the resource implementation.
type NetworkZoneResource struct {
	client *client.Client
}

// NetworkZoneResourceModel describes the resource data model.
type NetworkZoneResourceModel struct {
	// ID is the unique identifier (resource_id from audit completion)
	ID types.String `tfsdk:"id"`

	// Name is the zone name
	Name types.String `tfsdk:"name"`

	// EnvironmentID is the environment ID
	EnvironmentID types.Int64 `tfsdk:"environment_id"`

	// FirewallID is the firewall ID from firewall resource
	FirewallID types.Int64 `tfsdk:"firewall_id"`

	// NoOfIPs is the number of IPs (only used with Auto IPAM)
	NoOfIPs types.Int64 `tfsdk:"no_of_ips"`

	// Purpose is the purpose of the zone
	Purpose types.String `tfsdk:"purpose"`

	// DataPlane is the data plane type ("Auto IPAM" or "Data Plane CIDR")
	DataPlane types.String `tfsdk:"data_plane"`

	// CIDR is the IPv4 CIDR (required for Data Plane CIDR; computed after create for Auto IPAM)
	CIDR types.String `tfsdk:"cidr"`

	// ZoneType is the zone type ("overlay" or "vlan")
	ZoneType types.String `tfsdk:"zone_type"`

	// NoOfV6IPs is the number of IPv6 IPs (only used when dual_stack_mode is "yes")
	NoOfV6IPs types.Int64 `tfsdk:"no_of_v6_ips"`

	// Ipv6CIDR is the IPv6 CIDR value (only used when dual_stack_mode is "yes")
	IPv6CIDR types.String `tfsdk:"ipv6_cidr"`

	// Computed attributes
	// AuditID is the audit ID from the creation response
	AuditID types.String `tfsdk:"audit_id"`

	// Status is the final status from the audit log
	Status types.String `tfsdk:"status"`

	NetworkZoneId types.Int64 `tfsdk:"network_zone_id"`
}

// Metadata returns the resource type name.
func (r *NetworkZoneResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network_zone"
}

// Schema defines the schema for the network zone resource.
func (r *NetworkZoneResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Manages a network zone resource in VayuCloud.",
		MarkdownDescription: "Manages a network zone resource in VayuCloud.\n\nThis resource creates a network zone through an asynchronous provisioning process. The resource will poll the audit log until the network zone creation is complete.\n\n## Data Plane Options\n\n- **Auto IPAM**: Automatic IP address management. Requires `no_of_ips`.\n- **Data Plane CIDR**: Manual CIDR specification. Requires `cidr`.\n\n## Dual Stack Mode\n\nWhen `dual_stack_mode` is set to `yes`, you must provide `no_of_v6_ips`.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:         "The unique identifier of the network zone (resource ID).",
				MarkdownDescription: "The unique identifier of the network zone (resource ID).",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description:         "The name of the network zone.",
				MarkdownDescription: "The name of the network zone.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(5, 45),
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[a-zA-Z0-9 _-]+$`),
						"Value may only contain alphanumeric characters, underscores (_), and hyphens (-)",
					),
				},
			},
			"environment_id": schema.Int64Attribute{
				Description:         "The environment ID.",
				MarkdownDescription: "The environment ID.",
				Required:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"firewall_id": schema.Int64Attribute{
				Description:         "The firewall ID.",
				MarkdownDescription: "The firewall ID.",
				Required:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"no_of_ips": schema.Int64Attribute{
				Description:         "The number of IPs. Required when data_plane is 'Auto IPAM'.",
				MarkdownDescription: "The number of IPs. Required when `data_plane` is `Auto IPAM`.",
				Required:            true,
				
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"purpose": schema.StringAttribute{
				Description:         "The purpose of the network zone.",
				MarkdownDescription: "The purpose of the network zone.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("IPC"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"data_plane": schema.StringAttribute{
				Description:         "The data plane type. Must be 'Auto IPAM' or 'Data Plane CIDR'.",
				MarkdownDescription: "The data plane type. Must be `Auto IPAM` or `Data Plane CIDR`.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("Auto IPAM"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"cidr": schema.StringAttribute{
				Description:         "The IPv4 CIDR for the zone. Required when data_plane is 'Data Plane CIDR'; populated from the platform after create for Auto IPAM.",
				MarkdownDescription: "The IPv4 CIDR for the zone. Required when `data_plane` is `Data Plane CIDR`; populated from the platform after create for Auto IPAM.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"zone_type": schema.StringAttribute{
				Description:         "The network zone type. Must be 'overlay' or 'vlan'.",
				MarkdownDescription: "The network zone type. Must be `overlay` or `vlan`.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("overlay"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"no_of_v6_ips": schema.Int64Attribute{
				Description:         "The number of IPv6 IPs.",
				MarkdownDescription: "The number of IPv6 IPs.",
				Optional:            true,
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
			"ipv6_cidr": schema.StringAttribute{
				Description:         "The IPv6 CIDR value (only used when dual_stack_mode is 'yes').",
				MarkdownDescription: "The IPv6 CIDR value (only used when dual_stack_mode is 'yes').",
				Optional:            true,
			},
			"status": schema.StringAttribute{
				Description:         "The final status from the audit log.",
				MarkdownDescription: "The final status from the audit log.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"network_zone_id": schema.Int64Attribute{
				Description:         "The network zone ID.",
				MarkdownDescription: "The network zone ID.",
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *NetworkZoneResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// ModifyPlan validates firewall and environment IDs during terraform plan.
// Destroy plans use prior state when the plan value is null.
func (r *NetworkZoneResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if r.client == nil {
		return
	}

	var plan NetworkZoneResourceModel
	if req.Plan.Raw.IsNull() {
		resp.Diagnostics.Append(req.State.Get(ctx, &plan)...)
	} else {
		resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.FirewallID.IsUnknown() || plan.EnvironmentID.IsUnknown() {
		return
	}

	if err := network_firewall.ValidateFirewallExists(r.client, ctx, plan.FirewallID.ValueInt64()); err != nil {
		resp.Diagnostics.AddError(
			"Invalid Firewall ID",
			err.Error(),
		)
		return
	}

	if err := resource_group_environment.ValidateEnvironmentExists(r.client, ctx, plan.EnvironmentID.ValueInt64()); err != nil {
		resp.Diagnostics.AddError(
			"Invalid Environment ID",
			err.Error(),
		)
		return
	}
	if err := resource_group_environment.ValidateEnvironmentExistsForFirewall(r.client, ctx, plan.EnvironmentID.ValueInt64(), plan.FirewallID.ValueInt64()); err != nil {
		resp.Diagnostics.AddError(
			"Invalid Environment ID",
			err.Error(),
		)
		return
	}
}

// Create creates a new network zone resource.
func (r *NetworkZoneResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data NetworkZoneResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Validate that the firewall ID exists
	if err := network_firewall.ValidateFirewallExists(r.client, ctx, data.FirewallID.ValueInt64()); err != nil {
		resp.Diagnostics.AddError(
			"Invalid Firewall ID",
			err.Error(),
		)
		return
	}

	// Validate that the environment ID exists
	if err := resource_group_environment.ValidateEnvironmentExists(r.client, ctx, data.EnvironmentID.ValueInt64()); err != nil {
		resp.Diagnostics.AddError(
			"Invalid Environment ID",
			err.Error(),
		)
		return
	}
	if err := resource_group_environment.ValidateEnvironmentExistsForFirewall(r.client, ctx, data.EnvironmentID.ValueInt64(), data.FirewallID.ValueInt64()); err != nil {
		resp.Diagnostics.AddError(
			"Invalid Environment ID",
			err.Error(),
		)
		return
	}

	tflog.Debug(ctx, "Creating network zone", map[string]any{
		"name":           data.Name.ValueString(),
		"environment_id": data.EnvironmentID.ValueInt64(),
		"firewall_id":    data.FirewallID.ValueInt64(),
		"zone_type":      data.ZoneType.ValueString(),
		"data_plane":     data.DataPlane.ValueString(),
	})

	// Build the input config
	inputConfig := &NetworkZoneInputConfig{
		Name:          data.Name.ValueString(),
		EnvironmentID: data.EnvironmentID.ValueInt64(),
		FirewallID:    data.FirewallID.ValueInt64(),
		Purpose:       data.Purpose.ValueString(),
		DataPlaneType: data.DataPlane.ValueString(),
		ZoneType:      data.ZoneType.ValueString(),
	}

	// Set optional fields
	if !data.NoOfIPs.IsNull() && !data.NoOfIPs.IsUnknown() {
		noOfIPs := data.NoOfIPs.ValueInt64()
		inputConfig.NoOfIPs = &noOfIPs
	}

	if !data.NoOfV6IPs.IsNull() && !data.NoOfV6IPs.IsUnknown() {
		noOfV6IPs := data.NoOfV6IPs.ValueInt64()
		inputConfig.NoOfV6IPs = &noOfV6IPs
		inputConfig.DualStackMode = "yes"
	} else {
		inputConfig.DualStackMode = "no"
		inputConfig.NoOfV6IPs = nil
	}

	// Build the create request
	createReq := BuildNetworkZoneCreateRequest(inputConfig)

	// Create network zone and wait for completion
	auditLog, err := CreateNetworkZoneAndWait(r.client, ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Network Zone",
			"Could not create network zone: "+err.Error(),
		)
		return
	}

	// Set the ID and audit ID
	if auditLog.ResourceID.String() != "" {
		data.ID = types.StringValue(auditLog.ResourceID.String())
	} else {
		resp.Diagnostics.AddError(
			"Error Creating Network Zone",
			"Could not create network zone: ResourceID is empty",
		)
		return
	}
	data.AuditID = types.StringValue(auditLog.AuditID)
	data.Status = types.StringValue(auditLog.Status)
	data.ZoneType = types.StringValue(data.ZoneType.ValueString())
	data.DataPlane = types.StringValue(data.DataPlane.ValueString())
	data.Purpose = types.StringValue(data.Purpose.ValueString())
	if data.NetworkZoneId.IsUnknown() || data.NetworkZoneId.IsNull() {
		data.NetworkZoneId = types.Int64Null()
	}

	// Read the zone so platform-assigned CIDR (Auto IPAM) and related fields are in state.
	if err := r.refreshZoneState(ctx, &data); err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Network Zone After Create",
			"Network zone was created but could not read CIDR and zone details: "+err.Error(),
		)
		return
	}

	tflog.Info(ctx, "Network Zone created successfully", map[string]any{
		"id":       data.ID.ValueString(),
		"audit_id": data.AuditID.ValueString(),
		"status":   data.Status.ValueString(),
		"cidr":     data.CIDR.ValueString(),
	})

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read refreshes the Terraform state with the latest data from the API.
func (r *NetworkZoneResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data NetworkZoneResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Reading network zone", map[string]any{
		"id":       data.ID.ValueString(),
		"audit_id": data.AuditID.ValueString(),
	})

	if err := r.refreshZoneState(ctx, &data); err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Network Zone",
			"Could not read network zone: "+err.Error(),
		)
		return
	}

	tflog.Info(ctx, "Network zone read successfully", map[string]any{
		"id":     data.ID.ValueString(),
		"status": data.Status.ValueString(),
		"cidr":   data.CIDR.ValueString(),
	})

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// refreshZoneState reads the zone via action-state and updates computed fields (cidr, etc.).
func (r *NetworkZoneResource) refreshZoneState(ctx context.Context, data *NetworkZoneResourceModel) error {
	actionStateBody := map[string]any{
		"resourceId": data.ID.ValueString(),
	}

	actionStateResponse, err := common.UpdateActionState(ctx, r.client, "zone", "read", actionStateBody)
	if err != nil {
		return err
	}

	if len(actionStateResponse.Data) == 0 {
		tflog.Warn(ctx, "Empty zone action-state response; keeping existing state values", map[string]any{
			"id": data.ID.ValueString(),
		})
		return nil
	}

	var responseMap map[string]interface{}
	if err := json.Unmarshal(actionStateResponse.Data, &responseMap); err != nil {
		return fmt.Errorf("could not unmarshal response data: %w; response: %s", err, string(actionStateResponse.Data))
	}

	tflog.Debug(ctx, "Zone action-state response", map[string]any{
		"response_map": responseMap,
	})

	if v, ok := responseMap["name"].(string); ok {
		data.Name = types.StringValue(v)
	}
	if v, ok := responseMap["environment_id"].(float64); ok {
		data.EnvironmentID = types.Int64Value(int64(v))
	}
	if v, ok := responseMap["firewall_id"].(float64); ok {
		data.FirewallID = types.Int64Value(int64(v))
	}
	if v, ok := responseMap["no_of_ips"].(float64); ok {
		data.NoOfIPs = types.Int64Value(int64(v))
	}
	if v, ok := responseMap["purpose"].(string); ok {
		data.Purpose = types.StringValue(v)
	}
	if v, ok := responseMap["data_plane"].(string); ok {
		data.DataPlane = types.StringValue(v)
	}
	if v, ok := responseMap["zone_type"].(string); ok {
		data.ZoneType = types.StringValue(v)
	}
	if responseMap["dual_stack_mode"] != nil {
		if v, ok := responseMap["no_of_v6_ips"].(float64); ok {
			data.NoOfV6IPs = types.Int64Value(int64(v))
		}
		if v, ok := responseMap["ipv6_cidr"].(string); ok {
			data.IPv6CIDR = types.StringValue(v)
		}
	}
	if v, ok := responseMap["zone_ci_master_id"].(float64); ok {
		data.NetworkZoneId = types.Int64Value(int64(v))
	}
	if v, ok := responseMap["cidr"].(string); ok && v != "" {
		data.CIDR = types.StringValue(v)
	}

	// Ensure computed fields are always known (handles import where state is empty)
	if data.Purpose.IsNull() || data.Purpose.IsUnknown() {
		data.Purpose = types.StringValue("IPC")
	}
	if data.DataPlane.IsNull() || data.DataPlane.IsUnknown() {
		data.DataPlane = types.StringValue("Auto IPAM")
	}
	if data.ZoneType.IsNull() || data.ZoneType.IsUnknown() {
		data.ZoneType = types.StringValue("overlay")
	}

	return nil
}

// Update updates the resource.
func (r *NetworkZoneResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan NetworkZoneResourceModel
	var state NetworkZoneResourceModel

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

	// Validate that the firewall ID exists
	if err := network_firewall.ValidateFirewallExists(r.client, ctx, plan.FirewallID.ValueInt64()); err != nil {
		resp.Diagnostics.AddError(
			"Invalid Firewall ID",
			err.Error(),
		)
		return
	}

	// Validate that the environment ID exists
	if err := resource_group_environment.ValidateEnvironmentExists(r.client, ctx, plan.EnvironmentID.ValueInt64()); err != nil {
		resp.Diagnostics.AddError(
			"Invalid Environment ID",
			err.Error(),
		)
		return
	}
	if err := resource_group_environment.ValidateEnvironmentExistsForFirewall(r.client, ctx, plan.EnvironmentID.ValueInt64(), plan.FirewallID.ValueInt64()); err != nil {
		resp.Diagnostics.AddError(
			"Invalid Environment ID",
			err.Error(),
		)
		return
	}
	tflog.Debug(ctx, "Updating network zone", map[string]any{
		"id":       plan.ID.ValueString(),
		"old_name": state.Name.ValueString(),
		"new_name": plan.Name.ValueString(),
	})

	// Resolve computed fields from state if unknown in plan
	if plan.Purpose.IsUnknown() {
		plan.Purpose = state.Purpose
	}
	if plan.DataPlane.IsUnknown() {
		plan.DataPlane = state.DataPlane
	}
	if plan.ZoneType.IsUnknown() {
		plan.ZoneType = state.ZoneType
	}

	nameChanged := plan.Name.ValueString() != state.Name.ValueString()

	if nameChanged {
		tflog.Debug(ctx, "Updating network zone name", map[string]any{
			"zone_id":  plan.ID.ValueString(),
			"old_name": state.Name.ValueString(),
			"new_name": plan.Name.ValueString(),
		})

		_, err := UpdateNetworkZoneAndWait(r.client, ctx, plan.ID.ValueString(), plan.Name.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Updating Network Zone",
				"Could not update network zone name: "+err.Error(),
			)
			return
		}

		// plan.AuditID = types.StringValue(auditLog.AuditID)
		// plan.Status = types.StringValue(auditLog.Status)

		tflog.Info(ctx, "Network zone name updated successfully", map[string]any{
			"id":       plan.ID.ValueString(),
			"name":     plan.Name.ValueString(),
			"audit_id": plan.AuditID.ValueString(),
			"status":   plan.Status.ValueString(),
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
	if plan.NetworkZoneId.IsUnknown() || plan.NetworkZoneId.IsNull() {
		plan.NetworkZoneId = state.NetworkZoneId
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete deletes the network zone resource.
func (r *NetworkZoneResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data NetworkZoneResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting network zone", map[string]any{
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

	// Validate that the environment ID exists
	if err := resource_group_environment.ValidateEnvironmentExists(r.client, ctx, data.EnvironmentID.ValueInt64()); err != nil {
		resp.Diagnostics.AddError(
			"Invalid Environment ID",
			err.Error(),
		)
		return
	}
	if err := resource_group_environment.ValidateEnvironmentExistsForFirewall(r.client, ctx, data.EnvironmentID.ValueInt64(), data.FirewallID.ValueInt64()); err != nil {
		resp.Diagnostics.AddError(
			"Invalid Environment ID",
			err.Error(),
		)
		return
	}
	// Delete network zone and wait for completion
	_, err := DeleteNetworkZoneAndWait(r.client, ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Network Zone",
			"Could not delete network zone: "+err.Error(),
		)
		return
	}

	tflog.Info(ctx, "Network Zone deleted successfully", map[string]any{
		"id": data.ID.ValueString(),
	})
}

// ImportState imports an existing network zone into Terraform state.
func (r *NetworkZoneResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import using the id
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
