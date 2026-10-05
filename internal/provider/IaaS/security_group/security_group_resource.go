// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package security_group

import (
	"context"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/client"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/network_firewall"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/network_zone"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/virtualmachine"
)

var _ resource.Resource = &SecurityGroupResource{}
var _ resource.ResourceWithValidateConfig = &SecurityGroupResource{}
var _ resource.ResourceWithModifyPlan = &SecurityGroupResource{}
var _ resource.ResourceWithImportState = &SecurityGroupResource{}

// NewSecurityGroupResource creates the security group resource.
func NewSecurityGroupResource() resource.Resource {
	return &SecurityGroupResource{}
}

// SecurityGroupResource manages a VayuCloud security group and its rules.
type SecurityGroupResource struct {
	client *client.Client
}

// SecurityGroupRuleModel is one rule block in Terraform state.
type SecurityGroupRuleModel struct {
	ID                  types.String `tfsdk:"id"`
	Protocol            types.String `tfsdk:"protocol"`
	EtherType           types.String `tfsdk:"ether_type"`
	Direction           types.String `tfsdk:"direction"`
	PortRange           types.Int64  `tfsdk:"port_range"`
	RemoteIPPrefix      types.List   `tfsdk:"remote_ip_prefix"`
	RemoteSecurityGroup types.String `tfsdk:"remote_security_group"`
}

// SecurityGroupResourceModel is the Terraform model.
type SecurityGroupResourceModel struct {
	FirewallID           types.Int64             `tfsdk:"firewall_id"`
	ID                   types.String            `tfsdk:"id"`
	Name                 types.String            `tfsdk:"name"`
	Description          types.String            `tfsdk:"description"`
	Type                 types.String            `tfsdk:"type"`
	ResourceID           types.Int64             `tfsdk:"resource_id"`
	ManagedSecurityGroup types.Bool              `tfsdk:"managed_security_group"`
	Rules                []SecurityGroupRuleModel `tfsdk:"rule"`
	AuditID              types.String            `tfsdk:"audit_id"`
	Status               types.String            `tfsdk:"status"`
}

var (
	sgNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
)

func (r *SecurityGroupResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_security_group"
}

func (r *SecurityGroupResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a VayuCloud security group and its rules on a firewall.",
		MarkdownDescription: "Manages a security group on a firewall and optionally creates rules under it.\n\n" +
			"Choose **exactly one** creation mode:\n\n" +
			"1. **Create group** — set `name` (and optional `description`) to create a new security group.\n" +
			"2. **Adopt by ID** — set `id` to an existing security group UUID (group create is skipped).\n" +
			"3. **Adopt by type** — set `type` (`zone`, `virtualmachine`, or `firewall`) and `resource_id` to locate `SG_Zone_<id>`, `SG_VM_<id>`, or `SG_TR_<id>`.\n\n" +
			"Each `rule` block creates a security group rule. Provide either `remote_ip_prefix` or `remote_security_group` " +
			"(optional when `protocol` is `icmp`).",
		Attributes: map[string]schema.Attribute{
			"firewall_id": schema.Int64Attribute{
				Description: "Firewall CI ID that owns the security group.",
				Required:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"id": schema.StringAttribute{
				Description: "Security group UUID. Optional to adopt an existing group; otherwise computed after create/lookup.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Security group name (alphanumeric and underscore only). Used for create mode.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(sgNameRegex, "name may only contain letters, digits, and underscore (_)"),
				},
			},
			"description": schema.StringAttribute{
				Description: "Security group description (create mode).",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"type": schema.StringAttribute{
				Description: "Resource type used to frame a system SG name: zone, virtualmachine, or firewall.",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOfCaseInsensitive("zone", "virtualmachine", "firewall"),
				},
			},
			"resource_id": schema.Int64Attribute{
				Description: "Platform resource ID used with `type` to frame and look up the security group name.",
				Optional:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"managed_security_group": schema.BoolAttribute{
				Description: "True when this resource created the security group (destroy will delete the group). False when an existing group was adopted.",
				Computed:    true,
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
		Blocks: map[string]schema.Block{
			// Set (not list): removing/reordering a rule must not shift other rules by
			// index (NestingList would show false updates on surviving rules).
			"rule": schema.SetNestedBlock{
				Description: "Security group rules to create under the security group. " +
					"Order does not matter; rules are matched by their attributes.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "Security group rule UUID (known after apply).",
							Computed:    true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"protocol": schema.StringAttribute{
							Description: "Protocol: tcp, udp, or icmp.",
							Required:    true,
							Validators: []validator.String{
								stringvalidator.OneOfCaseInsensitive("tcp", "udp", "icmp"),
							},
						},
						"ether_type": schema.StringAttribute{
							Description: "Ether type: ipv4 or ipv6.",
							Required:    true,
							Validators: []validator.String{
								stringvalidator.OneOfCaseInsensitive("ipv4", "ipv6"),
							},
						},
						"direction": schema.StringAttribute{
							Description: "Direction: ingress or egress.",
							Required:    true,
							Validators: []validator.String{
								stringvalidator.OneOfCaseInsensitive("ingress", "egress"),
							},
						},
						"port_range": schema.Int64Attribute{
							Description: "Single port number (1–65535). Sent to the API as a one-element port_range_list. Omit for all ports.",
							Optional:    true,
							Validators: []validator.Int64{
								int64validator.Between(1, 65535),
							},
						},
						"remote_ip_prefix": schema.ListAttribute{
							Description: "Remote CIDR list. Mutually exclusive with remote_security_group.",
							Optional:    true,
							ElementType: types.StringType,
						},
						"remote_security_group": schema.StringAttribute{
							Description: "Remote security group name. Mutually exclusive with remote_ip_prefix.",
							Optional:    true,
						},
					},
				},
			},
		},
	}
}

