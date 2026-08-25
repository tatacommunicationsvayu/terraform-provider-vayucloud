# VayuCloud Virtual Machine Flavor Data Source Example
#
# Queries the flavor catalog for a zone with optional filters.

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
  type        = string
  description = "Zone ID whose flavor catalog to query."
}

variable "name_filter" {
  type        = list(string)
  description = "Optional flavor name filter values. Empty list skips the filter."
  default     = []
}

variable "os_model_filter" {
  type        = list(string)
  description = "Optional os_model filter values. Empty list skips the filter."
  default     = []
}

data "vayucloud_virtualmachine_flavor" "filtered" {
  zone_id = var.zone_id

  dynamic "filter" {
    for_each = length(var.name_filter) > 0 ? [var.name_filter] : []
    content {
      name   = "name"
      values = filter.value
    }
  }

  dynamic "filter" {
    for_each = length(var.os_model_filter) > 0 ? [var.os_model_filter] : []
    content {
      name   = "os_model"
      values = filter.value
    }
  }
}

output "first_matching_flavor_id" {
  description = "Flavor id of first match (if any)"
  value       = try(data.vayucloud_virtualmachine_flavor.filtered.flavors[0].id, null)
}
