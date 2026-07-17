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

# =============================================================================
# Example 1: List all firewalls for an engagement and endpoint
# =============================================================================

# =============================================================================
# Example 2: List firewalls with an optional filter
# =============================================================================

data "vayucloud_network_firewall_list" "filtered" {
  engagement_id = xxxxx
  endpoint_id   = xxxxx

  filter {
    name   = "hypervisor"
    values = ["xxxx"]
  }
}

output "firewalls" {
  description = "Network firewalls for the configured engagement and endpoint"
  value       = data.vayucloud_network_firewall_list.filtered.firewalls
}
