# VayuCloud C2S VPN resource example
#
# Creates a VPN on an existing firewall; async create with audit polling.

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
  description = "Network firewall resource ID hosting the VPN."
}

variable "c2s_vpn_pricing_model" {
  type        = string
  description = "e.g. daily, monthly, reserved_1 (see docs)."
  default     = "daily"
}

variable "c2s_vpn_users" {
  type = list(object({
    name     = string
    password = string
  }))
  description = "Initial VPN users; supply via tfvars or TF_VAR_* (password complexity per platform)."
  sensitive   = true
}

resource "vayucloud_network_c2s_vpn" "example" {
  firewall_id   = var.firewall_id
  pricing_model = var.c2s_vpn_pricing_model
  users         = var.c2s_vpn_users
}

output "vpn" {
  value = {
    firewall_id      = vayucloud_network_c2s_vpn.example.firewall_id
    id               = vayucloud_network_c2s_vpn.example.id
    vpn_name         = vayucloud_network_c2s_vpn.example.vpn_name
    vpn_ip           = vayucloud_network_c2s_vpn.example.vpn_ip
    vpn_no_of_users  = vayucloud_network_c2s_vpn.example.vpn_no_of_users
    audit_id         = vayucloud_network_c2s_vpn.example.audit_id
  }
}
