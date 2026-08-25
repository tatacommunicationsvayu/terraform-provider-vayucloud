// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package file_storage_export_policy

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/client"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/file_storage_volume"
)

const exportPolicyIDSeparator = "/"

var (
	_ resource.Resource                = &FileStorageExportPolicyResource{}
	_ resource.ResourceWithImportState = &FileStorageExportPolicyResource{}
	_ resource.ResourceWithModifyPlan  = &FileStorageExportPolicyResource{}
)

// exportPolicyID is the Terraform resource id: one attach per (volume, client_ip) pair.
func exportPolicyID(volumeID, clientIP string) string {
	return strings.TrimSpace(volumeID) + exportPolicyIDSeparator + strings.TrimSpace(clientIP)
}

// NewFileStorageExportPolicyResource registers the file_storage_export_policy resource.
func NewFileStorageExportPolicyResource() resource.Resource {
	return &FileStorageExportPolicyResource{}
}

// FileStorageExportPolicyResource manages a NAS export policy (attach/detach client IP on a volume).
type FileStorageExportPolicyResource struct {
	client *client.Client
}

// FileStorageExportPolicyResourceModel maps Terraform state to the API.
type FileStorageExportPolicyResourceModel struct {
	ID                  types.String `tfsdk:"id"`
	FileStorageVolumeID types.String `tfsdk:"file_storage_volume_id"`
	Name                types.String `tfsdk:"name"`
	ClientIP            types.String `tfsdk:"client_ip"`
	EngagementID        types.Int64  `tfsdk:"engagement_id"`
	FileServerID        types.String `tfsdk:"file_server_id"`
	QuotaGb             types.Int64  `tfsdk:"quota_gb"`
	AuditID             types.String `tfsdk:"audit_id"`
	Status              types.String `tfsdk:"status"`
}

func (r *FileStorageExportPolicyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_file_storage_export_policy"
}

func (r *FileStorageExportPolicyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Manages a file storage export policy (NAS client IP allowed on a volume).",
		MarkdownDescription: "Attaches and detaches a client IP on a NAS volume via `POST .../nas/attachClient` and `DELETE .../nas/volumes/{volumeId}/clients/{clientIp}`. Requires `file_storage_volume_id` (nasVolCi), `name` (volume name), and `client_ip`. Optional `engagement_id` and `file_server_id` enable drift detection on refresh via `fetchVolumeDetails` (client removed from state when `client_ip` is no longer attached on the platform). Use multiple resources (or `for_each`) for several client IPs on the same volume; `id` is `{file_storage_volume_id}/{client_ip}`.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Synthetic id `{file_storage_volume_id}/{client_ip}` so multiple export policies can target the same volume with different client IPs.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"file_storage_volume_id": schema.StringAttribute{
				Description: "NAS volume CI id (nasVolCi; same as `vayucloud_file_storage_volume.id` when managed in Terraform).",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "NAS volume name (same as `vayucloud_file_storage_volume.name`; sent as `name` to attachClient).",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 256),
				},
			},
			"client_ip": schema.StringAttribute{
				Description: "Client IP to attach to the volume export (sent as `ip` to attachClient; path segment on detach).",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 45),
				},
			},
			"engagement_id": schema.Int64Attribute{
				Description: "Optional engagement id used with `file_server_id` to read attached clients via fetchVolumeDetails and detect client_ip drift on refresh.",
				Optional:    true,
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
				},
			},
			"file_server_id": schema.StringAttribute{
				Description: "Optional file server id (vserver id) used with `engagement_id` for client_ip drift detection on refresh.",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 256),
				},
			},
			"quota_gb": schema.Int64Attribute{
				Description: "Optional quota in GB for documentation only; attachClient does not set quota (in-place resize is not supported).",
				Optional:    true,
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
				},
			},
			"audit_id": schema.StringAttribute{
				Description: "Audit ID from the last async operation.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"status": schema.StringAttribute{
				Description: "Status from the last async operation.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *FileStorageExportPolicyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// ModifyPlan sets id at plan time to volume_id/client_ip (no separate platform id from attachClient).
func (r *FileStorageExportPolicyResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}

	var plan FileStorageExportPolicyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.FileStorageVolumeID.IsUnknown() || plan.FileStorageVolumeID.IsNull() ||
		plan.ClientIP.IsUnknown() || plan.ClientIP.IsNull() {
		return
	}

	plan.ID = types.StringValue(exportPolicyID(plan.FileStorageVolumeID.ValueString(), plan.ClientIP.ValueString()))
	resp.Diagnostics.Append(resp.Plan.Set(ctx, plan)...)
}

