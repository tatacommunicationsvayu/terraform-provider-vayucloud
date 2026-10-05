// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package s3_object

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"os"
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

var _ resource.Resource = &S3ObjectResource{}
var _ resource.ResourceWithImportState = &S3ObjectResource{}
var _ resource.ResourceWithModifyPlan = &S3ObjectResource{}
var _ resource.ResourceWithValidateConfig = &S3ObjectResource{}

func NewS3ObjectResource() resource.Resource {
	return &S3ObjectResource{}
}

type S3ObjectResource struct {
	client *client.Client
}

type S3ObjectResourceModel struct {
	ID types.String `tfsdk:"id"`

	DomainID   types.String `tfsdk:"domain_id"`
	BucketName types.String `tfsdk:"bucket_name"`
	Key        types.String `tfsdk:"key"`

	Source        types.String `tfsdk:"source"`
	Content       types.String `tfsdk:"content"`
	ContentBase64 types.String `tfsdk:"content_base64"`
	ContentType   types.String `tfsdk:"content_type"`
	SourceHash    types.String `tfsdk:"source_hash"`

	ETag         types.String `tfsdk:"etag"`
	Size         types.Int64  `tfsdk:"size"`
	LastModified types.String `tfsdk:"last_modified"`
	Prefix       types.String `tfsdk:"prefix"`
	FileName     types.String `tfsdk:"file_name"`
	AuditID      types.String `tfsdk:"audit_id"`
}

func (r *S3ObjectResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_s3_object"
}

