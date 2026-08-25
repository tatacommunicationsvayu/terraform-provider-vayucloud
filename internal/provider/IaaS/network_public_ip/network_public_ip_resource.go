// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package network_public_ip

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/client"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/network_lb"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/network_zone"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/virtualmachine"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/network_firewall"
)

var _ resource.Resource = &NetworkPublicIPResource{}
var _ resource.ResourceWithModifyPlan = &NetworkPublicIPResource{}

var privateIPv4Regex = regexp.MustCompile(`^(25[0-5]|2[0-4][0-9]|1?[0-9]{1,2})(\.(25[0-5]|2[0-4][0-9]|1?[0-9]{1,2})){3}$`)

// NewNetworkPublicIPResource creates the network public IP association resource.
func NewNetworkPublicIPResource() resource.Resource {
	return &NetworkPublicIPResource{}
}

// NetworkPublicIPResource associates a public IP with a platform resource via the network operations API.
type NetworkPublicIPResource struct {
	client *client.Client
}

// NetworkPublicIPResourceModel is Terraform state for one association.
type NetworkPublicIPResourceModel struct {
	ID types.String `tfsdk:"id"`

	ResourceType         types.String `tfsdk:"resource_type"`
	ResourceID           types.Int64  `tfsdk:"resource_id"`
	PrivateIP            types.String `tfsdk:"private_ip"`
	PublicIP             types.String `tfsdk:"public_ip"`
	PublicIPPricingModel types.String `tfsdk:"public_ip_pricing_model"`
	RetainOnDissociate   types.Bool   `tfsdk:"retain_on_dissociate"`

	AuditID types.String `tfsdk:"audit_id"`
	Status  types.String `tfsdk:"status"`
}

func (r *NetworkPublicIPResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network_public_ip"
}

func (r *NetworkPublicIPResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Associates a public IP with a zone, virtual machine, firewall, bare metal host, or load balancer.",
		MarkdownDescription: "Calls the network operations associate API, waits for audit completion, then updates action state (`module=publicIp`, `action=create`). Destroy sends dissociate using **current Terraform state** for `retain_on_dissociate` (change this attribute only via apply so state matches before destroy). The association itself cannot be updated in the cloud API; only `retain_on_dissociate` may be updated in Terraform state in place.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Stable identifier: `{resource_type}|{resource_id}|{private_ip}`.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"resource_type": schema.StringAttribute{
				Description: "Target kind in the associate URL path: zone, virtualmachine, firewall, baremetal, or loadbalancer.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf(
						"zone",
						"virtualmachine",
						"firewall",
						"baremetal",
						"loadbalancer",
						"vcs",
					),
				},
			},
			"resource_id": schema.Int64Attribute{
				Description: "Platform resource ID the public IP is associated with (path segment after resource_type).",
				Required:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"private_ip": schema.StringAttribute{
				Description: "Private IP address to NAT with the public IP.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(privateIPv4Regex, "Must be a valid IPv4 address"),
				},
			},
			"public_ip": schema.StringAttribute{
				Description: "Public IP address from action-state read (`module=publicIp`, `action=read`) for this `resource_id`, `resource_type`, and `private_ip`.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"public_ip_pricing_model": schema.StringAttribute{
				Description: "Public IP pricing model for associate: daily or monthly (case-insensitive; sent to the API as Daily/Monthly).",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("daily"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					stringvalidator.OneOfCaseInsensitive("daily", "monthly", "reserved_1", "reserved_2", "reserved_3"),
				},
			},
			"retain_on_dissociate": schema.BoolAttribute{
				Description: "Whether to retain the public IP when dissociating (associate and dissociate request bodies).",
				Optional: true,
				Computed: true,
				Default: booldefault.StaticBool(true),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
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

func (r *NetworkPublicIPResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	cl, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T.", req.ProviderData),
		)
		return
	}
	r.client = cl
}

