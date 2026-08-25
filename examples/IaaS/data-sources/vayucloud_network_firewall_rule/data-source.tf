# VayuCloud Network Firewall Rule Data Source Example
#
# Reads one firewall rule by firewall ID and rule ID (action-state read).

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

variable "rule_id" {
  type        = string
  description = "Firewall rule ID."
}

data "vayucloud_network_firewall_rule" "example" {
  firewall_id = var.firewall_id
  id          = var.rule_id
}

output "rule_summary" {
  description = "Firewall rule summary from read state"
  value = {
    id                    = data.vayucloud_network_firewall_rule.example.id
    rule_name             = data.vayucloud_network_firewall_rule.example.rule_name
    source                = data.vayucloud_network_firewall_rule.example.source
    destination           = data.vayucloud_network_firewall_rule.example.destination
    action                = data.vayucloud_network_firewall_rule.example.action
    services              = data.vayucloud_network_firewall_rule.example.services
    status                = data.vayucloud_network_firewall_rule.example.status
    source_addresses      = data.vayucloud_network_firewall_rule.example.source_addresses
    destination_addresses = data.vayucloud_network_firewall_rule.example.destination_addresses
    schedule_start_date   = data.vayucloud_network_firewall_rule.example.schedule_start_date
    schedule_end_date     = data.vayucloud_network_firewall_rule.example.schedule_end_date
  }
}
