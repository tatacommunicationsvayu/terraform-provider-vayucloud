// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package network_c2s_vpn_user

import (
	"context"
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/client"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/network_c2s_vpn"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/network_firewall"
)

var _ resource.Resource = &NetworkC2SVPNUserResource{}
var _ resource.ResourceWithModifyPlan = &NetworkC2SVPNUserResource{}

// NewNetworkC2SVPNUserResource creates the C2S VPN user resource.
func NewNetworkC2SVPNUserResource() resource.Resource {
	return &NetworkC2SVPNUserResource{}
}

// NetworkC2SVPNUserResource manages VPN users on an existing C2S VPN (firewall).
type NetworkC2SVPNUserResource struct {
	client *client.Client
}

// VPNUserModel is one VPN user in Terraform state (same shape as vayucloud_network_c2s_vpn users).
type VPNUserModel struct {
	Name     types.String `tfsdk:"name"`
	Password types.String `tfsdk:"password"`
}

// NetworkC2SVPNUserResourceModel is the Terraform model.
type NetworkC2SVPNUserResourceModel struct {
	FirewallID types.Int64    `tfsdk:"firewall_id"`
	Users      []VPNUserModel `tfsdk:"users"`
}

var (
	passwordRegexCharsetLen = regexp.MustCompile(`^[a-zA-Z0-9!#$%^&*()_+\-=\[\]{}:;,.?/~]{8,50}$`)
	passwordRegexLower      = regexp.MustCompile(`[a-z]`)
	passwordRegexUpper      = regexp.MustCompile(`[A-Z]`)
	passwordRegexDigit      = regexp.MustCompile(`[0-9]`)
	passwordRegexSpecial    = regexp.MustCompile(`[!#$%^&*()_+\-=\[\]{}:;,.?/~]`)
	usernameRegex           = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
)

func userExistsOnVPN(read *network_c2s_vpn.C2SVPNReadData, name string) bool {
	if read == nil {
		return false
	}
	for _, u := range read.Users {
		if u.Name == name {
			return true
		}
	}
	return false
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

func remoteNameSet(read *network_c2s_vpn.C2SVPNReadData) map[string]struct{} {
	if read == nil {
		return map[string]struct{}{}
	}
	s := make(map[string]struct{})
	for _, u := range read.Users {
		s[u.Name] = struct{}{}
	}
	return s
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

func usersToAdd(planByName map[string]VPNUserModel, remoteNames map[string]struct{}) []network_c2s_vpn.VPNUserEntry {
	var out []network_c2s_vpn.VPNUserEntry
	for n, u := range planByName {
		if _, ok := remoteNames[n]; !ok {
			out = append(out, network_c2s_vpn.VPNUserEntry{
				Name:   u.Name.ValueString(),
				Passwd: u.Password.ValueString(),
			})
		}
	}
	return out
}

func usersToResetPassword(planByName map[string]VPNUserModel, stateByName map[string]VPNUserModel, remoteNames map[string]struct{}) []network_c2s_vpn.VPNUserEntry {
	var out []network_c2s_vpn.VPNUserEntry
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
		out = append(out, network_c2s_vpn.VPNUserEntry{
			Name:   pu.Name.ValueString(),
			Passwd: pu.Password.ValueString(),
		})
	}
	return out
}

func (r *NetworkC2SVPNUserResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network_c2s_vpn_user"
}

func (r *NetworkC2SVPNUserResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Manages C2S VPN users on a firewall (add, remove, password reset).",
		MarkdownDescription: "Manages VPN users for an existing C2S VPN. The VPN must already exist on `firewall_id` (for example via `vayucloud_network_c2s_vpn`). User list shape matches `users` on `vayucloud_network_c2s_vpn`.",

		Attributes: map[string]schema.Attribute{
			"firewall_id": schema.Int64Attribute{
				Description: "Firewall ID the C2S VPN is attached to.",
				Required:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
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
								stringvalidator.RegexMatches(passwordRegexLower, "Password must contain at least one lowercase letter"),
								stringvalidator.RegexMatches(passwordRegexUpper, "Password must contain at least one uppercase letter"),
								stringvalidator.RegexMatches(passwordRegexDigit, "Password must contain at least one digit"),
								stringvalidator.RegexMatches(passwordRegexSpecial, "Password must contain at least one special character from the allowed set"),
							},
						},
					},
				},
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
				},
			},
		},
	}
}

