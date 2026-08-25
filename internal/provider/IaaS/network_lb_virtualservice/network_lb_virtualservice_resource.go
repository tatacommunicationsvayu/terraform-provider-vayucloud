package network_lb_virtualservice

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/client"
)

var (
	_ resource.Resource                = &NetworkLBVirtualServiceResource{}
	_ resource.ResourceWithImportState = &NetworkLBVirtualServiceResource{}
	_ resource.ResourceWithModifyPlan  = &NetworkLBVirtualServiceResource{}
)

func NewNetworkLBVirtualServiceResource() resource.Resource {
	return &NetworkLBVirtualServiceResource{}
}

type NetworkLBVirtualServiceResource struct {
	client *client.Client
}

type VirtualServicePoolMemberModel struct {
	Name      types.String `tfsdk:"name"`
	IPAddress types.String `tfsdk:"ip_address"`
	Port      types.Int64  `tfsdk:"port"`
}

type NetworkLBVirtualServiceResourceModel struct {
	ID               types.String                  `tfsdk:"id"`
	LoadBalancerID   types.String                  `tfsdk:"load_balancer_id"`
	Name             types.String                  `tfsdk:"name"`
	ZoneID           types.Int64                   `tfsdk:"zone_id"`
	Port             types.String                  `tfsdk:"port"`
	Protocol         types.String                  `tfsdk:"protocol"`
	PoolAlgorithm    types.String                  `tfsdk:"pool_algorithm"`
	Monitor          types.List                    `tfsdk:"monitor"`
	VipIP            types.String                  `tfsdk:"vip_ip"`
	PersistenceType  types.String                  `tfsdk:"persistence_type"`
	PersistenceValue types.String                `tfsdk:"persistence_value"`
	CertificateName  types.String                  `tfsdk:"certificate_name"`
	PoolMembers      []VirtualServicePoolMemberModel `tfsdk:"pool_member"`
	CIStatus         types.String                  `tfsdk:"ci_status"`
	AuditID          types.String                  `tfsdk:"audit_id"`
	Status           types.String                  `tfsdk:"status"`
	LastUpdated      types.String                  `tfsdk:"last_updated"`
}

func (r *NetworkLBVirtualServiceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network_lb_virtualservice"
}

func (r *NetworkLBVirtualServiceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an HAProxy Virtual Service on a VayuCloud Load Balancer. " +
			"Use `vayucloud_network_lb_virtualservice_options` to discover valid dropdown values (set zone_id to also load VIPs and VMs).",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Composite identity `{load_balancer_id}/{name}` for HAProxy virtual services.",
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
			"name": schema.StringAttribute{
				MarkdownDescription: "Virtual Service name. Must be unique per load balancer and cannot be changed after creation.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"zone_id": schema.Int64Attribute{
				MarkdownDescription: "Zone ID for the virtual service. Use `vayucloud_network_lb.zone_id` from the parent load balancer.",
				Required:            true,
			},
			"port": schema.StringAttribute{
				MarkdownDescription: "Listener port (e.g. `80`, `443`, or comma-separated `80,443`).",
				Required:            true,
			},
			"protocol": schema.StringAttribute{
				MarkdownDescription: "Listener protocol from `vayucloud_network_lb_virtualservice_options.protocols` (`http`, `https`, or `tcp`). " +
					"For `https`, set `certificate_name` (`.pem` from SSL profile list) and an SSL port such as `443`.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("HTTP", "HTTPS", "TCP", "http", "https", "tcp"),
				},
			},
			"pool_algorithm": schema.StringAttribute{
				MarkdownDescription: "Pool algorithm API value (e.g. `roundrobin`, `leastconn`, `source`). Defaults to `roundrobin`.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("roundrobin"),
			},
			"monitor": schema.ListAttribute{
				MarkdownDescription: "Health monitor names (e.g. `httpchk` for HTTP/HTTPS, `tcp-check` for TCP). Defaults based on protocol.",
				ElementType:         types.StringType,
				Optional:            true,
				Computed:            true,
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
				},
			},
			"vip_ip": schema.StringAttribute{
				MarkdownDescription: "Virtual IP address. Optional; the platform auto-assigns from the zone VIP subnet when omitted. After create, the assigned value is kept from state on later plans when omitted in config (`UseStateForUnknown`), so dependents such as `vayucloud_network_public_ip.private_ip` stay stable without pinning.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"persistence_type": schema.StringAttribute{
				MarkdownDescription: "Persistence API value (e.g. `PERSISTENCE_TYPE_HTTP_COOKIE`). Optional.",
				Optional:            true,
			},
			"persistence_value": schema.StringAttribute{
				MarkdownDescription: "Persistence input value such as cookie name when required by the persistence type.",
				Optional:            true,
			},
			"certificate_name": schema.StringAttribute{
				MarkdownDescription: "Client SSL certificate storage name (e.g. `tf-test-cert.pem` from SSL profile id). Required when protocol is HTTPS or when using an SSL listener port.",
				Optional:            true,
			},
			"ci_status": schema.StringAttribute{
				MarkdownDescription: "CI status (ACTIVE, INACTIVE, etc.).",
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
		Blocks: map[string]schema.Block{
			"pool_member": schema.ListNestedBlock{
				MarkdownDescription: "Backend pool members. Discover VM names and IPs with `vayucloud_virtualmachine_list` using `vayucloud_network_lb.zone_id`.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							MarkdownDescription: "Pool member name (typically the VM name).",
							Required:            true,
						},
						"ip_address": schema.StringAttribute{
							MarkdownDescription: "Pool member IP address.",
							Required:            true,
						},
						"port": schema.Int64Attribute{
							MarkdownDescription: "Backend port on the pool member.",
							Required:            true,
						},
					},
				},
			},
		},
	}
}

func (r *NetworkLBVirtualServiceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T", req.ProviderData),
		)
		return
	}
	r.client = c
}

func (r *NetworkLBVirtualServiceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan NetworkLBVirtualServiceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	lbCiID, vsReq, diags := r.buildAPIRequest(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	auditResp, err := CreateVirtualService(r.client, ctx, lbCiID, vsReq)
	if err != nil {
		resp.Diagnostics.AddError("Failed to Create Virtual Service", err.Error())
		return
	}

	pollDiags := r.pollAuditLog(ctx, auditResp.Data.Audit.AuditID, "create", "virtualservice", lbCiID, vsReq.VirtualServerName, int64(vsReq.ZoneID))
	resp.Diagnostics.Append(pollDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state := plan
	state.ID = types.StringValue(compositeVSID(lbCiID, vsReq.VirtualServerName))
	state.AuditID = types.StringValue(auditResp.Data.Audit.AuditID)
	state.Status = types.StringValue(auditResp.Status)
	state.LastUpdated = types.StringValue(time.Now().Format(time.RFC3339))
	if monitorList, monitorDiags := types.ListValueFrom(ctx, types.StringType, vsReq.Monitors); !monitorDiags.HasError() {
		state.Monitor = monitorList
	}
	if state.PoolAlgorithm.IsNull() || state.PoolAlgorithm.IsUnknown() {
		state.PoolAlgorithm = types.StringValue(vsReq.PoolAlgorithm)
	}

	resp.Diagnostics.Append(r.refreshState(ctx, &state)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *NetworkLBVirtualServiceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state NetworkLBVirtualServiceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags := r.refreshState(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *NetworkLBVirtualServiceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state NetworkLBVirtualServiceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	lbCiID, vsReq, diags := r.buildAPIRequest(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Keep VIP stable when plan omits vip_ip (computed unknown) but state has the assigned address.
	if vsReq != nil && strings.TrimSpace(vsReq.VipIP) == "" &&
		!state.VipIP.IsNull() && !state.VipIP.IsUnknown() {
		vsReq.VipIP = strings.TrimSpace(state.VipIP.ValueString())
	}
	if vsReq == nil {
		resp.Diagnostics.AddError("Failed to Update Virtual Service", "No API request built")
		return
	}

	auditResp, err := EditVirtualService(r.client, ctx, lbCiID, vsReq)
	if err != nil {
		resp.Diagnostics.AddError("Failed to Update Virtual Service", err.Error())
		return
	}

	pollDiags := r.pollAuditLog(ctx, auditResp.Data.Audit.AuditID, "update", "virtualservice", lbCiID, vsReq.VirtualServerName, int64(vsReq.ZoneID))
	resp.Diagnostics.Append(pollDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updated := plan
	if state.ID.IsNull() || state.ID.IsUnknown() || strings.TrimSpace(state.ID.ValueString()) == "" {
		updated.ID = types.StringValue(compositeVSID(lbCiID, vsReq.VirtualServerName))
	} else {
		updated.ID = state.ID
	}
	updated.AuditID = types.StringValue(auditResp.Data.Audit.AuditID)
	updated.Status = types.StringValue(auditResp.Status)
	updated.LastUpdated = types.StringValue(time.Now().Format(time.RFC3339))

	resp.Diagnostics.Append(r.refreshState(ctx, &updated)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, updated)...)
}

func (r *NetworkLBVirtualServiceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state NetworkLBVirtualServiceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	lbCiID, err := strconv.ParseInt(state.LoadBalancerID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid LB ID", fmt.Sprintf("Could not parse LB ID: %s", err.Error()))
		return
	}

	auditResp, err := DeleteVirtualService(r.client, ctx, lbCiID, state.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to Delete Virtual Service", err.Error())
		return
	}

	diags := r.pollAuditLog(ctx, auditResp.Data.Audit.AuditID, "delete", "virtualservice", lbCiID, state.Name.ValueString(), state.ZoneID.ValueInt64())
	resp.Diagnostics.Append(diags...)
}

func (r *NetworkLBVirtualServiceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var data NetworkLBVirtualServiceResourceModel

	if isCompositeVSID(req.ID) {
		parts := strings.SplitN(req.ID, "/", 2)
		data.ID = types.StringValue(req.ID)
		data.LoadBalancerID = types.StringValue(parts[0])
		data.Name = types.StringValue(parts[1])
	} else {
		data.ID = types.StringValue(req.ID)
	}

	diags := r.refreshState(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *NetworkLBVirtualServiceResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if r.client == nil || req.Plan.Raw.IsNull() {
		return
	}

	var plan NetworkLBVirtualServiceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state NetworkLBVirtualServiceResourceModel
	hasState := !req.State.Raw.IsNull()
	if hasState {
		resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	resp.Diagnostics.Append(runVirtualServicePlanValidations(ctx, r.client, &plan, &state, hasState)...)

	// When vip_ip is omitted in config (null/unknown) but already assigned in state, keep it in the
	// plan so downstream resources (e.g. public IP private_ip) do not see an unknown VIP.
	if hasState && (plan.VipIP.IsNull() || plan.VipIP.IsUnknown()) &&
		!state.VipIP.IsNull() && !state.VipIP.IsUnknown() &&
		strings.TrimSpace(state.VipIP.ValueString()) != "" {
		plan.VipIP = state.VipIP
		resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
	}
}

func (r *NetworkLBVirtualServiceResource) buildAPIRequest(ctx context.Context, data *NetworkLBVirtualServiceResourceModel) (int64, *VirtualServiceRequest, diag.Diagnostics) {
	var diags diag.Diagnostics

	lbCiID, err := strconv.ParseInt(data.LoadBalancerID.ValueString(), 10, 64)
	if err != nil {
		diags.AddError("Invalid LB ID", fmt.Sprintf("Could not parse LB ID: %s", err.Error()))
		return 0, nil, diags
	}

	if len(data.PoolMembers) == 0 {
		diags.AddError("Missing Pool Members", "At least one pool_member block is required.")
		return 0, nil, diags
	}

	protocol := normalizeProtocol(data.Protocol.ValueString())
	port := data.Port.ValueString()
	certName := certificateNameFromModel(data)
	diags.Append(validateCertificateConfig(protocol, port, certName)...)
	if diags.HasError() {
		return 0, nil, diags
	}

	monitors, monitorDiags := resolveMonitors(data.Monitor, protocol)
	diags.Append(monitorDiags...)
	if diags.HasError() {
		return 0, nil, diags
	}

	members := make([]VirtualServicePoolMember, 0, len(data.PoolMembers))
	for _, member := range data.PoolMembers {
		members = append(members, VirtualServicePoolMember{
			Name:      member.Name.ValueString(),
			IPAddress: member.IPAddress.ValueString(),
			Port:      int(member.Port.ValueInt64()),
		})
	}

	req := &VirtualServiceRequest{
		VirtualServerName: data.Name.ValueString(),
		VirtualServerPort: data.Port.ValueString(),
		Protocol:          apiProtocolForCreate(protocol),
		ZoneID:            int(data.ZoneID.ValueInt64()),
		PoolAlgorithm:     normalizePoolAlgorithm(data.PoolAlgorithm.ValueString()),
		Monitors:          monitors,
		PoolMembers:       members,
	}

	if !data.VipIP.IsNull() && !data.VipIP.IsUnknown() {
		req.VipIP = strings.TrimSpace(data.VipIP.ValueString())
	}
	if !data.PersistenceType.IsNull() && !data.PersistenceType.IsUnknown() {
		req.PersistenceType = strings.TrimSpace(data.PersistenceType.ValueString())
	}
	if !data.PersistenceValue.IsNull() && !data.PersistenceValue.IsUnknown() {
		req.PersistenceValue = strings.TrimSpace(data.PersistenceValue.ValueString())
	}
	if certName != "" {
		resolved, certDiags := resolveCertificateForAPI(r.client, ctx, lbCiID, certName)
		diags.Append(certDiags...)
		if diags.HasError() {
			return 0, nil, diags
		}
		req.CertificateName = resolved
	}

	tflog.Debug(ctx, "Built virtual service API request", map[string]any{
		"name":     req.VirtualServerName,
		"zone_id":  req.ZoneID,
		"protocol": req.Protocol,
	})

	return lbCiID, req, diags
}

func resolveMonitors(monitorList types.List, protocol string) ([]string, diag.Diagnostics) {
	var diags diag.Diagnostics
	if !monitorList.IsNull() && !monitorList.IsUnknown() {
		var values []string
		diags.Append(monitorList.ElementsAs(context.Background(), &values, false)...)
		if diags.HasError() {
			return nil, diags
		}
		result := make([]string, 0, len(values))
		for _, value := range values {
			value = strings.TrimSpace(value)
			if value != "" {
				result = append(result, value)
			}
		}
		if len(result) > 0 {
			return result, diags
		}
	}
	return []string{defaultMonitorForProtocol(protocol)}, diags
}

func (r *NetworkLBVirtualServiceResource) pollAuditLog(ctx context.Context, auditID, action, module string, lbCiID int64, vsName string, zoneID int64) diag.Diagnostics {
	var diags diag.Diagnostics
	requestBody := map[string]any{
		"virtualServerName": strings.TrimSpace(vsName),
	}
	if zoneID > 0 {
		requestBody["zoneId"] = zoneID
	}
	_, err := r.client.WaitForAuditCompletion(ctx, auditID, action, module, requestBody)
	if err != nil {
		diags.AddError(
			"Audit Polling Timeout",
			fmt.Sprintf("Virtual Service operation did not complete within timeout: %s", err.Error()),
		)
	}
	return diags
}

func (r *NetworkLBVirtualServiceResource) refreshState(ctx context.Context, data *NetworkLBVirtualServiceResourceModel) diag.Diagnostics {
	id := strings.TrimSpace(data.ID.ValueString())
	var diags diag.Diagnostics
	if isCompositeVSID(id) || data.ID.IsUnknown() || id == "" {
		diags = r.refreshStateFromLBList(ctx, data)
	} else if _, err := strconv.ParseInt(id, 10, 64); err == nil {
		diags = r.refreshStateFromActionState(ctx, data)
	} else {
		diags = r.refreshStateFromLBList(ctx, data)
	}
	normalizeUnknownComputedValues(data)
	return diags
}

// HAProxy virtual services have no VS CI; list refresh does not populate ci_status.
func normalizeUnknownComputedValues(data *NetworkLBVirtualServiceResourceModel) {
	if data.CIStatus.IsUnknown() {
		data.CIStatus = types.StringNull()
	}
	if data.CertificateName.IsUnknown() {
		data.CertificateName = types.StringNull()
	}
	if data.PersistenceType.IsUnknown() {
		data.PersistenceType = types.StringNull()
	}
	if data.PersistenceValue.IsUnknown() {
		data.PersistenceValue = types.StringNull()
	}
	if data.VipIP.IsUnknown() {
		data.VipIP = types.StringNull()
	}
}

func (r *NetworkLBVirtualServiceResource) refreshStateFromLBList(ctx context.Context, data *NetworkLBVirtualServiceResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	lbIDStr := strings.TrimSpace(data.LoadBalancerID.ValueString())
	vsName := strings.TrimSpace(data.Name.ValueString())
	if lbIDStr == "" || vsName == "" {
		return diags
	}

	lbCiID, err := strconv.ParseInt(lbIDStr, 10, 64)
	if err != nil {
		diags.AddWarning("Invalid LB ID", fmt.Sprintf("Could not parse load balancer ID for list refresh: %s", err.Error()))
		return diags
	}

	zoneID := data.ZoneID.ValueInt64()
	list, err := ListVirtualServicesOnLB(r.client, ctx, lbCiID, zoneID)
	if err != nil {
		diags.AddWarning("Failed to Refresh Virtual Service State", fmt.Sprintf("List virtual services API call failed: %s", err.Error()))
		return diags
	}

	vsDetails, found := findVSInList(list, vsName)
	if !found {
		diags.AddError("Virtual Service Not Found", fmt.Sprintf("Virtual service %q was not found on load balancer %d", vsName, lbCiID))
		return diags
	}

	data.ID = types.StringValue(compositeVSID(lbCiID, vsName))
	data.LoadBalancerID = types.StringValue(lbIDStr)

	if vsPort := stringFromActionStateMap(vsDetails, "virtualServerport", "vs_port", "port"); vsPort != "" {
		data.Port = types.StringValue(vsPort)
	}
	if protocol := stringFromActionStateMap(vsDetails, "protocol", "vs_protocol"); protocol != "" {
		certName := stringFromActionStateMap(vsDetails, "certificateName", "certificate_name")
		port := data.Port.ValueString()
		if port == "" {
			port = stringFromActionStateMap(vsDetails, "virtualServerport", "vs_port", "port")
		}
		data.Protocol = types.StringValue(displayProtocolFromAPI(protocol, port, certName))
	}
	if poolAlg := stringFromActionStateMap(vsDetails, "poolAlgorithm", "pool_algorithm"); poolAlg != "" {
		data.PoolAlgorithm = types.StringValue(normalizePoolAlgorithm(poolAlg))
	}
	if vipIP := stringFromActionStateMap(vsDetails, "vipIp", "vip_ip"); vipIP != "" {
		data.VipIP = types.StringValue(vipIP)
	}
	if persistence := stringFromActionStateMap(vsDetails, "persistenceType", "persistence_type"); persistence != "" {
		data.PersistenceType = types.StringValue(persistence)
	}
	if persistenceValue := stringFromActionStateMap(vsDetails, "persistenceValue", "persistence_value"); persistenceValue != "" {
		data.PersistenceValue = types.StringValue(persistenceValue)
	}
	if certName := stringFromActionStateMap(vsDetails, "certificateName", "certificate_name"); certName != "" {
		data.CertificateName = types.StringValue(normalizeCertificateNameForState(certName))
	}
	if zoneIDFromAPI, ok := inferZoneIDFromVSDetails(vsDetails); ok {
		data.ZoneID = types.Int64Value(zoneIDFromAPI)
	}

	if monitors, ok := vsDetails["monitor"].([]interface{}); ok && len(monitors) > 0 {
		values := make([]string, 0, len(monitors))
		for _, item := range monitors {
			if s, ok := item.(string); ok && s != "" {
				values = append(values, s)
			}
		}
		if len(values) > 0 {
			monitorList, monitorDiags := types.ListValueFrom(ctx, types.StringType, values)
			diags.Append(monitorDiags...)
			if !monitorDiags.HasError() {
				data.Monitor = monitorList
			}
		}
	} else if data.Monitor.IsNull() || data.Monitor.IsUnknown() {
		defaults, _ := types.ListValueFrom(ctx, types.StringType, []string{defaultMonitorForProtocol(data.Protocol.ValueString())})
		data.Monitor = defaults
	}

	if rawMembers, ok := vsDetails["poolMembers"].([]interface{}); ok {
		members := make([]VirtualServicePoolMemberModel, 0, len(rawMembers))
		for _, item := range rawMembers {
			memberMap, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			port, _ := int64FromActionStateMap(memberMap, "port")
			members = append(members, VirtualServicePoolMemberModel{
				Name:      types.StringValue(stringFromActionStateMap(memberMap, "name")),
				IPAddress: types.StringValue(stringFromActionStateMap(memberMap, "ipAddress", "ip_address")),
				Port:      types.Int64Value(port),
			})
		}
		if len(members) > 0 {
			data.PoolMembers = members
		}
	}

	data.LastUpdated = types.StringValue(time.Now().Format(time.RFC3339))
	return diags
}

func (r *NetworkLBVirtualServiceResource) refreshStateFromActionState(ctx context.Context, data *NetworkLBVirtualServiceResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	if data.ID.IsNull() || data.ID.IsUnknown() || strings.TrimSpace(data.ID.ValueString()) == "" {
		return diags
	}

	vsCiID, err := strconv.ParseInt(strings.TrimSpace(data.ID.ValueString()), 10, 64)
	if err != nil {
		diags.AddWarning("Invalid VS ID", fmt.Sprintf("Could not parse VS ID for action-state call: %s", err.Error()))
		return diags
	}

	details, err := GetVirtualServiceDetails(r.client, ctx, vsCiID)
	if err != nil {
		diags.AddWarning("Failed to Refresh Virtual Service State", fmt.Sprintf("Action-state API call failed: %s", err.Error()))
		return diags
	}

	if vsName := stringFromActionStateMap(details, "vs_name", "name"); vsName != "" {
		data.Name = types.StringValue(vsName)
	}
	if vsPort := stringFromActionStateMap(details, "vs_port"); vsPort != "" {
		data.Port = types.StringValue(vsPort)
	}
	if protocol := stringFromActionStateMap(details, "vs_protocol", "protocol"); protocol != "" {
		certName := data.CertificateName.ValueString()
		if certName == "" {
			certName = stringFromActionStateMap(details, "certificate_name", "certificateName")
		}
		port := data.Port.ValueString()
		if port == "" {
			port = stringFromActionStateMap(details, "vs_port", "port")
		}
		data.Protocol = types.StringValue(displayProtocolFromAPI(protocol, port, certName))
	}
	if poolAlg := stringFromActionStateMap(details, "pool_algorithm"); poolAlg != "" {
		data.PoolAlgorithm = types.StringValue(normalizePoolAlgorithm(poolAlg))
	}
	if persistence := stringFromActionStateMap(details, "persistence_type"); persistence != "" {
		data.PersistenceType = types.StringValue(persistence)
	}
	if ciStatus := stringFromActionStateMap(details, "ci_status"); ciStatus != "" {
		data.CIStatus = types.StringValue(ciStatus)
	}
	if parentLBID, ok := int64FromActionStateMap(details, "parent_lb_id"); ok && parentLBID > 0 {
		if data.LoadBalancerID.IsNull() || data.LoadBalancerID.IsUnknown() || data.LoadBalancerID.ValueString() == "" {
			data.LoadBalancerID = types.StringValue(strconv.FormatInt(parentLBID, 10))
		}
	}

	if monitors, ok := details["monitor"].([]interface{}); ok && len(monitors) > 0 {
		values := make([]string, 0, len(monitors))
		for _, item := range monitors {
			if s, ok := item.(string); ok && s != "" {
				values = append(values, s)
			}
		}
		if len(values) > 0 {
			monitorList, monitorDiags := types.ListValueFrom(ctx, types.StringType, values)
			diags.Append(monitorDiags...)
			if !monitorDiags.HasError() {
				data.Monitor = monitorList
			}
		}
	} else if data.Monitor.IsNull() || data.Monitor.IsUnknown() {
		defaults, _ := types.ListValueFrom(ctx, types.StringType, []string{defaultMonitorForProtocol(data.Protocol.ValueString())})
		data.Monitor = defaults
	}

	data.LastUpdated = types.StringValue(time.Now().Format(time.RFC3339))
	return diags
}

