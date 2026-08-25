# VayuCloud Resource Group Environment List Data Source Example
#
# Lists environments in a business unit.

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

variable "business_unit_id" {
  type        = number
  description = "Business unit ID whose environments to list."
}

variable "status_filter" {
  type        = list(string)
  description = "Optional status filter values. Empty list skips the filter."
  default     = []
}

data "vayucloud_resource_group_environment_list" "filtered" {
  business_unit_id = var.business_unit_id

  dynamic "filter" {
    for_each = length(var.status_filter) > 0 ? [var.status_filter] : []
    content {
      name   = "status"
      values = filter.value
    }
  }
}

output "environments" {
  description = "Environments returned for the business unit"
  value       = data.vayucloud_resource_group_environment_list.filtered.environments
}
