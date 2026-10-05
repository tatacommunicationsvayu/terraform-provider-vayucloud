// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package s3_bucket

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/client"
)

var _ resource.Resource = &S3BucketResource{}
var _ resource.ResourceWithImportState = &S3BucketResource{}
var _ resource.ResourceWithModifyPlan = &S3BucketResource{}

func NewS3BucketResource() resource.Resource {
	return &S3BucketResource{}
}

type S3BucketResource struct {
	client *client.Client
}

type S3BucketResourceModel struct {
	ID types.String `tfsdk:"id"`

	DomainID   types.String `tfsdk:"domain_id"`
	BucketName types.String `tfsdk:"bucket_name"`

	VersioningEnabled types.Bool   `tfsdk:"versioning_enabled"`
	VersioningStatus  types.String `tfsdk:"versioning_status"`
	CreatedAt         types.String `tfsdk:"created_at"`
	StorageClass      types.String `tfsdk:"storage_class"`
	AuditID           types.String `tfsdk:"audit_id"`
}

func (r *S3BucketResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_s3_bucket"
}

func (r *S3BucketResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an S3 bucket within a VayuCloud S3 domain.",
		MarkdownDescription: "Manages an S3 bucket within a VayuCloud S3 domain.\n\n" +
			"Bucket operations complete synchronously. `audit_id` is for audit trail lookup via " +
			"`vayucloud_auditlog_details` only; no polling is required.\n\n" +
			"The parent `vayucloud_s3_domain` must exist and be active before creating buckets.\n\n" +
			"`bucket_name` is immutable (ForceNew). Only `versioning_enabled` supports in-place update.\n\n" +
			"`storage_class` is domain-derived and cannot be changed on the bucket.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Composite resource id `{domain_id}:{bucket_name}` from the API.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"domain_id": schema.StringAttribute{
				Description: "Parent S3 domain id from `vayucloud_s3_domain.id`. Changing forces a new bucket.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"bucket_name": schema.StringAttribute{
				Description: "Bucket name (3–63 chars; letters, numbers, `.`, `-`; alphanumeric start/end). Stored lowercase.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(3, 63),
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9.-]*[a-zA-Z0-9])?$`),
						"bucket_name must start and end with alphanumeric characters and may contain letters, numbers, dots, and hyphens",
					),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"versioning_enabled": schema.BoolAttribute{
				Description: "Whether object versioning is enabled. Optional in config; defaults to `false` and is always sent on create. On update, `false` suspends versioning when currently enabled.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"versioning_status": schema.StringAttribute{
				Description: "Server versioning state: `Disabled`, `Enabled`, or `Suspended`.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"created_at": schema.StringAttribute{
				Description: "Bucket creation timestamp (ISO-8601 UTC) when returned by the API.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"storage_class": schema.StringAttribute{
				Description: "Domain-derived storage class (e.g. STANDARD, HIGH_PERFORMANCE, AI_STANDARD). Immutable — set by the parent domain, not configurable on the bucket.",
				Computed:    true,
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

func (r *S3BucketResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *S3BucketResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if r.client == nil {
		return
	}

	var plan S3BucketResourceModel
	if req.Plan.Raw.IsNull() {
		resp.Diagnostics.Append(req.State.Get(ctx, &plan)...)
	} else {
		resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.DomainID.IsUnknown() || plan.BucketName.IsUnknown() {
		return
	}

	domainID := plan.DomainID.ValueString()
	bucketName := strings.TrimSpace(plan.BucketName.ValueString())

	// Destroy plan — skip parent checks (bucket may already be gone).
	if req.Plan.Raw.IsNull() {
		return
	}

	if err := ValidateDomainExistsByID(r.client, ctx, domainID); err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("domain_id"), "Invalid S3 Domain", err.Error())
		return
	}

	checkAvailability := plan.ID.IsUnknown() || plan.ID.IsNull() || plan.ID.ValueString() == ""
	if !checkAvailability && !req.State.Raw.IsNull() {
		var state S3BucketResourceModel
		resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
		if resp.Diagnostics.HasError() {
			return
		}
		// ForceNew on domain_id or bucket_name — validate the target name is free, not that it exists.
		if !plan.DomainID.Equal(state.DomainID) || !plan.BucketName.Equal(state.BucketName) {
			checkAvailability = true
		}
	}

	if checkAvailability {
		if err := ValidateBucketNameAvailable(r.client, ctx, domainID, bucketName); err != nil {
			attrPath, summary := bucketValidationDiagnostic(err)
			resp.Diagnostics.AddAttributeError(attrPath, summary, err.Error())
		}
		return
	}

	// Existing resource — confirm the managed bucket still exists.
	if err := ValidateBucketExists(r.client, ctx, domainID, bucketName); err != nil {
		attrPath, summary := bucketValidationDiagnostic(err)
		resp.Diagnostics.AddAttributeError(attrPath, summary, err.Error())
		return
	}

	// Versioning update returns new versioning_status, audit_id, and created_at from the API.
	// Mark them unknown in plan so apply does not expect stale UseStateForUnknown values.
	if req.State.Raw.IsNull() {
		return
	}
	var state S3BucketResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !plan.VersioningEnabled.Equal(state.VersioningEnabled) {
		plan.VersioningStatus = types.StringUnknown()
		plan.AuditID = types.StringUnknown()
		plan.CreatedAt = types.StringUnknown()
		resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
	}
}

func (r *S3BucketResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data S3BucketResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domainID := data.DomainID.ValueString()
	bucketName := strings.TrimSpace(data.BucketName.ValueString())

	tflog.Debug(ctx, "Creating S3 bucket", map[string]any{
		"domain_id":   domainID,
		"bucket_name": bucketName,
	})

	if err := ValidateBucketNameAvailable(r.client, ctx, domainID, bucketName); err != nil {
		attrPath, summary := bucketValidationDiagnostic(err)
		resp.Diagnostics.AddAttributeError(attrPath, summary, err.Error())
		return
	}

	createReq := &CreateBucketRequest{
		BucketName:        bucketName,
		VersioningEnabled: versioningEnabledFromPlan(data.VersioningEnabled),
	}

	payload, _, err := CreateBucket(r.client, ctx, domainID, createReq)
	if err != nil {
		var apiErr *BucketAPIError
		if errors.As(err, &apiErr) && apiErr.PartialCreate {
			resp.Diagnostics.AddWarning(
				"Partial Bucket Create",
				"The bucket may have been created before versioning configuration failed. Check the backend or import the bucket before retrying.",
			)
		}
		summary, detail := bucketOperationDiagnostic("create", err)
		resp.Diagnostics.AddError(summary, detail)
		return
	}

	applyPayloadToResourceModel(payload, &data)

	tflog.Info(ctx, "S3 bucket created successfully", map[string]any{
		"id":          data.ID.ValueString(),
		"domain_id":   domainID,
		"bucket_name": data.BucketName.ValueString(),
		"audit_id":    data.AuditID.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *S3BucketResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data S3BucketResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domainID := data.DomainID.ValueString()
	bucketName := data.BucketName.ValueString()
	if domainID == "" || bucketName == "" {
		if id := data.ID.ValueString(); id != "" {
			parsedDomain, parsedName, err := ParseBucketID(id)
			if err == nil {
				domainID = parsedDomain
				bucketName = parsedName
				data.DomainID = types.StringValue(domainID)
				data.BucketName = types.StringValue(bucketName)
			}
		}
	}

	tflog.Debug(ctx, "Reading S3 bucket", map[string]any{
		"domain_id":   domainID,
		"bucket_name": bucketName,
	})

	payload, _, err := GetBucket(r.client, ctx, domainID, bucketName)
	if err != nil {
		if isBucketNotFoundError(err) {
			tflog.Warn(ctx, "S3 bucket not found, removing from state", map[string]any{
				"domain_id":   domainID,
				"bucket_name": bucketName,
			})
			resp.State.RemoveResource(ctx)
			return
		}
		summary, detail := bucketOperationDiagnostic("read", err)
		resp.Diagnostics.AddError(summary, detail)
		return
	}

	savedAuditID := data.AuditID
	applyPayloadToResourceModel(payload, &data)
	preserveAuditIDFromState(&data, savedAuditID)

	tflog.Info(ctx, "S3 bucket read successfully", map[string]any{
		"id": data.ID.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *S3BucketResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan S3BucketResourceModel
	var state S3BucketResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domainID := plan.DomainID.ValueString()
	bucketName := plan.BucketName.ValueString()

	versioningChanged := !plan.VersioningEnabled.Equal(state.VersioningEnabled)

	tflog.Debug(ctx, "Updating S3 bucket", map[string]any{
		"domain_id":          domainID,
		"bucket_name":        bucketName,
		"versioning_changed": versioningChanged,
	})

	if versioningChanged {
		updateReq := &UpdateBucketRequest{
			VersioningEnabled: plan.VersioningEnabled.ValueBool(),
		}
		payload, _, err := UpdateBucket(r.client, ctx, domainID, bucketName, updateReq)
		if err != nil {
			summary, detail := bucketOperationDiagnostic("update", err)
			resp.Diagnostics.AddError(summary, detail)
			return
		}
		applyPayloadToResourceModel(payload, &plan)
		preserveImmutableBucketFields(&plan, &state)
	} else {
		payload, _, err := GetBucket(r.client, ctx, domainID, bucketName)
		if err != nil {
			summary, detail := bucketOperationDiagnostic("read", err)
			resp.Diagnostics.AddError(summary, detail)
			return
		}
		savedAuditID := plan.AuditID
		applyPayloadToResourceModel(payload, &plan)
		preserveAuditIDFromState(&plan, savedAuditID)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *S3BucketResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data S3BucketResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domainID := data.DomainID.ValueString()
	bucketName := data.BucketName.ValueString()

	tflog.Debug(ctx, "Deleting S3 bucket", map[string]any{
		"domain_id":   domainID,
		"bucket_name": bucketName,
	})

	if envelope, err := DeleteBucket(r.client, ctx, domainID, bucketName); err != nil {
		summary, detail := bucketOperationDiagnostic("delete", err)
		resp.Diagnostics.AddError(summary, detail)
		return
	} else if envelope != nil {
		if auditID := parseDeleteAuditID(envelope); auditID != "" {
			tflog.Info(ctx, "S3 bucket delete audit_id", map[string]any{"audit_id": auditID})
		}
	}

	tflog.Info(ctx, "S3 bucket deleted successfully", map[string]any{
		"domain_id":   domainID,
		"bucket_name": bucketName,
	})
}

// ImportState imports an existing bucket using composite id `{domain_id}:{bucket_name}`.
//
// Usage: terraform import vayucloud_s3_bucket.example 2413:my-bucket
func (r *S3BucketResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	domainID, bucketName, err := ParseBucketID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), FormatBucketID(domainID, bucketName))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain_id"), domainID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("bucket_name"), bucketName)...)
}

func bucketValidationDiagnostic(err error) (path.Path, string) {
	if err == nil {
		return path.Root("bucket_name"), "Bucket validation failed"
	}
	if isDomainNotFoundError(err) || isLikelyInvalidDomainError(err) {
		return path.Root("domain_id"), "Invalid S3 Domain"
	}
	if isBucketAlreadyExistsError(err) {
		return path.Root("bucket_name"), "Bucket name already exists"
	}
	if isBucketNotFoundError(err) {
		return path.Root("bucket_name"), "S3 bucket not found"
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "domain") && strings.Contains(msg, "not found"):
		return path.Root("domain_id"), "Invalid S3 Domain"
	case strings.Contains(msg, "already exists"):
		return path.Root("bucket_name"), "Bucket name already exists"
	case strings.Contains(msg, "was not found"):
		return path.Root("bucket_name"), "S3 bucket not found"
	default:
		return path.Root("bucket_name"), "Bucket validation failed"
	}
}

func bucketOperationDiagnostic(operation string, err error) (summary, detail string) {
	if err == nil {
		return "S3 bucket operation failed", "unknown error"
	}

	var apiErr *BucketAPIError
	if errors.As(err, &apiErr) {
		switch {
		case operation == "create" && isBucketAlreadyExistsError(err):
			return "Bucket Already Exists", apiErr.Error()
		case operation == "delete" && isBucketNotEmptyError(err):
			return "Bucket Not Empty", apiErr.Error()
		case isDomainNotFoundError(err):
			return "Invalid S3 Domain", apiErr.Error()
		case isBucketNotFoundError(err):
			return "S3 Bucket Not Found", apiErr.Error()
		default:
			return operationErrorSummary(operation), apiErr.Error()
		}
	}

	return operationErrorSummary(operation), err.Error()
}

func operationErrorSummary(operation string) string {
	switch operation {
	case "create":
		return "Error Creating S3 Bucket"
	case "update":
		return "Error Updating S3 Bucket"
	case "delete":
		return "Error Deleting S3 Bucket"
	default:
		return "Error Reading S3 Bucket"
	}
}

func preserveAuditIDFromState(data *S3BucketResourceModel, saved types.String) {
	if !saved.IsNull() && saved.ValueString() != "" &&
		(data.AuditID.IsNull() || data.AuditID.ValueString() == "") {
		data.AuditID = saved
	}
}

// versioningEnabledFromPlan returns false when versioning_enabled is omitted or unknown in config.
func versioningEnabledFromPlan(v types.Bool) bool {
	if v.IsNull() || v.IsUnknown() {
		return false
	}
	return v.ValueBool()
}

// preserveImmutableBucketFields keeps domain-derived / create-time fields from state when the API
// omits them on update responses (storage_class is never mutable on the bucket).
func preserveImmutableBucketFields(plan, state *S3BucketResourceModel) {
	if plan.StorageClass.IsNull() || plan.StorageClass.IsUnknown() {
		plan.StorageClass = state.StorageClass
	}
	if plan.CreatedAt.IsNull() || plan.CreatedAt.IsUnknown() {
		plan.CreatedAt = state.CreatedAt
	}
}

func applyPayloadToResourceModel(payload *BucketPayload, data *S3BucketResourceModel) {
	if payload.ID != "" {
		data.ID = types.StringValue(payload.ID)
	} else if payload.DomainID != "" && payload.BucketName != "" {
		data.ID = types.StringValue(FormatBucketID(payload.DomainID, payload.BucketName))
	}
	if payload.DomainID != "" {
		data.DomainID = types.StringValue(payload.DomainID)
	}
	if payload.BucketName != "" {
		data.BucketName = types.StringValue(payload.BucketName)
	}
	data.VersioningStatus = types.StringValue(payload.VersioningStatus)
	data.VersioningEnabled = types.BoolValue(versioningEnabledFromStatus(payload.VersioningStatus))
	if payload.CreatedAt != "" {
		data.CreatedAt = types.StringValue(payload.CreatedAt)
	} else {
		data.CreatedAt = types.StringNull()
	}
	if payload.StorageClass != "" {
		data.StorageClass = types.StringValue(payload.StorageClass)
	} else {
		data.StorageClass = types.StringNull()
	}
	if payload.AuditID != "" {
		data.AuditID = types.StringValue(payload.AuditID)
	}
}
