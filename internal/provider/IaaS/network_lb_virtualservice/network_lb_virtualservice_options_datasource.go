package network_lb_virtualservice

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/client"
)

var (
	_ datasource.DataSource              = &NetworkLBVirtualServiceOptionsDataSource{}
	_ datasource.DataSourceWithConfigure = &NetworkLBVirtualServiceOptionsDataSource{}
)

func NewNetworkLBVirtualServiceOptionsDataSource() datasource.DataSource {
	return &NetworkLBVirtualServiceOptionsDataSource{}
}

type NetworkLBVirtualServiceOptionsDataSource struct {
	client *client.Client
}

type LBOptionItem struct {
	Label types.String `tfsdk:"label"`
	Value types.String `tfsdk:"value"`
}

type LBProtocolOptionItem struct {
	Label       types.String `tfsdk:"label"`
	Value       types.String `tfsdk:"value"`
	SSLRequired types.Bool   `tfsdk:"ssl_required"`
}

type LBPersistenceOptionItem struct {
	Label         types.String `tfsdk:"label"`
	Value         types.String `tfsdk:"value"`
	InputRequired types.Bool   `tfsdk:"input_required"`
	InputLabel    types.String `tfsdk:"input_label"`
}

type LBMonitorOptionItem struct {
	Name     types.String `tfsdk:"name"`
	FullPath types.String `tfsdk:"full_path"`
}

type LBZoneOptionItem struct {
	ID              types.Int64  `tfsdk:"id"`
	Name            types.String `tfsdk:"name"`
	EnvironmentID   types.Int64  `tfsdk:"environment_id"`
	EnvironmentName types.String `tfsdk:"environment_name"`
	DepartmentID    types.Int64  `tfsdk:"department_id"`
	DepartmentName  types.String `tfsdk:"department_name"`
}

type NetworkLBVirtualServiceOptionsDataSourceModel struct {
	LoadBalancerID   types.String              `tfsdk:"load_balancer_id"`
	MonitorType      types.String              `tfsdk:"monitor_type"`
	Protocols        []LBProtocolOptionItem    `tfsdk:"protocols"`
	Algorithms       []LBOptionItem            `tfsdk:"algorithms"`
	PersistenceTypes []LBPersistenceOptionItem `tfsdk:"persistence_types"`
	Monitors         []LBMonitorOptionItem     `tfsdk:"monitors"`
	Zones            []LBZoneOptionItem        `tfsdk:"zones"`
	Filters          []client.FilterModel      `tfsdk:"filter"`
}

func (d *NetworkLBVirtualServiceOptionsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network_lb_virtualservice_options"
}

func (d *NetworkLBVirtualServiceOptionsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists HAProxy virtual service configuration options for a load balancer (step 1 of the UI wizard). " +
			"Returns protocols, algorithms, monitors, persistence types, and LB-eligible zones. " +
			"Set `zone_id` on `vayucloud_network_lb` for zone selection; use `vayucloud_virtualmachine_list` for pool member VMs. " +
			"Use `vayucloud_network_lb_ssl_profiles` for HTTPS certificate names.",

		Attributes: map[string]schema.Attribute{
			"load_balancer_id": schema.StringAttribute{
				MarkdownDescription: "Load Balancer CI Master ID.",
				Required:            true,
			},
			"monitor_type": schema.StringAttribute{
				MarkdownDescription: "Protocol family used to filter monitors (`http` or `tcp`). Defaults to `http`.",
				Optional:            true,
			},
			"protocols": schema.ListNestedAttribute{
				MarkdownDescription: "Supported protocols. Use the `value` field in the virtual service resource.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"label":        schema.StringAttribute{Computed: true},
						"value":        schema.StringAttribute{Computed: true},
						"ssl_required": schema.BoolAttribute{Computed: true},
					},
				},
			},
			"algorithms": schema.ListNestedAttribute{
				MarkdownDescription: "Supported pool algorithms.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"label": schema.StringAttribute{Computed: true},
						"value": schema.StringAttribute{Computed: true},
					},
				},
			},
			"persistence_types": schema.ListNestedAttribute{
				MarkdownDescription: "Supported persistence types.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"label":          schema.StringAttribute{Computed: true},
						"value":          schema.StringAttribute{Computed: true},
						"input_required": schema.BoolAttribute{Computed: true},
						"input_label":    schema.StringAttribute{Computed: true},
					},
				},
			},
			"monitors": schema.ListNestedAttribute{
				MarkdownDescription: "Supported health monitors for the selected monitor_type.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name":      schema.StringAttribute{Computed: true},
						"full_path": schema.StringAttribute{Computed: true},
					},
				},
			},
			"zones": schema.ListNestedAttribute{
				MarkdownDescription: "Zones available for virtual service placement on this load balancer.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":               schema.Int64Attribute{Computed: true},
						"name":             schema.StringAttribute{Computed: true},
						"environment_id":   schema.Int64Attribute{Computed: true},
						"environment_name": schema.StringAttribute{Computed: true},
						"department_id":    schema.Int64Attribute{Computed: true},
						"department_name":  schema.StringAttribute{Computed: true},
					},
				},
			},
		},
		Blocks: map[string]schema.Block{
			"filter": client.FilterBlockSchema(),
		},
	}
}

func (d *NetworkLBVirtualServiceOptionsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *NetworkLBVirtualServiceOptionsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data NetworkLBVirtualServiceOptionsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	lbCiID, err := strconv.ParseInt(data.LoadBalancerID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Load Balancer ID", err.Error())
		return
	}

	monitorType := strings.TrimSpace(strings.ToLower(data.MonitorType.ValueString()))
	if monitorType == "" {
		monitorType = "http"
	}
	data.MonitorType = types.StringValue(monitorType)

	tflog.Info(ctx, "Reading virtual service options", map[string]any{
		"load_balancer_id": lbCiID,
		"monitor_type":     monitorType,
	})

	protocols, err := ListLBProtocols(d.client, ctx, lbCiID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to List Protocols", err.Error())
		return
	}
	data.Protocols = mapProtocolOptions(protocols)

	algorithms, err := ListLBAlgorithms(d.client, ctx, lbCiID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to List Algorithms", err.Error())
		return
	}
	data.Algorithms = mapLabelValueOptions(algorithms)

	persistence, err := ListLBPersistenceTypes(d.client, ctx, lbCiID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to List Persistence Types", err.Error())
		return
	}
	data.PersistenceTypes = mapPersistenceOptions(persistence)

	monitors, err := ListLBMonitors(d.client, ctx, lbCiID, monitorType)
	if err != nil {
		resp.Diagnostics.AddError("Failed to List Monitors", err.Error())
		return
	}
	data.Monitors = mapMonitorOptions(monitors)

	zones, err := ListLBZones(d.client, ctx, lbCiID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to List Zones", err.Error())
		return
	}
	data.Zones = mapZoneOptions(zones)
	if len(data.Filters) > 0 {
		filtered, err := client.ApplyFilters(data.Zones, data.Filters)
		if err != nil {
			resp.Diagnostics.AddError("Failed to Filter Zones", err.Error())
			return
		}
		data.Zones = filtered
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func mapLabelValueOptions(items []map[string]interface{}) []LBOptionItem {
	result := make([]LBOptionItem, 0, len(items))
	for _, item := range items {
		result = append(result, LBOptionItem{
			Label: types.StringValue(stringFromActionStateMap(item, "label")),
			Value: types.StringValue(stringFromActionStateMap(item, "value")),
		})
	}
	return result
}

func mapProtocolOptions(items []map[string]interface{}) []LBProtocolOptionItem {
	result := make([]LBProtocolOptionItem, 0, len(items))
	for _, item := range items {
		sslRequired := false
		if raw, ok := item["sslRequired"]; ok {
			switch v := raw.(type) {
			case bool:
				sslRequired = v
			case string:
				sslRequired = strings.EqualFold(v, "true")
			}
		}
		result = append(result, LBProtocolOptionItem{
			Label:       types.StringValue(stringFromActionStateMap(item, "label")),
			Value:       types.StringValue(stringFromActionStateMap(item, "value")),
			SSLRequired: types.BoolValue(sslRequired),
		})
	}
	return result
}

func mapPersistenceOptions(items []map[string]interface{}) []LBPersistenceOptionItem {
	result := make([]LBPersistenceOptionItem, 0, len(items))
	for _, item := range items {
		inputRequired := false
		if raw, ok := item["inputRequired"]; ok {
			switch v := raw.(type) {
			case bool:
				inputRequired = v
			case string:
				inputRequired = strings.EqualFold(v, "true")
			}
		}
		result = append(result, LBPersistenceOptionItem{
			Label:         types.StringValue(stringFromActionStateMap(item, "label")),
			Value:         types.StringValue(stringFromActionStateMap(item, "value")),
			InputRequired: types.BoolValue(inputRequired),
			InputLabel:    types.StringValue(stringFromActionStateMap(item, "inputLabel")),
		})
	}
	return result
}

func mapMonitorOptions(items []map[string]interface{}) []LBMonitorOptionItem {
	result := make([]LBMonitorOptionItem, 0, len(items))
	for _, item := range items {
		result = append(result, LBMonitorOptionItem{
			Name:     types.StringValue(stringFromActionStateMap(item, "name")),
			FullPath: types.StringValue(stringFromActionStateMap(item, "fullPath", "full_path", "id")),
		})
	}
	return result
}

func mapZoneOptions(items []map[string]interface{}) []LBZoneOptionItem {
	result := make([]LBZoneOptionItem, 0, len(items))
	for _, zone := range items {
		item := LBZoneOptionItem{}
		if id, ok := int64FromActionStateMap(zone, "id"); ok {
			item.ID = types.Int64Value(id)
		}
		item.Name = types.StringValue(stringFromActionStateMap(zone, "name"))
		if envID, ok := int64FromActionStateMap(zone, "environmentId", "environment_id"); ok {
			item.EnvironmentID = types.Int64Value(envID)
		}
		item.EnvironmentName = types.StringValue(stringFromActionStateMap(zone, "environmentName", "environment_name"))
		if deptID, ok := int64FromActionStateMap(zone, "departmentId", "department_id"); ok {
			item.DepartmentID = types.Int64Value(deptID)
		}
		item.DepartmentName = types.StringValue(stringFromActionStateMap(zone, "departmentName", "department_name"))
		result = append(result, item)
	}
	return result
}
