# VayuCloud List Security Group Data Source Example
#
# Lists security groups under a firewall.

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
  description = "Firewall ID whose security groups to list."
}

variable "name_filter" {
  type        = list(string)
  description = "Optional security group name filter values. Empty list skips the filter."
  default     = []
}

data "vayucloud_list_security_group" "all" {
  firewall_id = var.firewall_id

  dynamic "filter" {
    for_each = length(var.name_filter) > 0 ? [var.name_filter] : []
    content {
      name   = "name"
      values = filter.value
    }
  }
}

output "security_groups" {
  value = data.vayucloud_list_security_group.all.security_groups
}
