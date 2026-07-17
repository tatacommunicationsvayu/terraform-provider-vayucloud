# VayuCloud Network Public IPs Data Source Example
#
# Lists public IP inventory for exactly one scope: firewall or engagement.

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
# Example 1: By firewall
# =============================================================================

# =============================================================================
# Example 2: By engagement (uncomment exactly one scope)
# =============================================================================

data "vayucloud_network_public_ips" "by_firewall" {
  firewall_id = xxxxx
}

# data "vayucloud_network_public_ips" "by_engagement" {
#   engagement_id = xxxxx
# }

output "public_ip_segments" {
  description = "publicIpSegment values from the API"
  value       = [for row in data.vayucloud_network_public_ips.by_firewall.public_ips : row.public_ip_segment]
}

output "row_count" {
  description = "Number of public IP rows returned"
  value       = length(data.vayucloud_network_public_ips.by_firewall.public_ips)
}
