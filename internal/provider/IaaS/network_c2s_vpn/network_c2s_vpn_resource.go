// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package network_c2s_vpn

import (
	"context"
	"fmt"
	"regexp"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
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
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/network_firewall"
)

var _ resource.Resource = &NetworkC2SVPNResource{}
var _ resource.ResourceWithImportState = &NetworkC2SVPNResource{}
var _ resource.ResourceWithModifyPlan = &NetworkC2SVPNResource{}

// NewNetworkC2SVPNResource creates the C2S VPN resource.
func NewNetworkC2SVPNResource() resource.Resource {
	return &NetworkC2SVPNResource{}
}

// NetworkC2SVPNResource manages a C2S VPN on a firewall.
type NetworkC2SVPNResource struct {
	client *client.Client
}

// VPNUserModel is one VPN user in Terraform state.
type VPNUserModel struct {
	Name     types.String `tfsdk:"name"`
	Password types.String `tfsdk:"password"`
}

// NetworkC2SVPNResourceModel is the Terraform model.
type NetworkC2SVPNResourceModel struct {
	FirewallID types.Int64  `tfsdk:"firewall_id"`
	ID         types.String `tfsdk:"id"`

	PricingModel types.String   `tfsdk:"pricing_model"`
	Users        []VPNUserModel `tfsdk:"users"`

	AuditID types.String `tfsdk:"audit_id"`
	Status  types.String `tfsdk:"status"`

	PreSharedKey types.String `tfsdk:"pre_shared_key"`
	VPNName      types.String `tfsdk:"vpn_name"`
	VPNStatus    types.String `tfsdk:"vpn_status"`
	VPNIP        types.String `tfsdk:"vpn_ip"`
	PeerID       types.String `tfsdk:"peer_id"`
	VPNNoOfUsers types.Int64  `tfsdk:"vpn_no_of_users"`
}

var (
	passwordRegexCharsetLen = regexp.MustCompile(`^[a-zA-Z0-9!#$%^&*()_+\-=\[\]{}:;,.?/~]{8,50}$`)
	passwordRegexLower      = regexp.MustCompile(`[a-z]`)
	passwordRegexUpper      = regexp.MustCompile(`[A-Z]`)
	passwordRegexDigit      = regexp.MustCompile(`[0-9]`)
	passwordRegexSpecial    = regexp.MustCompile(`[!#$%^&*()_+\-=\[\]{}:;,.?/~]`)
	usernameRegex           = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
)

func (r *NetworkC2SVPNResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network_c2s_vpn"
}

func (r *NetworkC2SVPNResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Manages a C2S VPN on a network firewall in VayuCloud.",
		MarkdownDescription: "Manages a C2S VPN attached to a firewall. Initial creation provisions one VPN user; additional users and password resets use separate API flows.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Firewall ID the VPN is bound to (same as `firewall_id`).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"firewall_id": schema.Int64Attribute{
				Description: "The firewall resource ID this VPN is created on.",
				Required:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"pricing_model": schema.StringAttribute{
				Description: "Pricing model: daily, monthly, reserved_1, reserved_2, or reserved_3.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOfCaseInsensitive(
						"daily", "monthly", "reserved_1", "reserved_2", "reserved_3",
					),
				},
			},
			"users": schema.ListNestedAttribute{
				Description: "VPN users. Passwords are sensitive; usernames cannot be changed in place—remove and add a user instead.",
				Required:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Description: "Username (letters, digits, underscore, hyphen only).",
							Required:    true,
							Validators: []validator.String{
								stringvalidator.RegexMatches(usernameRegex, "Username may only contain letters, digits, underscore (_), and hyphen (-)"),
							},
						},
						"password": schema.StringAttribute{
							Description: "Password (8–50 chars; upper, lower, digit, special from allowed set).",
							Required:    true,
							Sensitive:   true,
							Validators: []validator.String{
								stringvalidator.RegexMatches(
									passwordRegexCharsetLen,
									"Password must be 8–50 characters with uppercase, lowercase, digit, and special character from the allowed set",
								),
								stringvalidator.RegexMatches(
									passwordRegexLower,
									"Password must contain at least one lowercase letter",
								),
								stringvalidator.RegexMatches(
									passwordRegexUpper,
									"Password must contain at least one uppercase letter",
								),
								stringvalidator.RegexMatches(
									passwordRegexDigit,
									"Password must contain at least one digit",
								),
								stringvalidator.RegexMatches(
									passwordRegexSpecial,
									"Password must contain at least one special character from the allowed set",
								),
							},
						},
					},
				},
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
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
			"pre_shared_key": schema.StringAttribute{
				Description: "Pre-shared key from the platform (read-only).",
				Computed:    true,
				Sensitive:   true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"vpn_name": schema.StringAttribute{
				Description: "VPN name from the platform.",
				Computed:    true,
			},
			"vpn_status": schema.StringAttribute{
				Description: "VPN status from the platform.",
				Computed:    true,
			},
			"vpn_ip": schema.StringAttribute{
				Description: "VPN IP from the platform.",
				Computed:    true,
			},
			"peer_id": schema.StringAttribute{
				Description: "Peer ID from the platform.",
				Computed:    true,
			},
			"vpn_no_of_users": schema.Int64Attribute{
				Description: "Number of VPN users reported by the platform.",
				Computed:    true,
			},
		},
	}
}

