# VayuCloud C2S VPN users resource example
#
# Manages VPN users where the VPN already exists on the firewall.

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
  description = "Firewall where the VPN is already provisioned."
}

variable "c2s_vpn_managed_users" {
  type = list(object({
    name     = string
    password = string
  }))
  description = "User list via tfvars or TF_VAR_* (password complexity per platform)."
  sensitive   = true
}

resource "vayucloud_network_c2s_vpn_user" "managed" {
  firewall_id = var.firewall_id
  users       = var.c2s_vpn_managed_users
}

output "vpn_user_firewall_id" {
  value = vayucloud_network_c2s_vpn_user.managed.firewall_id
}
