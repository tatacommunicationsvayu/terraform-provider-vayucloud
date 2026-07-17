// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package network_c2s_vpn_user

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/client"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/network_c2s_vpn"
)

var _ datasource.DataSource = &NetworkC2SVPNUsersDataSource{}

// NewNetworkC2SVPNUsersDataSource returns a data source that lists C2S VPN users for a firewall.
func NewNetworkC2SVPNUsersDataSource() datasource.DataSource {
	return &NetworkC2SVPNUsersDataSource{}
}

// NetworkC2SVPNUsersDataSource reads VPN details and exposes the user name list (same API as `vayucloud_network_c2s_vpn` read).
type NetworkC2SVPNUsersDataSource struct {
	client *client.Client
}

// VPNUserNameModel is one user name from the read API.
type VPNUserNameModel struct {
	Name types.String `tfsdk:"name"`
}

// NetworkC2SVPNUsersDataSourceModel is the data source model.
type NetworkC2SVPNUsersDataSourceModel struct {
	ID         types.String       `tfsdk:"id"`
	FirewallID types.Int64        `tfsdk:"firewall_id"`
	Users      []VPNUserNameModel `tfsdk:"users"`
}

func (d *NetworkC2SVPNUsersDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network_c2s_vpn_users"
}

func (d *NetworkC2SVPNUsersDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Lists C2S VPN usernames for a firewall (action-state read, module=c2svpn).",
		MarkdownDescription: "Calls the same read path as `vayucloud_network_c2s_vpn` and returns the `users` list (names only).",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Same as `firewall_id` as a string.",
				Computed:    true,
			},
			"firewall_id": schema.Int64Attribute{
				Description: "Firewall resource ID the C2S VPN is attached to.",
				Required:    true,
			},
			"users": schema.ListNestedAttribute{
				Description: "VPN users (names only; passwords are not returned).",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Description: "Username.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *NetworkC2SVPNUsersDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	cl, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T.", req.ProviderData),
		)
		return
	}
	d.client = cl
}

func (d *NetworkC2SVPNUsersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data NetworkC2SVPNUsersDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	fwID := data.FirewallID.ValueInt64()
	tflog.Debug(ctx, "Reading C2S VPN users data source", map[string]any{"firewall_id": fwID})

	_, readData, err := network_c2s_vpn.ReadC2SVPN(d.client, ctx, fwID)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading C2S VPN", err.Error())
		return
	}

	data.ID = types.StringValue(strconv.FormatInt(fwID, 10))
	users := make([]VPNUserNameModel, 0, len(readData.Users))
	for _, u := range readData.Users {
		users = append(users, VPNUserNameModel{Name: types.StringValue(u.Name)})
	}
	data.Users = users

	tflog.Info(ctx, "C2S VPN users data source read", map[string]any{"firewall_id": fwID, "count": len(users)})
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
