# VayuCloud network zone example
#
# Creates a zone in an environment scoped to one firewall.

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

variable "network_zone_name" {
  type        = string
  description = "User-visible zone label."
}

variable "environment_id" {
  type        = number
  description = "Parent resource_group_environment id."
}

variable "firewall_id" {
  type        = number
  description = "Firewall enforcing the perimeter for this zone."
}

variable "no_of_ips" {
  type        = number
  description = "IPv4 pool size allocated to the zone."
  default     = 20
}

resource "vayucloud_network_zone" "auto_ipam_zone" {
  name           = var.network_zone_name
  environment_id = var.environment_id
  firewall_id    = var.firewall_id
  no_of_ips      = var.no_of_ips
}

output "network_zone" {
  value = {
    id     = vayucloud_network_zone.auto_ipam_zone.id
    status = vayucloud_network_zone.auto_ipam_zone.status
  }
}
