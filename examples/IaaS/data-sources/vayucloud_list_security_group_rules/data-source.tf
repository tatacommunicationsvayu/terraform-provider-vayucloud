# VayuCloud List Security Group Rules Data Source Example
#
# Lists rules for a security group under a firewall.

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
  description = "Firewall ID."
}

variable "security_group_id" {
  type        = string
  description = "Security group ID whose rules to list."
}

variable "protocol_filter" {
  type        = list(string)
  description = "Optional protocol filter values. Empty list skips the filter."
  default     = []
}

variable "direction_filter" {
  type        = list(string)
  description = "Optional direction filter values. Empty list skips the filter."
  default     = []
}

data "vayucloud_list_security_group_rules" "rules" {
  firewall_id       = var.firewall_id
  security_group_id = var.security_group_id

  dynamic "filter" {
    for_each = length(var.protocol_filter) > 0 ? [var.protocol_filter] : []
    content {
      name   = "protocol"
      values = filter.value
    }
  }

  dynamic "filter" {
    for_each = length(var.direction_filter) > 0 ? [var.direction_filter] : []
    content {
      name   = "direction"
      values = filter.value
    }
  }
}

output "rules" {
  value = data.vayucloud_list_security_group_rules.rules.rules
}
