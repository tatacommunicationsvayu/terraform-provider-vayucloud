// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package s3_domain

import (
	"context"
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/float64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
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
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/account_engagement"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/account_location"
)

// Ensure provider-defined types fully satisfy framework interfaces.
var _ resource.Resource = &S3DomainResource{}
var _ resource.ResourceWithImportState = &S3DomainResource{}
var _ resource.ResourceWithModifyPlan = &S3DomainResource{}

// NewS3DomainResource creates a new S3 domain resource.
func NewS3DomainResource() resource.Resource {
	return &S3DomainResource{}
}

// S3DomainResource defines the resource implementation.
type S3DomainResource struct {
	client *client.Client
}

// S3DomainResourceModel describes the resource data model.
// tfsdk tags must match Schema() attribute names exactly (snake_case).
type S3DomainResourceModel struct {
	// Computed from firewall_id at plan/create (IPC engagement and endpoint).
	EngagementID types.Int64 `tfsdk:"engagement_id"`
	EndpointID   types.Int64 `tfsdk:"endpoint_id"`

	// Required user inputs
	DomainName types.String `tfsdk:"domain_name"`
	Quota        types.Float64 `tfsdk:"quota"`
	StorageClass types.String  `tfsdk:"storage_class"`
	Variant      types.String  `tfsdk:"variant"`

	// Optional user inputs
	SecondaryEndpointID types.Int64  `tfsdk:"secondary_endpoint_id"`
	PricingModel        types.String `tfsdk:"pricing_model"`
	FirewallID          types.Int64  `tfsdk:"firewall_id"`

	// Computed — populated by provider after create/read.
	ID             types.String `tfsdk:"id"`
	AuditID        types.String `tfsdk:"audit_id"`
	Status         types.String `tfsdk:"status"`
	DomainNameFQDN types.String `tfsdk:"domain_name_fqdn"`
	QuotaUnit      types.String `tfsdk:"quota_unit"`
	// DomainAccessIP is the private IP for S3 access over the connected firewall.
	DomainAccessIP types.String `tfsdk:"domain_access_ip"`
	// DomainAccessPublicIP is the public IP when a mapping exists (optional).
	DomainAccessPublicIP types.String `tfsdk:"domain_access_public_ip"`
}

// Metadata returns the resource type name: "vayucloud_s3_domain".
func (r *S3DomainResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_s3_domain"
}

