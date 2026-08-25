// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

// Package provider implements the VayuCloud Terraform provider.
//
// This provider allows Terraform to manage resources on the VayuCloud platform,
// authenticating via a Spring Boot REST API that returns Bearer tokens.
package provider

import (
	"context"
	"os"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/client"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/account_engagement"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/file_server"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/file_storage_export_policy"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/file_storage_volume"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/nas"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/account_engagement_auditlog"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/account_location"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/auditlog_details"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/network_c2s_vpn"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/network_c2s_vpn_user"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/network_firewall"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/network_lb"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/network_lb_ssl_profile"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/network_lb_virtualservice"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/network_firewall_rule"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/network_public_ip"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/network_zone"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/resource_group_business_unit"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/resource_group_environment"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/s3_bucket"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/s3_domain"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/s3_object"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/s3_token"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/s3_user"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/security_group"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/virtualmachine"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/virtualmachine_blockstorage"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/virtualmachine_flavor"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/virtualmachine_image"
	"github.com/tatacommunications/terraform-provider-vayucloud/internal/provider/IaaS/virtualmachine_security_group_association"
)

// Ensure VayuCloudProvider satisfies various provider interfaces.
var _ provider.Provider = &VayuCloudProvider{}
var _ provider.ProviderWithValidateConfig = &VayuCloudProvider{}

// VayuCloudProvider defines the provider implementation.
type VayuCloudProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// VayuCloudProviderModel describes the provider data model.
// These fields map directly to the provider block in Terraform configuration.
type VayuCloudProviderModel struct {
	Username types.String `tfsdk:"username"`
	Password types.String `tfsdk:"password"`
	Timeout  types.Int64  `tfsdk:"timeout"`
}

// New creates a new provider factory function.
// This is called by the main.go to create provider instances.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &VayuCloudProvider{
			version: version,
		}
	}
}

// Metadata returns the provider type name.
func (p *VayuCloudProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "vayucloud"
	resp.Version = p.version
}

// Schema defines the provider-level schema for configuration data.
// These are the fields that users will set in the provider block.
func (p *VayuCloudProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The VayuCloud provider allows you to manage resources on the VayuCloud platform.",
		MarkdownDescription: `
The VayuCloud provider allows you to manage resources on the VayuCloud platform.

## Authentication

The provider authenticates using a username and password against the TATA Communications IDP.
The API returns a Bearer token that is used for all subsequent API calls.

You can configure credentials either in the provider block or via environment variables:

- ` + "`VAYU_USERNAME`" + ` - Username
- ` + "`VAYU_PASSWORD`" + ` - Password

## Example Usage

` + "```hcl" + `
provider "vayucloud" {
  # Or set VAYU_USERNAME and VAYU_PASSWORD in the environment.
}
` + "```" + `
`,
		Attributes: map[string]schema.Attribute{
			"username": schema.StringAttribute{
				Description:         "The username for authentication. Can also be set via VAYU_USERNAME environment variable.",
				MarkdownDescription: "The username for authentication. Can also be set via `VAYU_USERNAME` environment variable.",
				Optional:            true,
				Sensitive:           true,
			},
			"password": schema.StringAttribute{
				Description:         "The password for authentication. Can also be set via VAYU_PASSWORD environment variable.",
				MarkdownDescription: "The password for authentication. Can also be set via `VAYU_PASSWORD` environment variable.",
				Optional:            true,
				Sensitive:           true,
			},
			"timeout": schema.Int64Attribute{
				Description:         "HTTP client timeout in seconds. Defaults to 60. Can also be set via VAYU_TIMEOUT environment variable.",
				MarkdownDescription: "HTTP client timeout in seconds. Defaults to `60`. Can also be set via `VAYU_TIMEOUT` environment variable.",
				Optional:            true,
			},
		},
	}
}

// ValidateConfig validates the provider configuration before it is used.
// This runs during `terraform validate` and provides early feedback.
func (p *VayuCloudProvider) ValidateConfig(ctx context.Context, req provider.ValidateConfigRequest, resp *provider.ValidateConfigResponse) {
	var config VayuCloudProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	username := os.Getenv("VAYU_USERNAME")
	password := os.Getenv("VAYU_PASSWORD")


	if config.Username.IsUnknown() && username == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("username"),
			"Unknown VayuCloud Username",
			"The provider cannot create the VayuCloud API client as there is an unknown configuration value for the VayuCloud username. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the VAYU_USERNAME environment variable.",
		)
	}

	if config.Password.IsUnknown() && password == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("password"),
			"Unknown VayuCloud Password",
			"The provider cannot create the VayuCloud API client as there is an unknown configuration value for the VayuCloud password. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the VAYU_PASSWORD environment variable.",
		)
	}

	if config.Timeout.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("timeout"),
			"Unknown VayuCloud Timeout",
			"The provider cannot create the VayuCloud API client as there is an unknown configuration value for the timeout. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the VAYU_TIMEOUT environment variable.",
		)
	}
}