func (r *FileStorageExportPolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data FileStorageExportPolicyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	volumeID := data.FileStorageVolumeID.ValueString()
	clientIP := data.ClientIP.ValueString()
	volumeName := data.Name.ValueString()

	auditLog, err := AttachNASClientAndWait(r.client, ctx, clientIP, volumeName, volumeID)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating File Storage Export Policy", err.Error())
		return
	}

	data.ID = types.StringValue(exportPolicyID(volumeID, clientIP))
	data.AuditID = types.StringValue(auditLog.AuditID)
	data.Status = types.StringValue(auditLog.Status)

	tflog.Info(ctx, "File storage export policy created", map[string]any{
		"id": data.ID.ValueString(), "volume_id": volumeID, "client_ip": clientIP,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *FileStorageExportPolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state FileStorageExportPolicyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !state.FileStorageVolumeID.IsNull() && !state.ClientIP.IsNull() {
		expected := exportPolicyID(state.FileStorageVolumeID.ValueString(), state.ClientIP.ValueString())
		if state.ID.ValueString() != expected {
			state.ID = types.StringValue(expected)
		}
	}

	if r.client != nil && driftLookupEnabled(!state.EngagementID.IsNull(), state.FileServerID.ValueString()) {
		if removed, warn := r.reconcileClientIPDrift(ctx, &state); removed {
			resp.State.RemoveResource(ctx)
			return
		} else if warn != "" {
			resp.Diagnostics.AddWarning("Export Policy Drift Check Skipped", warn)
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// reconcileClientIPDrift returns true when the resource should be removed from state
// because the platform no longer has client_ip attached to the volume.
func (r *FileStorageExportPolicyResource) reconcileClientIPDrift(ctx context.Context, state *FileStorageExportPolicyResourceModel) (removed bool, warn string) {
	detail, err := file_storage_volume.GetFileStorageVolumeDetails(
		r.client, ctx,
		state.EngagementID.ValueInt64(),
		state.FileServerID.ValueString(),
		state.Name.ValueString(),
	)
	if err != nil {
		if errors.Is(err, file_storage_volume.ErrFileStorageVolumeNotFound) {
			tflog.Warn(ctx, "File storage export policy volume not found during drift check; removing from state", map[string]any{
				"id": state.ID.ValueString(),
			})
			return true, ""
		}
		return false, err.Error()
	}

	clientIP := state.ClientIP.ValueString()
	if clientIPInVolumeClients(detail.Clients, clientIP) {
		return false, ""
	}

	tflog.Warn(ctx, "File storage export policy client_ip drift detected; removing from state", map[string]any{
		"id":                state.ID.ValueString(),
		"client_ip":         clientIP,
		"platform_clients":  detail.Clients,
		"file_storage_name": state.Name.ValueString(),
	})
	return true, ""
}

func (r *FileStorageExportPolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, prior FileStorageExportPolicyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !plan.QuotaGb.Equal(prior.QuotaGb) && !plan.QuotaGb.IsNull() {
		resp.Diagnostics.AddError(
			"Quota Update Not Supported",
			"quota_gb cannot be changed in-place; attachClient/detachClient do not manage client quota.",
		)
		return
	}
	plan.AuditID = prior.AuditID
	plan.Status = prior.Status

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *FileStorageExportPolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state FileStorageExportPolicyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if _, err := DetachNASClientAndWait(r.client, ctx,
		state.ClientIP.ValueString(),
		state.FileStorageVolumeID.ValueString(),
	); err != nil {
		resp.Diagnostics.AddError("Error Deleting File Storage Export Policy", err.Error())
		return
	}

	tflog.Info(ctx, "File storage export policy deleted", map[string]any{"id": state.ID.ValueString()})
}

// Import ID format: file_storage_volume_id,volume_name,client_ip
func (r *FileStorageExportPolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, ",")
	if len(parts) != 3 {
		resp.Diagnostics.AddError(
			"Invalid Import Identifier",
			`Expected format "file_storage_volume_id,volume_name,client_ip".`,
		)
		return
	}
	vol := strings.TrimSpace(parts[0])
	volName := strings.TrimSpace(parts[1])
	clientIP := strings.TrimSpace(parts[2])
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("file_storage_volume_id"), types.StringValue(vol))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), types.StringValue(volName))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("client_ip"), types.StringValue(clientIP))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), types.StringValue(exportPolicyID(vol, clientIP)))...)
}
