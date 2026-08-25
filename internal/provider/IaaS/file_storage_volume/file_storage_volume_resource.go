// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package file_storage_volume

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
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
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/file_server"
)

var (
	_ resource.Resource                = &FileStorageVolumeResource{}
	_ resource.ResourceWithImportState = &FileStorageVolumeResource{}
	_ resource.ResourceWithModifyPlan  = &FileStorageVolumeResource{}
)

// NewFileStorageVolumeResource registers the file_storage_volume resource.
func NewFileStorageVolumeResource() resource.Resource {
	return &FileStorageVolumeResource{}
}

// FileStorageVolumeResource manages a NAS file storage volume.
type FileStorageVolumeResource struct {
	client *client.Client
}

// FileStorageVolumeResourceModel maps Terraform state to the createNASVolume API.
type FileStorageVolumeResourceModel struct {
	ID              types.String `tfsdk:"id"`
	EngagementID    types.Int64  `tfsdk:"engagement_id"`
	EndpointID      types.Int64  `tfsdk:"endpoint_id"`
	FileServerID    types.String `tfsdk:"file_server_id"`
	Name            types.String `tfsdk:"name"`
	SizeGb          types.Int64  `tfsdk:"size_gb"`
	VolumeType      types.String `tfsdk:"volume_type"`
	FileStorageType types.String `tfsdk:"file_storage_type"`
	UsageType       types.String `tfsdk:"usage_type"`
	PricingModel    types.String `tfsdk:"pricing_model"`
	Iops            types.Int64  `tfsdk:"iops"`
	AuditID         types.String `tfsdk:"audit_id"`
	Status          types.String `tfsdk:"status"`
}

func (r *FileStorageVolumeResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_file_storage_volume"
}

