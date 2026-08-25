package network_lb_virtualservice

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/client"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/network_lb"
)

// certificateRequiredForListener reports whether certificate_name must be set,
// mirroring HaProxyConfigServiceImpl.validateSSLCertificate.
func certificateRequiredForListener(protocol, port string) bool {
	protocol = normalizeProtocol(protocol)
	if protocol != "http" && protocol != "https" {
		return false
	}
	return protocol == "https" || listenerUsesSSLPort(port)
}

func certificateNameFromModel(data *NetworkLBVirtualServiceResourceModel) string {
	if data.CertificateName.IsNull() || data.CertificateName.IsUnknown() {
		return ""
	}
	return strings.TrimSpace(data.CertificateName.ValueString())
}

// normalizeCertificateStorageName returns the HAProxy storage name (e.g. test.pem).
func normalizeCertificateStorageName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	if strings.HasSuffix(strings.ToLower(name), ".pem") {
		return name
	}
	return name + ".pem"
}

func certificateBaseName(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if i := strings.LastIndex(path, "/"); i >= 0 {
		return path[i+1:]
	}
	return path
}

func certificateNamesMatch(profileName, requested string) bool {
	profileName = strings.TrimSpace(profileName)
	requested = strings.TrimSpace(requested)
	if profileName == "" || requested == "" {
		return false
	}
	if strings.EqualFold(profileName, requested) {
		return true
	}
	return strings.EqualFold(normalizeCertificateStorageName(profileName), normalizeCertificateStorageName(requested))
}

func certificateMatchesProfile(storageName, fullPath, requested string) bool {
	if certificateNamesMatch(storageName, requested) {
		return true
	}
	fullPath = strings.TrimSpace(fullPath)
	requested = strings.TrimSpace(requested)
	if fullPath == "" || requested == "" {
		return false
	}
	if strings.EqualFold(fullPath, requested) {
		return true
	}
	return certificateNamesMatch(certificateBaseName(fullPath), requested)
}

func findCertificateForAPI(profiles []map[string]interface{}, certName string) (string, bool) {
	for _, profile := range profiles {
		storageName := stringFromActionStateMap(profile, "name", "storage_name")
		fullPath := stringFromActionStateMap(profile, "fullPath", "full_path", "file")
		if !certificateMatchesProfile(storageName, fullPath, certName) {
			continue
		}
		if fullPath != "" {
			return fullPath, true
		}
		return normalizeCertificateStorageName(storageName), true
	}
	return "", false
}

func tryResolveCertificateForAPI(c *client.Client, ctx context.Context, lbCiID int64, certName string) (string, bool) {
	certName = strings.TrimSpace(certName)
	if certName == "" {
		return "", false
	}
	profiles, err := ListClientSSLProfiles(c, ctx, lbCiID)
	if err != nil {
		return "", false
	}
	return findCertificateForAPI(profiles, certName)
}

func resolveCertificateForAPI(c *client.Client, ctx context.Context, lbCiID int64, certName string) (string, diag.Diagnostics) {
	var diags diag.Diagnostics
	certName = strings.TrimSpace(certName)
	if certName == "" {
		return "", diags
	}

	profiles, err := ListClientSSLProfiles(c, ctx, lbCiID)
	if err != nil {
		diags.AddError("SSL Profile List Failed", fmt.Sprintf("Could not resolve certificate_name on load balancer %d: %s", lbCiID, err.Error()))
		return "", diags
	}

	apiCertName, found := findCertificateForAPI(profiles, certName)
	if !found {
		diags.AddError(
			"Certificate Not Found",
			fmt.Sprintf(
				`certificate_name %q was not found on load balancer %d. Upload it with vayucloud_network_lb_ssl_profile or list names via vayucloud_network_lb_ssl_profiles.`,
				normalizeCertificateStorageName(certName),
				lbCiID,
			),
		)
		return "", diags
	}
	return apiCertName, diags
}

// normalizeCertificateNameForState maps API full paths back to storage names for Terraform state.
func normalizeCertificateNameForState(certName string) string {
	certName = strings.TrimSpace(certName)
	if certName == "" {
		return ""
	}
	if strings.Contains(certName, "/") {
		return normalizeCertificateStorageName(certificateBaseName(certName))
	}
	return normalizeCertificateStorageName(certName)
}

func validateCertificateConfig(protocol, port, certName string) diag.Diagnostics {
	var diags diag.Diagnostics
	if !certificateRequiredForListener(protocol, port) {
		return diags
	}
	if certName != "" {
		return diags
	}
	diags.AddError(
		"Missing Certificate",
		"certificate_name is required when protocol is HTTPS or when the listener port includes an SSL port (443, 8443, 4433, 9443, 10443).",
	)
	return diags
}

