package network_lb_ssl_profile

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/client"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/network_lb"
)

var (
	_ resource.Resource                = &NetworkLBSSLProfileResource{}
	_ resource.ResourceWithImportState = &NetworkLBSSLProfileResource{}
	_ resource.ResourceWithModifyPlan  = &NetworkLBSSLProfileResource{}
)

func NewNetworkLBSSLProfileResource() resource.Resource {
	return &NetworkLBSSLProfileResource{}
}

type NetworkLBSSLProfileResource struct {
	client *client.Client
}

type NetworkLBSSLProfileResourceModel struct {
	ID              types.String `tfsdk:"id"`
	LoadBalancerID  types.String `tfsdk:"load_balancer_id"`
	CertificateName types.String `tfsdk:"certificate_name"`
	Certificate     types.String `tfsdk:"certificate"`
	PrivateKey      types.String `tfsdk:"private_key"`
	ValidFrom       types.String `tfsdk:"valid_from"`
	ValidTo         types.String `tfsdk:"valid_to"`
	CIStatus        types.String `tfsdk:"ci_status"`
	AuditID         types.String `tfsdk:"audit_id"`
	Status          types.String `tfsdk:"status"`
	LastUpdated     types.String `tfsdk:"last_updated"`
}

func (r *NetworkLBSSLProfileResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network_lb_ssl_profile"
}

func (r *NetworkLBSSLProfileResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an HAProxy SSL client profile on a VayuCloud Load Balancer. " +
			"Provide the certificate upload name and PEM file contents (same as uploading cert/key in the UI). " +
			"The upload name may omit `.pem`; HAProxy stores it as `{name}.pem`.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Composite id `{load_balancer_id}/{name}.pem` (e.g. 399729/test.pem).",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"load_balancer_id": schema.StringAttribute{
				MarkdownDescription: "Parent Load Balancer CI Master ID.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"certificate_name": schema.StringAttribute{
				MarkdownDescription: "Certificate upload name (UI label). May omit `.pem`; use `{name}.pem` on HTTPS virtual services.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"certificate": schema.StringAttribute{
				MarkdownDescription: "Certificate content (PEM format).",
				Required:            true,
				Sensitive:           true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"private_key": schema.StringAttribute{
				MarkdownDescription: "Private key content (PEM format).",
				Required:            true,
				Sensitive:           true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"valid_from": schema.StringAttribute{
				MarkdownDescription: "Certificate validity start date.",
				Computed:            true,
			},
			"valid_to": schema.StringAttribute{
				MarkdownDescription: "Certificate validity end date.",
				Computed:            true,
			},
			"ci_status": schema.StringAttribute{
				MarkdownDescription: "Parent load balancer CI status after refresh.",
				Computed:            true,
			},
			"audit_id": schema.StringAttribute{
				MarkdownDescription: "Audit ID from the latest operation.",
				Computed:            true,
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "Audit status.",
				Computed:            true,
			},
			"last_updated": schema.StringAttribute{
				MarkdownDescription: "Timestamp of last update.",
				Computed:            true,
			},
		},
	}
}

func (r *NetworkLBSSLProfileResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *NetworkLBSSLProfileResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if r.client == nil || req.Plan.Raw.IsNull() {
		return
	}

	var plan NetworkLBSSLProfileResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if plan.LoadBalancerID.IsUnknown() || plan.LoadBalancerID.IsNull() {
		return
	}

	lbCiID, err := strconv.ParseInt(strings.TrimSpace(plan.LoadBalancerID.ValueString()), 10, 64)
	if err != nil || lbCiID <= 0 {
		resp.Diagnostics.AddAttributeError(path.Root("load_balancer_id"), "Invalid Load Balancer ID", "load_balancer_id must be a positive integer.")
		return
	}
	if err := network_lb.ValidateLoadBalancerExists(r.client, ctx, lbCiID); err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("load_balancer_id"), "Invalid Load Balancer ID", err.Error())
	}
}

func (r *NetworkLBSSLProfileResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data NetworkLBSSLProfileResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.CertificateName.IsNull() || strings.TrimSpace(data.CertificateName.ValueString()) == "" {
		resp.Diagnostics.AddError("Missing certificate_name", "certificate_name is required when creating an SSL profile.")
		return
	}

	lbCiID, err := strconv.ParseInt(data.LoadBalancerID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid LB ID", fmt.Sprintf("Could not parse LB ID: %s", err.Error()))
		return
	}

	uploadName := stripPEMSuffix(data.CertificateName.ValueString())
	tflog.Info(ctx, "Creating SSL Profile", map[string]any{
		"upload_name":      uploadName,
		"load_balancer_id": lbCiID,
	})

	createReq := &SSLProfileCreateRequest{
		LoadBalancerCI:     lbCiID,
		CertificateName:    uploadName,
		CertificateContent: data.Certificate.ValueString(),
		PrivateKeyContent:  data.PrivateKey.ValueString(),
	}

	auditResp, err := CreateSSLProfile(r.client, ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Failed to Create SSL Profile", err.Error())
		return
	}

	pollDiags := r.pollAuditLog(ctx, auditResp.Data.Audit.AuditID, "create", "sslprofile", uploadName)
	resp.Diagnostics.Append(pollDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.AuditID = types.StringValue(auditResp.Data.Audit.AuditID)
	data.Status = types.StringValue(auditResp.Status)
	data.LastUpdated = types.StringValue(time.Now().Format(time.RFC3339))

	diags := r.refreshState(ctx, &data)
	normalizeUnknownSSLProfileComputedValues(&data)
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *NetworkLBSSLProfileResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data NetworkLBSSLProfileResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags := r.refreshState(ctx, &data)
	normalizeUnknownSSLProfileComputedValues(&data)
	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *NetworkLBSSLProfileResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"Update Not Supported",
		"SSL Profile attributes cannot be updated. Remove and recreate the resource.",
	)
}

func (r *NetworkLBSSLProfileResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data NetworkLBSSLProfileResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	lbCiID, err := r.resolveLoadBalancerID(&data)
	if err != nil {
		resp.Diagnostics.AddError("Invalid SSL Profile Identity", err.Error())
		return
	}

	storageName := ensurePEMSuffix(strings.TrimSpace(data.CertificateName.ValueString()))
	if storageName == "" {
		resp.Diagnostics.AddError("Invalid SSL Profile Identity", "certificate_name is required in state for delete")
		return
	}

	tflog.Info(ctx, "Deleting SSL Profile", map[string]any{
		"load_balancer_id": lbCiID,
		"certificate_name": storageName,
	})

	auditResp, err := DeleteSSLProfile(r.client, ctx, lbCiID, storageName)
	if err != nil {
		resp.Diagnostics.AddError("Failed to Delete SSL Profile", err.Error())
		return
	}

	pollDiags := r.pollAuditLog(ctx, auditResp.Data.Audit.AuditID, "delete", "sslprofile", stripPEMSuffix(storageName))
	resp.Diagnostics.Append(pollDiags...)
}

func (r *NetworkLBSSLProfileResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	lbCiID, certName, ok := parseCompositeSSLProfileID(req.ID)
	if !ok {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			"Expected format `{load_balancer_id}/{certificate_name}` (e.g. 399729/test.pem).",
		)
		return
	}

	storageName := ensurePEMSuffix(certName)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), compositeSSLProfileID(lbCiID, storageName))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("load_balancer_id"), strconv.FormatInt(lbCiID, 10))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("certificate_name"), stripPEMSuffix(certName))...)
}

func (r *NetworkLBSSLProfileResource) pollAuditLog(ctx context.Context, auditID, action, module, uploadName string) diag.Diagnostics {
	var diags diag.Diagnostics

	requestBody := map[string]any{
		"certificateName": stripPEMSuffix(uploadName),
	}

	_, err := r.client.WaitForAuditCompletion(ctx, auditID, action, module, requestBody)
	if err != nil {
		diags.AddError(
			"Audit Polling Timeout",
			fmt.Sprintf("SSL Profile operation did not complete within timeout: %s", err.Error()),
		)
	}
	return diags
}

func (r *NetworkLBSSLProfileResource) refreshState(ctx context.Context, data *NetworkLBSSLProfileResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	lbCiID, lookupName, err := r.resolveLBAndCertLookup(data)
	if err != nil {
		diags.AddError("Invalid SSL Profile Identity", err.Error())
		return diags
	}

	profile, foundErr := FindSSLProfileOnLB(r.client, ctx, lbCiID, lookupName)
	if foundErr != nil {
		diags.AddError("SSL Profile Not Found", foundErr.Error())
		return diags
	}

	storageName := stringFromActionStateMap(profile, "name", "storage_name")
	if storageName == "" {
		diags.AddError("SSL Profile Not Found", "list API returned profile without a name")
		return diags
	}

	data.ID = types.StringValue(compositeSSLProfileID(lbCiID, storageName))
	data.LoadBalancerID = types.StringValue(strconv.FormatInt(lbCiID, 10))

	details, err := GetSSLProfileDetails(r.client, ctx, lbCiID, storageName)
	if err != nil {
		diags.AddWarning("Failed to Refresh SSL Profile State", fmt.Sprintf("Action-state read failed: %s", err.Error()))
	} else {
		if v := stringFromActionStateMap(details, "valid_from"); v != "" {
			data.ValidFrom = types.StringValue(v)
		}
		if v := stringFromActionStateMap(details, "valid_to"); v != "" {
			data.ValidTo = types.StringValue(v)
		}
		if v := stringFromActionStateMap(details, "ci_status"); v != "" {
			data.CIStatus = types.StringValue(v)
		}
	}

	data.LastUpdated = types.StringValue(time.Now().Format(time.RFC3339))
	normalizeUnknownSSLProfileComputedValues(data)
	return diags
}

func normalizeUnknownSSLProfileComputedValues(data *NetworkLBSSLProfileResourceModel) {
	if data.ValidFrom.IsUnknown() {
		data.ValidFrom = types.StringNull()
	}
	if data.ValidTo.IsUnknown() {
		data.ValidTo = types.StringNull()
	}
	if data.CIStatus.IsUnknown() {
		data.CIStatus = types.StringNull()
	}
}

func (r *NetworkLBSSLProfileResource) resolveLoadBalancerID(data *NetworkLBSSLProfileResourceModel) (int64, error) {
	if lbCiID, _, ok := parseCompositeSSLProfileID(data.ID.ValueString()); ok {
		return lbCiID, nil
	}

	lbCiID, err := strconv.ParseInt(strings.TrimSpace(data.LoadBalancerID.ValueString()), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("could not parse load_balancer_id: %w", err)
	}
	return lbCiID, nil
}

// resolveLBAndCertLookup returns LB id and a name used only to find the cert in the list API.
func (r *NetworkLBSSLProfileResource) resolveLBAndCertLookup(data *NetworkLBSSLProfileResourceModel) (int64, string, error) {
	lbCiID, err := r.resolveLoadBalancerID(data)
	if err != nil {
		return 0, "", err
	}

	if _, certName, ok := parseCompositeSSLProfileID(data.ID.ValueString()); ok && certName != "" {
		return lbCiID, certName, nil
	}

	lookupName := strings.TrimSpace(data.CertificateName.ValueString())
	if lookupName == "" {
		return 0, "", fmt.Errorf("certificate_name is required")
	}
	return lbCiID, lookupName, nil
}