func (r *S3ObjectResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an S3 object within a VayuCloud S3 bucket.",
		MarkdownDescription: "Manages an S3 object within a VayuCloud S3 bucket.\n\n" +
			"Object operations complete synchronously. `audit_id` is for audit trail lookup via " +
			"`vayucloud_auditlog_details` only; no polling is required.\n\n" +
			"The parent `vayucloud_s3_bucket` must exist before uploading objects.\n\n" +
			"Provide exactly one of `source`, `content`, or `content_base64` for the object body. " +
			"Upload uses raw `PUT` bytes (`object_key` query param). Updates overwrite the same key in place.\n\n" +
			"`domain_id`, `bucket_name`, and `key` are immutable (ForceNew).",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Composite resource id `{domain_id}:{bucket_name}:{object_key}` from the API.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"domain_id": schema.StringAttribute{
				Description: "Parent S3 domain id from `vayucloud_s3_domain.id`. Changing forces a new object.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"bucket_name": schema.StringAttribute{
				Description: "Parent bucket name from `vayucloud_s3_bucket.bucket_name` (3–63 chars). Changing forces a new object.",
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
			"key": schema.StringAttribute{
				Description: "S3 object key (maps to API `object_key`). Must not end with `/`. Changing forces a new object.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 1024),
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^(?:[^/]+/)*[^/]+$`),
						"key must refer to a file, not a folder prefix (trailing '/' is not allowed)",
					),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"source": schema.StringAttribute{
				Description: "Path to a local file whose contents are uploaded. Conflicts with `content` and `content_base64`.",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName("content"), path.MatchRelative().AtParent().AtName("content_base64")),
				},
			},
			"content": schema.StringAttribute{
				Description: "Inline UTF-8 string body. Conflicts with `source` and `content_base64`.",
				Optional:    true,
				Sensitive:   true,
				Validators: []validator.String{
					stringvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName("source"), path.MatchRelative().AtParent().AtName("content_base64")),
				},
			},
			"content_base64": schema.StringAttribute{
				Description: "Base64-encoded object body. Conflicts with `source` and `content`.",
				Optional:    true,
				Sensitive:   true,
				Validators: []validator.String{
					stringvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName("source"), path.MatchRelative().AtParent().AtName("content")),
				},
			},
			"content_type": schema.StringAttribute{
				Description: "MIME type sent as the `Content-Type` header on upload. Defaults to `application/octet-stream`.",
				Optional:    true,
			},
			"source_hash": schema.StringAttribute{
				Description: "Hex SHA-256 hash of the local `source` file at last upload. Used to detect file content changes when the path is unchanged.",
				Computed:    true,
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"etag": schema.StringAttribute{
				Description: "Entity tag from the API (without surrounding quotes).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"size": schema.Int64Attribute{
				Description: "Object size in bytes.",
				Computed:    true,
			},
			"last_modified": schema.StringAttribute{
				Description: "Last modification timestamp (ISO-8601 UTC) from the API.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"prefix": schema.StringAttribute{
				Description: "Parent path prefix derived from `key` by the API.",
				Computed:    true,
			},
			"file_name": schema.StringAttribute{
				Description: "Final path segment derived from `key` by the API.",
				Computed:    true,
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

func (r *S3ObjectResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *S3ObjectResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config S3ObjectResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := validateObjectBodyAttributes(config); err != nil {
		resp.Diagnostics.AddError("Invalid Object Body", err.Error())
	}
}

func (r *S3ObjectResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if r.client == nil {
		return
	}

	var plan S3ObjectResourceModel
	if req.Plan.Raw.IsNull() {
		return
	}
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.DomainID.IsUnknown() || plan.BucketName.IsUnknown() || plan.Key.IsUnknown() {
		return
	}

	domainID := plan.DomainID.ValueString()
	bucketName := strings.TrimSpace(plan.BucketName.ValueString())

	if err := s3_bucket.ValidateDomainExistsByID(r.client, ctx, domainID); err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("domain_id"), "Invalid S3 Domain", err.Error())
		return
	}

	if err := validateObjectBodyAttributes(plan); err != nil {
		resp.Diagnostics.AddError("Invalid Object Body", err.Error())
		return
	}

	if plan.ID.IsUnknown() || plan.ID.IsNull() || plan.ID.ValueString() == "" {
		if err := s3_bucket.ValidateBucketExists(r.client, ctx, domainID, bucketName); err != nil {
			resp.Diagnostics.AddAttributeError(path.Root("bucket_name"), "Parent S3 bucket not found", err.Error())
		}
		if !plan.Source.IsNull() && !plan.Source.IsUnknown() && plan.Source.ValueString() != "" {
			if hash, err := hashSourceFile(plan.Source.ValueString()); err != nil {
				resp.Diagnostics.AddAttributeError(path.Root("source"), "Invalid Source File", err.Error())
			} else {
				plan.SourceHash = types.StringValue(hash)
			}
		} else {
			plan.SourceHash = types.StringNull()
		}
		resp.Diagnostics.Append(resp.Plan.Set(ctx, plan)...)
		return
	}

	if err := s3_bucket.ValidateBucketExists(r.client, ctx, domainID, bucketName); err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("bucket_name"), "Invalid S3 Bucket", err.Error())
		return
	}

	var state S3ObjectResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !plan.Source.IsNull() && !plan.Source.IsUnknown() && plan.Source.ValueString() != "" {
		hash, err := hashSourceFile(plan.Source.ValueString())
		if err != nil {
			resp.Diagnostics.AddAttributeError(path.Root("source"), "Invalid Source File", err.Error())
			return
		}
		plan.SourceHash = types.StringValue(hash)
	} else {
		plan.SourceHash = types.StringNull()
	}

	// A re-upload changes API-derived metadata; mark unknown in plan so apply does not
	// fail with "inconsistent result after apply" (UseStateForUnknown would otherwise
	// keep the previous etag/audit_id/etc. while apply writes new values).
	if shouldReupload(plan, state) {
		plan.ETag = types.StringUnknown()
		plan.Size = types.Int64Unknown()
		plan.LastModified = types.StringUnknown()
		plan.AuditID = types.StringUnknown()
		plan.Prefix = types.StringUnknown()
		plan.FileName = types.StringUnknown()
	}

	resp.Diagnostics.Append(resp.Plan.Set(ctx, plan)...)
}

func (r *S3ObjectResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data S3ObjectResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.uploadFromModel(ctx, &data); err != nil {
		resp.Diagnostics.AddError("Error Creating S3 Object", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *S3ObjectResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data S3ObjectResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domainID, bucketName, objectKey, err := resolveObjectIdentity(&data)
	if err != nil {
		resp.Diagnostics.AddError("Invalid S3 Object State", err.Error())
		return
	}

	tflog.Debug(ctx, "Reading S3 object", map[string]any{
		"domain_id":   domainID,
		"bucket_name": bucketName,
		"object_key":  objectKey,
	})

	payload, _, err := GetObject(r.client, ctx, domainID, bucketName, objectKey)
	if err != nil {
		if isObjectNotFoundError(err) {
			tflog.Warn(ctx, "S3 object not found, removing from state", map[string]any{
				"domain_id":   domainID,
				"bucket_name": bucketName,
				"object_key":  objectKey,
			})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading S3 Object", err.Error())
		return
	}

	applyPayloadToResourceModel(payload, &data, true)
	// Preserve source_hash from state (hash at last upload). Do not re-hash the
	// local file on read — that would hide content changes when source path is unchanged.
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *S3ObjectResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state S3ObjectResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if shouldReupload(plan, state) {
		if err := r.uploadFromModel(ctx, &plan); err != nil {
			resp.Diagnostics.AddError("Error Updating S3 Object", err.Error())
			return
		}
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
		return
	}

	domainID, bucketName, objectKey, err := resolveObjectIdentity(&plan)
	if err != nil {
		resp.Diagnostics.AddError("Invalid S3 Object State", err.Error())
		return
	}

	payload, _, err := GetObject(r.client, ctx, domainID, bucketName, objectKey)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading S3 Object", err.Error())
		return
	}

	applyPayloadToResourceModel(payload, &plan, true)
	preserveBodyFields(&plan, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *S3ObjectResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data S3ObjectResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domainID, bucketName, objectKey, err := resolveObjectIdentity(&data)
	if err != nil {
		resp.Diagnostics.AddError("Invalid S3 Object State", err.Error())
		return
	}

	tflog.Debug(ctx, "Deleting S3 object", map[string]any{
		"domain_id":   domainID,
		"bucket_name": bucketName,
		"object_key":  objectKey,
	})

	if _, err := DeleteObject(r.client, ctx, domainID, bucketName, objectKey); err != nil {
		resp.Diagnostics.AddError("Error Deleting S3 Object", err.Error())
		return
	}

	tflog.Info(ctx, "S3 object deleted successfully", map[string]any{
		"domain_id":   domainID,
		"bucket_name": bucketName,
		"object_key":  objectKey,
	})
}

func (r *S3ObjectResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	domainID, bucketName, objectKey, err := ParseObjectID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), FormatObjectID(domainID, bucketName, objectKey))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain_id"), domainID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("bucket_name"), bucketName)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("key"), objectKey)...)
}

func (r *S3ObjectResource) uploadFromModel(ctx context.Context, data *S3ObjectResourceModel) error {
	domainID := data.DomainID.ValueString()
	bucketName := strings.TrimSpace(data.BucketName.ValueString())
	objectKey := normalizeObjectKey(data.Key.ValueString())

	body, length, contentType, err := resolveUploadBody(data)
	if err != nil {
		return err
	}
	if closer, ok := body.(io.Closer); ok {
		defer closer.Close()
	}

	tflog.Debug(ctx, "Uploading S3 object", map[string]any{
		"domain_id":   domainID,
		"bucket_name": bucketName,
		"object_key":  objectKey,
	})

	payload, _, err := UploadObject(r.client, ctx, domainID, bucketName, objectKey, contentType, body, length)
	if err != nil {
		return err
	}

	applyPayloadToResourceModel(payload, data, false)
	setSourceHashField(data)

	tflog.Info(ctx, "S3 object uploaded successfully", map[string]any{
		"id":         data.ID.ValueString(),
		"object_key": data.Key.ValueString(),
		"audit_id":   data.AuditID.ValueString(),
	})

	return nil
}

func resolveObjectIdentity(data *S3ObjectResourceModel) (domainID, bucketName, objectKey string, err error) {
	domainID = data.DomainID.ValueString()
	bucketName = data.BucketName.ValueString()
	objectKey = data.Key.ValueString()

	if domainID != "" && bucketName != "" && objectKey != "" {
		return domainID, bucketName, normalizeObjectKey(objectKey), nil
	}

	if id := data.ID.ValueString(); id != "" {
		return ParseObjectID(id)
	}

	return "", "", "", fmt.Errorf("domain_id, bucket_name, and key are required")
}

func validateObjectBodyAttributes(data S3ObjectResourceModel) error {
	set := 0
	if !data.Source.IsNull() && !data.Source.IsUnknown() && data.Source.ValueString() != "" {
		set++
	}
	if !data.Content.IsNull() && !data.Content.IsUnknown() {
		set++
	}
	if !data.ContentBase64.IsNull() && !data.ContentBase64.IsUnknown() && data.ContentBase64.ValueString() != "" {
		set++
	}
	if set != 1 {
		return fmt.Errorf("exactly one of source, content, or content_base64 must be set")
	}
	return nil
}

func resolveUploadBody(data *S3ObjectResourceModel) (io.Reader, int64, string, error) {
	contentType := defaultObjectContentType
	if !data.ContentType.IsNull() && !data.ContentType.IsUnknown() && data.ContentType.ValueString() != "" {
		contentType = data.ContentType.ValueString()
	}

	if !data.Source.IsNull() && !data.Source.IsUnknown() && data.Source.ValueString() != "" {
		path := data.Source.ValueString()
		info, err := os.Stat(path)
		if err != nil {
			return nil, 0, "", fmt.Errorf("failed to stat source file %q: %w", path, err)
		}
		if info.IsDir() {
			return nil, 0, "", fmt.Errorf("source %q is a directory, expected a file", path)
		}
		file, err := os.Open(path)
		if err != nil {
			return nil, 0, "", fmt.Errorf("failed to open source file %q: %w", path, err)
		}
		return file, info.Size(), contentType, nil
	}

	if !data.Content.IsNull() && !data.Content.IsUnknown() {
		raw := []byte(data.Content.ValueString())
		return io.NopCloser(bytes.NewReader(raw)), int64(len(raw)), contentType, nil
	}

	if !data.ContentBase64.IsNull() && !data.ContentBase64.IsUnknown() {
		raw, err := base64.StdEncoding.DecodeString(data.ContentBase64.ValueString())
		if err != nil {
			return nil, 0, "", fmt.Errorf("invalid content_base64: %w", err)
		}
		return io.NopCloser(bytes.NewReader(raw)), int64(len(raw)), contentType, nil
	}

	return nil, 0, "", fmt.Errorf("exactly one of source, content, or content_base64 must be set")
}

func setSourceHashField(data *S3ObjectResourceModel) {
	if !data.Source.IsNull() && !data.Source.IsUnknown() && data.Source.ValueString() != "" {
		if hash, err := hashSourceFile(data.Source.ValueString()); err == nil {
			data.SourceHash = types.StringValue(hash)
			return
		}
	}
	data.SourceHash = types.StringNull()
}

func hashSourceFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("failed to open source file %q: %w", path, err)
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("failed to hash source file %q: %w", path, err)
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func shouldReupload(plan, state S3ObjectResourceModel) bool {

	plan.ETag = types.StringUnknown()
	plan.Size = types.Int64Unknown()
	plan.LastModified = types.StringUnknown()
	plan.AuditID = types.StringUnknown()
	plan.Prefix = types.StringUnknown()
	plan.FileName = types.StringUnknown()
	return false
}

func applyPayloadToResourceModel(payload *ObjectPayload, data *S3ObjectResourceModel, preserveBody bool) {
	if payload == nil {
		return
	}

	if payload.ID != "" {
		data.ID = types.StringValue(payload.ID)
	} else if payload.DomainID != "" && payload.BucketName != "" && payload.ObjectKey != "" {
		data.ID = types.StringValue(FormatObjectID(payload.DomainID, payload.BucketName, payload.ObjectKey))
	}
	if payload.DomainID != "" {
		data.DomainID = types.StringValue(payload.DomainID)
	}
	if payload.BucketName != "" {
		data.BucketName = types.StringValue(payload.BucketName)
	}
	if payload.ObjectKey != "" {
		data.Key = types.StringValue(payload.ObjectKey)
	}
	if payload.ETag != "" {
		data.ETag = types.StringValue(payload.ETag)
	}
	data.Size = types.Int64Value(payload.Size)
	if payload.LastModified != "" {
		data.LastModified = types.StringValue(payload.LastModified)
	}
	if payload.Prefix != "" {
		data.Prefix = types.StringValue(payload.Prefix)
	} else {
		data.Prefix = types.StringNull()
	}
	if payload.FileName != "" {
		data.FileName = types.StringValue(payload.FileName)
	} else {
		data.FileName = types.StringNull()
	}
	if !preserveBody && payload.ContentType != "" && (data.ContentType.IsNull() || data.ContentType.IsUnknown() || data.ContentType.ValueString() == "") {
		data.ContentType = types.StringValue(payload.ContentType)
	}
	if payload.AuditID != "" {
		data.AuditID = types.StringValue(payload.AuditID)
	}
}

func preserveBodyFields(plan, state *S3ObjectResourceModel) {
	plan.Source = state.Source
	plan.Content = state.Content
	plan.ContentBase64 = state.ContentBase64
	if plan.ContentType.IsNull() || plan.ContentType.IsUnknown() {
		plan.ContentType = state.ContentType
	}
	if plan.Source.IsNull() || plan.Source.IsUnknown() || plan.Source.ValueString() == "" {
		plan.SourceHash = types.StringNull()
	} else {
		plan.SourceHash = state.SourceHash
	}
}
