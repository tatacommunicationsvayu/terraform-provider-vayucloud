package network_lb_ssl_profile

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/client"
)

var (
	_ datasource.DataSource              = &NetworkLBSSLProfilesDataSource{}
	_ datasource.DataSourceWithConfigure = &NetworkLBSSLProfilesDataSource{}
)

func NewNetworkLBSSLProfilesDataSource() datasource.DataSource {
	return &NetworkLBSSLProfilesDataSource{}
}

type NetworkLBSSLProfilesDataSource struct {
	client *client.Client
}

type LBSSLProfileOptionItem struct {
	Name     types.String `tfsdk:"name"`
	FullPath types.String `tfsdk:"full_path"`
}

type NetworkLBSSLProfilesDataSourceModel struct {
	LoadBalancerID types.String           `tfsdk:"load_balancer_id"`
	SSLProfiles    []LBSSLProfileOptionItem `tfsdk:"ssl_profiles"`
}

func (d *NetworkLBSSLProfilesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network_lb_ssl_profiles"
}

func (d *NetworkLBSSLProfilesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists uploaded client SSL certificate profiles on a load balancer. " +
			"Use `name` as `certificate_name` for HTTPS virtual services.",

		Attributes: map[string]schema.Attribute{
			"load_balancer_id": schema.StringAttribute{
				MarkdownDescription: "Load Balancer CI Master ID.",
				Required:            true,
			},
			"ssl_profiles": schema.ListNestedAttribute{
				MarkdownDescription: "SSL certificate profiles on the load balancer.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name":      schema.StringAttribute{Computed: true},
						"full_path": schema.StringAttribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *NetworkLBSSLProfilesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected DataSource Configure Type", fmt.Sprintf("Expected *client.Client, got: %T", req.ProviderData))
		return
	}
	d.client = c
}

func (d *NetworkLBSSLProfilesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data NetworkLBSSLProfilesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	lbCiID, err := strconv.ParseInt(data.LoadBalancerID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Load Balancer ID", err.Error())
		return
	}

	tflog.Info(ctx, "Reading SSL profiles", map[string]any{
		"load_balancer_id": lbCiID,
	})

	sslProfiles, err := ListClientSSLProfiles(d.client, ctx, lbCiID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to List SSL Profiles", err.Error())
		return
	}
	data.SSLProfiles = mapSSLProfileOptions(sslProfiles)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func mapSSLProfileOptions(items []map[string]interface{}) []LBSSLProfileOptionItem {
	result := make([]LBSSLProfileOptionItem, 0, len(items))
	for _, item := range items {
		name := stringFromActionStateMap(item, "name", "storage_name")
		fullPath := stringFromActionStateMap(item, "fullPath", "full_path", "file")
		result = append(result, LBSSLProfileOptionItem{
			Name:     types.StringValue(name),
			FullPath: types.StringValue(fullPath),
		})
	}
	return result
}
