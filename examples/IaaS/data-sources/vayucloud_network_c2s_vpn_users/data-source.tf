# VayuCloud Network C2S VPN Users Data Source Example
#
# Lists VPN usernames for an existing C2S VPN on a firewall.

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
  description = "Firewall ID hosting the C2S VPN."
}

data "vayucloud_network_c2s_vpn_users" "vpn_users" {
  firewall_id = var.firewall_id
}

output "user_names" {
  description = "C2S VPN user names"
  value       = [for u in data.vayucloud_network_c2s_vpn_users.vpn_users.users : u.name]
}
