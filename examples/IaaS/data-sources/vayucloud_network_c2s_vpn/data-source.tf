# VayuCloud C2S VPN Data Source Example
#
# Reads C2S VPN state for a firewall from the action-state API.

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

data "vayucloud_network_c2s_vpn" "example" {
  firewall_id = xxxxx
}

output "vpn_summary" {
  description = "C2S VPN summary from read state"
  value = {
    vpn_name       = data.vayucloud_network_c2s_vpn.example.vpn_name
    vpn_status     = data.vayucloud_network_c2s_vpn.example.vpn_status
    vpn_ip         = data.vayucloud_network_c2s_vpn.example.vpn_ip
    vpn_no_of_users = data.vayucloud_network_c2s_vpn.example.vpn_no_of_users
  }
}

output "vpn_user_names" {
  description = "VPN user names returned by read (passwords are not exposed)"
  value       = [for u in data.vayucloud_network_c2s_vpn.example.users : u.name]
}

output "peer_id" {
  description = "Peer ID from read state"
  value       = data.vayucloud_network_c2s_vpn.example.peer_id
}

output "pre_shared_key" {
  description = "Pre-shared key from read state"
  sensitive   = true
  value       = data.vayucloud_network_c2s_vpn.example.pre_shared_key
}
