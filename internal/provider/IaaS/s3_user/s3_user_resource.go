// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package s3_user

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/client"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/s3_bucket"
)

var _ resource.Resource = &S3UserResource{}
var _ resource.ResourceWithImportState = &S3UserResource{}
var _ resource.ResourceWithModifyPlan = &S3UserResource{}

func NewS3UserResource() resource.Resource {
	return &S3UserResource{}
}

type S3UserResource struct {
	client *client.Client
}

type S3UserResourceModel struct {
	ID types.String `tfsdk:"id"`

	DomainID types.String `tfsdk:"domain_id"`
	UserName types.String `tfsdk:"user_name"`
	AuditID  types.String `tfsdk:"audit_id"`
}

func (r *S3UserResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_s3_user"
}

func (r *S3UserResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an S3 service user within a VayuCloud S3 domain.",
		MarkdownDescription: "Manages an S3 service user within a VayuCloud S3 domain.\n\n" +
			"User operations complete synchronously. `audit_id` is for audit trail lookup via " +
			"`vayucloud_auditlog_details` only; no polling is required.\n\n" +
			"The parent `vayucloud_s3_domain` must exist and be active.\n\n" +
			"`user_name` is immutable (ForceNew). There is **no update** API — changes require replacement.\n\n" +
			"Deleting a user **cascades** and removes all tokens for that user.\n\n" +
			"**AI_STANDARD (DDN OSS) domains:** user create/delete is **not supported** — use the `root` user " +
			"provisioned at domain creation and manage tokens only via `vayucloud_s3_token`.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Composite resource id `{domain_id}:{user_name}` from the API.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"domain_id": schema.StringAttribute{
				Description: "Parent S3 domain id from `vayucloud_s3_domain.id`. Changing forces a new user.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"user_name": schema.StringAttribute{
				Description: "S3 username (3–128 chars; alphanumeric start; letters, numbers, `.`, `_`, `@`, `-`). Sent as `username` to the API.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(3, 128),
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._@-]*$`),
						"user_name must start with alphanumeric and may contain letters, numbers, dots, underscores, at-signs, and hyphens",
					),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"audit_id": schema.StringAttribute{
				Description: "Audit identifier for the last create, update, or delete API call tracked by this resource.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *S3UserResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}
	r.client = c
}

func (r *S3UserResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if r.client == nil {
		return
	}

	var plan S3UserResourceModel
	if req.Plan.Raw.IsNull() {
		// Destroy plan — block user delete on AI_STANDARD domains.
		if req.State.Raw.IsNull() {
			return
		}
		var state S3UserResourceModel
		resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
		if resp.Diagnostics.HasError() {
			return
		}
		storageClass, err := ResolveDomainStorageClass(r.client, ctx, state.DomainID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Invalid S3 Domain", err.Error())
			return
		}
		if storageClass == StorageClassAIStandard {
			resp.Diagnostics.AddError(
				"S3 user delete not supported on AI_STANDARD domains",
				"DDN OSS domains provision the root user at domain creation. Manage tokens only via vayucloud_s3_token with user_name = \"root\".",
			)
		}
		return
	}
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.DomainID.IsUnknown() || plan.UserName.IsUnknown() {
		return
	}

	domainID := plan.DomainID.ValueString()
	userName := strings.TrimSpace(plan.UserName.ValueString())

	if err := s3_bucket.ValidateDomainExistsByID(r.client, ctx, domainID); err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("domain_id"), "Invalid S3 Domain", err.Error())
		return
	}

	// Create plan — block AI_STANDARD domains and duplicate usernames.
	if plan.ID.IsUnknown() || plan.ID.IsNull() || plan.ID.ValueString() == "" {
		storageClass, err := ResolveDomainStorageClass(r.client, ctx, domainID)
		if err != nil {
			resp.Diagnostics.AddAttributeError(path.Root("domain_id"), "Invalid S3 Domain", err.Error())
			return
		}
		if storageClass == StorageClassAIStandard {
			resp.Diagnostics.AddAttributeError(
				path.Root("domain_id"),
				"S3 user create not supported on AI_STANDARD domains",
				"DDN OSS domains provision the root user at domain creation. Use vayucloud_s3_token with user_name = \"root\" instead.",
			)
			return
		}

		if err := ValidateUserNameAvailable(r.client, ctx, domainID, userName); err != nil {
			resp.Diagnostics.AddAttributeError(path.Root("user_name"), "Username not available", err.Error())
		}
		return
	}

	// Existing resource — confirm user still exists.
	if err := ValidateUserExists(r.client, ctx, domainID, userName); err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("id"), "Invalid S3 User", err.Error())
	}
}

func (r *S3UserResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data S3UserResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domainID := data.DomainID.ValueString()
	userName := strings.TrimSpace(data.UserName.ValueString())

	tflog.Debug(ctx, "Creating S3 user", map[string]any{
		"domain_id": domainID,
		"user_name": userName,
	})

	if err := ValidateUserNameAvailable(r.client, ctx, domainID, userName); err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("user_name"),
			"Username not available",
			err.Error(),
		)
		return
	}

	createReq := &CreateUserRequest{Username: userName}

	payload, _, err := CreateUser(r.client, ctx, domainID, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating S3 User", err.Error())
		return
	}

	applyPayloadToResourceModel(payload, &data)

	tflog.Info(ctx, "S3 user created successfully", map[string]any{
		"id":        data.ID.ValueString(),
		"domain_id": domainID,
		"user_name": data.UserName.ValueString(),
		"audit_id":  data.AuditID.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *S3UserResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data S3UserResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domainID := data.DomainID.ValueString()
	userName := data.UserName.ValueString()
	if domainID == "" || userName == "" {
		if id := data.ID.ValueString(); id != "" {
			parsedDomain, parsedName, err := ParseUserID(id)
			if err == nil {
				domainID = parsedDomain
				userName = parsedName
				data.DomainID = types.StringValue(domainID)
				data.UserName = types.StringValue(userName)
			}
		}
	}

	tflog.Debug(ctx, "Reading S3 user", map[string]any{
		"domain_id": domainID,
		"user_name": userName,
	})

	payload, _, err := GetUser(r.client, ctx, domainID, userName)
	if err != nil {
		if isUserNotFoundError(err) {
			tflog.Warn(ctx, "S3 user not found, removing from state", map[string]any{
				"domain_id": domainID,
				"user_name": userName,
			})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading S3 User", err.Error())
		return
	}

	applyPayloadToResourceModel(payload, &data)

	tflog.Info(ctx, "S3 user read successfully", map[string]any{
		"id": data.ID.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *S3UserResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan S3UserResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// No mutable fields — refresh from API.
	domainID := plan.DomainID.ValueString()
	userName := plan.UserName.ValueString()

	payload, _, err := GetUser(r.client, ctx, domainID, userName)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading S3 User", err.Error())
		return
	}

	applyPayloadToResourceModel(payload, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *S3UserResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data S3UserResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domainID := data.DomainID.ValueString()
	userName := data.UserName.ValueString()

	tflog.Debug(ctx, "Deleting S3 user", map[string]any{
		"domain_id": domainID,
		"user_name": userName,
	})

	if envelope, err := DeleteUser(r.client, ctx, domainID, userName); err != nil {
		resp.Diagnostics.AddError("Error Deleting S3 User", err.Error())
		return
	} else if envelope != nil && auditIDFromData(envelope.Data) != "" {
		tflog.Info(ctx, "S3 user delete audit_id", map[string]any{"audit_id": auditIDFromData(envelope.Data)})
	}

	tflog.Info(ctx, "S3 user deleted successfully", map[string]any{
		"domain_id": domainID,
		"user_name": userName,
	})
}

// ImportState imports an existing user using composite id `{domain_id}:{user_name}`.
//
// Usage: terraform import vayucloud_s3_user.example 123:app-service-user
func (r *S3UserResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	domainID, userName, err := ParseUserID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), FormatUserID(domainID, userName))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain_id"), domainID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("user_name"), userName)...)
}

func applyPayloadToResourceModel(payload *UserPayload, data *S3UserResourceModel) {
	if payload.ID != "" {
		data.ID = types.StringValue(payload.ID)
	} else if payload.DomainID != "" && payload.Username != "" {
		data.ID = types.StringValue(FormatUserID(payload.DomainID, payload.Username))
	}
	if payload.DomainID != "" {
		data.DomainID = types.StringValue(payload.DomainID)
	}
	if payload.Username != "" {
		data.UserName = types.StringValue(payload.Username)
	}
	if payload.AuditID != "" {
		data.AuditID = types.StringValue(payload.AuditID)
	}
}
