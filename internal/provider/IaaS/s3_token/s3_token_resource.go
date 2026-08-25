// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package s3_token

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
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/s3_user"
)

var _ resource.Resource = &S3TokenResource{}
var _ resource.ResourceWithImportState = &S3TokenResource{}
var _ resource.ResourceWithModifyPlan = &S3TokenResource{}

func NewS3TokenResource() resource.Resource {
	return &S3TokenResource{}
}

type S3TokenResource struct {
	client *client.Client
}

type S3TokenResourceModel struct {
	ID types.String `tfsdk:"id"`

	DomainID         types.String `tfsdk:"domain_id"`
	UserName         types.String `tfsdk:"user_name"`
	TokenExpiryDate  types.String `tfsdk:"token_expiry_date"`
	TokenDescription types.String `tfsdk:"token_description"`

	AccessKeyID     types.String `tfsdk:"access_key_id"`
	SecretAccessKey types.String `tfsdk:"secret_access_key"`
	AuditID         types.String `tfsdk:"audit_id"`
}

func (r *S3TokenResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_s3_token"
}

func (r *S3TokenResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an S3 access token for a user within a VayuCloud S3 domain.",
		MarkdownDescription: "Manages an S3 access token for a user within a VayuCloud S3 domain.\n\n" +
			"Token operations complete synchronously. `audit_id` is for audit trail lookup via " +
			"`vayucloud_auditlog_details` only; no polling is required.\n\n" +
			"The parent user must exist in the domain before creating a token.\n\n" +
			"`secret_access_key` is returned **only at create time** and preserved in Terraform state on read/refresh. " +
			"It is **not recoverable** via import or read API.\n\n" +
			"**STANDARD** domains require `token_expiry_date` and `token_description` on create. " +
			"**HIGH_PERFORMANCE** and **AI_STANDARD** allow an empty body.\n\n" +
			"**AI_STANDARD (DDN OSS):** tokens may only be created for `user_name = \"root\"`. " +
			"Token **delete is blocked** — create a new token to rotate credentials.\n\n" +
			"There is **no update** API — changes require replacement.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Composite resource id `{domain_id}:{user_name}:{access_key_id}`.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"domain_id": schema.StringAttribute{
				Description: "Parent S3 domain id. Changing forces a new token.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"user_name": schema.StringAttribute{
				Description: "S3 username the token belongs to (3–128 chars; use `root` on AI_STANDARD). Changing forces a new token.",
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
			"token_expiry_date": schema.StringAttribute{
				Description: "Token expiry date (yyyy-MM-dd). Required on create for STANDARD domains; optional otherwise. Changing forces a new token.",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`),
						"token_expiry_date must be yyyy-MM-dd",
					),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"token_description": schema.StringAttribute{
				Description: "Token description (max 256 chars). Required on create for STANDARD domains; optional otherwise. Changing forces a new token.",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtMost(256),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"access_key_id": schema.StringAttribute{
				Description: "Access key / token id (`access_token` in the API).",
				Computed:    true,
				Sensitive:   true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"secret_access_key": schema.StringAttribute{
				Description: "Secret key returned only at create time (`secret_key` in the API). Preserved in state on refresh.",
				Computed:    true,
				Sensitive:   true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
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

func (r *S3TokenResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *S3TokenResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if r.client == nil {
		return
	}

	var plan S3TokenResourceModel
	if req.Plan.Raw.IsNull() {
		// Destroy plan — block token delete on AI_STANDARD domains.
		if req.State.Raw.IsNull() {
			return
		}
		var state S3TokenResourceModel
		resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
		if resp.Diagnostics.HasError() {
			return
		}
		storageClass, err := s3_user.ResolveDomainStorageClass(r.client, ctx, state.DomainID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Invalid S3 Domain", err.Error())
			return
		}
		if storageClass == s3_user.StorageClassAIStandard {
			resp.Diagnostics.AddError(
				"S3 token delete not supported on AI_STANDARD domains",
				"DDN OSS tokens cannot be deleted. Create a new token to rotate credentials.",
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
	expiry := stringValueOrEmpty(plan.TokenExpiryDate)
	description := stringValueOrEmpty(plan.TokenDescription)

	if err := s3_bucket.ValidateDomainExistsByID(r.client, ctx, domainID); err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("domain_id"), "Invalid S3 Domain", err.Error())
		return
	}

	// Create plan validations.
	if plan.ID.IsUnknown() || plan.ID.IsNull() || plan.ID.ValueString() == "" {
		storageClass, err := s3_user.ResolveDomainStorageClass(r.client, ctx, domainID)
		if err != nil {
			resp.Diagnostics.AddAttributeError(path.Root("domain_id"), "Invalid S3 Domain", err.Error())
			return
		}

		if storageClass == s3_user.StorageClassAIStandard && userName != "root" {
			resp.Diagnostics.AddAttributeError(
				path.Root("user_name"),
				"Invalid user for AI_STANDARD domain",
				"On DDN OSS (AI_STANDARD) domains, tokens may only be created for user_name = \"root\".",
			)
			return
		}

		if err := s3_user.ValidateUserExists(r.client, ctx, domainID, userName); err != nil {
			resp.Diagnostics.AddAttributeError(path.Root("user_name"), "Parent S3 user not found", err.Error())
			return
		}

		if err := ValidateCreateTokenRequest(storageClass, expiry, description); err != nil {
			resp.Diagnostics.AddError("Invalid token create request", err.Error())
		}
		return
	}

	// Existing resource — confirm token still exists when access_key_id is known.
	accessKey := stringValueOrEmpty(plan.AccessKeyID)
	if accessKey == "" {
		return
	}
	if _, _, err := GetToken(r.client, ctx, domainID, userName, accessKey); err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("id"), "Invalid S3 Token", err.Error())
	}
}

func (r *S3TokenResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data S3TokenResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domainID := data.DomainID.ValueString()
	userName := strings.TrimSpace(data.UserName.ValueString())

	createReq := &CreateTokenRequest{
		TokenExpiryDate:  stringValueOrEmpty(data.TokenExpiryDate),
		TokenDescription: stringValueOrEmpty(data.TokenDescription),
	}

	tflog.Debug(ctx, "Creating S3 token", map[string]any{
		"domain_id": domainID,
		"user_name": userName,
	})

	payload, _, err := CreateToken(r.client, ctx, domainID, userName, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating S3 Token", err.Error())
		return
	}

	applyPayloadToResourceModel(payload, &data, true)

	tflog.Info(ctx, "S3 token created successfully", map[string]any{
		"id":            data.ID.ValueString(),
		"domain_id":     domainID,
		"user_name":     data.UserName.ValueString(),
		"access_key_id": data.AccessKeyID.ValueString(),
		"audit_id":      data.AuditID.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *S3TokenResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data S3TokenResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domainID := data.DomainID.ValueString()
	userName := data.UserName.ValueString()
	accessKey := data.AccessKeyID.ValueString()

	if domainID == "" || userName == "" || accessKey == "" {
		if id := data.ID.ValueString(); id != "" {
			parsedDomain, parsedUser, parsedKey, err := ParseTokenID(id)
			if err == nil {
				domainID = parsedDomain
				userName = parsedUser
				accessKey = parsedKey
				data.DomainID = types.StringValue(domainID)
				data.UserName = types.StringValue(userName)
				data.AccessKeyID = types.StringValue(accessKey)
			}
		}
	}

	tflog.Debug(ctx, "Reading S3 token", map[string]any{
		"domain_id":     domainID,
		"user_name":     userName,
		"access_key_id": accessKey,
	})

	existingSecret := data.SecretAccessKey

	payload, _, err := GetToken(r.client, ctx, domainID, userName, accessKey)
	if err != nil {
		if isTokenNotFoundError(err) {
			tflog.Warn(ctx, "S3 token not found, removing from state", map[string]any{
				"domain_id":     domainID,
				"user_name":     userName,
				"access_key_id": accessKey,
			})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading S3 Token", err.Error())
		return
	}

	applyPayloadToResourceModel(payload, &data, false)
	preserveSecretFromState(&data, existingSecret)

	tflog.Info(ctx, "S3 token read successfully", map[string]any{
		"id": data.ID.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *S3TokenResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan S3TokenResourceModel
	var state S3TokenResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domainID := plan.DomainID.ValueString()
	userName := plan.UserName.ValueString()
	accessKey := plan.AccessKeyID.ValueString()
	if accessKey == "" {
		accessKey = state.AccessKeyID.ValueString()
	}

	payload, _, err := GetToken(r.client, ctx, domainID, userName, accessKey)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading S3 Token", err.Error())
		return
	}

	applyPayloadToResourceModel(payload, &plan, false)
	preserveSecretFromState(&plan, state.SecretAccessKey)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *S3TokenResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data S3TokenResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domainID := data.DomainID.ValueString()
	userName := data.UserName.ValueString()
	accessKey := data.AccessKeyID.ValueString()

	tflog.Debug(ctx, "Deleting S3 token", map[string]any{
		"domain_id":     domainID,
		"user_name":     userName,
		"access_key_id": accessKey,
	})

	if envelope, err := DeleteToken(r.client, ctx, domainID, userName, accessKey); err != nil {
		resp.Diagnostics.AddError("Error Deleting S3 Token", err.Error())
		return
	} else if envelope != nil && auditIDFromData(envelope.Data) != "" {
		tflog.Info(ctx, "S3 token delete audit_id", map[string]any{"audit_id": auditIDFromData(envelope.Data)})
	}

	tflog.Info(ctx, "S3 token deleted successfully", map[string]any{
		"domain_id":     domainID,
		"user_name":     userName,
		"access_key_id": accessKey,
	})
}

// ImportState imports an existing token using composite id `{domain_id}:{user_name}:{access_key_id}`.
//
// Usage: terraform import vayucloud_s3_token.example 123:app-service-user:AKIAEXAMPLE
func (r *S3TokenResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	domainID, userName, accessKey, err := ParseTokenID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), FormatTokenID(domainID, userName, accessKey))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain_id"), domainID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("user_name"), userName)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("access_key_id"), accessKey)...)
}

func applyPayloadToResourceModel(payload *TokenPayload, data *S3TokenResourceModel, includeSecret bool) {
	if payload.ID != "" {
		data.ID = types.StringValue(payload.ID)
	} else if payload.DomainID != "" && payload.Username != "" && payload.AccessToken != "" {
		data.ID = types.StringValue(FormatTokenID(payload.DomainID, payload.Username, payload.AccessToken))
	}
	if payload.DomainID != "" {
		data.DomainID = types.StringValue(payload.DomainID)
	}
	if payload.Username != "" {
		data.UserName = types.StringValue(payload.Username)
	}
	if payload.AccessToken != "" {
		data.AccessKeyID = types.StringValue(payload.AccessToken)
	}
	if includeSecret && payload.SecretKey != "" {
		data.SecretAccessKey = types.StringValue(payload.SecretKey)
	}
	if payload.ExpiryDate != "" {
		data.TokenExpiryDate = types.StringValue(NormalizeTokenExpiryDate(payload.ExpiryDate))
	}
	if payload.TokenDescription != "" {
		data.TokenDescription = types.StringValue(payload.TokenDescription)
	}
	if payload.AuditID != "" {
		data.AuditID = types.StringValue(payload.AuditID)
	}
}

func preserveSecretFromState(data *S3TokenResourceModel, existing types.String) {
	if !existing.IsNull() && !existing.IsUnknown() && existing.ValueString() != "" {
		data.SecretAccessKey = existing
	}
}

func stringValueOrEmpty(v types.String) string {
	if v.IsNull() || v.IsUnknown() {
		return ""
	}
	return strings.TrimSpace(v.ValueString())
}