// Schema defines all attributes the user can set or that the provider computes.
func (r *S3DomainResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an S3 domain resource in VayuCloud.",
		MarkdownDescription: "Manages an S3 domain resource in VayuCloud.\n\n" +
			"Domain creation is an asynchronous operation. The provider polls the audit log " +
			"until provisioning completes (up to ~30 minutes; longer when base-extension " +
			"firewall mapping is in progress).\n\n" +
			"`firewall_id` is required so the domain is mapped to a firewall for private S3 access. " +
			"`engagement_id` and `endpoint_id` are derived from `firewall_id` at plan time.\n\n" +
			"`quota` supports in-place update. All other input attributes use `RequiresReplace`.",

		Attributes: map[string]schema.Attribute{
			// ── Computed identity ──────────────────────────────────────────────
			"id": schema.StringAttribute{
				Description: "The unique identifier of the S3 domain (numeric ics_domain.id, set from audit resourceId).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"audit_id": schema.StringAttribute{
				Description:         "The audit ID from the creation response.",
				MarkdownDescription: "The audit ID from the creation response.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"status": schema.StringAttribute{
				Description:         "The final status from the audit log.",
				MarkdownDescription: "The final status from the audit log.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},

			"engagement_id": schema.Int64Attribute{
				Description: "IPC engagement ID derived from `firewall_id` at plan/create time.",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"endpoint_id": schema.Int64Attribute{
				Description: "Endpoint ID derived from `firewall_id` at plan/create time.",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},

			// ── Required user inputs ───────────────────────────────────────────
			"domain_name": schema.StringAttribute{
				Description: "The domain name (3–24 chars; alphanumeric, underscore, hyphen only). " +
					"The server appends .ipstorage.tatacommunications.com to form the FQDN. " +
					"Changing this forces a new resource.",
				Required: true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(3, 24),
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[a-zA-Z0-9_-]+$`),
						"domain_name may only contain alphanumeric characters, underscores (_), and hyphens (-)",
					),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"quota": schema.Float64Attribute{
				Description: "Storage quota in GB (> 0, ≤ 10000). Supports in-place update via PUT /ics-operations/domain/{id}.",
				Required:    true,
				Validators: []validator.Float64{
					float64validator.Between(0.001, 10000),
				},
			},
			"storage_class": schema.StringAttribute{
				Description: "Storage class: `STANDARD`, `HIGH_PERFORMANCE`, or `AI_STANDARD`. " +
					"Determines the underlying vendor/cluster. Changing this forces a new resource.",
				Required: true,
				Validators: []validator.String{
					stringvalidator.OneOfCaseInsensitive("STANDARD", "HIGH_PERFORMANCE", "AI_STANDARD"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"variant": schema.StringAttribute{
				Description: "Domain variant: `value`, `geoResilient`, or `resilient`. " +
					"Determines the P2R order item and topology. " +
					"When set to `geoResilient`, `secondary_endpoint_id` is required. " +
					"Changing this forces a new resource.",
				Required: true,
				Validators: []validator.String{
					stringvalidator.OneOfCaseInsensitive("value", "geoResilient", "resilient"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},

			// ── Optional user inputs ───────────────────────────────────────────
			"secondary_endpoint_id": schema.Int64Attribute{
				Description: "Secondary endpoint ID required when `variant` is `geoResilient`. " +
					"Changing this forces a new resource.",
				Optional: true,
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
				},
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"pricing_model": schema.StringAttribute{
				Description: "Pricing model: `daily`, `monthly`, `reserved_1`, `reserved_3`, or `reserved_5`. " +
					"Defaults to `daily` when omitted. Changing this forces a new resource.",
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString("daily"),
				Validators: []validator.String{
					stringvalidator.OneOfCaseInsensitive(
						"daily", "monthly", "reserved_1", "reserved_3", "reserved_5",
					),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"firewall_id": schema.Int64Attribute{
				Description: "IPC firewall ci_master.id used to map private S3 access to the domain via base extension. " +
					"Required at create. Changing this forces a new resource.",
				Required: true,
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
				},
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},

			// ── Computed server-populated fields ──────────────────────────────
			"domain_name_fqdn": schema.StringAttribute{
				Description: "Fully-qualified domain name as assigned by the server " +
					"(domain_name + .ipstorage.tatacommunications.com).",
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"quota_unit": schema.StringAttribute{
				Description: "Unit of the provisioned quota as reported by the server. Always GB after read/list.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"domain_access_ip": schema.StringAttribute{
				Description: "Private IP address for S3 access over the connected firewall. " +
					"Populated after create when the base-extension mapping completes.",
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"domain_access_public_ip": schema.StringAttribute{
				Description: "Public IP address for S3 access when a mapping exists. " +
					"Optional; only populated when firewall_id is set and the backend has a public IP.",
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

// Configure receives the authenticated client from the provider and stores it
// on the resource for use in CRUD methods.
func (r *S3DomainResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// ModifyPlan resolves engagement/endpoint from firewall_id and validates cross-field rules.
// Destroy plans use prior state when the plan value is null.
func (r *S3DomainResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if r.client == nil {
		return
	}

	var plan S3DomainResourceModel
	if req.Plan.Raw.IsNull() {
		resp.Diagnostics.Append(req.State.Get(ctx, &plan)...)
	} else {
		resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.FirewallID.IsUnknown() || plan.FirewallID.IsNull() {
		return
	}

	firewallID := plan.FirewallID.ValueInt64()
	engagementID, endpointID, err := ResolveEngagementEndpointFromFirewall(r.client, ctx, firewallID)
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("firewall_id"), "Invalid firewall ID", err.Error())
		return
	}

	plan.EngagementID = types.Int64Value(engagementID)
	plan.EndpointID = types.Int64Value(endpointID)

	tflog.Debug(ctx, "Planning S3 domain", map[string]any{
		"engagement_id": engagementID,
		"endpoint_id":   endpointID,
		"firewall_id":   firewallID,
		"domain_name":   plan.DomainName.ValueString(),
		"variant":       plan.Variant.ValueString(),
		"quota":         plan.Quota.ValueFloat64(),
	})

	if err := account_engagement.ValidateEngagementExists(r.client, ctx, engagementID); err != nil {
		resp.Diagnostics.AddError("Invalid Engagement ID", err.Error())
		return
	}

	if err := account_location.ValidateEndpointExists(r.client, ctx, engagementID, endpointID); err != nil {
		resp.Diagnostics.AddError("Invalid Endpoint ID", err.Error())
		return
	}

	if !plan.Variant.IsUnknown() && plan.Variant.ValueString() == "geoResilient" {
		if plan.SecondaryEndpointID.IsNull() || plan.SecondaryEndpointID.IsUnknown() {
			resp.Diagnostics.AddAttributeError(
				path.Root("secondary_endpoint_id"),
				"Missing secondary_endpoint_id",
				"secondary_endpoint_id is required when variant is 'geoResilient'.",
			)
			return
		}
	}

	// Destroy plan — firewall/engagement/endpoint checks only.
	if req.Plan.Raw.IsNull() {
		return
	}

	// Create plan — validate domain name availability; set resolved ids on plan.
	if plan.ID.IsUnknown() || plan.ID.IsNull() || plan.ID.ValueString() == "" {
		if !plan.DomainName.IsUnknown() && plan.DomainName.ValueString() != "" {
			if err := ValidateDomainNameAvailable(
				r.client, ctx,
				engagementID, endpointID,
				plan.DomainName.ValueString(),
				firewallID,
			); err != nil {
				resp.Diagnostics.AddAttributeError(
					path.Root("domain_name"),
					"Domain name already exists",
					err.Error(),
				)
				return
			}
		}
		resp.Diagnostics.Append(resp.Plan.Set(ctx, plan)...)
		return
	}

	if err := ValidateDomainExists(
		r.client, ctx,
		engagementID,
		plan.ID.ValueString(),
		firewallID,
	); err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("id"),
			"Invalid S3 Domain ID",
			err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.Plan.Set(ctx, plan)...)
}

// Create provisions a new S3 domain.
//
// Execution order (mirrors FIREWALL pattern but with ICS-specific nuances):
//  1. Read plan into model
//  2. E5 provider-side cross-field and range validations
//  3. Resolve engagement_id + endpoint_id from firewall_id; pre-check availability
//  4. Build S3DomainCreateRequest (unchanged ICS payload shape)
//  5. Build actionStateBody (quota, quotaUnit, firewallId)
//  6. CreateS3DomainAndWait → POST createDomain → poll audit → action-state create
//  7. Set id + audit_id from audit response
//  8. refreshStateFromRead → action-state read populates FQDN, tenant_id, etc.
//  9. Save to state (plan values authoritative for engagement_id, pricing_model,
//     firewall_id, secondary_endpoint_id; quota refreshed from server)
func (r *S3DomainResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data S3DomainResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.FirewallID.IsNull() || data.FirewallID.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("firewall_id"),
			"Missing firewall_id",
			"firewall_id is required to create an S3 domain.",
		)
		return
	}

	firewallID := data.FirewallID.ValueInt64()
	engagementID, endpointID, err := ResolveEngagementEndpointFromFirewall(r.client, ctx, firewallID)
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("firewall_id"), "Invalid firewall ID", err.Error())
		return
	}
	data.EngagementID = types.Int64Value(engagementID)
	data.EndpointID = types.Int64Value(endpointID)

	tflog.Debug(ctx, "Creating S3 domain", map[string]any{
		"engagement_id": engagementID,
		"endpoint_id":   endpointID,
		"firewall_id":   firewallID,
		"domain_name":   data.DomainName.ValueString(),
		"quota":         data.Quota.ValueFloat64(),
		"storage_class": data.StorageClass.ValueString(),
		"variant":       data.Variant.ValueString(),
	})

	// ── E5: cross-field and range validation ──────────────────────────────────
	if data.Variant.ValueString() == "geoResilient" {
		if data.SecondaryEndpointID.IsNull() || data.SecondaryEndpointID.IsUnknown() {
			resp.Diagnostics.AddAttributeError(
				path.Root("secondary_endpoint_id"),
				"Missing secondary_endpoint_id",
				"secondary_endpoint_id is required when variant is 'geoResilient'.",
			)
			return
		}
	}

	quota := data.Quota.ValueFloat64()
	if quota <= 0 || quota > 10000 {
		resp.Diagnostics.AddAttributeError(
			path.Root("quota"),
			"Invalid quota",
			fmt.Sprintf("quota must be greater than 0 and at most 10000 GB / 10 TB (got %.4f).", quota),
		)
		return
	}

	// ── F: pre-checks ─────────────────────────────────────────────────────────
	if err := account_engagement.ValidateEngagementExists(r.client, ctx, engagementID); err != nil {
		resp.Diagnostics.AddError("Invalid Engagement ID", err.Error())
		return
	}
	if err := account_location.ValidateEndpointExists(r.client, ctx, engagementID, endpointID); err != nil {
		resp.Diagnostics.AddError("Invalid Endpoint ID", err.Error())
		return
	}

	if err := ValidateDomainNameAvailable(
		r.client, ctx,
		engagementID, endpointID,
		data.DomainName.ValueString(),
		firewallID,
	); err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("domain_name"),
			"Domain name already exists",
			err.Error(),
		)
		return
	}

	// ── Build create request (ICS camelCase payload unchanged) ────────────────
	createReq := &S3DomainCreateRequest{
		EngagementID: engagementID,
		EndpointID:   endpointID,
		DomainName:   data.DomainName.ValueString(),
		Quota:        data.Quota.ValueFloat64(),
		StorageClass: data.StorageClass.ValueString(),
		Variant:      data.Variant.ValueString(),
		FirewallID:   firewallID,
	}
	if !data.SecondaryEndpointID.IsNull() && !data.SecondaryEndpointID.IsUnknown() {
		v := data.SecondaryEndpointID.ValueInt64()
		createReq.SecondaryEndpointID = &v
	}
	pricingModel := "daily"
	if !data.PricingModel.IsNull() && !data.PricingModel.IsUnknown() {
		pricingModel = data.PricingModel.ValueString()
	}
	createReq.PricingModel = &pricingModel
	data.PricingModel = types.StringValue(pricingModel)

	// ── Build action-state body (merged with resourceId inside WaitForAuditCompletion) ──
	actionStateBody := map[string]any{
		"quota":      data.Quota.ValueFloat64(),
		"quotaUnit":  "GB",
		"firewallId": firewallID,
	}

	// ── Fire + poll + action-state create ────────────────────────────────────
	auditLog, err := CreateS3DomainAndWait(r.client, ctx, createReq, actionStateBody)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating S3 Domain", err.Error())
		return
	}

	// ── Set id + audit_id from audit ──────────────────────────────────────────
	data.ID = types.StringValue(auditLog.ResourceID.String())
	data.AuditID = types.StringValue(auditLog.AuditID)
	data.Status = types.StringValue(auditLog.Status)

	tflog.Info(ctx, "S3 domain created successfully", map[string]any{
		"id":       data.ID.ValueString(),
		"audit_id": data.AuditID.ValueString(),
		"status":   data.Status.ValueString(),
	})

	// ── action-state read: populate server-computed attrs ─────────────────────
	refreshDiags := r.refreshStateFromActionState(ctx, &data)
	resp.Diagnostics.Append(refreshDiags...)
	if refreshDiags.HasError() {
		return
	}

	// ── Save to state ─────────────────────────────────────────────────────────
	// engagement_id, endpoint_id, quota, pricing_model, firewall_id,
	// secondary_endpoint_id: kept from plan (authoritative user inputs).
	// domain_name_fqdn, quota_unit, domain_access_ip, domain_access_public_ip: from read.
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read refreshes Terraform state with the latest data from the action-state read API.
//
// Plan-authoritative fields (never overwritten from read):
//   - engagement_id  — read may return ICS id; provider keeps IPC id resolved from firewall
//   - quota          — server may normalise to TB internally; keep user-supplied GB
//   - pricing_model  — not returned by read
//   - firewall_id    — not returned by read
//   - secondary_endpoint_id — not returned by read
//
// Fields refreshed from server (out-of-band changes surface as drift):
//   - quota          — backend may change quota manually; always re-read
func (r *S3DomainResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data S3DomainResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Reading S3 domain", map[string]any{
		"id": data.ID.ValueString(),
	})

	if diags := r.ensureEngagementEndpointFromFirewall(ctx, &data); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	refreshDiags := r.refreshStateFromActionState(ctx, &data)
	resp.Diagnostics.Append(refreshDiags...)
	if refreshDiags.HasError() {
		return
	}

	tflog.Info(ctx, "S3 domain read successfully", map[string]any{
		"id": data.ID.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update changes domain quota in-place when quota differs from state.
func (r *S3DomainResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan S3DomainResourceModel
	var state S3DomainResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Updating S3 domain", map[string]any{
		"id":          plan.ID.ValueString(),
		"plan_quota":  plan.Quota.ValueFloat64(),
		"state_quota": state.Quota.ValueFloat64(),
	})

	if diags := r.ensureEngagementEndpointFromFirewall(ctx, &plan); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}
	engagementID := plan.EngagementID.ValueInt64()
	endpointID := plan.EndpointID.ValueInt64()

	if err := account_engagement.ValidateEngagementExists(r.client, ctx, engagementID); err != nil {
		resp.Diagnostics.AddError("Invalid Engagement ID", err.Error())
		return
	}
	if err := account_location.ValidateEndpointExists(r.client, ctx, engagementID, endpointID); err != nil {
		resp.Diagnostics.AddError("Invalid Endpoint ID", err.Error())
		return
	}

	quotaChanged := plan.Quota.ValueFloat64() != state.Quota.ValueFloat64()

	if quotaChanged {
		newQuota := plan.Quota.ValueFloat64()
		if newQuota <= 0 || newQuota > 10000 {
			resp.Diagnostics.AddAttributeError(
				path.Root("quota"),
				"Invalid quota",
				fmt.Sprintf("quota must be greater than 0 and at most 10000 GB (got %.4f).", newQuota),
			)
			return
		}

		auditLog, err := UpdateS3DomainAndWait(r.client, ctx, plan.ID.ValueString(), newQuota)
		if err != nil {
			resp.Diagnostics.AddError("Error Updating S3 Domain", err.Error())
			return
		}

		if auditLog.AuditID != "" {
			plan.AuditID = types.StringValue(auditLog.AuditID)
		}
		plan.Status = types.StringValue(auditLog.Status)

		tflog.Info(ctx, "S3 domain quota updated successfully", map[string]any{
			"id":       plan.ID.ValueString(),
			"quota":    newQuota,
			"audit_id": plan.AuditID.ValueString(),
			"status":   plan.Status.ValueString(),
		})
	} else {
		tflog.Debug(ctx, "No quota change detected, skipping update API call", map[string]any{
			"id": plan.ID.ValueString(),
		})
	}

	out := plan
	refreshDiags := r.refreshStateFromActionState(ctx, &out)
	resp.Diagnostics.Append(refreshDiags...)
	if refreshDiags.HasError() {
		return
	}

	// Action-state read does not populate audit_id / status; keep values from the plan or last state
	if out.AuditID.IsUnknown() || out.AuditID.IsNull() {
		out.AuditID = state.AuditID
	}
	if out.Status.IsUnknown() || out.Status.IsNull() {
		out.Status = state.Status
	}
	if out.DomainNameFQDN.IsUnknown() || out.DomainNameFQDN.IsNull() {
		out.DomainNameFQDN = state.DomainNameFQDN
	}
	if out.QuotaUnit.IsUnknown() || out.QuotaUnit.IsNull() {
		out.QuotaUnit = state.QuotaUnit
	}
	if out.DomainAccessIP.IsUnknown() {
		out.DomainAccessIP = state.DomainAccessIP
	}
	if out.DomainAccessPublicIP.IsUnknown() {
		out.DomainAccessPublicIP = state.DomainAccessPublicIP
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &out)...)
}

// Delete removes the S3 domain via ICS operations DELETE + audit poll + action-state delete.
func (r *S3DomainResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data S3DomainResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting S3 domain", map[string]any{
		"id":       data.ID.ValueString(),
		"audit_id": data.AuditID.ValueString(),
	})

	if err := account_engagement.ValidateEngagementExists(r.client, ctx, data.EngagementID.ValueInt64()); err != nil {
		resp.Diagnostics.AddError("Invalid Engagement ID", err.Error())
		return
	}
	if err := account_location.ValidateEndpointExists(r.client, ctx, data.EngagementID.ValueInt64(), data.EndpointID.ValueInt64()); err != nil {
		resp.Diagnostics.AddError("Invalid Endpoint ID", err.Error())
		return
	}

	if err := DeleteS3DomainAndWait(r.client, ctx, data.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error Deleting S3 Domain", err.Error())
		return
	}

	tflog.Info(ctx, "S3 domain deleted successfully", map[string]any{
		"id": data.ID.ValueString(),
	})
}

// ImportState imports an existing S3 domain into Terraform state by its numeric id.
//
// Usage: terraform import vayucloud_s3_domain.<name> <domain_id>
func (r *S3DomainResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// ── Helpers ───────────────────────────────────────────────────────────────────

// ensureEngagementEndpointFromFirewall fills engagement_id and endpoint_id from firewall_id
// when they are not already in state (e.g. after import).
func (r *S3DomainResource) ensureEngagementEndpointFromFirewall(ctx context.Context, data *S3DomainResourceModel) diag.Diagnostics {
	var d diag.Diagnostics

	if data.FirewallID.IsNull() || data.FirewallID.IsUnknown() {
		return d
	}
	if !data.EngagementID.IsNull() && !data.EngagementID.IsUnknown() &&
		!data.EndpointID.IsNull() && !data.EndpointID.IsUnknown() {
		return d
	}

	engagementID, endpointID, err := ResolveEngagementEndpointFromFirewall(r.client, ctx, data.FirewallID.ValueInt64())
	if err != nil {
		d.AddAttributeError(path.Root("firewall_id"), "Invalid firewall ID", err.Error())
		return d
	}
	data.EngagementID = types.Int64Value(engagementID)
	data.EndpointID = types.Int64Value(endpointID)
	return d
}

// refreshStateFromActionState calls action-state read and maps the payload onto data.
// It does not set id, audit_id, or status (not returned in the read payload).
// A warning (not error) is emitted when the domain is not yet readable so Create()
// can still complete.
func (r *S3DomainResource) refreshStateFromActionState(ctx context.Context, data *S3DomainResourceModel) diag.Diagnostics {
	var d diag.Diagnostics

	if data.ID.IsNull() || data.ID.IsUnknown() || data.ID.ValueString() == "" {
		return d
	}

	tflog.Debug(ctx, "Calling action-state read for S3 domain", map[string]any{
		"id": data.ID.ValueString(),
	})

	payload, err := ReadS3Domain(r.client, ctx, data.EngagementID.ValueInt64(), data.ID.ValueString(), data.FirewallID.ValueInt64())
	if err != nil {
		d.AddWarning(
			"Empty S3 Domain Read Response",
			"Could not read S3 domain state; server-computed attributes will be "+
				"populated on the next terraform refresh. "+err.Error(),
		)
		return d
	}

	applyPayloadToModel(payload, data)
	return d
}

func applyPayloadToModel(payload *S3DomainPayload, data *S3DomainResourceModel) {
	if payload.DomainName != "" {
		data.DomainNameFQDN = types.StringValue(payload.DomainName)
	}
	if payload.Quota > 0 {
		data.Quota = types.Float64Value(payload.Quota)
	}
	if payload.QuotaUnit != "" {
		data.QuotaUnit = types.StringValue(payload.QuotaUnit)
	}
	if payload.DomainAccessIP != "" {
		data.DomainAccessIP = types.StringValue(payload.DomainAccessIP)
	} else {
		data.DomainAccessIP = types.StringNull()
	}
	if payload.DomainAccessPublicIP != "" {
		data.DomainAccessPublicIP = types.StringValue(payload.DomainAccessPublicIP)
	} else {
		data.DomainAccessPublicIP = types.StringNull()
	}
}
