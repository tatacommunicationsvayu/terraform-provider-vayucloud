// Copyright (c) TATA Communications
// SPDX-License-Identifier: Apache-2.0

// Package common holds shared constants and helpers for the Terraform provider.
package common

// API endpoint constants
const (
	// NetworkOperationsPath is the path for the network operations API
	//NetworkOperationsPath = "networkservice/network_operations"
	NetworkOperationsPath = "network_operations"

	//ConfigServicePath = "portalservice/configservice"
	ConfigServicePath = "configservice"
	//SecurityServicePath = "portalservice/securityservice"
	SecurityServicePath = "securityservice"

	//InstanceServicePath = "portalservice/vm-instances"
	InstanceServicePath = "vm-instances"

	//AuditLogServicePath = "auditlogservice/auditlog"
	AuditLogServicePath = "auditlog"

	//NetworkServicePath = "networkservice/network"
	NetworkServicePath = "network"

	// LoadBalancerServicePath is LBConfigController (@RequestMapping("/loadbalancer")) under network-service context.
	//LoadBalancerServicePath = "networkservice/loadbalancer"
	LoadBalancerServicePath = "loadbalancer"
	//FirewallConfigPath = "networkservice/firewallconfig"
	FirewallConfigPath = "firewallconfig"
	//FileStorageServicePath = "portalservice/nas"
	FileStorageServicePath = "nas"

	//ICSOperationsPath = "ics-service/ics-operations"
	ICSOperationsPath = "ics-operations"

	// SecurityGroupServicePath is the path for security group and security group rule APIs.
	//SecurityGroupServicePath = "networkservice/security-group"
	SecurityGroupServicePath = "security-group"
)
