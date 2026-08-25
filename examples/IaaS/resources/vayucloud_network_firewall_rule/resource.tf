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

variable "rule_name" {
  type        = string
  description = "Firewall rule name."
}

variable "firewall_id" {
  type        = string
  description = "Firewall CI Master ID."
}

variable "source" {
  type        = string
  description = "Rule source (for example zone, internet, nas, vcs)."
}

variable "destination" {
  type        = string
  description = "Rule destination (for example vcs, zone, internet, nas)."
}

variable "source_zone_id" {
  type        = number
  description = "Source zone ID when source is a zone."
  default     = null
  nullable    = true
}

variable "destination_zone_id" {
  type        = number
  description = "Destination zone ID when destination is a zone."
  default     = null
  nullable    = true
}

variable "action" {
  type        = string
  description = "Rule action (for example allow)."
}

variable "services" {
  type        = list(string)
  description = "Service names (for example HTTP, HTTPS)."
}

variable "source_addresses" {
  type        = list(string)
  description = "Source CIDRs."
}

variable "destination_addresses" {
  type        = list(string)
  description = "Destination CIDRs."
}

variable "schedule_start_date" {
  type        = string
  description = "Optional schedule start date (YYYY-MM-DD)."
  default     = null
  nullable    = true
}

variable "schedule_end_date" {
  type        = string
  description = "Optional schedule end date (YYYY-MM-DD)."
  default     = null
  nullable    = true
}

resource "vayucloud_network_firewall_rule" "example" {
  rule_name              = var.rule_name
  firewall_id            = var.firewall_id
  source                 = var.source
  destination            = var.destination
  source_zone_id         = var.source_zone_id
  destination_zone_id    = var.destination_zone_id
  action                 = var.action
  services               = var.services
  source_addresses       = var.source_addresses
  destination_addresses  = var.destination_addresses
  schedule_start_date    = var.schedule_start_date
  schedule_end_date      = var.schedule_end_date
}

output "firewall_rule" {
  value = {
    id                    = vayucloud_network_firewall_rule.example.id
    rule_name             = vayucloud_network_firewall_rule.example.rule_name
    firewall_id           = vayucloud_network_firewall_rule.example.firewall_id
    source                = vayucloud_network_firewall_rule.example.source
    destination           = vayucloud_network_firewall_rule.example.destination
    destination_addresses = vayucloud_network_firewall_rule.example.destination_addresses
    action                = vayucloud_network_firewall_rule.example.action
    status                = vayucloud_network_firewall_rule.example.status
    audit_id              = vayucloud_network_firewall_rule.example.audit_id
  }
}
