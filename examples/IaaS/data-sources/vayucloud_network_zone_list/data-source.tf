# VayuCloud Network Zone List Data Source Example
#
# Lists network zones for an environment.

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

variable "environment_id" {
  type        = number
  description = "Environment ID whose zones to list."
}

variable "zone_type_filter" {
  type        = list(string)
  description = "Optional zone_type filter values. Empty list skips the filter."
  default     = []
}

data "vayucloud_network_zone_list" "filtered" {
  environment_id = var.environment_id

  dynamic "filter" {
    for_each = length(var.zone_type_filter) > 0 ? [var.zone_type_filter] : []
    content {
      name   = "zone_type"
      values = filter.value
    }
  }
}

output "network_zones" {
  description = "Network zones for the environment"
  value       = data.vayucloud_network_zone_list.filtered.network_zones
}
