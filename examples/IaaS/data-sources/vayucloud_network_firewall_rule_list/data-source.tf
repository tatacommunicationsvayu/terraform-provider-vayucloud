# VayuCloud Network Firewall Rule List Data Source Example
#
# Lists firewall rules for a firewall (action-state list).

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
  description = "Firewall ID whose rules to list."
}

variable "action_filter" {
  type        = list(string)
  description = "Optional action filter values. Empty list skips the filter."
  default     = []
}

data "vayucloud_network_firewall_rule_list" "all" {
  firewall_id = var.firewall_id

  dynamic "filter" {
    for_each = length(var.action_filter) > 0 ? [var.action_filter] : []
    content {
      name   = "action"
      values = filter.value
    }
  }
}

output "rule_count" {
  description = "Number of firewall rules on the firewall"
  value       = length(data.vayucloud_network_firewall_rule_list.all.rules)
}

output "rule_ids" {
  description = "Rule IDs from the list"
  value       = data.vayucloud_network_firewall_rule_list.all.rules
}