func (r *NetworkC2SVPNUserResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *NetworkC2SVPNUserResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if r.client == nil {
		return
	}

	var plan NetworkC2SVPNUserResourceModel
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

// refreshReadIntoModel is the only path that calls ReadC2SVPN for this resource.
//
// When data is nil (or has no users), it only returns API read data—used before Create/Update/Delete
// logic without mutating state.
//
// When data is non-nil with users, it keeps only users still present on the API and preserves each
// entry from state (including sensitive passwords). It must never write null passwords—Terraform
// rejects nested sensitive required attributes that differ from the plan as "inconsistent values".
func (r *NetworkC2SVPNUserResource) refreshReadIntoModel(ctx context.Context, data *NetworkC2SVPNUserResourceModel, firewallID int64) (*network_c2s_vpn.C2SVPNReadData, error) {
	_, readData, err := network_c2s_vpn.ReadC2SVPN(r.client, ctx, firewallID)
	if err != nil {
		return nil, err
	}
	if readData == nil {
		readData = &network_c2s_vpn.C2SVPNReadData{}
	}
	if data == nil || len(data.Users) == 0 {
		return readData, nil
	}
	api := remoteNameSet(readData)
	var merged []VPNUserModel
	for _, u := range data.Users {
		n := u.Name.ValueString()
		if _, ok := api[n]; !ok {
			return nil, fmt.Errorf("user %q not found on C2S VPN for firewall_id %d", n, firewallID)
		}
		merged = append(merged, u)
	}
	data.Users = merged
	return readData, nil
}

func (r *NetworkC2SVPNUserResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data NetworkC2SVPNUserResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := duplicateUserNames(data.Users); err != nil {
		resp.Diagnostics.AddError("Invalid users", err.Error())
		return
	}

	fwID := data.FirewallID.ValueInt64()
	if err := network_firewall.ValidateFirewallExists(r.client, ctx, fwID); err != nil {
		resp.Diagnostics.AddError("Invalid firewall_id", err.Error())
		return
	}

	readData, err := r.refreshReadIntoModel(ctx, nil, fwID)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading C2S VPN", err.Error())
		return
	}

	for _, u := range data.Users {
		if userExistsOnVPN(readData, u.Name.ValueString()) {
			resp.Diagnostics.AddError(
				"VPN User Already Exists",
				fmt.Sprintf("User %q already exists on the C2S VPN for firewall_id %d.", u.Name.ValueString(), fwID),
			)
			return
		}
	}

	toAdd := make([]network_c2s_vpn.VPNUserEntry, 0, len(data.Users))
	for _, u := range data.Users {
		toAdd = append(toAdd, network_c2s_vpn.VPNUserEntry{
			Name:   u.Name.ValueString(),
			Passwd: u.Password.ValueString(),
		})
	}
	targetCount := readData.EffectiveVPNUserCount() + len(toAdd)
	_, err = network_c2s_vpn.AddVPNUsersAndWait(r.client, ctx, fwID, toAdd, targetCount)
	if err != nil {
		resp.Diagnostics.AddError("Error Adding VPN Users", err.Error())
		return
	}

	// Keep plan users/passwords as-is; re-merging from the API can introduce null passwords and
	// trigger Terraform "inconsistent values for sensitive attribute" on nested blocks.

	tflog.Info(ctx, "C2S VPN users created", map[string]any{"firewall_id": fwID, "count": len(data.Users)})
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *NetworkC2SVPNUserResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data NetworkC2SVPNUserResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	fwID := data.FirewallID.ValueInt64()
	if _, err := r.refreshReadIntoModel(ctx, &data, fwID); err != nil {
		resp.Diagnostics.AddError("Error Reading C2S VPN", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *NetworkC2SVPNUserResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state NetworkC2SVPNUserResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := duplicateUserNames(plan.Users); err != nil {
		resp.Diagnostics.AddError("Invalid users", err.Error())
		return
	}

	fwID := plan.FirewallID.ValueInt64()
	if err := network_firewall.ValidateFirewallExists(r.client, ctx, fwID); err != nil {
		resp.Diagnostics.AddError("Invalid firewall_id", err.Error())
		return
	}

	planByName := userMap(plan.Users)
	stateByName := userMap(state.Users)

	remoteNames := userNamesSet(state.Users)
	planNames := userNamesSet(plan.Users)

	toRemove := usersToRemove(remoteNames, planNames)
	targetCount := len(plan.Users)

	if len(toRemove) > 0 {
		_, err := network_c2s_vpn.RemoveVPNUsersAndWait(r.client, ctx, fwID, toRemove, targetCount)
		if err != nil {
			resp.Diagnostics.AddError("Error Removing VPN Users", err.Error())
			return
		}
		readData, err := r.refreshReadIntoModel(ctx, nil, fwID)
		if err != nil {
			resp.Diagnostics.AddError("Error Reading C2S VPN After User Removal", err.Error())
			return
		}
		remoteNames = remoteNameSet(readData)
	}

	toAdd := usersToAdd(planByName, remoteNames)

	if len(toAdd) > 0 {
		_, err := network_c2s_vpn.AddVPNUsersAndWait(r.client, ctx, fwID, toAdd, targetCount)
		if err != nil {
			resp.Diagnostics.AddError("Error Adding VPN Users", err.Error())
			return
		}
		readData, err := r.refreshReadIntoModel(ctx, nil, fwID)
		if err != nil {
			resp.Diagnostics.AddError("Error Reading C2S VPN After Adding Users", err.Error())
			return
		}
		remoteNames = remoteNameSet(readData)
	}

	toReset := usersToResetPassword(planByName, stateByName, remoteNames)

	if len(toReset) > 0 {
		_, err := network_c2s_vpn.ResetVPNUserPasswordsAndWait(r.client, ctx, fwID, toReset, targetCount)
		if err != nil {
			resp.Diagnostics.AddError("Error Resetting VPN User Passwords", err.Error())
			return
		}
	}

	// Save desired users from plan only; do not merge API users with null passwords into state.

	tflog.Info(ctx, "C2S VPN users updated", map[string]any{"firewall_id": fwID})
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *NetworkC2SVPNUserResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data NetworkC2SVPNUserResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	fwID := data.FirewallID.ValueInt64()
	if err := network_firewall.ValidateFirewallExists(r.client, ctx, fwID); err != nil {
		resp.Diagnostics.AddError("Invalid firewall_id", err.Error())
		return
	}

	readData, err := r.refreshReadIntoModel(ctx, nil, fwID)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading C2S VPN", err.Error())
		return
	}

	var namesToRemove []string
	for _, u := range data.Users {
		n := u.Name.ValueString()
		if userExistsOnVPN(readData, n) {
			namesToRemove = append(namesToRemove, n)
		}
	}
	if len(namesToRemove) == 0 {
		tflog.Info(ctx, "C2S VPN users already absent", map[string]any{"firewall_id": fwID})
		return
	}

	targetCount := readData.EffectiveVPNUserCount() - len(namesToRemove)
	if targetCount < 0 {
		targetCount = 0
	}

	_, err = network_c2s_vpn.RemoveVPNUsersAndWait(r.client, ctx, fwID, namesToRemove, targetCount)
	if err != nil {
		resp.Diagnostics.AddError("Error Removing VPN Users", err.Error())
		return
	}

	tflog.Info(ctx, "C2S VPN users deleted", map[string]any{"firewall_id": fwID, "removed": len(namesToRemove)})
}
