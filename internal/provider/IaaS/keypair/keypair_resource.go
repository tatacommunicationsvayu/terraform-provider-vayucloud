// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package keypair

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/client"
)

// Ensure provider-defined types fully satisfy framework interfaces.
var _ resource.Resource = &KeypairResource{}
var _ resource.ResourceWithImportState = &KeypairResource{}

// NewKeypairResource creates a new keypair resource.
func NewKeypairResource() resource.Resource {
	return &KeypairResource{}
}

// KeypairResource defines the resource implementation.
type KeypairResource struct {
	client *client.Client
}

// KeypairResourceModel describes the resource data model.
// This maps the Terraform schema to Go types.
type KeypairResourceModel struct {
	// ID is the unique identifier of the keypair (computed after creation)
	ID types.String `tfsdk:"id"`

	// Name is the name of the keypair
	Name types.String `tfsdk:"name"`

	// EngagementID is the engagement ID associated with the keypair
	EngagementID types.String `tfsdk:"engagement_id"`

	// Mode determines whether to create a new keypair or upload an existing public key
	// Values: "create" or "upload"
	Mode types.String `tfsdk:"mode"`

	// KeypairType is the type of keypair (e.g., "RSA", "ED25519") - used only for create mode
	KeypairType types.String `tfsdk:"keypair_type"`

	// PrivateKeyFileFormat is the format for the private key file ("pem" or "ppk") - used only for create mode
	PrivateKeyFileFormat types.String `tfsdk:"private_key_file_format"`

	// PublicKey is the public key content to upload - used only for upload mode
	PublicKey types.String `tfsdk:"public_key"`

	// PrivateKeyContent contains the private key content (computed, only for create mode)
	// This is sensitive and will be stored in state
	PrivateKeyContent types.String `tfsdk:"private_key_content"`

	// PrivateKeyFilename is the filename of the private key (computed, only for create mode)
	PrivateKeyFilename types.String `tfsdk:"private_key_filename"`

	// Fingerprint is the key fingerprint (computed)
	Fingerprint types.String `tfsdk:"fingerprint"`

	// Status is the response status from the API (computed)
	Status types.String `tfsdk:"status"`

	// Message is the response message from the API (computed)
	Message types.String `tfsdk:"message"`
}

// Metadata returns the resource type name.
func (r *KeypairResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_keypair"
}

