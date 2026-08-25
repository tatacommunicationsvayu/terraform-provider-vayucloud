// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package network_firewall

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
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
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/common"
)

// Ensure provider-defined types fully satisfy framework interfaces.
var _ resource.Resource = &NetworkFirewallResource{}
var _ resource.ResourceWithImportState = &NetworkFirewallResource{}
var _ resource.ResourceWithModifyPlan = &NetworkFirewallResource{}

// NewNetworkFirewallResource creates a new network firewall resource.
func NewNetworkFirewallResource() resource.Resource {
	return &NetworkFirewallResource{}
}

// NetworkFirewallResource defines the resource implementation.
type NetworkFirewallResource struct {
	client *client.Client
}

// NetworkFirewallResourceModel describes the resource data model.
type NetworkFirewallResourceModel struct {
	// ID is the unique identifier (audit_id from creation)
	ID types.String `tfsdk:"id"`

	// EngagementID is the engagement ID
	EngagementID types.Int64 `tfsdk:"engagement_id"`

	// EndpointID is the endpoint ID
	EndpointID types.Int64 `tfsdk:"endpoint_id"`

	// FirewallType is the type of firewall (e.g., "VFAAS")
	FirewallType types.String `tfsdk:"firewall_type"`

	// FirewallDisplayName is the display name for the firewall
	FirewallDisplayName types.String `tfsdk:"firewall_display_name"`

	// FirewallThroughput is the throughput setting (e.g., "2Mbps")
	FirewallThroughput types.String `tfsdk:"firewall_throughput"`

	// Bandwidth is the bandwidth setting (e.g., "2Mbps")
	InternetBandwidth types.String `tfsdk:"internet_bandwidth"`
	// AccessType is the access type (e.g., "Bandwidth", "DataTransfer")
	AccessType types.String `tfsdk:"access_type"`

	// MinimumCommitment is the minimum commitment (e.g., "500GB")
	MinimumCommitment types.String `tfsdk:"minimum_commitment"`

	// IsInternetEnabled indicates if internet is enabled
	IsInternetEnabled types.Bool `tfsdk:"is_internet_enabled"`

	// NonDistributedEnabled indicates if non-distributed mode is enabled
	NonDistributedEnabled types.Bool `tfsdk:"non_distributed_enabled"`

	// FirewallPricingModel is the pricing model (e.g., "daily", "monthly")
	FirewallPricingModel types.String `tfsdk:"firewall_pricing_model"`

	// InternetPricingModel is the internet pricing model (e.g., "daily", "monthly")
	InternetPricingModel types.String `tfsdk:"internet_pricing_model"`

	// Hypervisor is the hypervisor type (e.g., "KVM", "VCD_ESXI")
	Hypervisor types.String `tfsdk:"hypervisor"`

	// Computed attributes
	// AuditID is the audit ID from the creation response
	AuditID types.String `tfsdk:"audit_id"`

	// Status is the final status from the audit log
	Status types.String `tfsdk:"status"`
}

// Metadata returns the resource type name.
func (r *NetworkFirewallResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network_firewall"
}

