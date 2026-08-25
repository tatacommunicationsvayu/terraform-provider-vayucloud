// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package file_server

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
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
)

var (
	_ resource.Resource                = &FileServerResource{}
	_ resource.ResourceWithImportState = &FileServerResource{}
	_ resource.ResourceWithModifyPlan  = &FileServerResource{}
)

// NewFileServerResource registers the file_server resource.
func NewFileServerResource() resource.Resource {
	return &FileServerResource{}
}

// FileServerResource manages a NAS vserver: create (createNasVserver) and delete only; there is no resize API.
type FileServerResource struct {
	client *client.Client
}

// FileServerResourceModel maps Terraform state to API payloads.
type FileServerResourceModel struct {
	ID              types.String `tfsdk:"id"`
	EngagementID    types.Int64  `tfsdk:"engagement_id"`
	EndpointID      types.Int64  `tfsdk:"endpoint_id"`
	VserverName     types.String `tfsdk:"vserver_name"`
	FileStorageType types.String `tfsdk:"file_storage_type"`
	AuditID         types.String `tfsdk:"audit_id"`
	Status          types.String `tfsdk:"status"`
}

func (r *FileServerResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_file_server"
}

func (r *FileServerResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Creates and deletes a NAS vserver (file server). No resize operation is supported.",
		MarkdownDescription: "Creates a NAS vserver via `POST .../nas/createNasVserver/{engagement_id}` and deletes it on destroy via `DELETE .../nas/vservers/{vserver_id}`. Changing `engagement_id`, `endpoint_id`, `vserver_name`, or `file_storage_type` forces replacement (there is no in-place resize or update API for the vserver).\n\n**Import:** use `terraform import ... ID` with either the numeric vserver id only, or `ENGAGEMENT_ID/VSERVER_ID` so `engagement_id` is stored in state.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Platform NAS File Server id after create; must be numeric (used as vserverId in the delete API body).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"engagement_id": schema.Int64Attribute{
				Description: "Engagement ID; must match the path segment in createNasVserver/{engagement_id} and the request body.",
				Required:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
				},
			},
			"endpoint_id": schema.Int64Attribute{
				Description: "Endpoint ID for the NAS vserver.",
				Required:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
				},
			},
			"vserver_name": schema.StringAttribute{
				Description: "vserverName in the API (File Server Name).",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 256),
				},
			},
			"file_storage_type": schema.StringAttribute{
				Description: "Storage protocol: NAS-NFS or CIFS. Defaults to NAS-NFS; portal may resolve order details when omitted from volume create.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("NAS-NFS"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf("NAS-NFS", "CIFS"),
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

func (r *FileServerResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// ModifyPlan validates engagement and endpoint during terraform plan.
func (r *FileServerResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if r.client == nil {
		return
	}

	var plan FileServerResourceModel
	if req.Plan.Raw.IsNull() {
		resp.Diagnostics.Append(req.State.Get(ctx, &plan)...)
	} else {
		resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.EngagementID.IsUnknown() || plan.EndpointID.IsUnknown() || plan.FileStorageType.IsUnknown() || plan.FileStorageType.IsNull() {
		return
	}

	if err := ValidateNASOrderForFileServerCreate(r.client, ctx,
		plan.EngagementID.ValueInt64(),
		plan.EndpointID.ValueInt64(),
		plan.FileStorageType.ValueString(),
	); err != nil {
		resp.Diagnostics.AddError("NAS Order Validation Failed", err.Error())
	}
}

func (r *FileServerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data FileServerResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := ValidateNASOrderForFileServerCreate(r.client, ctx,
		data.EngagementID.ValueInt64(),
		data.EndpointID.ValueInt64(),
		data.FileStorageType.ValueString(),
	); err != nil {
		resp.Diagnostics.AddError("NAS Order Validation Failed", err.Error())
		return
	}

	apiReq := &FileServerCreateRequest{
		EndpointID:      data.EndpointID.ValueInt64(),
		EngagementID:    data.EngagementID.ValueInt64(),
		VserverName:     data.VserverName.ValueString(),
		FileStorageType: data.FileStorageType.ValueString(),
	}

	auditLog, err := CreateFileServerAndWait(r.client, ctx, apiReq)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating File Server", err.Error())
		return
	}

	rid := auditLog.ResourceID.String()
	if rid == "" {
		resp.Diagnostics.AddError("Error Creating File Server", "ResourceID from audit log is empty")
		return
	}

	data.ID = types.StringValue(rid)
	data.AuditID = types.StringValue(auditLog.AuditID)
	data.Status = types.StringValue(auditLog.Status)

	tflog.Info(ctx, "File server created", map[string]any{"id": rid})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *FileServerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state FileServerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueString()
	detail, err := ReadFileServerDetail(r.client, ctx, id)
	if err != nil {
		if errors.Is(err, ErrFileServerNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddWarning("Read File Server Failed", err.Error())
		return
	}

	vserver := detail.FileServerName
	if vserver != "" {
		state.VserverName = types.StringValue(vserver)
	}
	if detail.EndpointID != 0 {
		state.EndpointID = types.Int64Value(detail.EndpointID)
	}
	if detail.EngagementID != 0 {
		state.EngagementID = types.Int64Value(detail.EngagementID)
	}
	if ft := canonicalFileServerStorageType(detail.FileStorageType); ft != "" {
		state.FileStorageType = types.StringValue(ft)
	} else if prior := canonicalFileServerStorageType(state.FileStorageType.ValueString()); prior != "" {
		// Refresh API (e.g. fetchVserverDetails) may return component type only; keep last known good value.
		state.FileStorageType = types.StringValue(prior)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *FileServerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, prior FileServerResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if plan.AuditID.IsUnknown() || plan.AuditID.IsNull() {
		plan.AuditID = prior.AuditID
	}
	if plan.Status.IsUnknown() || plan.Status.IsNull() {
		plan.Status = prior.Status
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *FileServerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state FileServerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	idStr := state.ID.ValueString()
	vserverID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting File Server",
			fmt.Sprintf("id must be a numeric vserver id for DELETE /nas/vservers/{vserverId}: %q", idStr),
		)
		return
	}

	if _, err := DeleteFileServerAndWait(r.client, ctx, vserverID); err != nil {
		resp.Diagnostics.AddError("Error Deleting File Server", err.Error())
		return
	}

	tflog.Info(ctx, "File server deleted", map[string]any{"vserver_id": vserverID})
}

func (r *FileServerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id := strings.TrimSpace(req.ID)
	if id == "" {
		resp.Diagnostics.AddError("Error Importing File Server", "import id is empty")
		return
	}
	// Composite ID: ENGAGEMENT_ID/VSERVER_ID — stores engagement_id in state so destroy and outputs work
	// when refresh APIs are unavailable or import only had the vserver id before.
	if strings.Contains(id, "/") {
		parts := strings.SplitN(id, "/", 2)
		if len(parts) != 2 {
			resp.Diagnostics.AddError("Error Importing File Server", `composite id must be "ENGAGEMENT_ID/VSERVER_ID"`)
			return
		}
		engStr, vserverStr := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
		if engStr == "" || vserverStr == "" {
			resp.Diagnostics.AddError("Error Importing File Server", `composite id must be "ENGAGEMENT_ID/VSERVER_ID" with non-empty parts`)
			return
		}
		eng, err := strconv.ParseInt(engStr, 10, 64)
		if err != nil || eng < 1 {
			resp.Diagnostics.AddError("Error Importing File Server", fmt.Sprintf("invalid engagement id %q in composite import id", engStr))
			return
		}
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("engagement_id"), types.Int64Value(eng))...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), types.StringValue(vserverStr))...)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), types.StringValue(id))...)
}
