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

variable "firewall_id" {
  type        = number
  description = "Firewall ID scope. Set this or engagement_id, not both."
  default     = null
  nullable    = true
}

variable "engagement_id" {
  type        = number
  description = "Engagement ID scope. Set this or firewall_id, not both."
  default     = null
  nullable    = true
}

variable "purpose_filter" {
  type        = list(string)
  description = "Optional purpose filter values. Empty list skips the filter."
  default     = []
}

data "vayucloud_network_public_ips" "example" {
  firewall_id   = var.firewall_id
  engagement_id = var.engagement_id

  dynamic "filter" {
    for_each = length(var.purpose_filter) > 0 ? [var.purpose_filter] : []
    content {
      name   = "purpose"
      values = filter.value
    }
  }
}

output "public_ip_segments" {
  description = "publicIpSegment values from the API"
  value       = data.vayucloud_network_public_ips.example.public_ips
}

output "row_count" {
  description = "Number of public IP rows returned"
  value       = length(data.vayucloud_network_public_ips.example.public_ips)
}
