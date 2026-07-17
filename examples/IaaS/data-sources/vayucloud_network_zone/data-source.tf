# VayuCloud Network Zone Data Source Example
#
# Reads one network zone by resource ID.

terraform {
  required_providers {
    vayucloud = {
      source = "tatacommunicationsvayu/vayucloud"
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

# =============================================================================
# Example: Read a network zone by ID
# =============================================================================

data "vayucloud_network_zone" "example" {
  network_zone_id = xxxxx
}

output "network_zone_summary" {
  description = "Key attributes of the network zone"
  value = {
    name            = data.vayucloud_network_zone.example.name
    environment_id  = data.vayucloud_network_zone.example.environment_id
    firewall_id     = data.vayucloud_network_zone.example.firewall_id
    purpose         = data.vayucloud_network_zone.example.purpose
    zone_type       = data.vayucloud_network_zone.example.zone_type
    data_plane      = data.vayucloud_network_zone.example.data_plane
    no_of_ips       = data.vayucloud_network_zone.example.no_of_ips
    ipv6_cidr       = data.vayucloud_network_zone.example.ipv6_cidr
  }
}