func (r *SecurityGroupResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config SecurityGroupResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	hasName := !config.Name.IsNull() && !config.Name.IsUnknown() && config.Name.ValueString() != ""
	hasID := !config.ID.IsNull() && !config.ID.IsUnknown() && config.ID.ValueString() != ""
	hasType := !config.Type.IsNull() && !config.Type.IsUnknown() && config.Type.ValueString() != ""
	hasResourceID := !config.ResourceID.IsNull() && !config.ResourceID.IsUnknown()

	modes := 0
	if hasName {
		modes++
	}
	if hasID {
		modes++
	}
	if hasType {
		modes++
	}
	if modes > 1 {
		resp.Diagnostics.AddError(
			"Invalid Security Group Configuration",
			"Set exactly one of: `name` (create), `id` (adopt existing), or `type` + `resource_id` (lookup system SG).",
		)
		return
	}
	if modes == 0 && !config.Name.IsUnknown() && !config.ID.IsUnknown() && !config.Type.IsUnknown() {
		resp.Diagnostics.AddError(
			"Invalid Security Group Configuration",
			"Set exactly one of: `name` (create), `id` (adopt existing), or `type` + `resource_id` (lookup system SG).",
		)
		return
	}

	if hasType && !hasResourceID {
		resp.Diagnostics.AddAttributeError(
			path.Root("resource_id"),
			"Missing resource_id",
			"`resource_id` is required when `type` is set.",
		)
	}
	if hasResourceID && !hasType {
		resp.Diagnostics.AddAttributeError(
			path.Root("type"),
			"Missing type",
			"`type` is required when `resource_id` is set.",
		)
	}

	for i, rule := range config.Rules {
		resp.Diagnostics.Append(validateRuleConfig(rule, path.Root("rule").AtListIndex(i))...)
	}
}

func validateRuleConfig(rule SecurityGroupRuleModel, p path.Path) diag.Diagnostics {
	var diags diag.Diagnostics

	hasCIDR := !rule.RemoteIPPrefix.IsNull() && !rule.RemoteIPPrefix.IsUnknown() && len(rule.RemoteIPPrefix.Elements()) > 0
	hasRemoteSG := !rule.RemoteSecurityGroup.IsNull() && !rule.RemoteSecurityGroup.IsUnknown() && rule.RemoteSecurityGroup.ValueString() != ""
	protocol := strings.ToLower(rule.Protocol.ValueString())

	if hasCIDR && hasRemoteSG {
		diags.AddAttributeError(p, "Invalid rule remotes",
			"Set either remote_ip_prefix or remote_security_group, not both.")
	}
	if protocol != "icmp" && !hasCIDR && !hasRemoteSG && !rule.Protocol.IsUnknown() {
		diags.AddAttributeError(p, "Missing rule remote",
			"remote_ip_prefix or remote_security_group is required unless protocol is icmp.")
	}

	if hasCIDR {
		var prefixes []string
		diags.Append(rule.RemoteIPPrefix.ElementsAs(context.Background(), &prefixes, false)...)
		for _, cidr := range prefixes {
			if _, _, err := net.ParseCIDR(cidr); err != nil {
				diags.AddAttributeError(p.AtName("remote_ip_prefix"), "Invalid CIDR",
					fmt.Sprintf("%q is not a valid CIDR: %s", cidr, err.Error()))
			}
		}
	}

	return diags
}

