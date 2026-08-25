// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package nas

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/client"
)

var (
	_ resource.Resource                = &ExtendNasZoneResource{}
	_ resource.ResourceWithImportState = &ExtendNasZoneResource{}
)

// NewExtendNasZoneResource registers POST .../nas/extendNASZone/{zoneId}/{vserverId}.
func NewExtendNasZoneResource() resource.Resource {
	return &ExtendNasZoneResource{}
}

// ExtendNasZoneResource triggers portal extend NAS zone workflow for a zone + vserver pair.
type ExtendNasZoneResource struct {
	client *client.Client
}

type extendNasZoneResourceModel struct {
	ID types.String `tfsdk:"id"`

	ZoneID    types.Int64 `tfsdk:"zone_id"`
	VserverID types.Int64 `tfsdk:"vserver_id"`

	IsFirewallConfigured types.Bool   `tfsdk:"is_firewall_configured"`
	TicketID             types.String `tfsdk:"ticket_id"`

	RawResponse types.String `tfsdk:"raw_response"`
}

func (r *ExtendNasZoneResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_extend_nas_zone"
}

func (r *ExtendNasZoneResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Invokes extend NAS zone for a zone and vserver (portal POST extendNASZone). Destroy calls deconfigureNASZone only — does not delete the zone.",
		MarkdownDescription: "Calls `POST .../nas/extendNASZone/{zoneId}/{vserverId}` with body `{\"isFirewallConfigured\": ...}` matching the portal contract. Optional `ticket_id` is sent as query `ticketId` for audit correlation.\n\n**Destroy** calls **only** `DELETE .../nas/deconfigureNASZone/{zoneId}` (same optional `ticketId`) **before** this resource is removed from state. That endpoint **deconfigures** NAS for the zone (portal workflow); it **does not** delete or remove the zone entity itself. Because this resource should `depends_on` the file server, Terraform typically destroys dependents of the file server first (e.g. volumes), then this resource (deconfigure), then the file server — so NAS is deconfigured before vserver deletion when this resource is in the stack.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Synthetic id: extend-nas-zone-{zone_id}-{vserver_id}.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"zone_id": schema.Int64Attribute{
				Description: "Zone id path segment (same as portal `zoneId`).",
				Required:    true,
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
				},
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"vserver_id": schema.Int64Attribute{
				Description: "Vserver / endpoint-component id path segment (same as portal `vserverId`; typically `vayucloud_file_server.id`).",
				Required:    true,
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
				},
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"is_firewall_configured": schema.BoolAttribute{
				Description: "JSON body `isFirewallConfigured` sent to the portal. Defaults to false.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"ticket_id": schema.StringAttribute{
				Description: "Optional audit ticket id; sent as query parameter `ticketId`.",
				Optional:    true,
			},
			"raw_response": schema.StringAttribute{
				Description: "Last successful HTTP JSON response body from extend NAS zone.",
				Computed:    true,
			},
		},
	}
}

func (r *ExtendNasZoneResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ExtendNasZoneResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan extendNasZoneResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	raw, err := ExtendNASZone(r.client, ctx, plan.ZoneID.ValueInt64(), plan.VserverID.ValueInt64(),
		plan.IsFirewallConfigured.ValueBool(), plan.TicketID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Extend NAS Zone Failed", err.Error())
		return
	}
	plan.ID = types.StringValue(extendNasZoneSyntheticID(plan.ZoneID.ValueInt64(), plan.VserverID.ValueInt64()))
	plan.RawResponse = types.StringValue(raw)
	tflog.Info(ctx, "extend_nas_zone created", map[string]any{"id": plan.ID.ValueString()})
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ExtendNasZoneResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state extendNasZoneResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ExtendNasZoneResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, prior extendNasZoneResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
	if resp.Diagnostics.HasError() {
		return
	}
	ticket := plan.TicketID.ValueString()
	raw, err := ExtendNASZone(r.client, ctx, plan.ZoneID.ValueInt64(), plan.VserverID.ValueInt64(),
		plan.IsFirewallConfigured.ValueBool(), ticket)
	if err != nil {
		resp.Diagnostics.AddError("Extend NAS Zone Failed", err.Error())
		return
	}
	plan.ID = prior.ID
	plan.RawResponse = types.StringValue(raw)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ExtendNasZoneResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state extendNasZoneResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	ticket := ""
	if !state.TicketID.IsNull() {
		ticket = state.TicketID.ValueString()
	}
	raw, err := DeconfigureNASZone(r.client, ctx, state.ZoneID.ValueInt64(), ticket)
	if err != nil {
		resp.Diagnostics.AddError("Deconfigure NAS Zone Failed", err.Error())
		return
	}
	tflog.Info(ctx, "extend_nas_zone deconfigured before state removal",
		map[string]any{"zone_id": state.ZoneID.ValueInt64(), "response_len": len(raw)})
}

func (r *ExtendNasZoneResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id := strings.TrimSpace(req.ID)
	if id == "" {
		resp.Diagnostics.AddError("Import Extend NAS Zone", "import id is empty")
		return
	}
	parts := strings.SplitN(id, ",", 2)
	if len(parts) != 2 {
		resp.Diagnostics.AddError("Import Extend NAS Zone", `expected import id "zone_id,vserver_id"`)
		return
	}
	zStr, vStr := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
	zid, err := strconv.ParseInt(zStr, 10, 64)
	if err != nil || zid < 1 {
		resp.Diagnostics.AddError("Import Extend NAS Zone", fmt.Sprintf("invalid zone_id %q", zStr))
		return
	}
	vid, err := strconv.ParseInt(vStr, 10, 64)
	if err != nil || vid < 1 {
		resp.Diagnostics.AddError("Import Extend NAS Zone", fmt.Sprintf("invalid vserver_id %q", vStr))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("zone_id"), types.Int64Value(zid))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("vserver_id"), types.Int64Value(vid))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), types.StringValue(extendNasZoneSyntheticID(zid, vid)))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("is_firewall_configured"), types.BoolValue(false))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("ticket_id"), types.StringNull())...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("raw_response"), types.StringValue(""))...)
}

func extendNasZoneSyntheticID(zoneID, vserverID int64) string {
	return fmt.Sprintf("extend-nas-zone-%d-%d", zoneID, vserverID)
}