func validatePublicIPTargetResource(ctx context.Context, c *client.Client, resourceType string, resourceID int64) error {
	switch resourceType {
	case "zone":
		return network_zone.ValidateNetworkZoneExists(c, ctx, resourceID)
	case "virtualmachine":
		return virtualmachine.ValidateInstanceExists(c, ctx, resourceID)
	case "loadbalancer":
		return network_lb.ValidateLoadBalancerExists(c, ctx, resourceID)
	case "vcs":
		return network_firewall.ValidateFirewallExists(c, ctx, resourceID)
	default:
		return nil
	}
}

func (r *NetworkPublicIPResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if r.client == nil {
		return
	}
	if req.Plan.Raw.IsNull() {
		return
	}

	var plan NetworkPublicIPResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.ResourceType.IsUnknown() || plan.ResourceID.IsUnknown() {
		return
	}
	if plan.ResourceType.IsNull() || plan.ResourceID.IsNull() {
		return
	}

	if err := validatePublicIPTargetResource(ctx, r.client, plan.ResourceType.ValueString(), plan.ResourceID.ValueInt64()); err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("resource_id"),
			"Invalid resource_id for resource_type",
			err.Error(),
		)
	}
}

func (r *NetworkPublicIPResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data NetworkPublicIPResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError("Client not configured", "Expected configured API client")
		return
	}
	if data.ResourceType.ValueString() == "vcs" {
		data.ResourceType = types.StringValue("firewall")
	}

	if err := validatePublicIPTargetResource(ctx, r.client, data.ResourceType.ValueString(), data.ResourceID.ValueInt64()); err != nil {
		resp.Diagnostics.AddError("Invalid resource_id for resource_type", err.Error())
		return
	}
	pubIP, readErr := ReadPublicIPAssociation(
		r.client,
		ctx,
		data.ResourceID.ValueInt64(),
		data.ResourceType.ValueString(),
		data.PrivateIP.ValueString(),
	)
	if readErr != nil {
		tflog.Debug(ctx, "No public ip associated with this resource. Proceeding with public ip action", map[string]any{
			"resource_id": data.ResourceID.ValueInt64(),
			"resource_type": data.ResourceType.ValueString(),
			"private_ip": data.PrivateIP.ValueString(),
		})
	} else if pubIP != "NA" {
		resp.Diagnostics.AddError(pubIP+ " is already associated with this resource "+data.ResourceType.ValueString()+" "+strconv.FormatInt(data.ResourceID.ValueInt64(), 10), "Public ip is already associated with this resource")
		return
	}

	pricingAPI, err := MapPricingModelToAPI(data.PublicIPPricingModel.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid public_ip_pricing_model", err.Error())
		return
	}

	body := &associatePublicIPRequest{
		PrivateIP:            data.PrivateIP.ValueString(),
		PublicIPPricingModel: pricingAPI,
		RetainOnDissociate:   data.RetainOnDissociate.ValueBool(),
		AllowFetchOrProvisionPublicIp: true,
	}

	resIDStr := strconv.FormatInt(data.ResourceID.ValueInt64(), 10)
	auditLog, err := AssociatePublicIPAndWait(
		r.client,
		ctx,
		data.ResourceType.ValueString(),
		resIDStr,
		body,
	)
	if err != nil {
		resp.Diagnostics.AddError("Error associating public IP", err.Error())
		return
	}

	data.ID = types.StringValue(resIDStr)
	data.AuditID = types.StringValue(auditLog.AuditID)
	data.Status = types.StringValue(auditLog.Status)
	if data.RetainOnDissociate.IsUnknown() || data.RetainOnDissociate.IsNull() {
		data.RetainOnDissociate = types.BoolValue(true)
	}
	if data.PublicIPPricingModel.IsUnknown() || data.PublicIPPricingModel.IsNull() {
		data.PublicIPPricingModel = types.StringNull()
	}
	if data.PrivateIP.IsUnknown() || data.PrivateIP.IsNull() {
		data.PrivateIP = types.StringNull()
	}
	if data.ResourceType.IsUnknown() || data.ResourceType.IsNull() {
		data.ResourceType = types.StringNull()
	}
	if data.ResourceID.IsUnknown() || data.ResourceID.IsNull() {
		data.ResourceID = types.Int64Null()
	}

	pub, readErr := ReadPublicIPAssociation(
		r.client,
		ctx,
		data.ResourceID.ValueInt64(),
		data.ResourceType.ValueString(),
		data.PrivateIP.ValueString(),
	)
	if readErr != nil {
		resp.Diagnostics.AddWarning(
			"Could not read public IP after associate",
			readErr.Error(),
		)
		data.PublicIP = types.StringNull()
	} else {
		data.PublicIP = types.StringValue(pub)
	}

	tflog.Info(ctx, "Public IP associated", map[string]any{
		"id":       data.ID.ValueString(),
		"audit_id": auditLog.AuditID,
	})
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *NetworkPublicIPResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data NetworkPublicIPResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError("Client not configured", "Expected configured API client")
		return
	}

	pub, err := ReadPublicIPAssociation(
		r.client,
		ctx,
		data.ResourceID.ValueInt64(),
		data.ResourceType.ValueString(),
		data.PrivateIP.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Error reading public IP (action-state)", err.Error())
		return
	}
	data.PublicIP = types.StringValue(pub)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *NetworkPublicIPResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state NetworkPublicIPResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Immutable identity — replacement uses destroy+create via RequiresReplace on those attributes.
	if plan.ResourceType.ValueString() != state.ResourceType.ValueString() ||
		plan.ResourceID.ValueInt64() != state.ResourceID.ValueInt64() ||
		plan.PrivateIP.ValueString() != state.PrivateIP.ValueString() {
		resp.Diagnostics.AddError(
			"Unexpected resource identity change",
			"resource_type, resource_id, and private_ip cannot change without replacement; use terraform apply with forced replacement if needed.",
		)
		return
	}

	// Pricing model is fixed at associate time in the API.
	normalizePricing := func(s types.String) string {
		if s.IsNull() || s.IsUnknown() {
			return ""
		}
		return strings.TrimSpace(s.ValueString())
	}
	planPM := normalizePricing(plan.PublicIPPricingModel)
	statePM := normalizePricing(state.PublicIPPricingModel)
	if planPM != "" && statePM != "" && !strings.EqualFold(planPM, statePM) {
		resp.Diagnostics.AddError(
			"Updates not supported",
			"public_ip_pricing_model cannot be changed in place; replace this resource.",
		)
		return
	}

	// Only retain_on_dissociate may change without API call; it affects the dissociate payload on destroy.
	out := state
	if !plan.RetainOnDissociate.IsUnknown() {
		out.RetainOnDissociate = plan.RetainOnDissociate
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &out)...)
}

func (r *NetworkPublicIPResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data NetworkPublicIPResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError("Client not configured", "Expected configured API client")
		return
	}

	resIDStr := strconv.FormatInt(data.ResourceID.ValueInt64(), 10)
	publicIP := ""
	if !data.PublicIP.IsNull() && !data.PublicIP.IsUnknown() {
		publicIP = strings.TrimSpace(data.PublicIP.ValueString())
	}
	if publicIP == "" {
		resp.Diagnostics.AddError(
			"Cannot dissociate public IP",
			"The public_ip attribute is empty in Terraform state. Run terraform refresh so the provider can read the associated address from the platform, then try destroy again.",
		)
		return
	}

	_, err := DissociatePublicIPAndWait(
		r.client,
		ctx,
		data.ResourceType.ValueString(),
		resIDStr,
		data.PrivateIP.ValueString(),
		publicIP,
		data.RetainOnDissociate.ValueBool(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Error dissociating public IP", err.Error())
		return
	}

	tflog.Info(ctx, "Public IP dissociated", map[string]any{"id": data.ID.ValueString()})
}
