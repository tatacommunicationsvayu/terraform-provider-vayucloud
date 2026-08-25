# VayuCloud Network Firewall List Data Source Example
#
# Lists network firewalls for an engagement and endpoint.

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

variable "engagement_id" {
  type        = number
  description = "Engagement ID."
}

variable "endpoint_id" {
  type        = number
  description = "Endpoint ID."
}

variable "hypervisor_filter" {
  type        = list(string)
  description = "Optional hypervisor filter values. Empty list skips the filter."
  default     = []
}

data "vayucloud_network_firewall_list" "all" {
  engagement_id = var.engagement_id
  endpoint_id   = var.endpoint_id
}

data "vayucloud_network_firewall_list" "filtered" {
  engagement_id = var.engagement_id
  endpoint_id   = var.endpoint_id

  dynamic "filter" {
    for_each = length(var.hypervisor_filter) > 0 ? [var.hypervisor_filter] : []
    content {
      name   = "hypervisor"
      values = filter.value
    }
  }
}

output "firewalls" {
  description = "All network firewalls for the configured engagement and endpoint"
  value       = data.vayucloud_network_firewall_list.all.firewalls
}

output "filtered_firewalls" {
  description = "Network firewalls after optional hypervisor filter"
  value       = data.vayucloud_network_firewall_list.filtered.firewalls
}
