# VayuCloud Network Firewall Data Source Example
#
# Reads one network firewall by resource ID.

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

variable "network_firewall_id" {
  type        = string
  description = "Network firewall resource ID."
}

data "vayucloud_network_firewall" "example" {
  network_firewall_id = var.network_firewall_id
}

output "firewall_summary" {
  description = "Key attributes of the network firewall"
  value = {
    display_name       = data.vayucloud_network_firewall.example.firewall_display_name
    throughput        = data.vayucloud_network_firewall.example.firewall_throughput
    internet_bandwidth = data.vayucloud_network_firewall.example.internet_bandwidth
    engagement_id      = data.vayucloud_network_firewall.example.engagement_id
    endpoint_id        = data.vayucloud_network_firewall.example.endpoint_id
    hypervisor         = data.vayucloud_network_firewall.example.hypervisor
  }
}