func (r *NetworkC2SVPNResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// ModifyPlan validates duplicate usernames and firewall existence during terraform plan.
// Destroy plans use prior state when the plan value is null.
func (r *NetworkC2SVPNResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if r.client == nil {
		return
	}

	var plan NetworkC2SVPNResourceModel
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
		var usersList types.List
		resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("users"), &usersList)...)
		if resp.Diagnostics.HasError() {
			return
		}
		if usersList.IsUnknown() {
			return
		}
	}
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.FirewallID.IsUnknown() {
		return
	}
	for _, u := range plan.Users {
		if u.Name.IsUnknown() {
			return
		}
	}

	if err := duplicateUserNames(plan.Users); err != nil {
		resp.Diagnostics.AddError("Invalid users", err.Error())
		return
	}

	if err := network_firewall.ValidateFirewallExists(r.client, ctx, plan.FirewallID.ValueInt64()); err != nil {
		resp.Diagnostics.AddError("Invalid firewall_id", err.Error())
		return
	}
}

func duplicateUserNames(users []VPNUserModel) error {
	seen := make(map[string]struct{})
	for _, u := range users {
		n := u.Name.ValueString()
		if _, ok := seen[n]; ok {
			return fmt.Errorf("duplicate VPN username %q", n)
		}
		seen[n] = struct{}{}
	}
	return nil
}

func userMap(users []VPNUserModel) map[string]VPNUserModel {
	m := make(map[string]VPNUserModel, len(users))
	for _, u := range users {
		m[u.Name.ValueString()] = u
	}
	return m
}

func userNamesSet(users []VPNUserModel) map[string]struct{} {
	s := make(map[string]struct{}, len(users))
	for _, u := range users {
		s[u.Name.ValueString()] = struct{}{}
	}
	return s
}

func remoteNameSet(read *C2SVPNReadData) map[string]struct{} {
	s := make(map[string]struct{})
	for _, u := range read.Users {
		s[u.Name] = struct{}{}
	}
	return s
}

