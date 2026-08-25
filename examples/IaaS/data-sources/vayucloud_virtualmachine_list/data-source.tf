# VayuCloud Virtual Machine List Data Source Example
#
# Lists virtual machines in a zone with optional filters.

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

variable "zone_id" {
  type        = number
  description = "Zone ID whose virtual machines to list."
}

variable "power_status_filter" {
  type        = list(string)
  description = "Optional power_status filter values. Empty list skips the filter."
  default     = []
}

variable "os_type_filter" {
  type        = list(string)
  description = "Optional os_type filter values. Empty list skips the filter."
  default     = []
}

data "vayucloud_virtualmachine_list" "filtered" {
  zone_id = var.zone_id

  dynamic "filter" {
    for_each = length(var.power_status_filter) > 0 ? [var.power_status_filter] : []
    content {
      name   = "power_status"
      values = filter.value
    }
  }

  dynamic "filter" {
    for_each = length(var.os_type_filter) > 0 ? [var.os_type_filter] : []
    content {
      name   = "os_type"
      values = filter.value
    }
  }
}

output "virtual_machines" {
  description = "Virtual machines returned for the zone"
  value       = data.vayucloud_virtualmachine_list.filtered.virtual_machines
}
