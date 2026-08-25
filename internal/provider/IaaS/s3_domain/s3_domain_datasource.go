// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package s3_domain

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/client"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/account_engagement"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/network_firewall"
)

var _ datasource.DataSource = &S3DomainDataSource{}

func NewS3DomainDataSource() datasource.DataSource {
	return &S3DomainDataSource{}
}

type S3DomainDataSource struct {
	client *client.Client
}

type S3DomainDataSourceModel struct {
	ID types.String `tfsdk:"id"`

	S3DomainID types.String `tfsdk:"s3_domain_id"`
	FirewallID types.Int64  `tfsdk:"firewall_id"`

	DomainNameFQDN       types.String  `tfsdk:"domain_name_fqdn"`
	Quota                types.Float64 `tfsdk:"quota"`
	QuotaUnit            types.String  `tfsdk:"quota_unit"`
	StorageClass         types.String  `tfsdk:"storage_class"`
	Variant              types.String  `tfsdk:"variant"`
	EngagementID         types.Int64   `tfsdk:"engagement_id"`
	EndpointID           types.Int64   `tfsdk:"endpoint_id"`
	DomainAccessIP       types.String  `tfsdk:"domain_access_ip"`
	DomainAccessPublicIP types.String  `tfsdk:"domain_access_public_ip"`

	Status       types.String `tfsdk:"status"`
	Message      types.String `tfsdk:"message"`
	ResponseCode types.Int64  `tfsdk:"response_code"`
}

func (d *S3DomainDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_s3_domain"
}

func (d *S3DomainDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves details of an S3 domain from the VayuCloud API.",
		MarkdownDescription: "Retrieves details of an S3 domain from the VayuCloud API.\n\n" +
			"This data source calls the action-state API with `module=domain` and `action=read`. " +
			"`engagement_id` and `firewall_id` are required on the read request.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Placeholder identifier for Terraform (same as s3_domain_id).",
				Computed:    true,
			},
			"s3_domain_id": schema.StringAttribute{
				Description: "The numeric S3 domain ID to look up.",
				Required:    true,
			},
			"engagement_id": schema.Int64Attribute{
				Description: "The IPC engagement ID. Required for the action-state read request.",
				Required:    true,
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
				},
			},
			"firewall_id": schema.Int64Attribute{
				Description: "IPC firewall ci_master.id mapped to the domain for private S3 access. Required.",
				Required:    true,
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
				},
			},
			"domain_name_fqdn": schema.StringAttribute{
				Description: "Fully-qualified domain name (domain_name + .ipstorage.tatacommunications.com).",
				Computed:    true,
			},
			"quota": schema.Float64Attribute{
				Description: "Storage quota in GB as reported by the server.",
				Computed:    true,
			},
			"quota_unit": schema.StringAttribute{
				Description: "Unit of the provisioned quota. Always GB from the action-state read API.",
				Computed:    true,
			},
			"storage_class": schema.StringAttribute{
				Description: "Storage class (STANDARD, HIGH_PERFORMANCE, AI_STANDARD).",
				Computed:    true,
			},
			"variant": schema.StringAttribute{
				Description: "Domain variant (value, geoResilient, resilient).",
				Computed:    true,
			},
			"endpoint_id": schema.Int64Attribute{
				Description: "Endpoint ID as returned by the API.",
				Computed:    true,
			},
			"domain_access_ip": schema.StringAttribute{
				Description: "Private IP for S3 access over the firewall. Null when not available.",
				Computed:    true,
			},
			"domain_access_public_ip": schema.StringAttribute{
				Description: "Public IP for S3 access when a mapping exists. Null when not available.",
				Computed:    true,
			},
			"status": schema.StringAttribute{
				Description: "The API response status (e.g. success).",
				Computed:    true,
			},
			"message": schema.StringAttribute{
				Description: "Additional information from the API response.",
				Computed:    true,
			},
			"response_code": schema.Int64Attribute{
				Description: "The API response code (0 = success).",
				Computed:    true,
			},
		},
	}
}

func (d *S3DomainDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}
	d.client = c
}

func (d *S3DomainDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data S3DomainDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Reading S3 domain data source", map[string]any{
		"s3_domain_id":  data.S3DomainID.ValueString(),
		"engagement_id": data.EngagementID.ValueInt64(),
		"firewall_id":   data.FirewallID.ValueInt64(),
	})

	if err := account_engagement.ValidateEngagementExists(d.client, ctx, data.EngagementID.ValueInt64()); err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("engagement_id"),
			"Invalid engagement ID",
			err.Error(),
		)
		return
	}

	if err := network_firewall.ValidateFirewallExists(d.client, ctx, data.FirewallID.ValueInt64()); err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("firewall_id"),
			"Invalid firewall ID",
			err.Error(),
		)
		return
	}

	payload, actionStateResp, err := ReadS3DomainActionState(
		d.client, ctx, data.EngagementID.ValueInt64(), data.S3DomainID.ValueString(), data.FirewallID.ValueInt64(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading S3 Domain", err.Error())
		return
	}

	data.ID = data.S3DomainID
	data.Status = types.StringValue(actionStateResp.Status)
	data.Message = types.StringValue(actionStateResp.Message)
	data.ResponseCode = types.Int64Value(int64(actionStateResp.ResponseCode))

	mapPayloadToDataSourceModel(payload, &data)

	tflog.Info(ctx, "S3 domain data source read successfully", map[string]any{
		"s3_domain_id":     data.S3DomainID.ValueString(),
		"domain_name_fqdn": data.DomainNameFQDN.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func mapPayloadToDataSourceModel(payload *S3DomainPayload, data *S3DomainDataSourceModel) {
	if payload.DomainName != "" {
		data.DomainNameFQDN = types.StringValue(payload.DomainName)
	}
	if payload.Quota > 0 {
		data.Quota = types.Float64Value(payload.Quota)
	}
	if payload.QuotaUnit != "" {
		data.QuotaUnit = types.StringValue(payload.QuotaUnit)
	}
	if payload.StorageClass != "" {
		data.StorageClass = types.StringValue(payload.StorageClass)
	}
	if payload.Variant != "" {
		data.Variant = types.StringValue(payload.Variant)
	}
	if payload.EndpointID > 0 {
		data.EndpointID = types.Int64Value(payload.EndpointID)
	}
	data.DomainAccessIP = stringAttrOrNull(payload.DomainAccessIP)
	data.DomainAccessPublicIP = stringAttrOrNull(payload.DomainAccessPublicIP)
}

func stringAttrOrNull(s string) types.String {
	if s == "" {
		return types.StringNull()
	}
	return types.StringValue(s)
}

func optionalInt64FromTypes(v types.Int64) *int64 {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	val := v.ValueInt64()
	return &val
}