// Schema defines the schema for the keypair resource.
func (r *KeypairResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Manages a keypair resource in VayuCloud.",
		MarkdownDescription: "Manages a keypair resource in VayuCloud.\n\nA keypair can be created (generating a new key pair) or uploaded (using an existing public key).",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:         "The unique identifier of the keypair.",
				MarkdownDescription: "The unique identifier of the keypair.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description:         "The name of the keypair. Must be unique within the engagement.",
				MarkdownDescription: "The name of the keypair. Must be unique within the engagement.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"engagement_id": schema.StringAttribute{
				Description:         "The engagement ID associated with the keypair.",
				MarkdownDescription: "The engagement ID associated with the keypair.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"mode": schema.StringAttribute{
				Description:         "The mode of keypair operation. Must be 'create' (generate new keypair) or 'upload' (upload existing public key).",
				MarkdownDescription: "The mode of keypair operation. Must be `create` (generate new keypair) or `upload` (upload existing public key).",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"keypair_type": schema.StringAttribute{
				Description:         "The type of keypair (e.g., 'RSA', 'ED25519'). Required when mode is 'create'.",
				MarkdownDescription: "The type of keypair (e.g., `RSA`, `ED25519`). Required when mode is `create`.",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"private_key_file_format": schema.StringAttribute{
				Description:         "The format for the private key file ('pem' or 'ppk'). Required when mode is 'create'.",
				MarkdownDescription: "The format for the private key file (`pem` or `ppk`). Required when mode is `create`.",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"public_key": schema.StringAttribute{
				Description:         "The public key content to upload. Required when mode is 'upload'.",
				MarkdownDescription: "The public key content to upload. Required when mode is `upload`.",
				Optional:            true,
				Sensitive:           true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"private_key_content": schema.StringAttribute{
				Description:         "The private key content generated by the API. Only populated when mode is 'create'.",
				MarkdownDescription: "The private key content generated by the API. Only populated when mode is `create`.",
				Computed:            true,
				Sensitive:           true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"private_key_filename": schema.StringAttribute{
				Description:         "The filename of the private key. Only populated when mode is 'create'.",
				MarkdownDescription: "The filename of the private key. Only populated when mode is `create`.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"fingerprint": schema.StringAttribute{
				Description:         "The fingerprint of the keypair.",
				MarkdownDescription: "The fingerprint of the keypair.",
				Computed:            true,
			},
			"status": schema.StringAttribute{
				Description:         "The status from the API response.",
				MarkdownDescription: "The status from the API response.",
				Computed:            true,
			},
			"message": schema.StringAttribute{
				Description:         "The message from the API response.",
				MarkdownDescription: "The message from the API response.",
				Computed:            true,
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *KeypairResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

// Create creates a new keypair resource.
func (r *KeypairResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data KeypairResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating keypair", map[string]any{
		"name":          data.Name.ValueString(),
		"engagement_id": data.EngagementID.ValueString(),
		"mode":          data.Mode.ValueString(),
	})

	mode := data.Mode.ValueString()

	if mode == "create" {
		// Validate required fields for create mode
		if data.KeypairType.IsNull() || data.KeypairType.ValueString() == "" {
			resp.Diagnostics.AddError(
				"Missing Required Field",
				"keypair_type is required when mode is 'create'",
			)
			return
		}
		if data.PrivateKeyFileFormat.IsNull() || data.PrivateKeyFileFormat.ValueString() == "" {
			resp.Diagnostics.AddError(
				"Missing Required Field",
				"private_key_file_format is required when mode is 'create'",
			)
			return
		}

		// Create a new keypair
		createReq := &KeypairCreateRequest{
			KeypairName:          data.Name.ValueString(),
			EngagementID:         data.EngagementID.ValueString(),
			KeypairType:          data.KeypairType.ValueString(),
			PrivateKeyFileFormat: data.PrivateKeyFileFormat.ValueString(),
		}

		createResp, err := CreateKeypair(r.client, ctx, createReq)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Creating Keypair",
				"Could not create keypair: "+err.Error(),
			)
			return
		}

		// Store the private key content
		data.PrivateKeyContent = types.StringValue(createResp.PrivateKeyContent)
		data.PrivateKeyFilename = types.StringValue(createResp.Filename)
		data.Status = types.StringValue("success")
		data.Message = types.StringValue("Keypair created successfully")

	} else if mode == "upload" {
		// Validate required fields for upload mode
		if data.PublicKey.IsNull() || data.PublicKey.ValueString() == "" {
			resp.Diagnostics.AddError(
				"Missing Required Field",
				"public_key is required when mode is 'upload'",
			)
			return
		}

		// Upload an existing public key
		uploadReq := &KeypairUploadRequest{
			KeypairName:        data.Name.ValueString(),
			EncryptedPublicKey: data.PublicKey.ValueString(),
			EngagementID:       data.EngagementID.ValueString(),
		}

		uploadResp, err := UploadKeypair(r.client, ctx, uploadReq)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Uploading Keypair",
				"Could not upload keypair: "+err.Error(),
			)
			return
		}

		data.Status = types.StringValue(uploadResp.Status)
		data.Message = types.StringValue(uploadResp.Data)
		data.PrivateKeyContent = types.StringNull()
		data.PrivateKeyFilename = types.StringNull()

	} else {
		resp.Diagnostics.AddError(
			"Invalid Mode",
			fmt.Sprintf("mode must be 'create' or 'upload', got: %s", mode),
		)
		return
	}

	// After create/upload, fetch the keypair to get its ID
	keypair, err := GetKeypairByName(r.client, ctx, data.EngagementID.ValueString(), data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Keypair",
			"Could not read keypair after creation: "+err.Error(),
		)
		return
	}

	// Set computed attributes
	data.ID = types.StringValue(FormatKeypairID(keypair.ID))
	if keypair.Fingerprint != "" {
		data.Fingerprint = types.StringValue(keypair.Fingerprint)
	} else {
		data.Fingerprint = types.StringNull()
	}

	tflog.Info(ctx, "Keypair created successfully", map[string]any{
		"id":   data.ID.ValueString(),
		"name": data.Name.ValueString(),
		"mode": mode,
	})

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read refreshes the Terraform state with the latest data from the API.
func (r *KeypairResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data KeypairResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Reading keypair", map[string]any{
		"id":            data.ID.ValueString(),
		"engagement_id": data.EngagementID.ValueString(),
	})

	// Parse the keypair ID
	keypairID, err := ParseKeypairID(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Parsing Keypair ID",
			"Could not parse keypair ID: "+err.Error(),
		)
		return
	}

	// Fetch the keypair from the API
	keypair, err := GetKeypairByID(r.client, ctx, data.EngagementID.ValueString(), keypairID)
	if err != nil {
		// If the keypair is not found, remove it from state
		resp.Diagnostics.AddWarning(
			"Keypair Not Found",
			fmt.Sprintf("Keypair with ID %s not found, removing from state: %s", data.ID.ValueString(), err.Error()),
		)
		resp.State.RemoveResource(ctx)
		return
	}

	// Update model with API response
	data.Name = types.StringValue(keypair.Name)
	if keypair.Fingerprint != "" {
		data.Fingerprint = types.StringValue(keypair.Fingerprint)
	}
	data.Status = types.StringValue("success")

	tflog.Info(ctx, "Keypair read successfully", map[string]any{
		"id":   data.ID.ValueString(),
		"name": data.Name.ValueString(),
	})

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update updates the resource. Since keypairs cannot be updated, this triggers recreation.
func (r *KeypairResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data KeypairResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Keypairs cannot be updated - all attributes that could change require replacement
	// This function should not be called due to RequiresReplace plan modifiers
	resp.Diagnostics.AddError(
		"Update Not Supported",
		"Keypairs cannot be updated. Please recreate the resource.",
	)
}

// Delete deletes the keypair resource.
func (r *KeypairResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data KeypairResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting keypair", map[string]any{
		"id":   data.ID.ValueString(),
		"name": data.Name.ValueString(),
	})

	// Parse the keypair ID
	keypairID, err := ParseKeypairID(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Parsing Keypair ID",
			"Could not parse keypair ID: "+err.Error(),
		)
		return
	}

	// Delete the keypair
	deleteResp, err := DeleteKeypair(r.client, ctx, keypairID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Keypair",
			"Could not delete keypair: "+err.Error(),
		)
		return
	}

	tflog.Info(ctx, "Keypair deleted successfully", map[string]any{
		"id":     data.ID.ValueString(),
		"name":   data.Name.ValueString(),
		"status": deleteResp.Status,
	})
}

// ImportState imports an existing keypair into Terraform state.
func (r *KeypairResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import using the format: engagement_id/keypair_id
	// e.g., "ENG123/456"
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