func (r *NetworkC2SVPNResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data NetworkC2SVPNResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := duplicateUserNames(data.Users); err != nil {
		resp.Diagnostics.AddError("Invalid users", err.Error())
		return
	}

	fwID := data.FirewallID.ValueInt64()

	first := data.Users[0]
	createReq := &CreateC2SVPNRequest{
		PricingModel: data.PricingModel.ValueString(),
		User: c2svpnUserCreds{
			Passwd: first.Password.ValueString(),
			Name:   first.Name.ValueString(),
		},
	}

	auditLog, err := CreateC2SVPNAndWait(r.client, ctx, fwID, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating C2S VPN", err.Error())
		return
	}

	if len(data.Users) > 1 {
		extra := make([]VPNUserEntry, 0, len(data.Users)-1)
		for i := 1; i < len(data.Users); i++ {
			u := data.Users[i]
			extra = append(extra, VPNUserEntry{
				Name:   u.Name.ValueString(),
				Passwd: u.Password.ValueString(),
			})
		}
		auditLog, err = AddVPNUsersAndWait(r.client, ctx, fwID, extra, len(data.Users))
		if err != nil {
			resp.Diagnostics.AddError("Error Adding Additional VPN Users", err.Error())
			return
		}
	}

	data.ID = types.StringValue(strconv.FormatInt(fwID, 10))
	data.AuditID = types.StringValue(auditLog.AuditID)
	data.Status = types.StringValue(auditLog.Status)

	if err := r.refreshReadIntoModel(ctx, &data, fwID); err != nil {
		resp.Diagnostics.AddError("Error Reading C2S VPN After Create", err.Error())
		return
	}

	tflog.Info(ctx, "C2S VPN created", map[string]any{"firewall_id": fwID, "audit_id": data.AuditID.ValueString()})
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *NetworkC2SVPNResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data NetworkC2SVPNResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	fwID := data.FirewallID.ValueInt64()
	if err := r.refreshReadIntoModel(ctx, &data, fwID); err != nil {
		resp.Diagnostics.AddError("Error Reading C2S VPN", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *NetworkC2SVPNResource) refreshReadIntoModel(ctx context.Context, data *NetworkC2SVPNResourceModel, firewallID int64) error {
	_, readData, err := ReadC2SVPN(r.client, ctx, firewallID)
	if err != nil {
		return err
	}

	data.PreSharedKey = types.StringValue(readData.PreSharedKey)
	data.VPNName = types.StringValue(readData.VPNName)
	data.VPNStatus = types.StringValue(readData.VPNStatus)
	data.VPNIP = types.StringValue(readData.VPNIP)
	data.PeerID = types.StringValue(readData.PeerID)
	data.VPNNoOfUsers = types.Int64Value(int64(readData.EffectiveVPNUserCount()))

	// Preserve passwords from state by username; order follows API users.
	prev := userMap(data.Users)
	var merged []VPNUserModel
	for _, ru := range readData.Users {
		name := ru.Name
		if u, ok := prev[name]; ok {
			merged = append(merged, u)
		} else {
			merged = append(merged, VPNUserModel{
				Name:     types.StringValue(name),
				Password: types.StringNull(),
			})
		}
	}
	data.Users = merged

	return nil
}

func (r *NetworkC2SVPNResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state NetworkC2SVPNResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	fwID := plan.FirewallID.ValueInt64()

	planByName := userMap(plan.Users)
	stateByName := userMap(state.Users)
	if err := duplicateUserNames(plan.Users); err != nil {
		resp.Diagnostics.AddError("Invalid users", err.Error())
		return
	}

	if err := network_firewall.ValidateFirewallExists(r.client, ctx, fwID); err != nil {
		resp.Diagnostics.AddError("Invalid firewall_id", err.Error())
		return
	}
	remoteNames := userNamesSet(state.Users)
	planNames := userNamesSet(plan.Users)

	toRemove := usersToRemove(remoteNames, planNames)
	targetCount := len(plan.Users)

	// var lastAudit *client.AuditLogResponse

	if len(toRemove) > 0 {
		_, err := RemoveVPNUsersAndWait(r.client, ctx, fwID, toRemove, targetCount)
		if err != nil {
			resp.Diagnostics.AddError("Error Removing VPN Users", err.Error())
			return
		}
		// lastAudit = al
		_, readData, err := ReadC2SVPN(r.client, ctx, fwID)
		if err != nil {
			resp.Diagnostics.AddError("Error Reading C2S VPN After User Removal", err.Error())
			return
		}
		remoteNames = remoteNameSet(readData)
	}

	toAdd := usersToAdd(planByName, remoteNames)

	if len(toAdd) > 0 {
		_, err := AddVPNUsersAndWait(r.client, ctx, fwID, toAdd, targetCount)
		if err != nil {
			resp.Diagnostics.AddError("Error Adding VPN Users", err.Error())
			return
		}
		// lastAudit = al
		_, readData, err := ReadC2SVPN(r.client, ctx, fwID)
		if err != nil {
			resp.Diagnostics.AddError("Error Reading C2S VPN After Adding Users", err.Error())
			return
		}
		remoteNames = remoteNameSet(readData)
	}

	toReset := usersToResetPassword(planByName, stateByName, remoteNames)

	if len(toReset) > 0 {
		_, err := ResetVPNUserPasswordsAndWait(r.client, ctx, fwID, toReset, targetCount)
		if err != nil {
			resp.Diagnostics.AddError("Error Resetting VPN User Passwords", err.Error())
			return
		}
		// lastAudit = al
	}

	// Refresh state fields
	plan.ID = types.StringValue(strconv.FormatInt(fwID, 10))
	if err := r.refreshReadIntoModel(ctx, &plan, fwID); err != nil {
		resp.Diagnostics.AddError("Error Reading C2S VPN After Update", err.Error())
		return
	}

	// if lastAudit != nil {
	// 	plan.AuditID = types.StringValue(lastAudit.AuditID)
	// 	plan.Status = types.StringValue(lastAudit.Status)
	// } else {
	// 	if plan.AuditID.IsNull() || plan.AuditID.IsUnknown() {
	// 		plan.AuditID = state.AuditID
	// 	}
	// 	if plan.Status.IsNull() || plan.Status.IsUnknown() {
	// 		plan.Status = state.Status
	// 	}
	// }

	tflog.Info(ctx, "C2S VPN updated", map[string]any{"firewall_id": fwID})
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func uniqueStrings(in []string) []string {
	seen := make(map[string]struct{})
	var out []string
	for _, s := range in {
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

func usersToRemove(remoteNames, planNames map[string]struct{}) []string {
	var out []string
	for n := range remoteNames {
		if _, ok := planNames[n]; !ok {
			out = append(out, n)
		}
	}
	return uniqueStrings(out)
}

func usersToAdd(planByName map[string]VPNUserModel, remoteNames map[string]struct{}) []VPNUserEntry {
	var out []VPNUserEntry
	for n, u := range planByName {
		if _, ok := remoteNames[n]; !ok {
			out = append(out, VPNUserEntry{
				Name:   u.Name.ValueString(),
				Passwd: u.Password.ValueString(),
			})
		}
	}
	return out
}

func usersToResetPassword(planByName map[string]VPNUserModel, stateByName map[string]VPNUserModel, remoteNames map[string]struct{}) []VPNUserEntry {
	var out []VPNUserEntry
	for n := range planByName {
		if _, ok := remoteNames[n]; !ok {
			continue
		}
		pu := planByName[n]
		su, so := stateByName[n]
		if !so {
			continue
		}
		if pu.Password.Equal(su.Password) {
			continue
		}
		out = append(out, VPNUserEntry{
			Name:   pu.Name.ValueString(),
			Passwd: pu.Password.ValueString(),
		})
	}
	return out
}

func (r *NetworkC2SVPNResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data NetworkC2SVPNResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	fwID := data.FirewallID.ValueInt64()

	_, err := DeleteC2SVPNAndWait(r.client, ctx, fwID)
	if err != nil {
		resp.Diagnostics.AddError("Error Deleting C2S VPN", err.Error())
		if err := network_firewall.ValidateFirewallExists(r.client, ctx, fwID); err != nil {
			resp.Diagnostics.AddError("Invalid firewall_id", err.Error())
			return
		}
		return
	}

	tflog.Info(ctx, "C2S VPN deleted", map[string]any{"firewall_id": fwID})
}

func (r *NetworkC2SVPNResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	fwID, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			"Import ID must be the numeric firewall_id.",
		)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("firewall_id"), types.Int64Value(fwID))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), types.StringValue(strconv.FormatInt(fwID, 10)))...)
}
