// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package nas

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/client"
)

var _ datasource.DataSource = &NasVlanZoneDataSource{}

// NewNasVlanZoneDataSource reads NAS VLAN zone eligibility for an engagement/endpoint pair.
func NewNasVlanZoneDataSource() datasource.DataSource {
	return &NasVlanZoneDataSource{}
}

// NasVlanZoneDataSource wraps GET .../nas/isNasVlanZone.
type NasVlanZoneDataSource struct {
	client *client.Client
}

// nasVlanZoneDataSourceModel maps GET isNasVlanZone into Terraform state.
type nasVlanZoneDataSourceModel struct {
	ID types.String `tfsdk:"id"`

	EngagementID types.Int64 `tfsdk:"engagement_id"`
	EndpointID   types.Int64 `tfsdk:"endpoint_id"`

	IsNasVlanZone types.Bool `tfsdk:"is_nas_vlan_zone"`

	NasVlanZoneIDs types.List `tfsdk:"nas_vlan_zone_ids"`

	RawResponse types.String `tfsdk:"raw_response"`
}

func (d *NasVlanZoneDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_nas_vlan_zone"
}

func (d *NasVlanZoneDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Reads whether any NAS zone for the engagement/endpoint is a NAS VLAN zone (portal isNasZone + VLAN id).",
		MarkdownDescription: "Calls `GET .../nas/isNasVlanZone?engagementId=&endpointId=` (same prefix as other NAS APIs). Returns `is_nas_vlan_zone` and `nas_vlan_zone_ids` from the wrapped `data` object. Use this before or alongside `vayucloud_file_server` to decide reuse vs new zone flows; Terraform cannot prompt interactively—set variables from these outputs.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Synthetic id (nas-vlan-zone-{engagement_id}-{endpoint_id}).",
				Computed:    true,
			},
			"engagement_id": schema.Int64Attribute{
				Description: "Engagement id query parameter.",
				Required:    true,
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
				},
			},
			"endpoint_id": schema.Int64Attribute{
				Description: "Endpoint id query parameter.",
				Required:    true,
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
				},
			},
			"is_nas_vlan_zone": schema.BoolAttribute{
				Description: "True when the portal reports at least one qualifying NAS VLAN zone.",
				Computed:    true,
			},
			"nas_vlan_zone_ids": schema.ListAttribute{
				Description: "Zone ids that qualify (deduplicated); empty when none.",
				Computed:    true,
				ElementType: types.Int64Type,
			},
			"raw_response": schema.StringAttribute{
				Description: "Full JSON body for debugging.",
				Computed:    true,
			},
		},
	}
}

func (d *NasVlanZoneDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T.", req.ProviderData),
		)
		return
	}
	d.client = c
}

func (d *NasVlanZoneDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data nasVlanZoneDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	engID := data.EngagementID.ValueInt64()
	epID := data.EndpointID.ValueInt64()
	tflog.Debug(ctx, "Reading nas_vlan_zone data source", map[string]any{"engagement_id": engID, "endpoint_id": epID})

	check, err := GetNasVlanZoneCheck(d.client, ctx, engID, epID)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading NAS VLAN Zone Check", err.Error())
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("nas-vlan-zone-%d-%d", engID, epID))
	data.IsNasVlanZone = types.BoolValue(check.IsNasVlanZone)
	data.RawResponse = types.StringValue(check.RawBody)

	ids := make([]int64, 0, len(check.NasVlanZoneIDs))
	ids = append(ids, check.NasVlanZoneIDs...)
	listVal, diags := types.ListValueFrom(ctx, types.Int64Type, ids)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.NasVlanZoneIDs = listVal

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
