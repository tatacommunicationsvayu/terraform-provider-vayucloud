// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package s3_domain

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/client"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/account_engagement"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/account_location"
)

var _ datasource.DataSource = &S3DomainListDataSource{}

func NewS3DomainListDataSource() datasource.DataSource {
	return &S3DomainListDataSource{}
}

type S3DomainListDataSource struct {
	client *client.Client
}

type S3DomainListItemModel struct {
	S3DomainID           types.String  `tfsdk:"s3_domain_id"`
	DomainNameFQDN       types.String  `tfsdk:"domain_name_fqdn"`
	Quota                types.Float64 `tfsdk:"quota"`
	QuotaUnit            types.String  `tfsdk:"quota_unit"`
	StorageClass         types.String  `tfsdk:"storage_class"`
	Variant              types.String  `tfsdk:"variant"`
	EngagementID         types.Int64   `tfsdk:"engagement_id"`
	EndpointID           types.Int64   `tfsdk:"endpoint_id"`
	DomainAccessIP       types.String  `tfsdk:"domain_access_ip"`
	DomainAccessPublicIP types.String  `tfsdk:"domain_access_public_ip"`
}

type S3DomainListDataSourceModel struct {
	ID types.Int64 `tfsdk:"id"`

	EngagementID types.Int64 `tfsdk:"engagement_id"`
	EndpointID   types.Int64 `tfsdk:"endpoint_id"`
	FirewallID   types.Int64 `tfsdk:"firewall_id"`

	Filters []client.FilterModel `tfsdk:"filter"`

	Domains []S3DomainListItemModel `tfsdk:"domains"`

	Status       types.String `tfsdk:"status"`
	Message      types.String `tfsdk:"message"`
	ResponseCode types.Int64  `tfsdk:"response_code"`
}

func (d *S3DomainListDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_s3_domain_list"
}