func (r *SecurityGroupResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// ModifyPlan validates firewall (and related IDs) during terraform plan.
// Destroy plans use prior state when the plan value is null.
func (r *SecurityGroupResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// Align planned rule ids with state by fingerprint (not list index).
	// Surviving rules keep their UUIDs; new/changed fingerprints get unknown ids
	// so Update can attach the correct UUID without "inconsistent result after apply".
	if !req.Plan.Raw.IsNull() && !req.State.Raw.IsNull() {
		var plan, state SecurityGroupResourceModel
		resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
		resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
		if resp.Diagnostics.HasError() {
			return
		}
		if alignPlanRuleIDsByFingerprint(&plan, state.Rules) {
			resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
			if resp.Diagnostics.HasError() {
				return
			}
		}
	}

	if r.client == nil {
		return
	}

	var plan SecurityGroupResourceModel
	if req.Plan.Raw.IsNull() {
		resp.Diagnostics.Append(req.State.Get(ctx, &plan)...)
	} else {
		resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
		var fw types.Int64
		resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("firewall_id"), &fw)...)
		if resp.Diagnostics.HasError() {
			return
		}
		if fw.IsUnknown() {
			return
		}
	}
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.FirewallID.IsUnknown() || plan.FirewallID.IsNull() {
		return
	}

	tflog.Debug(ctx, "Planning Security Group", map[string]any{
		"firewall_id": plan.FirewallID.ValueInt64(),
		"name":        plan.Name.ValueString(),
		"id":          plan.ID.ValueString(),
		"type":        plan.Type.ValueString(),
		"resource_id": plan.ResourceID.ValueInt64(),
	})

	if err := network_firewall.ValidateFirewallExists(r.client, ctx, plan.FirewallID.ValueInt64()); err != nil {
		resp.Diagnostics.AddError("Invalid Firewall ID", err.Error())
		return
	}

	// Mode: adopt by existing security group id
	if !plan.ID.IsNull() && !plan.ID.IsUnknown() && plan.ID.ValueString() != "" {
		if err := ValidateSecurityGroupExists(r.client, ctx, plan.FirewallID.ValueInt64(), plan.ID.ValueString()); err != nil {
			resp.Diagnostics.AddError("Invalid Security Group ID", err.Error())
			return
		}
	}

	// Mode: lookup system SG by type + resource_id
	if !plan.Type.IsNull() && !plan.Type.IsUnknown() && plan.Type.ValueString() != "" {
		if plan.ResourceID.IsUnknown() || plan.ResourceID.IsNull() {
			return
		}
		if err := validateTypedResource(r.client, ctx, plan.Type.ValueString(), plan.ResourceID.ValueInt64()); err != nil {
			resp.Diagnostics.AddError("Invalid Resource ID", err.Error())
			return
		}
		framedName, err := FrameSecurityGroupName(plan.Type.ValueString(), plan.ResourceID.ValueInt64())
		if err != nil {
			resp.Diagnostics.AddError("Invalid Type", err.Error())
			return
		}
		if _, err := FindSecurityGroupIDByName(r.client, ctx, plan.FirewallID.ValueInt64(), framedName); err != nil {
			resp.Diagnostics.AddError("Security Group Not Found", err.Error())
			return
		}
	}

	for i, rule := range plan.Rules {
		if rule.Protocol.IsUnknown() || rule.EtherType.IsUnknown() || rule.Direction.IsUnknown() {
			continue
		}
		resp.Diagnostics.Append(validateRuleConfig(rule, path.Root("rule").AtListIndex(i))...)
	}
}

