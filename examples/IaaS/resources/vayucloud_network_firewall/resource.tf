# VayuCloud Network Firewall Resource Example
#
# Create is asynchronous: the provider polls the audit log until completion.
#
# Bandwidth access (default): set firewall_throughput and internet_bandwidth.
#   Throughput must be >= internet_bandwidth (same unit, e.g. Mbps).
# DataTransfer access: set access_type = "DataTransfer", firewall_throughput, and minimum_commitment
#   (e.g. "500GB"); omit internet_bandwidth for that mode.

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

variable "engagement_id" {
  type        = number
  description = "Engagement for the firewall."
}

variable "endpoint_id" {
  type        = number
  description = "Endpoint (location) for the firewall."
}

variable "firewall_display_name" {
  type        = string
  description = "Administrative firewall name."
}

variable "firewall_throughput" {
  type        = string
  description = "Throughput (e.g. 50Mbps); must be compatible with api rules."
}

variable "internet_bandwidth" {
  type        = string
  description = "Internet bandwidth (e.g. 50Mbps); typically <= throughput for Bandwidth access."
}

resource "vayucloud_network_firewall" "terraform_firewall" {
  engagement_id           = var.engagement_id
  endpoint_id             = var.endpoint_id
  firewall_display_name   = var.firewall_display_name
  firewall_throughput     = var.firewall_throughput
  internet_bandwidth      = var.internet_bandwidth
}

output "firewall" {
  value = {
    id       = vayucloud_network_firewall.terraform_firewall.id
    audit_id = vayucloud_network_firewall.terraform_firewall.audit_id
    status   = vayucloud_network_firewall.terraform_firewall.status
  }
}