func validateCertificateExistsOnLB(c *client.Client, ctx context.Context, lbCiID int64, certName string) diag.Diagnostics {
	_, diags := resolveCertificateForAPI(c, ctx, lbCiID, certName)
	return diags
}

func buildValidateAPIRequest(data *NetworkLBVirtualServiceResourceModel) (int64, *VirtualServiceRequest, diag.Diagnostics) {
	var diags diag.Diagnostics

	if data.LoadBalancerID.IsNull() || data.LoadBalancerID.IsUnknown() {
		return 0, nil, diags
	}
	if data.Name.IsNull() || data.Name.IsUnknown() {
		return 0, nil, diags
	}
	if data.Port.IsNull() || data.Port.IsUnknown() {
		return 0, nil, diags
	}
	if data.Protocol.IsNull() || data.Protocol.IsUnknown() {
		return 0, nil, diags
	}
	if data.ZoneID.IsNull() || data.ZoneID.IsUnknown() {
		if !data.LoadBalancerID.IsNull() && !data.LoadBalancerID.IsUnknown() {
			diags.AddError(
				"Missing Zone ID",
				"zone_id is required. Use vayucloud_network_lb.<name>.zone_id from the parent load balancer.",
			)
		}
		return 0, nil, diags
	}

	lbCiID, err := parsePositiveInt64(data.LoadBalancerID.ValueString())
	if err != nil {
		diags.AddError("Invalid LB ID", err.Error())
		return 0, nil, diags
	}

	protocol := normalizeProtocol(data.Protocol.ValueString())
	monitors, monitorDiags := resolveMonitors(data.Monitor, protocol)
	diags.Append(monitorDiags...)
	if diags.HasError() {
		return 0, nil, diags
	}

	req := &VirtualServiceRequest{
		VirtualServerName: strings.TrimSpace(data.Name.ValueString()),
		VirtualServerPort: strings.TrimSpace(data.Port.ValueString()),
		Protocol:          apiProtocolForCreate(protocol),
		ZoneID:            int(data.ZoneID.ValueInt64()),
		PoolAlgorithm:     normalizePoolAlgorithm(data.PoolAlgorithm.ValueString()),
		Monitors:          monitors,
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
	if certName := certificateNameFromModel(data); certName != "" {
		req.CertificateName = normalizeCertificateStorageName(certName)
	}

	return lbCiID, req, diags
}

func shouldValidateVirtualServiceAtPlan(plan, state *NetworkLBVirtualServiceResourceModel, hasState bool) bool {
	if !hasState {
		return true
	}
	if vsIdentityRequiresReplace(plan.LoadBalancerID, state.LoadBalancerID) {
		return true
	}
	return vsIdentityRequiresReplace(plan.Name, state.Name)
}

func vsIdentityRequiresReplace(plan, state types.String) bool {
	if plan.IsNull() || plan.IsUnknown() || state.IsNull() || state.IsUnknown() {
		return false
	}
	planID := strings.TrimSpace(plan.ValueString())
	stateID := strings.TrimSpace(state.ValueString())
	if planID == "" || stateID == "" {
		return false
	}
	return planID != stateID
}

func parsePositiveInt64(value string) (int64, error) {
	parsed, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil || parsed <= 0 {
		return 0, fmt.Errorf("must be a positive integer: %q", value)
	}
	return parsed, nil
}

func validateVirtualServiceZoneAtPlan(ctx context.Context, c *client.Client, plan *NetworkLBVirtualServiceResourceModel, lbCiID int64) diag.Diagnostics {
	var diags diag.Diagnostics
	if plan.ZoneID.IsNull() || plan.ZoneID.IsUnknown() {
		diags.AddError(
			"Missing Zone ID",
			"zone_id is required. Use vayucloud_network_lb.<name>.zone_id from the parent load balancer.",
		)
		return diags
	}
	zoneID := plan.ZoneID.ValueInt64()
	if zoneID <= 0 {
		diags.AddError("Invalid Zone ID", "zone_id must be a positive integer.")
		return diags
	}
	if err := network_lb.ValidateLoadBalancerExists(c, ctx, lbCiID); err != nil {
		diags.AddError("Invalid Load Balancer ID", err.Error())
		return diags
	}
	firewallID, err := network_lb.ResolveFirewallCIFromLB(c, ctx, lbCiID)
	if err != nil {
		diags.AddError("Invalid Load Balancer ID", err.Error())
		return diags
	}
	if err := network_lb.ValidateLBZoneForFirewall(c, ctx, zoneID, firewallID); err != nil {
		diags.AddError("Invalid Zone ID", err.Error())
	}
	return diags
}

func validatePoolMembersAtPlan(plan *NetworkLBVirtualServiceResourceModel, isCreate bool) diag.Diagnostics {
	var diags diag.Diagnostics
	if len(plan.PoolMembers) == 0 {
		if isCreate {
			diags.AddError("Missing Pool Members", "At least one pool_member block is required.")
		}
		return diags
	}

	seen := map[string]bool{}
	for i, member := range plan.PoolMembers {
		name := ""
		if !member.Name.IsNull() && !member.Name.IsUnknown() {
			name = strings.TrimSpace(member.Name.ValueString())
		}
		label := fmt.Sprintf("pool_member[%d]", i)
		if name == "" && !member.Name.IsUnknown() {
			diags.AddError("Invalid Pool Member", fmt.Sprintf("%s: name is required.", label))
		}
		if member.IPAddress.IsNull() || member.IPAddress.IsUnknown() {
			continue
		}
		ip := strings.TrimSpace(member.IPAddress.ValueString())
		if ip == "" {
			diags.AddError(
				"Invalid Pool Member",
				fmt.Sprintf("%s (%s): ip_address is required. Discover IPs with vayucloud_virtualmachine_list for the LB zone.", label, name),
			)
		}
		if member.Port.IsNull() || member.Port.IsUnknown() {
			continue
		}
		port := member.Port.ValueInt64()
		if port <= 0 || port > 65535 {
			diags.AddError("Invalid Pool Member", fmt.Sprintf("%s (%s): port must be between 1 and 65535.", label, name))
		}
		if name != "" {
			key := name + ":" + strconv.FormatInt(port, 10)
			if seen[key] {
				diags.AddError("Invalid Pool Member", fmt.Sprintf("duplicate pool member %q on port %d.", name, port))
			}
			seen[key] = true
		}
	}
	return diags
}

func vsCertificateRequiresPlanCheck(plan, state *NetworkLBVirtualServiceResourceModel, hasState bool) bool {
	if !hasState {
		return true
	}
	if plan.CertificateName.IsUnknown() {
		return false
	}
	return !plan.CertificateName.Equal(state.CertificateName)
}

func runVirtualServicePlanValidations(ctx context.Context, c *client.Client, plan, state *NetworkLBVirtualServiceResourceModel, hasState bool) diag.Diagnostics {
	var diags diag.Diagnostics
	if c == nil {
		return diags
	}

	protocol := ""
	if !plan.Protocol.IsNull() && !plan.Protocol.IsUnknown() {
		protocol = plan.Protocol.ValueString()
	}
	port := ""
	if !plan.Port.IsNull() && !plan.Port.IsUnknown() {
		port = plan.Port.ValueString()
	}
	certName := certificateNameFromModel(plan)

	diags.Append(validateCertificateConfig(protocol, port, certName)...)

	isCreate := !hasState
	var lbCiID int64
	if !plan.LoadBalancerID.IsNull() && !plan.LoadBalancerID.IsUnknown() {
		var err error
		lbCiID, err = parsePositiveInt64(plan.LoadBalancerID.ValueString())
		if err != nil {
			diags.AddError("Invalid LB ID", err.Error())
			return diags
		}
		diags.Append(validateVirtualServiceZoneAtPlan(ctx, c, plan, lbCiID)...)
		diags.Append(validatePoolMembersAtPlan(plan, isCreate)...)
		// On first VS create, cert may be uploaded in the same apply (depends_on ssl_profile).
		// Only require cert to exist at plan time when updating an existing VS or changing cert.
		if certName != "" && hasState && vsCertificateRequiresPlanCheck(plan, state, hasState) {
			diags.Append(validateCertificateExistsOnLB(c, ctx, lbCiID, certName)...)
		}
	}

	if diags.HasError() {
		return diags
	}

	lbCiID, validateReq, buildDiags := buildValidateAPIRequest(plan)
	diags.Append(buildDiags...)
	if diags.HasError() {
		return diags
	}

	if !shouldValidateVirtualServiceAtPlan(plan, state, hasState) {
		return diags
	}

	if validateReq == nil {
		return diags
	}

	if certName != "" {
		if resolved, ok := tryResolveCertificateForAPI(c, ctx, lbCiID, certName); ok {
			validateReq.CertificateName = resolved
		}
	}

	tflog.Debug(ctx, "Validating virtual service creation at plan time", map[string]any{
		"load_balancer_id": lbCiID,
		"name":             validateReq.VirtualServerName,
		"port":             validateReq.VirtualServerPort,
		"vip_ip":           validateReq.VipIP,
	})

	if err := ValidateVirtualServiceCreation(c, ctx, lbCiID, validateReq); err != nil {
		diags.AddError("Virtual Service Validation Failed", err.Error())
	}

	return diags
}
