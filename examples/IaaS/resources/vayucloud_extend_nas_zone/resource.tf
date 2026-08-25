# Copyright (c) TATA Communications
# SPDX-License-Identifier: Apache-2.0
#
# VayuCloud extend NAS zone + NAS VLAN zone check (data source)
#
# API:
#   GET  .../nas/isNasVlanZone?engagementId=&endpointId=
#   POST .../nas/extendNASZone/{zoneId}/{vserverId}
#   POST .../nas/deconfigureNASZone/{zoneId}  (on destroy only — deconfigure NAS; does not delete the zone)
#
# Set var.extend_zone_id to a real zone id (e.g. from data.vayucloud_nas_vlan_zone.example.nas_vlan_zone_ids)
# before apply, or leave null to skip the extend resource (data source still runs).

terraform {
  required_providers {
    vayucloud = {
      source  = "tatacommunicationsvayu/vayucloud"
      version = "1.0.0"
    }
  }
}

provider "vayucloud" {
  username = var.vayucloud_username
  password = var.vayucloud_password
}

variable "vayucloud_username" {
  type        = string
  description = "VayuCloud API username (or set VAYU_USERNAME)."
  sensitive   = true
}

variable "vayucloud_password" {
  type        = string
  description = "VayuCloud API password (or set VAYU_PASSWORD)."
  sensitive   = true
}

variable "engagement_id" {
  type        = number
  description = "Engagement ID."
}

variable "endpoint_id" {
  type        = number
  description = "Endpoint ID."
}

variable "vserver_name" {
  type        = string
  description = "NAS vserver name used for the extend example."
}

variable "file_storage_type" {
  type        = string
  description = "File storage type (for example NAS-NFS)."
}

variable "extend_zone_id" {
  type        = number
  description = "Zone id for extendNASZone. Use a value from data.vayucloud_nas_vlan_zone.example.nas_vlan_zone_ids, or leave null to skip extend."
  default     = null
  nullable    = true
}

variable "is_firewall_configured" {
  type        = bool
  description = "Whether the NAS zone already has firewall configuration."
}

data "vayucloud_nas_vlan_zone" "example" {
  engagement_id = var.engagement_id
  endpoint_id   = var.endpoint_id
}

resource "vayucloud_file_server" "example" {
  engagement_id     = var.engagement_id
  endpoint_id       = var.endpoint_id
  vserver_name      = var.vserver_name
  file_storage_type = var.file_storage_type
}

resource "vayucloud_extend_nas_zone" "example" {
  count = var.extend_zone_id != null ? 1 : 0

  zone_id                = var.extend_zone_id
  vserver_id             = tonumber(vayucloud_file_server.example.id)
  is_firewall_configured = var.is_firewall_configured
  depends_on             = [vayucloud_file_server.example]
}

output "nas_vlan_zone_is_nas_vlan_zone" {
  description = "Portal isNasVlanZone flag for engagement/endpoint"
  value       = data.vayucloud_nas_vlan_zone.example.is_nas_vlan_zone
}

output "nas_vlan_zone_ids" {
  description = "Qualifying zone ids from isNasVlanZone"
  value       = data.vayucloud_nas_vlan_zone.example.nas_vlan_zone_ids
}

output "file_server_id" {
  value = vayucloud_file_server.example.id
}

output "extend_nas_zone_raw_response" {
  description = "Last extend API JSON (empty list if extend was skipped)"
  value       = try(vayucloud_extend_nas_zone.example[0].raw_response, null)
}
