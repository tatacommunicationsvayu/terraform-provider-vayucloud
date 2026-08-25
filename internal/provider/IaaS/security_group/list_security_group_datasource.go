// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

package security_group

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/client"
)

var _ datasource.DataSource = &ListSecurityGroupDataSource{}

// NewListSecurityGroupDataSource lists security groups under a firewall.
func NewListSecurityGroupDataSource() datasource.DataSource {
	return &ListSecurityGroupDataSource{}
}

// ListSecurityGroupDataSource implements vayucloud_list_security_group.
type ListSecurityGroupDataSource struct {
	client *client.Client
}

// ListSecurityGroupDataSourceModel is the Terraform model.
type ListSecurityGroupDataSourceModel struct {
	ID             types.String                 `tfsdk:"id"`
	FirewallID     types.Int64                  `tfsdk:"firewall_id"`
	Filters        []client.FilterModel         `tfsdk:"filter"`
	SecurityGroups []ListSecurityGroupItemModel `tfsdk:"security_groups"`
}

// ListSecurityGroupItemModel is one security group from the list API.
type ListSecurityGroupItemModel struct {
	ID             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	Description    types.String `tfsdk:"description"`
	TenantID       types.String `tfsdk:"tenant_id"`
	ProjectID      types.String `tfsdk:"project_id"`
	Shared         types.Bool   `tfsdk:"shared"`
	Stateful       types.Bool   `tfsdk:"stateful"`
	RevisionNumber types.Int64  `tfsdk:"revision_number"`
	CreatedAt      types.String `tfsdk:"created_at"`
	UpdatedAt      types.String `tfsdk:"updated_at"`
}

func (d *ListSecurityGroupDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_list_security_group"
}

func (d *ListSecurityGroupDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists security groups attached to a firewall.",
		MarkdownDescription: "Lists security groups for a firewall via " +
			"`GET .../security-group/{firewall_id}`.\n\n" +
			"Optionally, use `filter` blocks to narrow results client-side " +
			"(e.g. `name`, `id`, `description`).",
		Blocks: map[string]schema.Block{
			"filter": client.FilterBlockSchema(),
		},
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Placeholder identifier (`firewall_id`).",
				Computed:    true,
			},
			"firewall_id": schema.Int64Attribute{
				Description: "Firewall ID whose security groups should be listed.",
				Required:    true,
			},
			"security_groups": schema.ListNestedAttribute{
				Description: "Security groups returned by the API.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "Security group UUID.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "Security group name.",
							Computed:    true,
						},
						"description": schema.StringAttribute{
							Description: "Security group description.",
							Computed:    true,
						},
						"tenant_id": schema.StringAttribute{
							Description: "Tenant ID.",
							Computed:    true,
						},
						"project_id": schema.StringAttribute{
							Description: "Project ID.",
							Computed:    true,
						},
						"shared": schema.BoolAttribute{
							Description: "Whether the security group is shared.",
							Computed:    true,
						},
						"stateful": schema.BoolAttribute{
							Description: "Whether the security group is stateful.",
							Computed:    true,
						},
						"revision_number": schema.Int64Attribute{
							Description: "Revision number.",
							Computed:    true,
						},
						"created_at": schema.StringAttribute{
							Description: "Creation timestamp (RFC3339).",
							Computed:    true,
						},
						"updated_at": schema.StringAttribute{
							Description: "Last update timestamp (RFC3339).",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *ListSecurityGroupDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T.", req.ProviderData),
		)
		return
	}
	d.client = c
}

func (d *ListSecurityGroupDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ListSecurityGroupDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	firewallID := data.FirewallID.ValueInt64()
	tflog.Debug(ctx, "Reading list_security_group data source", map[string]any{
		"firewall_id": firewallID,
	})

	items, err := ListSecurityGroups(d.client, ctx, firewallID)
	if err != nil {
		resp.Diagnostics.AddError("Error Listing Security Groups", err.Error())
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("%d", firewallID))
	securityGroups := make([]ListSecurityGroupItemModel, 0, len(items))
	for _, item := range items {
		securityGroups = append(securityGroups, ListSecurityGroupItemModel{
			ID:             types.StringValue(item.ID),
			Name:           types.StringValue(item.Name),
			Description:    types.StringValue(item.Description),
			TenantID:       types.StringValue(item.TenantID),
			ProjectID:      types.StringValue(item.ProjectID),
			Shared:         types.BoolValue(item.Shared),
			Stateful:       types.BoolValue(item.Stateful),
			RevisionNumber: types.Int64Value(item.RevisionNumber),
			CreatedAt:      types.StringValue(item.CreatedAt),
			UpdatedAt:      types.StringValue(item.UpdatedAt),
		})
	}

	if len(data.Filters) > 0 {
		tflog.Debug(ctx, "Applying filters to security groups", map[string]any{
			"filter_count":     len(data.Filters),
			"pre_filter_count": len(securityGroups),
		})
		filtered, filterErr := client.ApplyFilters(securityGroups, data.Filters)
		if filterErr != nil {
			resp.Diagnostics.AddError("Error Filtering Security Groups", filterErr.Error())
			return
		}
		securityGroups = filtered
		tflog.Debug(ctx, "Filters applied to security groups", map[string]any{
			"post_filter_count": len(securityGroups),
		})
	}

	data.SecurityGroups = securityGroups

	tflog.Info(ctx, "Read list_security_group data source", map[string]any{
		"firewall_id": firewallID,
		"count":       len(securityGroups),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