func (r *FileStorageVolumeResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a NAS file storage volume.",
		MarkdownDescription: "Creates, resizes, and deletes a NAS volume via portal APIs `createNASVolume`, `resizeNasVolume`, and `DELETE /nas/volumes/{volumeId}`. " +
			"`file_server_id` is the platform vserver id (`vayucloud_file_server.id`). Request body fields align with `CreateNasVolumeVO`. " +
			"`size_gb` can be increased in place. If the volume is resized in the portal, `Read` and plan refresh adopt the larger size (decrease via Terraform is not supported).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Platform volume CI id (volCi) after create.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"engagement_id": schema.Int64Attribute{
				Description: "Engagement id (path segment for createNASVolume).",
				Required:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
				},
			},
			"endpoint_id": schema.Int64Attribute{
				Description: "Endpoint id for the NAS volume.",
				Required:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
				},
			},
			"file_server_id": schema.StringAttribute{
				Description: "Parent file server / vserver id (`vserverId` in the API; same as `vayucloud_file_server.id`).",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 256),
				},
			},
			"name": schema.StringAttribute{
				Description: "Volume name (`volumeName` in the API; 2–18 characters).",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthBetween(2, 18),
				},
			},
			"size_gb": schema.Int64Attribute{
				Description: "Volume size in GB (`volumeSize` in the API). Minimum 10. Increase triggers resizeNasVolume. Out-of-band portal growth is adopted on plan (same pattern as vayucloud_virtualmachine_blockstorage); shrink is not supported.",
				Required:    true,
				Validators: []validator.Int64{
					int64validator.Between(10, 10000),
				},
			},
			"volume_type": schema.StringAttribute{
				Description: "Volume type in the API (e.g. Flex Volume, Flex Group Volume).",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("Flex Volume"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"file_storage_type": schema.StringAttribute{
				Description: "File storage protocol for the volume (e.g. NFS, CIFS). Omit to let the portal resolve from NAS order / P2R defaults.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 64),
				},
			},
			"usage_type": schema.StringAttribute{
				Description: "Usage type for metering (e.g. reserved).",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"pricing_model": schema.StringAttribute{
				Description: "Pricing model for metering (e.g. reserved_1).",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"iops": schema.Int64Attribute{
				Description: "IOPS for the volume. Omit to let the portal set from NAS order (P2R). Not sent on update; does not force replacement when omitted.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
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

func (r *FileStorageVolumeResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *FileStorageVolumeResource) validatePlan(ctx context.Context, plan FileStorageVolumeResourceModel) error {
	if plan.FileServerID.IsUnknown() || plan.FileServerID.IsNull() {
		return nil
	}
	if _, err := ParseVserverID(plan.FileServerID.ValueString()); err != nil {
		return err
	}
	// Do not call getNASVservers/fetchVserverDetails here: new vservers (e.g. from
	// vayucloud_file_server.id) may not appear in those APIs until linkage propagates.
	if plan.EngagementID.IsUnknown() || plan.EndpointID.IsUnknown() {
		return nil
	}
	if err := file_server.ValidateEngagementAndEndpoint(r.client, ctx,
		plan.EngagementID.ValueInt64(),
		plan.EndpointID.ValueInt64(),
	); err != nil {
		return err
	}
	if plan.FileStorageType.IsUnknown() || plan.FileStorageType.IsNull() {
		return nil
	}
	return ValidateNASOrder(r.client, ctx,
		plan.EngagementID.ValueInt64(),
		plan.EndpointID.ValueInt64(),
		plan.FileStorageType.ValueString(),
	)
}

func (r *FileStorageVolumeResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if r.client == nil {
		return
	}
	var plan FileStorageVolumeResourceModel
	if req.Plan.Raw.IsNull() {
		resp.Diagnostics.Append(req.State.Get(ctx, &plan)...)
	} else {
		resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	}
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.validatePlan(ctx, plan); err != nil {
		resp.Diagnostics.AddError("File Storage Volume Plan Validation Failed", err.Error())
		return
	}

	r.adoptNasVolumeOutOfBandSizeInPlan(ctx, req, resp, plan)
	r.markAsyncFieldsUnknownOnResizePlan(ctx, req, resp)
}

// markAsyncFieldsUnknownOnResizePlan ensures plan audit_id/status are unknown when size_gb
// increases so apply can set the new resize audit without inconsistent result errors.
func (r *FileStorageVolumeResource) markAsyncFieldsUnknownOnResizePlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() || req.State.Raw.IsNull() {
		return
	}

	var plan, state FileStorageVolumeResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if plan.SizeGb.IsUnknown() || plan.SizeGb.IsNull() || state.SizeGb.IsUnknown() || state.SizeGb.IsNull() {
		return
	}
	if plan.SizeGb.ValueInt64() <= state.SizeGb.ValueInt64() {
		return
	}

	plan.AuditID = types.StringUnknown()
	plan.Status = types.StringUnknown()
	resp.Diagnostics.Append(resp.Plan.Set(ctx, plan)...)
}

// adoptNasVolumeOutOfBandSizeInPlan aligns plan.size_gb with portal size when the NAS volume grew outside Terraform.
func (r *FileStorageVolumeResource) adoptNasVolumeOutOfBandSizeInPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse, plan FileStorageVolumeResourceModel) {
	if req.Plan.Raw.IsNull() || req.State.Raw.IsNull() {
		return
	}
	if plan.ID.IsUnknown() || plan.ID.IsNull() || plan.SizeGb.IsUnknown() || plan.SizeGb.IsNull() {
		return
	}

	var state FileStorageVolumeResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if state.SizeGb.IsUnknown() || state.SizeGb.IsNull() {
		return
	}

	liveSize := int64(0)
	if !plan.EngagementID.IsUnknown() && !plan.EngagementID.IsNull() &&
		!plan.FileServerID.IsUnknown() && !plan.FileServerID.IsNull() &&
		!plan.Name.IsUnknown() && !plan.Name.IsNull() {
		if detail, err := GetFileStorageVolume(r.client, ctx,
			plan.EngagementID.ValueInt64(),
			plan.FileServerID.ValueString(),
			plan.Name.ValueString(),
		); err == nil && detail.SizeGb > 0 {
			liveSize = detail.SizeGb
		}
	}

	configured := plan.SizeGb.ValueInt64()
	stateSize := state.SizeGb.ValueInt64()
	adopted, changed := adoptNasVolumeOutOfBandSize(configured, stateSize, liveSize)
	if !changed {
		return
	}

	plan.SizeGb = types.Int64Value(adopted)
	tflog.Info(ctx, "Adopted out-of-band NAS volume size for plan",
		map[string]any{
			"volume_id":          plan.ID.ValueString(),
			"configured_size_gb": configured,
			"state_size_gb":      stateSize,
			"live_size_gb":       liveSize,
			"adopted_size_gb":    adopted,
		})
	resp.Diagnostics.Append(resp.Plan.Set(ctx, plan)...)
}

func (r *FileStorageVolumeResource) buildCreateRequest(data FileStorageVolumeResourceModel) (*FileStorageVolumeCreateRequest, error) {
	vserverID, err := ParseVserverID(data.FileServerID.ValueString())
	if err != nil {
		return nil, err
	}
	req := &FileStorageVolumeCreateRequest{
		EndpointID:   data.EndpointID.ValueInt64(),
		EngagementID: data.EngagementID.ValueInt64(),
		VolumeName:   data.Name.ValueString(),
		VolumeSize:   data.SizeGb.ValueInt64(),
		VolumeType:   data.VolumeType.ValueString(),
		VserverID:    vserverID,
		UsageType:    data.UsageType.ValueString(),
		PricingModel: data.PricingModel.ValueString(),
	}
	if !data.FileStorageType.IsNull() && !data.FileStorageType.IsUnknown() {
		if v := strings.TrimSpace(data.FileStorageType.ValueString()); v != "" {
			req.FileStorageType = v
		}
	}
	if !data.Iops.IsNull() && !data.Iops.IsUnknown() && data.Iops.ValueInt64() > 0 {
		req.Iops = int(data.Iops.ValueInt64())
	}
	return req, nil
}

// finalizeComputedFieldsAfterCreate ensures optional Computed attributes are known in state after Create.
// Terraform rejects apply results that still contain unknown values (e.g. omitted iops).
func (r *FileStorageVolumeResource) finalizeComputedFieldsAfterCreate(data *FileStorageVolumeResourceModel, apiReq *FileStorageVolumeCreateRequest) {
	if data.Iops.IsUnknown() || data.Iops.IsNull() {
		if apiReq.Iops > 0 {
			data.Iops = types.Int64Value(int64(apiReq.Iops))
		} else {
			// Portal resolves IOPS from NAS order when omitted; 1 matches P2R default used in examples.
			data.Iops = types.Int64Value(1)
		}
	}
	if (data.VolumeType.IsUnknown() || data.VolumeType.IsNull()) && strings.TrimSpace(apiReq.VolumeType) != "" {
		data.VolumeType = types.StringValue(apiReq.VolumeType)
	}
}

func (r *FileStorageVolumeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data FileStorageVolumeResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.validatePlan(ctx, data); err != nil {
		resp.Diagnostics.AddError("File Storage Volume Validation Failed", err.Error())
		return
	}

	apiReq, err := r.buildCreateRequest(data)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating File Storage Volume", err.Error())
		return
	}

	auditLog, err := CreateFileStorageVolumeAndWait(r.client, ctx, apiReq)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating File Storage Volume", err.Error())
		return
	}

	engID := data.EngagementID.ValueInt64()
	fileServerID := data.FileServerID.ValueString()
	volumeName := data.Name.ValueString()

	rid, resolveErr := ResolveNasVolumeCIAfterCreate(r.client, ctx, engID, fileServerID, volumeName)
	if resolveErr != nil {
		rid = strings.TrimSpace(auditLog.ResourceID.String())
		if rid == "" {
			resp.Diagnostics.AddError("Error Creating File Storage Volume",
				"could not resolve nasVolCi (volCi) from fetchVolumeDetails and audit resourceId is empty: "+resolveErr.Error())
			return
		}
		resp.Diagnostics.AddWarning("NAS Volume ID Resolution",
			fmt.Sprintf("Using audit resourceId %q as volume id; fetchVolumeDetails did not return volCi yet: %v", rid, resolveErr))
	} else if auditRid := strings.TrimSpace(auditLog.ResourceID.String()); auditRid != "" && auditRid != rid {
		tflog.Warn(ctx, "Audit resourceId differs from fetchVolumeDetails volCi; using volCi for attachClient", map[string]any{
			"audit_resource_id": auditRid,
			"nas_vol_ci":        rid,
			"volume":            volumeName,
		})
	}
	if rid == "" {
		resp.Diagnostics.AddError("Error Creating File Storage Volume", "volume id (nasVolCi / volCi) could not be determined after create")
		return
	}

	data.ID = types.StringValue(rid)
	data.AuditID = types.StringValue(auditLog.AuditID)
	data.Status = types.StringValue(auditLog.Status)
	r.finalizeComputedFieldsAfterCreate(&data, apiReq)

	tflog.Info(ctx, "File storage volume created", map[string]any{
		"id": rid, "file_server_id": data.FileServerID.ValueString(), "name": data.Name.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *FileStorageVolumeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state FileStorageVolumeResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	detail, err := GetFileStorageVolumeDetails(r.client, ctx, state.EngagementID.ValueInt64(), state.FileServerID.ValueString(), state.Name.ValueString())
	if err != nil {
		if errors.Is(err, ErrFileStorageVolumeNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddWarning("Read File Storage Volume Failed", err.Error())
		return
	}

	if detail.VolumeID != "" {
		state.ID = types.StringValue(detail.VolumeID)
	}
	if detail.Name != "" {
		state.Name = types.StringValue(detail.Name)
	}
	if detail.SizeGb > 0 {
		state.SizeGb = types.Int64Value(detail.SizeGb)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *FileStorageVolumeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, prior FileStorageVolumeResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
	if resp.Diagnostics.HasError() {
		return
	}

	planSize := plan.SizeGb.ValueInt64()
	priorSize := prior.SizeGb.ValueInt64()
	switch {
	case planSize > priorSize:
		volCi, err := strconv.ParseInt(prior.ID.ValueString(), 10, 64)
		if err != nil {
			resp.Diagnostics.AddError("Error Resizing File Storage Volume", fmt.Sprintf("invalid volume id %q: %v", prior.ID.ValueString(), err))
			return
		}
		resizeReq := &FileStorageVolumeResizeRequest{
			VolumeID:   volCi,
			VolumeSize: planSize,
			Unit:       "GB",
		}
		auditLog, err := ResizeFileStorageVolumeAndWait(r.client, ctx, resizeReq)
		if err != nil {
			resp.Diagnostics.AddError("Error Resizing File Storage Volume", err.Error())
			return
		}
		plan.AuditID = types.StringValue(auditLog.AuditID)
		plan.Status = types.StringValue(auditLog.Status)
		tflog.Info(ctx, "File storage volume resized", map[string]any{"id": prior.ID.ValueString(), "size_gb": planSize})
	case planSize < priorSize:
		resp.Diagnostics.AddError(
			"NAS Volume Size Decrease Not Supported",
			fmt.Sprintf("size_gb %d is less than the current volume size %d GB. Portal/out-of-band growth is adopted automatically; decreasing size via Terraform is not supported.", planSize, priorSize),
		)
		return
	default:
		plan.AuditID = prior.AuditID
		plan.Status = prior.Status
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *FileStorageVolumeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state FileStorageVolumeResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	volCi, err := strconv.ParseInt(state.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Error Deleting File Storage Volume", fmt.Sprintf("invalid volume id %q: %v", state.ID.ValueString(), err))
		return
	}

	if _, err := DeleteFileStorageVolumeAndWait(r.client, ctx, volCi); err != nil {
		resp.Diagnostics.AddError("Error Deleting File Storage Volume", err.Error())
		return
	}

	tflog.Info(ctx, "File storage volume deleted", map[string]any{"id": state.ID.ValueString()})
}

// Import ID format: engagement_id,file_server_id,volume_id,name
// Legacy format file_server_id,volume_id is accepted but name and engagement_id must be set in config for destroy/read.
func (r *FileStorageVolumeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, ",")
	switch len(parts) {
	case 4:
		engStr := strings.TrimSpace(parts[0])
		fs := strings.TrimSpace(parts[1])
		vol := strings.TrimSpace(parts[2])
		name := strings.TrimSpace(parts[3])
		engID, err := strconv.ParseInt(engStr, 10, 64)
		if err != nil {
			resp.Diagnostics.AddError("Invalid Import Identifier", fmt.Sprintf("engagement_id %q is not numeric: %v", engStr, err))
			return
		}
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("engagement_id"), types.Int64Value(engID))...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("file_server_id"), types.StringValue(fs))...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), types.StringValue(vol))...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), types.StringValue(name))...)
	case 2:
		fs := strings.TrimSpace(parts[0])
		vol := strings.TrimSpace(parts[1])
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("file_server_id"), types.StringValue(fs))...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), types.StringValue(vol))...)
	default:
		resp.Diagnostics.AddError(
			"Invalid Import Identifier",
			`Expected "engagement_id,file_server_id,volume_id,name" or legacy "file_server_id,volume_id".`,
		)
	}
}
