// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package network_c2s_vpn

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/client"
)

var _ datasource.DataSource = &NetworkC2SVPNDataSource{}

// NewNetworkC2SVPNDataSource creates a data source that reads C2S VPN state (same API as the resource Read).
func NewNetworkC2SVPNDataSource() datasource.DataSource {
	return &NetworkC2SVPNDataSource{}
}

// NetworkC2SVPNDataSource reads VPN details via action-state (module=c2svpn, action=read).
type NetworkC2SVPNDataSource struct {
	client *client.Client
}

// VPNUserNameModel is a VPN user as returned by the read API (name only).
type VPNUserNameModel struct {
	Name types.String `tfsdk:"name"`
}

// NetworkC2SVPNDataSourceModel is the data source state model.
type NetworkC2SVPNDataSourceModel struct {
	ID types.String `tfsdk:"id"`

	FirewallID types.Int64 `tfsdk:"firewall_id"`

	PreSharedKey types.String `tfsdk:"pre_shared_key"`
	VPNName      types.String `tfsdk:"vpn_name"`
	VPNStatus    types.String `tfsdk:"vpn_status"`
	VPNIP        types.String `tfsdk:"vpn_ip"`
	PeerID       types.String `tfsdk:"peer_id"`
	VPNNoOfUsers types.Int64  `tfsdk:"vpn_no_of_users"`

	Users []VPNUserNameModel `tfsdk:"users"`

	Status       types.String `tfsdk:"status"`
	Message      types.String `tfsdk:"message"`
	ResponseCode types.Int64  `tfsdk:"response_code"`
	RawResponse  types.String `tfsdk:"raw_response"`
}

func (d *NetworkC2SVPNDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network_c2s_vpn"
}

func (d *NetworkC2SVPNDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Reads C2S VPN details for a firewall from the VayuCloud API via GET /network_operations/vpn-state/{firewallId}.",
		MarkdownDescription: "Reads C2S VPN details for a firewall. This uses the same `ReadC2SVPN` flow as the `vayucloud_network_c2s_vpn` resource read operation.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Same as `firewall_id` as a string (for Terraform).",
				Computed:    true,
			},
			"firewall_id": schema.Int64Attribute{
				Description: "Firewall resource ID the VPN is attached to.",
				Required:    true,
			},
			"pre_shared_key": schema.StringAttribute{
				Description: "Pre-shared key from the platform.",
				Computed:    true,
				Sensitive:   true,
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
			"users": schema.ListNestedAttribute{
				Description: "VPN users (names only; passwords are not returned by the API).",
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
			"status": schema.StringAttribute{
				Description: "Top-level API response status (e.g. success).",
				Computed:    true,
			},
			"message": schema.StringAttribute{
				Description: "API message text.",
				Computed:    true,
			},
			"response_code": schema.Int64Attribute{
				Description: "API response code (0 = success).",
				Computed:    true,
			},
			"raw_response": schema.StringAttribute{
				Description: "Raw JSON `data` payload from the action-state response (for debugging).",
				Computed:    true,
			},
		},
	}
}

func (d *NetworkC2SVPNDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *NetworkC2SVPNDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data NetworkC2SVPNDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	fwID := data.FirewallID.ValueInt64()
	tflog.Debug(ctx, "Reading C2S VPN data source", map[string]any{"firewall_id": fwID})

	actionResp, readData, err := ReadC2SVPN(d.client, ctx, fwID)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading C2S VPN", err.Error())
		return
	}

	data.ID = types.StringValue(strconv.FormatInt(fwID, 10))
	data.Status = types.StringValue(actionResp.Status)
	data.Message = types.StringValue(actionResp.Message)
	data.ResponseCode = types.Int64Value(int64(actionResp.ResponseCode))

	raw, err := json.Marshal(actionResp.Data)
	if err != nil {
		resp.Diagnostics.AddError("Error Encoding Raw Response", err.Error())
		return
	}
	data.RawResponse = types.StringValue(string(raw))

	if len(actionResp.Data) == 0 {
		resp.Diagnostics.AddWarning(
			"Empty Response Data",
			"The action-state read returned empty data.",
		)
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
		return
	}

	data.PreSharedKey = types.StringValue(readData.PreSharedKey)
	data.VPNName = types.StringValue(readData.VPNName)
	data.VPNStatus = types.StringValue(readData.VPNStatus)
	data.VPNIP = types.StringValue(readData.VPNIP)
	data.PeerID = types.StringValue(readData.PeerID)
	data.VPNNoOfUsers = types.Int64Value(int64(readData.EffectiveVPNUserCount()))

	users := make([]VPNUserNameModel, 0, len(readData.Users))
	for _, u := range readData.Users {
		users = append(users, VPNUserNameModel{Name: types.StringValue(u.Name)})
	}
	data.Users = users

	tflog.Info(ctx, "C2S VPN data source read successfully", map[string]any{
		"firewall_id": fwID,
		"vpn_name":    readData.VPNName,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
