# VayuCloud Resource Group Business Unit List Data Source Example
#
# Lists business units for a firewall.

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
  description = "Firewall ID whose business units to list."
}

variable "business_unit_filter" {
  type        = list(string)
  description = "Optional business_unit name filter values. Empty list skips the filter."
  default     = []
}

data "vayucloud_resource_group_business_unit_list" "filtered" {
  firewall_id = var.firewall_id

  dynamic "filter" {
    for_each = length(var.business_unit_filter) > 0 ? [var.business_unit_filter] : []
    content {
      name   = "business_unit"
      values = filter.value
    }
  }
}

output "business_units" {
  description = "Business units returned for the firewall"
  value       = data.vayucloud_resource_group_business_unit_list.filtered.business_units
}