// Configure prepares a VayuCloud API client for data sources and resources.
// This is where we authenticate and create the client that will be shared
// across all resources and data sources.
func (p *VayuCloudProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	tflog.Info(ctx, "Configuring VayuCloud provider")

	var config VayuCloudProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Default values to environment variables, but override with
	// Terraform configuration value if set.
	username := os.Getenv("VAYU_USERNAME")
	password := os.Getenv("VAYU_PASSWORD")

	if !config.Username.IsNull() {
		username = config.Username.ValueString()
	}

	if !config.Password.IsNull() {
		password = config.Password.ValueString()
	}

	// Validate required configuration
	if username == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("username"),
			"Missing VayuCloud Username",
			"The provider cannot create the VayuCloud API client as there is a missing or empty value for the VayuCloud username. "+
				"Set the username value in the configuration or use the VAYU_USERNAME environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if password == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("password"),
			"Missing VayuCloud Password",
			"The provider cannot create the VayuCloud API client as there is a missing or empty value for the VayuCloud password. "+
				"Set the password value in the configuration or use the VAYU_PASSWORD environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// Resolve timeout: env var -> config -> default (60s)
	timeout := int64(client.DefaultTimeout)
	if envTimeout := os.Getenv("VAYU_TIMEOUT"); envTimeout != "" {
		if parsed, err := strconv.ParseInt(envTimeout, 10, 64); err == nil && parsed > 0 {
			timeout = parsed
		}
	}
	if !config.Timeout.IsNull() {
		timeout = config.Timeout.ValueInt64()
	}

	tflog.Debug(ctx, "Creating VayuCloud client", map[string]any{
		"username": username,
		"timeout":  timeout,
	})

	clientConfig := &client.Config{
		Username: username,
		Password: password,
		Timeout:  timeout,
	}

	vayuCloudClient, err := client.NewClient(ctx, clientConfig)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create VayuCloud API Client",
			"An unexpected error occurred when creating the VayuCloud API client. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"VayuCloud Client Error: "+err.Error(),
		)
		return
	}

	resp.DataSourceData = vayuCloudClient
	resp.ResourceData = vayuCloudClient
	tflog.Info(ctx, "VayuCloud provider configured successfully")
}

// Resources defines the resources implemented in the provider.
func (p *VayuCloudProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		file_server.NewFileServerResource,
		file_storage_volume.NewFileStorageVolumeResource,
		file_storage_export_policy.NewFileStorageExportPolicyResource,
		network_c2s_vpn.NewNetworkC2SVPNResource,
		network_c2s_vpn_user.NewNetworkC2SVPNUserResource,
		network_firewall.NewNetworkFirewallResource,
		network_lb.NewNetworkLBResource,
		network_lb_virtualservice.NewNetworkLBVirtualServiceResource,
		network_lb_ssl_profile.NewNetworkLBSSLProfileResource,
		network_firewall_rule.NewNetworkFirewallRuleResource,
		network_public_ip.NewNetworkPublicIPResource,
		network_zone.NewNetworkZoneResource,
		virtualmachine.NewVirtualMachineResource,
		virtualmachine_blockstorage.NewVirtualMachineBlockStorageResource,
		virtualmachine.NewVirtualMachineStateResource,
		resource_group_business_unit.NewResourceGroupBusinessUnitResource,
		resource_group_environment.NewResourceGroupEnvironmentResource,
		nas.NewExtendNasZoneResource,
		s3_domain.NewS3DomainResource,
		s3_bucket.NewS3BucketResource,
		s3_object.NewS3ObjectResource,
		s3_user.NewS3UserResource,
		s3_token.NewS3TokenResource,
		security_group.NewSecurityGroupResource,
		virtualmachine_security_group_association.NewVirtualMachineSecurityGroupAssociationResource,
	}
}

// DataSources defines the data sources implemented in the provider.
func (p *VayuCloudProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		network_c2s_vpn.NewNetworkC2SVPNDataSource,
		network_c2s_vpn_user.NewNetworkC2SVPNUsersDataSource,
		network_firewall.NewNetworkFirewallDataSource,
		network_firewall.NewNetworkFirewallListDataSource,
		network_firewall_rule.NewNetworkFirewallRuleDataSource,
		network_firewall_rule.NewNetworkFirewallRuleListDataSource,
		network_lb.NewNetworkLBDataSource,
		network_lb.NewNetworkLBListDataSource,
		network_lb_virtualservice.NewNetworkLBVirtualServiceOptionsDataSource,
		network_lb_ssl_profile.NewNetworkLBSSLProfilesDataSource,
		network_public_ip.NewNetworkPublicIPsDataSource,
		network_zone.NewNetworkZoneDataSource,
		network_zone.NewNetworkZoneListDataSource,
		account_engagement.NewAccountEngagementDataSource,
		account_location.NewAccountLocationDataSource,
		resource_group_business_unit.NewResourceGroupBusinessUnitDataSource,
		resource_group_business_unit.NewResourceGroupBusinessUnitListDataSource,
		resource_group_environment.NewResourceGroupEnvironmentDataSource,
		resource_group_environment.NewResourceGroupEnvironmentListDataSource,
		s3_domain.NewS3DomainDataSource,
		s3_domain.NewS3DomainListDataSource,
		s3_bucket.NewS3BucketDataSource,
		s3_bucket.NewS3BucketListDataSource,
		s3_object.NewS3ObjectDataSource,
		s3_object.NewS3ObjectListDataSource,
		s3_user.NewS3UserDataSource,
		s3_user.NewS3UserListDataSource,
		s3_token.NewS3TokenDataSource,
		s3_token.NewS3TokenListDataSource,
		account_engagement_auditlog.NewAccountEngagementAuditLogDataSource,
		auditlog_details.NewAuditLogDetailsDataSource,
		virtualmachine_image.NewVirtualMachineImageDataSource,
		virtualmachine_flavor.NewVirtualMachineFlavorDataSource,
		virtualmachine.NewVirtualMachineDataSource,
		virtualmachine.NewVirtualMachineListDataSource,
		file_server.NewFileServerDataSource,
		file_server.NewFileServerListDataSource,
		virtualmachine_blockstorage.NewVirtualMachineBlockStorageDataSource,
		file_storage_volume.NewFileStorageDataSource,
		nas.NewNasVlanZoneDataSource,
		security_group.NewListSecurityGroupDataSource,
		security_group.NewListSecurityGroupRulesDataSource,
		security_group.NewVirtualMachineListSecurityGroupRulesDataSource,
	}
}