func (d *S3DomainListDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves the list of S3 domains for an IPC engagement and endpoint.",
		MarkdownDescription: "Retrieves the list of S3 domains for an IPC engagement and endpoint.\n\n" +
			"This data source calls the action-state API with `module=domain` and `action=list`. " +
			"The backend resolves IPC to ICS engagement internally.\n\n" +
			"Optionally set `firewall_id` to include access IPs in list results. " +
			"Use `filter` blocks to narrow results client-side.",

		Blocks: map[string]schema.Block{
			"filter": client.FilterBlockSchema(),
		},

		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "Placeholder identifier for Terraform (engagement_id).",
				Computed:    true,
			},
			"engagement_id": schema.Int64Attribute{
				Description: "The IPC engagement ID to list domains for.",
				Required:    true,
			},
			"endpoint_id": schema.Int64Attribute{
				Description: "The endpoint ID to list domains for.",
				Required:    true,
			},
			"firewall_id": schema.Int64Attribute{
				Description: "Optional IPC firewall ID. When set, the list request includes firewallId so access IPs may be returned.",
				Optional:    true,
			},
			"domains": schema.ListNestedAttribute{
				Description: "The list of S3 domains.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"s3_domain_id": schema.StringAttribute{
							Description: "The S3 domain resource ID.",
							Computed:    true,
						},
						"domain_name_fqdn": schema.StringAttribute{
							Description: "Fully-qualified domain name.",
							Computed:    true,
						},
						"quota": schema.Float64Attribute{
							Description: "Storage quota in GB.",
							Computed:    true,
						},
						"quota_unit": schema.StringAttribute{
							Description: "Quota unit (always GB from the API).",
							Computed:    true,
						},
						"storage_class": schema.StringAttribute{
							Description: "Storage class.",
							Computed:    true,
						},
						"variant": schema.StringAttribute{
							Description: "Domain variant.",
							Computed:    true,
						},
						"engagement_id": schema.Int64Attribute{
							Description: "Engagement ID as returned by the API.",
							Computed:    true,
						},
						"endpoint_id": schema.Int64Attribute{
							Description: "Endpoint ID as returned by the API.",
							Computed:    true,
						},
						"domain_access_ip": schema.StringAttribute{
							Description: "Private IP for S3 access over the firewall.",
							Computed:    true,
						},
						"domain_access_public_ip": schema.StringAttribute{
							Description: "Public IP when available.",
							Computed:    true,
						},
					},
				},
			},
			"status": schema.StringAttribute{
				Description: "The API response status.",
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

func (d *S3DomainListDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *S3DomainListDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data S3DomainListDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	engagementID := data.EngagementID.ValueInt64()
	endpointID := data.EndpointID.ValueInt64()

	if err := account_engagement.ValidateEngagementExists(d.client, ctx, engagementID); err != nil {
		resp.Diagnostics.AddError("Error Validating Engagement", err.Error())
		return
	}
	if err := account_location.ValidateEndpointExists(d.client, ctx, engagementID, endpointID); err != nil {
		resp.Diagnostics.AddError("Error Validating Endpoint", err.Error())
		return
	}

	tflog.Debug(ctx, "Reading S3 domain list data source", map[string]any{
		"engagement_id": engagementID,
		"endpoint_id":   endpointID,
	})

	items, actionStateResp, err := ListS3Domains(
		d.client, ctx, engagementID, endpointID, optionalInt64FromTypes(data.FirewallID),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Listing S3 Domains",
			fmt.Sprintf("Could not list domains for engagement %d / endpoint %d: %s",
				engagementID, endpointID, err.Error()),
		)
		return
	}

	data.ID = types.Int64Value(engagementID)
	data.Status = types.StringValue(actionStateResp.Status)
	data.Message = types.StringValue(actionStateResp.Message)
	data.ResponseCode = types.Int64Value(int64(actionStateResp.ResponseCode))

	if len(items) == 0 {
		resp.Diagnostics.AddWarning(
			"Empty Response Data",
			"No S3 domains found for the given engagement and endpoint.",
		)
		data.Domains = []S3DomainListItemModel{}
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
		return
	}

	domains := make([]S3DomainListItemModel, len(items))
	for i, item := range items {
		domains[i] = payloadToListItemModel(&item)
	}

	if len(data.Filters) > 0 {
		filtered, filterErr := client.ApplyFilters(domains, data.Filters)
		if filterErr != nil {
			resp.Diagnostics.AddError("Error Filtering S3 Domains", filterErr.Error())
			return
		}
		domains = filtered
	}

	data.Domains = domains

	tflog.Info(ctx, "S3 domain list read successfully", map[string]any{
		"count":         len(domains),
		"engagement_id": engagementID,
		"endpoint_id":   endpointID,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func payloadToListItemModel(payload *S3DomainPayload) S3DomainListItemModel {
	item := S3DomainListItemModel{
		S3DomainID:           types.StringValue(fmt.Sprintf("%d", payload.ID)),
		DomainAccessIP:       stringAttrOrNull(payload.DomainAccessIP),
		DomainAccessPublicIP: stringAttrOrNull(payload.DomainAccessPublicIP),
	}
	if payload.DomainName != "" {
		item.DomainNameFQDN = types.StringValue(payload.DomainName)
	}
	if payload.Quota > 0 {
		item.Quota = types.Float64Value(payload.Quota)
	}
	if payload.QuotaUnit != "" {
		item.QuotaUnit = types.StringValue(payload.QuotaUnit)
	}
	if payload.StorageClass != "" {
		item.StorageClass = types.StringValue(payload.StorageClass)
	}
	if payload.Variant != "" {
		item.Variant = types.StringValue(payload.Variant)
	}
	if payload.EngagementID > 0 {
		item.EngagementID = types.Int64Value(payload.EngagementID)
	}
	if payload.EndpointID > 0 {
		item.EndpointID = types.Int64Value(payload.EndpointID)
	}
	return item
}