func (r *SecurityGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data SecurityGroupResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	firewallID := data.FirewallID.ValueInt64()
	if err := network_firewall.ValidateFirewallExists(r.client, ctx, firewallID); err != nil {
		resp.Diagnostics.AddError("Invalid Firewall ID", err.Error())
		return
	}

	sgID, managed, name, description, auditID, status, diags := r.resolveSecurityGroup(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.ID = types.StringValue(sgID)
	data.ManagedSecurityGroup = types.BoolValue(managed)
	if name != "" {
		data.Name = types.StringValue(name)
	} else if data.Name.IsUnknown() {
		data.Name = types.StringNull()
	}
	if description != "" {
		data.Description = types.StringValue(description)
	} else if data.Description.IsUnknown() {
		data.Description = types.StringNull()
	}
	if auditID != "" {
		data.AuditID = types.StringValue(auditID)
	}
	if status != "" {
		data.Status = types.StringValue(status)
	}

	createdRules, lastAudit, lastStatus, ruleDiags := r.createRules(ctx, firewallID, sgID, data.Rules)
	resp.Diagnostics.Append(ruleDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Rules = createdRules
	if lastAudit != "" {
		data.AuditID = types.StringValue(lastAudit)
	}
	if lastStatus != "" {
		data.Status = types.StringValue(lastStatus)
	}
	if data.AuditID.IsUnknown() {
		data.AuditID = types.StringNull()
	}
	if data.Status.IsUnknown() {
		data.Status = types.StringNull()
	}
	if data.Name.IsUnknown() {
		data.Name = types.StringNull()
	}
	if data.Description.IsUnknown() {
		data.Description = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SecurityGroupResource) resolveSecurityGroup(
	ctx context.Context,
	data *SecurityGroupResourceModel,
) (sgID string, managed bool, name, description, auditID, status string, diags diag.Diagnostics) {
	firewallID := data.FirewallID.ValueInt64()

	hasName := !data.Name.IsNull() && !data.Name.IsUnknown() && data.Name.ValueString() != ""
	hasID := !data.ID.IsNull() && !data.ID.IsUnknown() && data.ID.ValueString() != ""
	hasType := !data.Type.IsNull() && !data.Type.IsUnknown() && data.Type.ValueString() != ""

	switch {
	case hasType:
		resourceID := data.ResourceID.ValueInt64()
		resourceType := data.Type.ValueString()
		if err := validateTypedResource(r.client, ctx, resourceType, resourceID); err != nil {
			diags.AddError("Invalid Resource ID", err.Error())
			return
		}
		framedName, err := FrameSecurityGroupName(resourceType, resourceID)
		if err != nil {
			diags.AddError("Invalid Type", err.Error())
			return
		}
		foundID, err := FindSecurityGroupIDByName(r.client, ctx, firewallID, framedName)
		if err != nil {
			diags.AddError("Security Group Not Found", err.Error())
			return
		}
		return foundID, false, framedName, "", "", "", diags

	case hasID:
		candidate := data.ID.ValueString()
		if err := ValidateSecurityGroupExists(r.client, ctx, firewallID, candidate); err != nil {
			diags.AddError("Invalid Security Group ID", err.Error())
			return
		}
		return candidate, true, "", "", "", "", diags

	case hasName:
		desc := ""
		if !data.Description.IsNull() && !data.Description.IsUnknown() {
			desc = data.Description.ValueString()
		}
		auditLog, err := CreateSecurityGroupAndWait(r.client, ctx, firewallID, &SecurityGroupCreateRequest{
			Name:        data.Name.ValueString(),
			Description: desc,
		})
		if err != nil {
			diags.AddError("Security Group Create Failed", err.Error())
			return
		}
		id, err := auditLog.ProvisioningResourceID()
		if err != nil {
			diags.AddError("Security Group Create Failed", fmt.Sprintf("could not read security group id from audit provisioningDetails: %s", err.Error()))
			return
		}
		return id, true, data.Name.ValueString(), desc, auditLog.AuditID, auditLog.Status, diags

	default:
		diags.AddError(
			"Invalid Security Group Configuration",
			"Set exactly one of: `name` (create), `id` (adopt existing), or `type` + `resource_id` (lookup system SG).",
		)
		return
	}
}

func validateTypedResource(c *client.Client, ctx context.Context, resourceType string, resourceID int64) error {
	switch strings.ToLower(resourceType) {
	case "zone":
		return network_zone.ValidateNetworkZoneExists(c, ctx, resourceID)
	case "virtualmachine":
		return virtualmachine.ValidateInstanceExists(c, ctx, resourceID)
	case "firewall":
		return network_firewall.ValidateFirewallExists(c, ctx, resourceID)
	default:
		return fmt.Errorf("unsupported type %q", resourceType)
	}
}

func (r *SecurityGroupResource) createRules(
	ctx context.Context,
	firewallID int64,
	sgID string,
	rules []SecurityGroupRuleModel,
) ([]SecurityGroupRuleModel, string, string, diag.Diagnostics) {
	var diags diag.Diagnostics
	out := make([]SecurityGroupRuleModel, 0, len(rules))
	var lastAudit, lastStatus string

	for i, rule := range rules {
		reqBody, buildDiags := buildRuleCreateRequest(ctx, rule)
		diags.Append(buildDiags...)
		if diags.HasError() {
			return nil, "", "", diags
		}

		auditLog, err := CreateSecurityGroupRuleAndWait(r.client, ctx, firewallID, sgID, reqBody)
		if err != nil {
			diags.AddError("Security Group Rule Create Failed",
				fmt.Sprintf("rule[%d]: %s", i, err.Error()))
			return nil, "", "", diags
		}
		ruleID, err := auditLog.ProvisioningResourceID()
		if err != nil {
			diags.AddError("Security Group Rule Create Failed",
				fmt.Sprintf("rule[%d]: could not read rule id from audit provisioningDetails: %s", i, err.Error()))
			return nil, "", "", diags
		}

		rule.ID = types.StringValue(ruleID)
		out = append(out, rule)
		lastAudit = auditLog.AuditID
		lastStatus = auditLog.Status
	}

	return out, lastAudit, lastStatus, diags
}

func buildRuleCreateRequest(ctx context.Context, rule SecurityGroupRuleModel) (*SecurityGroupRuleCreateRequest, diag.Diagnostics) {
	var diags diag.Diagnostics

	protocol := strings.ToLower(rule.Protocol.ValueString())
	direction := strings.ToLower(rule.Direction.ValueString())
	etherType := normalizeEtherType(rule.EtherType.ValueString())

	req := &SecurityGroupRuleCreateRequest{
		Protocol:  protocol,
		Direction: direction,
		EtherType: etherType,
		PortType:  "allPort",
	}

	if !rule.PortRange.IsNull() && !rule.PortRange.IsUnknown() {
		req.PortType = "customPort"
		req.PortRangeList = []string{fmt.Sprintf("%d", rule.PortRange.ValueInt64())}
	}

	hasCIDR := !rule.RemoteIPPrefix.IsNull() && !rule.RemoteIPPrefix.IsUnknown() && len(rule.RemoteIPPrefix.Elements()) > 0
	hasRemoteSG := !rule.RemoteSecurityGroup.IsNull() && !rule.RemoteSecurityGroup.IsUnknown() && rule.RemoteSecurityGroup.ValueString() != ""

	switch {
	case hasCIDR:
		var prefixes []string
		diags.Append(rule.RemoteIPPrefix.ElementsAs(ctx, &prefixes, false)...)
		if diags.HasError() {
			return nil, diags
		}
		req.RemoteType = "cidr"
		req.RemoteIPPrefixList = prefixes
	case hasRemoteSG:
		req.RemoteType = "securityGroup"
		req.RemoteSecurityGroup = rule.RemoteSecurityGroup.ValueString()
	}

	return req, diags
}

func normalizeEtherType(v string) string {
	switch strings.ToLower(v) {
	case "ipv6":
		return "IPv6"
	default:
		return "IPv4"
	}
}

func (r *SecurityGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data SecurityGroupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	firewallID := data.FirewallID.ValueInt64()
	sgID := data.ID.ValueString()
	if sgID == "" {
		resp.State.RemoveResource(ctx)
		return
	}

	sg, err := GetSecurityGroupWithRules(r.client, ctx, firewallID, sgID)
	if err != nil {
		tflog.Warn(ctx, "Security group no longer exists; removing from state", map[string]any{
			"firewall_id": firewallID,
			"sg_id":       sgID,
			"error":       err.Error(),
		})
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(r.applySecurityGroupWithRules(ctx, &data, sg)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// ImportState imports an existing security group using `{firewall_id}/{sg_id}`.
//
// Usage: terraform import vayucloud_security_group.example 413964/caccc1cf-a4b9-469a-802a-e7426f851066
func (r *SecurityGroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			"Expected format: `{firewall_id}/{security_group_id}` "+
				"(e.g. `413964/caccc1cf-a4b9-469a-802a-e7426f851066`).",
		)
		return
	}

	firewallID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", fmt.Sprintf("firewall_id must be an integer: %s", err.Error()))
		return
	}
	sgID := parts[1]

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("firewall_id"), firewallID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), sgID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("managed_security_group"), false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var data SecurityGroupResourceModel
	resp.Diagnostics.Append(resp.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	sg, err := GetSecurityGroupWithRules(r.client, ctx, firewallID, sgID)
	if err != nil {
		resp.Diagnostics.AddError("Import Failed", err.Error())
		return
	}

	resp.Diagnostics.Append(r.applySecurityGroupWithRules(ctx, &data, sg)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SecurityGroupResource) applySecurityGroupWithRules(ctx context.Context, data *SecurityGroupResourceModel, sg *SecurityGroupWithRules) diag.Diagnostics {
	var diags diag.Diagnostics

	if sg.ID != "" {
		data.ID = types.StringValue(sg.ID)
	}
	data.Name = types.StringValue(sg.Name)
	if sg.Description != "" {
		data.Description = types.StringValue(sg.Description)
	} else {
		data.Description = types.StringNull()
	}

	remoteNameByID := map[string]string{}
	needsRemoteLookup := false
	for _, rule := range sg.Rules {
		if rule.RemoteGroupID != nil && *rule.RemoteGroupID != "" {
			needsRemoteLookup = true
			break
		}
	}
	if needsRemoteLookup {
		items, err := ListSecurityGroups(r.client, ctx, data.FirewallID.ValueInt64())
		if err != nil {
			tflog.Warn(ctx, "Could not list security groups to resolve remote_security_group names", map[string]any{
				"error": err.Error(),
			})
		} else {
			for _, item := range items {
				remoteNameByID[item.ID] = item.Name
			}
		}
	}

	mappedAPIRules := make([]SecurityGroupRuleModel, 0, len(sg.Rules))
	for _, apiRule := range sg.Rules {
		rule, ruleDiags := mapAPIRuleToModel(ctx, apiRule, remoteNameByID)
		diags.Append(ruleDiags...)
		if diags.HasError() {
			return diags
		}
		mappedAPIRules = append(mappedAPIRules, rule)
	}

	// Keep Terraform state rule order; match API rules by id (then fingerprint).
	// Replacing state with API order would shuffle ids across rule blocks and cause false diffs.
	data.Rules = mergeRulesPreservingStateOrder(data.Rules, mappedAPIRules)

	if data.ManagedSecurityGroup.IsNull() || data.ManagedSecurityGroup.IsUnknown() {
		data.ManagedSecurityGroup = types.BoolValue(false)
	}
	if data.AuditID.IsUnknown() {
		data.AuditID = types.StringNull()
	}
	if data.Status.IsUnknown() {
		data.Status = types.StringNull()
	}

	return diags
}

// mergeRulesPreservingStateOrder refreshes rule attributes from the API while keeping
// the existing state block order. Unmatched API rules are appended (import / remote adds).
func mergeRulesPreservingStateOrder(stateRules, apiRules []SecurityGroupRuleModel) []SecurityGroupRuleModel {
	if len(stateRules) == 0 {
		return apiRules
	}

	type apiSlot struct {
		rule SecurityGroupRuleModel
		used bool
	}
	slots := make([]apiSlot, len(apiRules))
	apiByID := make(map[string]int, len(apiRules))
	for i, r := range apiRules {
		slots[i] = apiSlot{rule: r}
		if id := r.ID.ValueString(); id != "" {
			apiByID[id] = i
		}
	}

	takeByFingerprint := func(fp string) (SecurityGroupRuleModel, bool) {
		for i := range slots {
			if slots[i].used {
				continue
			}
			if ruleFingerprint(slots[i].rule) != fp {
				continue
			}
			slots[i].used = true
			return slots[i].rule, true
		}
		return SecurityGroupRuleModel{}, false
	}

	out := make([]SecurityGroupRuleModel, 0, len(apiRules))
	for _, stateRule := range stateRules {
		stateID := stateRule.ID.ValueString()
		if stateID != "" {
			if idx, ok := apiByID[stateID]; ok && !slots[idx].used {
				slots[idx].used = true
				out = append(out, slots[idx].rule)
				continue
			}
			// Id gone from API (delete+recreate / failed apply). Keep this config
			// slot by matching state attributes to an unused API rule — do not drop
			// and append leftovers (that reshuffles blocks onto the wrong rules).
			if matched, ok := takeByFingerprint(ruleFingerprint(stateRule)); ok {
				out = append(out, matched)
			}
			continue
		}

		if matched, ok := takeByFingerprint(ruleFingerprint(stateRule)); ok {
			out = append(out, matched)
		}
	}

	// Append remaining API rules (import extras / added outside Terraform), preserving API order.
	for i := range slots {
		if slots[i].used {
			continue
		}
		out = append(out, slots[i].rule)
	}

	return out
}

func mapAPIRuleToModel(ctx context.Context, apiRule SecurityGroupRule, remoteNameByID map[string]string) (SecurityGroupRuleModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	rule := SecurityGroupRuleModel{
		ID:        types.StringValue(apiRule.ID),
		Protocol:  types.StringValue(strings.ToLower(apiRule.Protocol)),
		EtherType: types.StringValue(normalizeEtherTypeForState(apiRule.EtherType)),
		Direction: types.StringValue(strings.ToLower(apiRule.Direction)),
	}

	switch {
	case apiRule.PortRangeMin != nil && isValidPort(*apiRule.PortRangeMin):
		rule.PortRange = types.Int64Value(*apiRule.PortRangeMin)
	case apiRule.PortRangeMax != nil && isValidPort(*apiRule.PortRangeMax):
		rule.PortRange = types.Int64Value(*apiRule.PortRangeMax)
	default:
		// API often returns -1/0 for icmp / "all ports"; keep null so fingerprints
		// match config that omitted port_range.
		rule.PortRange = types.Int64Null()
	}

	if apiRule.RemoteIPPrefix != nil && *apiRule.RemoteIPPrefix != "" {
		list, listDiags := types.ListValueFrom(ctx, types.StringType, []string{*apiRule.RemoteIPPrefix})
		diags.Append(listDiags...)
		rule.RemoteIPPrefix = list
		rule.RemoteSecurityGroup = types.StringNull()
	} else if apiRule.RemoteGroupID != nil && *apiRule.RemoteGroupID != "" {
		name := remoteNameByID[*apiRule.RemoteGroupID]
		if name == "" {
			name = *apiRule.RemoteGroupID
		}
		rule.RemoteSecurityGroup = types.StringValue(name)
		rule.RemoteIPPrefix = types.ListNull(types.StringType)
	} else {
		rule.RemoteIPPrefix = types.ListNull(types.StringType)
		rule.RemoteSecurityGroup = types.StringNull()
	}

	return rule, diags
}

func normalizeEtherTypeForState(v string) string {
	switch strings.ToLower(v) {
	case "ipv6":
		return "ipv6"
	default:
		return "ipv4"
	}
}

func isValidPort(p int64) bool {
	return p >= 1 && p <= 65535
}

func (r *SecurityGroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state SecurityGroupResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	firewallID := state.FirewallID.ValueInt64()
	sgID := state.ID.ValueString()

	// Identity changes are RequiresReplace; Update only syncs rules.
	toDelete, toCreate := diffRules(state.Rules, plan.Rules)

	for _, rule := range toDelete {
		if rule.ID.IsNull() || rule.ID.ValueString() == "" {
			continue
		}
		if _, err := DeleteSecurityGroupRuleAndWait(r.client, ctx, firewallID, sgID, rule.ID.ValueString()); err != nil {
			resp.Diagnostics.AddError("Security Group Rule Delete Failed", err.Error())
			return
		}
	}

	created, _, _, diags := r.createRules(ctx, firewallID, sgID, toCreate)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Keep plan rule order. append(unchanged, created) shuffled blocks and caused
	// "inconsistent result after apply" (ids/attrs from one rule appeared on another).
	plan.Rules = rulesInPlanOrder(state.Rules, plan.Rules, created)
	plan.ID = state.ID
	plan.ManagedSecurityGroup = state.ManagedSecurityGroup
	if plan.Name.IsUnknown() {
		plan.Name = state.Name
	}
	if plan.Description.IsUnknown() {
		plan.Description = state.Description
	}
	// audit_id/status use UseStateForUnknown — keep planned values (do not replace
	// with the latest rule-op audit id or Terraform reports an inconsistent apply).
	if plan.AuditID.IsUnknown() || plan.AuditID.IsNull() {
		plan.AuditID = state.AuditID
	}
	if plan.Status.IsUnknown() || plan.Status.IsNull() {
		plan.Status = state.Status
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// alignPlanRuleIDsByFingerprint copies state rule ids onto matching plan rules
// (same fingerprint). Unmatched plan rules get an unknown id (create/recreate).
// Matching is by attributes, not list index, so removing a middle rule does not
// force surviving rules to look like in-place updates with new ids.
func alignPlanRuleIDsByFingerprint(plan *SecurityGroupResourceModel, stateRules []SecurityGroupRuleModel) bool {
	used := make([]bool, len(stateRules))
	changed := false

	for i := range plan.Rules {
		fp := ruleFingerprint(plan.Rules[i])
		matched := false
		for j := range stateRules {
			if used[j] {
				continue
			}
			if ruleFingerprint(stateRules[j]) != fp {
				continue
			}
			used[j] = true
			matched = true
			stateID := stateRules[j].ID
			if stateID.IsNull() || stateID.IsUnknown() || stateID.ValueString() == "" {
				if !plan.Rules[i].ID.IsUnknown() {
					plan.Rules[i].ID = types.StringUnknown()
					changed = true
				}
				break
			}
			if !plan.Rules[i].ID.Equal(stateID) {
				plan.Rules[i].ID = stateID
				changed = true
			}
			break
		}
		if matched {
			continue
		}
		if !plan.Rules[i].ID.IsUnknown() {
			plan.Rules[i].ID = types.StringUnknown()
			changed = true
		}
	}
	return changed
}

func ruleFingerprint(rule SecurityGroupRuleModel) string {
	port := ""
	if !rule.PortRange.IsNull() && !rule.PortRange.IsUnknown() {
		port = fmt.Sprintf("%d", rule.PortRange.ValueInt64())
	}
	prefixes := listStrings(rule.RemoteIPPrefix)
	return strings.ToLower(fmt.Sprintf("%s|%s|%s|%s|%s|%s",
		rule.Protocol.ValueString(),
		rule.EtherType.ValueString(),
		rule.Direction.ValueString(),
		port,
		strings.Join(prefixes, ","),
		rule.RemoteSecurityGroup.ValueString(),
	))
}

func listStrings(list types.List) []string {
	if list.IsNull() || list.IsUnknown() {
		return nil
	}
	var out []string
	_ = list.ElementsAs(context.Background(), &out, false)
	return out
}

func diffRules(stateRules, planRules []SecurityGroupRuleModel) (toDelete, toCreate []SecurityGroupRuleModel) {
	planCounts := map[string]int{}
	for _, r := range planRules {
		planCounts[ruleFingerprint(r)]++
	}

	stateCounts := map[string]int{}
	for _, r := range stateRules {
		fp := ruleFingerprint(r)
		stateCounts[fp]++
		if planCounts[fp] > 0 {
			planCounts[fp]--
			continue
		}
		toDelete = append(toDelete, r)
	}

	for _, r := range planRules {
		fp := ruleFingerprint(r)
		if stateCounts[fp] > 0 {
			stateCounts[fp]--
			continue
		}
		toCreate = append(toCreate, r)
	}
	return toDelete, toCreate
}

func unchangedRules(stateRules, planRules []SecurityGroupRuleModel) []SecurityGroupRuleModel {
	stateByFP := map[string][]SecurityGroupRuleModel{}
	for _, r := range stateRules {
		fp := ruleFingerprint(r)
		stateByFP[fp] = append(stateByFP[fp], r)
	}

	var out []SecurityGroupRuleModel
	for _, planRule := range planRules {
		fp := ruleFingerprint(planRule)
		if len(stateByFP[fp]) == 0 {
			continue
		}
		kept := stateByFP[fp][0]
		stateByFP[fp] = stateByFP[fp][1:]
		// Keep plan attribute values but retain known rule id from state.
		planRule.ID = kept.ID
		normalizeRuleNulls(&planRule)
		out = append(out, planRule)
	}
	return out
}

// rulesInPlanOrder rebuilds the rule list in the same order as the Terraform plan,
// attaching IDs from unchanged state rules or newly created rules.
func rulesInPlanOrder(stateRules, planRules, created []SecurityGroupRuleModel) []SecurityGroupRuleModel {
	pool := map[string][]SecurityGroupRuleModel{}
	for _, r := range unchangedRules(stateRules, planRules) {
		fp := ruleFingerprint(r)
		pool[fp] = append(pool[fp], r)
	}
	for _, r := range created {
		fp := ruleFingerprint(r)
		pool[fp] = append(pool[fp], r)
	}

	out := make([]SecurityGroupRuleModel, 0, len(planRules))
	for _, planRule := range planRules {
		fp := ruleFingerprint(planRule)
		if len(pool[fp]) == 0 {
			normalizeRuleNulls(&planRule)
			out = append(out, planRule)
			continue
		}
		matched := pool[fp][0]
		pool[fp] = pool[fp][1:]
		planRule.ID = matched.ID
		normalizeRuleNulls(&planRule)
		out = append(out, planRule)
	}
	return out
}

func normalizeRuleNulls(rule *SecurityGroupRuleModel) {
	if rule.PortRange.IsNull() {
		rule.PortRange = types.Int64Null()
	}
	if rule.RemoteIPPrefix.IsNull() {
		rule.RemoteIPPrefix = types.ListNull(types.StringType)
	}
	if rule.RemoteSecurityGroup.IsNull() {
		rule.RemoteSecurityGroup = types.StringNull()
	}
}

func (r *SecurityGroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data SecurityGroupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	firewallID := data.FirewallID.ValueInt64()
	sgID := data.ID.ValueString()
	if sgID == "" {
		return
	}

	for i, rule := range data.Rules {
		ruleID := rule.ID.ValueString()
		if ruleID == "" {
			continue
		}
		if _, err := DeleteSecurityGroupRuleAndWait(r.client, ctx, firewallID, sgID, ruleID); err != nil {
			resp.Diagnostics.AddError("Security Group Rule Delete Failed",
				fmt.Sprintf("rule[%d] (%s): %s", i, ruleID, err.Error()))
			return
		}
	}

	if data.ManagedSecurityGroup.ValueBool() {
		if _, err := DeleteSecurityGroupAndWait(r.client, ctx, firewallID, sgID); err != nil {
			resp.Diagnostics.AddError("Security Group Delete Failed", err.Error())
			return
		}
	}
}