// Schema defines the schema for the network firewall resource.
func (r *NetworkFirewallResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Manages a firewall resource in VayuCloud.",
		MarkdownDescription: "Manages a firewall resource in VayuCloud.\n\nThis resource creates a firewall through an asynchronous provisioning process. The resource will poll the audit log until the firewall creation is complete.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:         "The unique identifier of the firewall (audit ID).",
				MarkdownDescription: "The unique identifier of the firewall (audit ID).",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"engagement_id": schema.Int64Attribute{
				Description:         "The engagement ID.",
				MarkdownDescription: "The engagement ID.",
				Required:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"endpoint_id": schema.Int64Attribute{
				Description:         "The endpoint ID.",
				MarkdownDescription: "The endpoint ID.",
				Required:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"firewall_type": schema.StringAttribute{
				Description:         "The type of firewall (e.g., 'VFAAS').",
				MarkdownDescription: "The type of firewall (e.g., `VFAAS`).",
				Required:            false,
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("VFAAS"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"firewall_display_name": schema.StringAttribute{
				Description:         "The display name for the firewall. Only alphanumeric characters, underscores (_), and hyphens (-) are allowed.",
				MarkdownDescription: "The display name for the firewall. Only alphanumeric characters, underscores (`_`), and hyphens (`-`) are allowed.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(5, 45),
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[a-zA-Z0-9_-]+$`),
						"Value may only contain alphanumeric characters, underscores (_), and hyphens (-)",
					),
				},
			},
			"firewall_throughput": schema.StringAttribute{
				Description:         "The throughput setting (e.g., '2Mbps'). The value should be in the format of 'XMbps' (e.g., '2Mbps')",
				MarkdownDescription: "The throughput setting (e.g., `2Mbps`). The value should be in the format of `XMbps` (e.g., `2Mbps`)",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile(`^\d+[M]bps$`), "The value should be in the format of 'XMbps' (e.g., '2Mbps')"),
				},
			},
			"internet_bandwidth": schema.StringAttribute{
				Description:         "The internet bandwidth setting (e.g., '2Mbps'). The value should be in the format of 'XMbps' (e.g., '2Mbps')",
				MarkdownDescription: "The internet bandwidth setting (e.g., `2Mbps`). The value should be in the format of `XMbps` (e.g., `2Mbps`)",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile(`^\d+[M]bps$`), "The value should be in the format of 'XMbps' (e.g., '2Mbps')"),
				},
			},
			"access_type": schema.StringAttribute{
				Description:         "The access type (e.g., 'Bandwidth', 'DataTransfer'). Defaults to 'Bandwidth'.",
				MarkdownDescription: "The access type (e.g., `Bandwidth`, `DataTransfer`). Defaults to `Bandwidth`.",
				Optional:            true,
				Computed:            true,
				// Default:             stringdefault.StaticString("Bandwidth"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOfCaseInsensitive("Bandwidth", "DataTransfer"),
				},
			},
			"minimum_commitment": schema.StringAttribute{
				Description:         "The minimum commitment (e.g., '500GB'). The value should be in the format of 'XGB' or 'XMB' or 'XTB' (e.g., '500GB')",
				MarkdownDescription: "The minimum commitment (e.g., `500GB`). The value should be in the format of `XGB` or `XMB` or `XTB` (e.g., `500GB`)",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile(`^\d+[G]B$|^\d+[M]B$|^\d+[T]B$`), "The value can be in MB or GB or TB' (e.g., '500GB')"),
				},
			},
			"is_internet_enabled": schema.BoolAttribute{
				Description:         "Whether internet is enabled. Defaults to true.",
				MarkdownDescription: "Whether internet is enabled. Defaults to `true`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"non_distributed_enabled": schema.BoolAttribute{
				Description:         "Whether non-distributed mode is enabled. Defaults to true.",
				MarkdownDescription: "Whether non-distributed mode is enabled. Defaults to `true`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"firewall_pricing_model": schema.StringAttribute{
				Description:         "The firewall pricing model (e.g., 'daily', 'monthly', 'reserved_1', 'reserved_3', 'reserved_5'). Only sent to the API when explicitly provided.",
				MarkdownDescription: "The firewall pricing model (e.g., `daily`, `monthly`, `reserved_1`, `reserved_3`, `reserved_5`). Only sent to the API when explicitly provided.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("daily"),
				Validators: []validator.String{
					stringvalidator.OneOfCaseInsensitive(
						"daily", "monthly", "reserved_1", "reserved_3", "reserved_5",
					),
				},
			},
			"internet_pricing_model": schema.StringAttribute{
				Description:         "The internet pricing model (e.g., 'daily', 'monthly', 'reserved_1', 'reserved_3', 'reserved_5'). Only sent to the API when explicitly provided.",
				MarkdownDescription: "The internet pricing model (e.g., `daily`, `monthly`, `reserved_1`, `reserved_3`, `reserved_5`). Only sent to the API when explicitly provided.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("daily"),
				Validators: []validator.String{
					stringvalidator.OneOfCaseInsensitive(
						"daily", "monthly", "reserved_1", "reserved_3", "reserved_5",
					),
				},
			},
			"hypervisor": schema.StringAttribute{
				Description:         "The hypervisor type, allowed values are 'KVM' or 'VCD_ESXI'. Defaults to 'KVM'.",
				MarkdownDescription: "The hypervisor type (`KVM` or `VCD_ESXI`). Any other value will cause an error. Defaults to `KVM`.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("KVM"),
				Validators: []validator.String{
					stringvalidator.OneOfCaseInsensitive("KVM", "VCD_ESXI"),
				},
			},
			// Computed attributes
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
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *NetworkFirewallResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// ModifyPlan validates engagement, endpoint, and throughput/bandwidth during terraform plan.
// Destroy plans use prior state when the plan value is null.
func (r *NetworkFirewallResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if r.client == nil {
		return
	}

	var plan NetworkFirewallResourceModel
	if req.Plan.Raw.IsNull() {
		resp.Diagnostics.Append(req.State.Get(ctx, &plan)...)
	} else {
		resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	// Do not require internet_bandwidth to be known here: it stays unknown when unset or for
	// access_type DataTransfer, which previously caused an immediate return and skipped all
	// of ModifyPlan (including engagement/endpoint checks).
	if plan.EngagementID.IsUnknown() || plan.EndpointID.IsUnknown() {
		return
	}

	tflog.Debug(ctx, "Planning Network Firewall", map[string]any{
		"engagement_id":           plan.EngagementID.ValueInt64(),
		"endpoint_id":             plan.EndpointID.ValueInt64(),
		"firewall_throughput":     plan.FirewallThroughput.ValueString(),
		"internet_bandwidth":      plan.InternetBandwidth.ValueString(),
		"access_type":             plan.AccessType.ValueString(),
		"minimum_commitment":      plan.MinimumCommitment.ValueString(),
		"is_internet_enabled":     plan.IsInternetEnabled.ValueBool(),
		"non_distributed_enabled": plan.NonDistributedEnabled.ValueBool(),
		"firewall_pricing_model":  plan.FirewallPricingModel.ValueString(),
	})

	if err := account_engagement.ValidateEngagementExists(r.client, ctx, plan.EngagementID.ValueInt64()); err != nil {
		resp.Diagnostics.AddError(
			"Invalid Engagement ID",
			err.Error(),
		)
		return
	}

	if err := account_location.ValidateEndpointExists(r.client, ctx, plan.EngagementID.ValueInt64(), plan.EndpointID.ValueInt64()); err != nil {
		resp.Diagnostics.AddError(
			"Invalid Endpoint ID",
			err.Error(),
		)
		return
	}

	if plan.FirewallThroughput.IsUnknown() {
		return
	}
	// Set access type based on presence of InternetBandwidth or MinimumCommitment
	if plan.InternetBandwidth.IsUnknown() && plan.MinimumCommitment.IsUnknown() {
		resp.Diagnostics.AddError(
			"Missing Internet Bandwidth and Minimum Commitment",
			"Either internet_bandwidth or minimum_commitment must be specified. Please provide a valid value for one of them.",
		)
		return
	}

	if plan.MinimumCommitment.IsNull() || plan.MinimumCommitment.IsUnknown() || plan.MinimumCommitment.ValueString() == "" {
		plan.AccessType = types.StringValue("Bandwidth")
	} else if plan.InternetBandwidth.IsNull() || plan.InternetBandwidth.IsUnknown() || plan.InternetBandwidth.ValueString() == "" {
		plan.AccessType = types.StringValue("DataTransfer")
	}
	// Validate throughput >= bandwidth
	if plan.AccessType.ValueString() == "Bandwidth" {
		if plan.InternetBandwidth.IsNull() || plan.InternetBandwidth.IsUnknown() || plan.InternetBandwidth.ValueString() == "" {
			resp.Diagnostics.AddError(
				"Missing Internet Bandwidth",
				"Internet bandwidth must be specified when access_type is 'Bandwidth'. Please provide a valid value for internet_bandwidth.",
			)
			return
		}
		if err := client.ValidateThroughputAndBandwidth(
			ctx,
			plan.FirewallThroughput.ValueString(),
			plan.InternetBandwidth.ValueString(),
		); err != nil {
			resp.Diagnostics.AddError(
				"Invalid Network Firewall Configuration",
				err.Error(),
			)
			return
		}
	}
	if plan.AccessType.ValueString() == "DataTransfer" {
		if plan.MinimumCommitment.IsNull() || plan.MinimumCommitment.IsUnknown() || plan.MinimumCommitment.ValueString() == "" {
			resp.Diagnostics.AddError(
				"Missing Minimum Commitment",
				"Minimum commitment must be specified when access_type is 'DataTransfer'. Please provide a valid value for minimum_commitment.",
			)
			return
		}
		if err := client.ValidateMinimumCommitment(ctx,
			plan.FirewallThroughput.ValueString(),
			plan.MinimumCommitment.ValueString(),
		); err != nil {
			resp.Diagnostics.AddWarning(
				"Invalid Network Firewall Configuration",
				err.Error(),
			)
			return
		}
	}
}

// Create creates a new firewall resource.
func (r *NetworkFirewallResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data NetworkFirewallResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating network firewall", map[string]any{
		"firewall_display_name": data.FirewallDisplayName.ValueString(),
		"engagement_id":         data.EngagementID.ValueInt64(),
		"endpoint_id":           data.EndpointID.ValueInt64(),
	})

	// Validate that the engagement ID exists
	if err := account_engagement.ValidateEngagementExists(r.client, ctx, data.EngagementID.ValueInt64()); err != nil {
		resp.Diagnostics.AddError(
			"Invalid Engagement ID",
			err.Error(),
		)
		return
	}

	// Validate that the endpoint ID exists under the engagement
	if err := account_location.ValidateEndpointExists(r.client, ctx, data.EngagementID.ValueInt64(), data.EndpointID.ValueInt64()); err != nil {
		resp.Diagnostics.AddError(
			"Invalid Endpoint ID",
			err.Error(),
		)
		return
	}

	// Set access type based on presence of InternetBandwidth or MinimumCommitment
	if data.InternetBandwidth.IsUnknown() && data.MinimumCommitment.IsUnknown() {
		resp.Diagnostics.AddError(
			"Missing Internet Bandwidth and Minimum Commitment",
			"Either internet_bandwidth or minimum_commitment must be specified. Please provide a valid value for one of them.",
		)
		return
	}
	resp.Diagnostics.Append(validateAccessAndDetails(&data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.MinimumCommitment.IsNull() || data.MinimumCommitment.IsUnknown() || data.MinimumCommitment.ValueString() == "" {
		data.AccessType = types.StringValue("Bandwidth")
	} else if data.InternetBandwidth.IsNull() || data.InternetBandwidth.IsUnknown() || data.InternetBandwidth.ValueString() == "" {
		data.AccessType = types.StringValue("DataTransfer")
	}

	// Validate throughput >= bandwidth
	if data.AccessType.ValueString() == "Bandwidth" {
		if data.InternetBandwidth.IsNull() || data.InternetBandwidth.IsUnknown() || data.InternetBandwidth.ValueString() == "" {
			resp.Diagnostics.AddError(
				"Missing Internet Bandwidth",
				"Internet bandwidth must be specified when access_type is 'Bandwidth'. Please provide a valid value for internet_bandwidth.",
			)
			return
		}
		if err := client.ValidateThroughputAndBandwidth(
			ctx,
			data.FirewallThroughput.ValueString(),
			data.InternetBandwidth.ValueString(),
		); err != nil {
			resp.Diagnostics.AddError(
				"Invalid Network Firewall Configuration",
				err.Error(),
			)
			return
		}
	}
	if data.AccessType.ValueString() == "DataTransfer" {
		if data.MinimumCommitment.IsNull() || data.MinimumCommitment.IsUnknown() || data.MinimumCommitment.ValueString() == "" {
			resp.Diagnostics.AddError(
				"Missing Minimum Commitment",
				"Minimum commitment must be specified when access_type is 'DataTransfer'. Please provide a valid value for minimum_commitment.",
			)
			return
		}
		if err := client.ValidateMinimumCommitment(ctx,
			data.FirewallThroughput.ValueString(),
			data.MinimumCommitment.ValueString(),
		); err != nil {
			resp.Diagnostics.AddWarning(
				"Invalid Network Firewall Configuration",
				err.Error(),
			)
			// return
		}
	}

	// Build the create request
	createReq := &NetworkFirewallCreateRequest{
		EngagementID:          data.EngagementID.ValueInt64(),
		EndpointID:            data.EndpointID.ValueInt64(),
		FirewallDisplayName:   data.FirewallDisplayName.ValueString(),
		FirewallThroughput:    data.FirewallThroughput.ValueString(),
		AccessType:            data.AccessType.ValueString(),
		NonDistributedEnabled: data.NonDistributedEnabled.ValueBool(),
		Hypervisor:            data.Hypervisor.ValueString(),
		IsInternetEnabled:     data.IsInternetEnabled.ValueBool(),
	}

	// Only include pricing models if explicitly provided
	if !data.FirewallPricingModel.IsNull() && !data.FirewallPricingModel.IsUnknown() {
		val := data.FirewallPricingModel.ValueString()
		createReq.FirewallPricingModel = &val
	}
	if !data.InternetPricingModel.IsNull() && !data.InternetPricingModel.IsUnknown() {
		val := data.InternetPricingModel.ValueString()
		createReq.InternetPricingModel = &val
	}

	if data.AccessType.ValueString() == "Bandwidth" {
		createReq.Bandwidth = data.InternetBandwidth.ValueString()
		createReq.MinimumCommitment = ""
	}
	if data.AccessType.ValueString() == "DataTransfer" {
		createReq.MinimumCommitment = data.MinimumCommitment.ValueString()
		minimumCommitmentValue, err := client.ParseMinimumCommitmentValue(data.MinimumCommitment.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Invalid Minimum Commitment",
				"Could not parse minimum commitment: "+err.Error(),
			)
			return
		}
		equivalentBandwidthData, err := client.GetDataTransferBaseBandwidthTiers(minimumCommitmentValue)
		if err != nil {
			resp.Diagnostics.AddError(
				"Invalid Minimum Commitment",
				"Could not get data transfer base bandwidth tiers: "+err.Error(),
			)
			return
		}
		createReq.Bandwidth = equivalentBandwidthData[0]
	}

	// Create firewall and wait for completion
	auditLog, err := CreateNetworkFirewallAndWait(r.client, ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Network Firewall",
			"Could not create network firewall: "+err.Error(),
		)
		return
	}

	// Set the ID and audit ID
	if auditLog.ResourceID.String() != "" {
		data.ID = types.StringValue(auditLog.ResourceID.String())
	} else {
		resp.Diagnostics.AddError(
			"Error Creating Network Firewall",
			"Could not create network firewall: ResourceID is empty",
		)
		return
	}
	data.AuditID = types.StringValue(auditLog.AuditID)
	data.Status = types.StringValue(auditLog.Status)

	tflog.Info(ctx, "Network firewall created successfully", map[string]any{
		"id":       data.ID.ValueString(),
		"audit_id": data.AuditID.ValueString(),
		"status":   data.Status.ValueString(),
	})

	refreshDiags := r.refreshStateFromActionState(ctx, &data)
	resp.Diagnostics.Append(refreshDiags...)
	if refreshDiags.HasError() {
		return
	}
	// If action-state did not return optional+computed fields, avoid leaving Unknown in state
	if data.MinimumCommitment.IsUnknown() {
		data.MinimumCommitment = types.StringNull()
	}
	if data.InternetBandwidth.IsUnknown() {
		data.InternetBandwidth = types.StringNull()
	}

	// Skipping auditLog.Output parsing logic as per instructions.
	// The auditLog.Output sometimes contains only a wrapper string or null data,
	// so don't attempt to parse its contents or depend on it for state population.

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// refreshStateFromActionState loads firewall display, throughput, bandwidth, minimum
// commitment, access type, and pricing from the action-state read API. It does not
// set id, audit_id, or status (not returned in the inner Data payload in this flow).
func (r *NetworkFirewallResource) refreshStateFromActionState(ctx context.Context, data *NetworkFirewallResourceModel) diag.Diagnostics {
	var d diag.Diagnostics
	actionStateBody := map[string]any{
		"resourceId": data.ID.ValueString(),
	}
	tflog.Debug(ctx, "Data object", map[string]any{
		"data_object":            data.ID.ValueString(),
		"engagement_id":          data.EngagementID.ValueInt64(),
		"endpoint_id":            data.EndpointID.ValueInt64(),
		"firewall_display_name":  data.FirewallDisplayName.ValueString(),
		"firewall_throughput":    data.FirewallThroughput.ValueString(),
		"internet_bandwidth":     data.InternetBandwidth.ValueString(),
		"minimum_commitment":     data.MinimumCommitment.ValueString(),
		"access_type":            data.AccessType.ValueString(),
		"firewall_pricing_model": data.FirewallPricingModel.ValueString(),
		"internet_pricing_model": data.InternetPricingModel.ValueString(),
	})
	actionStateResponse, err := common.UpdateActionState(ctx, r.client, "firewall", "read", actionStateBody)
	if err != nil {
		d.AddError("Error Reading Network Firewall", "Could not read network firewall: "+err.Error())
		return d
	}
	if len(actionStateResponse.Data) == 0 {
		d.AddWarning(
			"Empty Response Data",
			"Could not read network firewall: response data is empty, keeping existing values for API-derived attributes",
		)
		return d
	}
	var responseMap map[string]interface{}
	if err := json.Unmarshal(actionStateResponse.Data, &responseMap); err != nil {
		d.AddError(
			"Error Unmarshalling Response Data",
			fmt.Sprintf("Could not unmarshal response data: %s. Response: %s", err.Error(), string(actionStateResponse.Data)),
		)
		return d
	}
	data.FirewallDisplayName = types.StringValue(responseMap["firewall_display_name"].(string))
	data.FirewallThroughput = types.StringValue(responseMap["firewall_throughput"].(string))
	if v, ok := responseMap["bandwidth"]; ok && v != nil {
		data.InternetBandwidth = types.StringValue(v.(string))
		data.AccessType = types.StringValue(responseMap["access_type"].(string))
		// data.AccessType = types.StringValue("Bandwidth")
	}
	if v, ok := responseMap["minimum_commitment"]; ok && v != nil {
		data.MinimumCommitment = types.StringValue(v.(string))
		data.AccessType = types.StringValue(responseMap["access_type"].(string))
		// data.AccessType = types.StringValue("DataTransfer")
	}
	data.FirewallPricingModel = types.StringValue(responseMap["firewall_pricing_model"].(string))
	data.InternetPricingModel = types.StringValue(responseMap["internet_pricing_model"].(string))
	return d
}

// Read refreshes the Terraform state with the latest data from the API.
func (r *NetworkFirewallResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data NetworkFirewallResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, "Reading network firewall", map[string]any{
		"id":       data.ID.ValueString(),
		"audit_id": data.AuditID.ValueString(),
	})
	refreshDiags := r.refreshStateFromActionState(ctx, &data)
	resp.Diagnostics.Append(refreshDiags...)
	if refreshDiags.HasError() {
		return
	}
	tflog.Info(ctx, "Network firewall read completed", map[string]any{
		"id":     data.ID.ValueString(),
		"status": data.Status.ValueString(),
	})
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update updates the resource.
func (r *NetworkFirewallResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan NetworkFirewallResourceModel
	var state NetworkFirewallResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Read current state
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Updating network firewall", map[string]any{
		"id":                    plan.ID.ValueString(),
		"firewall_display_name": plan.FirewallDisplayName.ValueString(),
		"firewall_throughput":   plan.FirewallThroughput.ValueString(),
		"access_type":           plan.AccessType.ValueString(),
	})

	// Validate that the engagement ID exists
	if err := account_engagement.ValidateEngagementExists(r.client, ctx, plan.EngagementID.ValueInt64()); err != nil {
		resp.Diagnostics.AddError(
			"Invalid Engagement ID",
			err.Error(),
		)
		return
	}

	// Validate that the endpoint ID exists under the engagement
	if err := account_location.ValidateEndpointExists(r.client, ctx, plan.EngagementID.ValueInt64(), plan.EndpointID.ValueInt64()); err != nil {
		resp.Diagnostics.AddError(
			"Invalid Endpoint ID",
			err.Error(),
		)
		return
	}

	// Convert firewall ID to int64 (needed for both update operations)
	firewallID, err := client.StringToInt64(plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Network Firewall",
			"Could not parse network firewall ID: "+err.Error(),
		)
		return
	}

	// Check if firewall_display_name changed
	displayNameChanged := plan.FirewallDisplayName.ValueString() != state.FirewallDisplayName.ValueString()

	// Check if firewall_throughput / bandwidth / minimum commitment changed (must be function-scoped for use below)
	var throughputOrBandwidthChanged bool
	var throughputOrMinimumCommitmentChanged bool // default value before any execution: false
	if plan.AccessType.ValueString() == "Bandwidth" {
		throughputOrBandwidthChanged = plan.FirewallThroughput.ValueString() != state.FirewallThroughput.ValueString() ||
			plan.InternetBandwidth.ValueString() != state.InternetBandwidth.ValueString()
	}
	if plan.AccessType.ValueString() == "DataTransfer" {
		throughputOrMinimumCommitmentChanged = plan.FirewallThroughput.ValueString() != state.FirewallThroughput.ValueString() ||
			plan.MinimumCommitment.ValueString() != state.MinimumCommitment.ValueString()
	}

	// Update display name if it changed
	if displayNameChanged {
		tflog.Debug(ctx, "Updating firewall display name", map[string]any{
			"firewall_id": firewallID,
			"old_name":    state.FirewallDisplayName.ValueString(),
			"new_name":    plan.FirewallDisplayName.ValueString(),
		})

		// Update firewall display name (synchronous operation, no audit polling)
		err := UpdateNetworkFirewallDisplayName(
			r.client,
			ctx,
			firewallID,
			plan.FirewallDisplayName.ValueString(),
		)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Updating Network Firewall Display Name",
				"Could not update network firewall display name: "+err.Error(),
			)
			return
		}

		tflog.Info(ctx, "Network firewall display name updated successfully", map[string]any{
			"id":           plan.ID.ValueString(),
			"display_name": plan.FirewallDisplayName.ValueString(),
		})
		// Update status to COMPLETED
		// plan.Status = types.StringValue("COMPLETED")
	}

	// Update throughput and bandwidth if they changed
	if throughputOrBandwidthChanged {
		// Validate throughput >= bandwidth
		if err := client.ValidateThroughputAndBandwidth(
			ctx,
			plan.FirewallThroughput.ValueString(),
			plan.InternetBandwidth.ValueString(),
		); err != nil {
			resp.Diagnostics.AddError(
				"Invalid Network Firewall Configuration",
				err.Error(),
			)
			return
		}

		tflog.Debug(ctx, "Updating network firewall throughput and bandwidth", map[string]any{
			"firewall_id":         firewallID,
			"firewall_throughput": plan.FirewallThroughput.ValueString(),
			"internet_bandwidth":  plan.InternetBandwidth.ValueString(),
		})

		// Update firewall and wait for completion
		_, err := UpdateNetworkFirewallAndWait(
			r.client,
			ctx,
			firewallID,
			plan.FirewallThroughput.ValueString(),
			plan.InternetBandwidth.ValueString(),
			"",
		)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Updating Network Firewall",
				"Could not update network firewall throughput/bandwidth: "+err.Error(),
			)
			return
		}

		// Update status from audit log
		// plan.Status = types.StringValue(auditLog.Status)
		// plan.AuditID = types.StringValue(auditLog.AuditID)

		tflog.Info(ctx, "Network firewall throughput and bandwidth updated successfully", map[string]any{
			"id":                  plan.ID.ValueString(),
			"firewall_throughput": plan.FirewallThroughput.ValueString(),
			"internet_bandwidth":  plan.InternetBandwidth.ValueString(),
			"audit_id":            plan.AuditID.ValueString(),
			"status":              plan.Status.ValueString(),
		})
	}

	// Update throughput and bandwidth if they changed
	if throughputOrMinimumCommitmentChanged {
		// Validate throughput >= bandwidth
		if err := client.ValidateMinimumCommitment(ctx,
			plan.FirewallThroughput.ValueString(),
			plan.MinimumCommitment.ValueString(),
		); err != nil {
			resp.Diagnostics.AddWarning(
				"Invalid Network Firewall Configuration",
				err.Error(),
			)
			// return
		}
		minimumCommitmentValue, err := client.ParseMinimumCommitmentValue(plan.MinimumCommitment.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Invalid Minimum Commitment",
				"Could not parse minimum commitment: "+err.Error(),
			)
			return
		}
		equivalentBandwidthData, err := client.GetDataTransferBaseBandwidthTiers(minimumCommitmentValue)
		if err != nil {
			resp.Diagnostics.AddError(
				"Invalid Minimum Commitment",
				"Could not get data transfer base bandwidth tiers: "+err.Error(),
			)
			return
		}
		plan.InternetBandwidth = types.StringValue(equivalentBandwidthData[0])

		tflog.Debug(ctx, "Updating network firewall throughput and minimum commitment", map[string]any{
			"firewall_id":         firewallID,
			"firewall_throughput": plan.FirewallThroughput.ValueString(),
			"minimum_commitment":  plan.MinimumCommitment.ValueString(),
		})

		// Update firewall and wait for completion
		auditLog, err := UpdateNetworkFirewallAndWait(
			r.client,
			ctx,
			firewallID,
			plan.FirewallThroughput.ValueString(),
			plan.InternetBandwidth.ValueString(),
			plan.MinimumCommitment.ValueString(),
		)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Updating Network Firewall",
				"Could not update network firewall throughput/minimum commitment: "+err.Error(),
			)
			return
		}
		tflog.Info(ctx, "Network firewall throughput and minimum commitment updated successfully", map[string]any{
			"id":                  auditLog.ResourceID.String(),
			"firewall_throughput": plan.FirewallThroughput.ValueString(),
			"minimum_commitment":  plan.MinimumCommitment.ValueString(),
			"audit_id":            auditLog.AuditID,
			"status":              auditLog.Status,
		})
	}

	// If nothing changed, just save the state
	if !displayNameChanged && !throughputOrBandwidthChanged && !throughputOrMinimumCommitmentChanged {
		tflog.Debug(ctx, "No changes detected, skipping update", map[string]any{
			"id": plan.ID.ValueString(),
		})
	}

	out := plan
	// refreshDiags := r.refreshStateFromActionState(ctx, &out)
	// resp.Diagnostics.Append(refreshDiags...)
	// if refreshDiags.HasError() {
	// 	return
	// }
	// Action-state read does not populate audit_id / status; keep values from the plan or last state
	if out.AuditID.IsUnknown() || out.AuditID.IsNull() {
		out.AuditID = state.AuditID
	}
	if out.Status.IsUnknown() || out.Status.IsNull() {
		out.Status = state.Status
	}
	if out.MinimumCommitment.IsUnknown() {
		out.MinimumCommitment = types.StringNull()
	}
	if out.InternetBandwidth.IsUnknown() {
		out.InternetBandwidth = types.StringNull()
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &out)...)
}

// Delete deletes the firewall resource.
func (r *NetworkFirewallResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data NetworkFirewallResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting network firewall", map[string]any{
		"id":       data.ID.ValueString(),
		"audit_id": data.AuditID.ValueString(),
	})

	// Validate that the engagement ID exists
	if err := account_engagement.ValidateEngagementExists(r.client, ctx, data.EngagementID.ValueInt64()); err != nil {
		resp.Diagnostics.AddError(
			"Invalid Engagement ID",
			err.Error(),
		)
		return
	}

	// Validate that the endpoint ID exists under the engagement
	if err := account_location.ValidateEndpointExists(r.client, ctx, data.EngagementID.ValueInt64(), data.EndpointID.ValueInt64()); err != nil {
		resp.Diagnostics.AddError(
			"Invalid Endpoint ID",
			err.Error(),
		)
		return
	}

	// Delete firewall and wait for completion
	_, err := DeleteNetworkFirewallAndWait(r.client, ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Network Firewall",
			"Could not delete network firewall: "+err.Error(),
		)
		return
	}

	tflog.Info(ctx, "Network firewall deleted successfully", map[string]any{
		"id": data.ID.ValueString(),
	})
}

// ImportState imports an existing network firewall into Terraform state.
func (r *NetworkFirewallResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import using the audit_id
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func validateAccessAndDetails(data *NetworkFirewallResourceModel) diag.Diagnostics {
	var d diag.Diagnostics
	if !(data.AccessType.IsNull() || data.AccessType.IsUnknown() || data.AccessType.ValueString() == "") {
		if data.AccessType.ValueString() == "Bandwidth" && (data.InternetBandwidth.IsNull() || data.InternetBandwidth.IsUnknown() || data.InternetBandwidth.ValueString() == "") {
			d.AddError("Invalid Network Firewall Configuration", "Internet bandwidth must be specified when access_type is 'Bandwidth'")
		}
		if data.AccessType.ValueString() == "DataTransfer" && (data.MinimumCommitment.IsNull() || data.MinimumCommitment.IsUnknown() || data.MinimumCommitment.ValueString() == "") {
			d.AddError("Invalid Network Firewall Configuration", "Minimum commitment must be specified when access_type is 'DataTransfer'")
		}
	}
	return d
}
