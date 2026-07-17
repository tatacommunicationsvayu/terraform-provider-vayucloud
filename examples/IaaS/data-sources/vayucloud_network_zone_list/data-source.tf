# VayuCloud Network Zone List Data Source Example
#
# Lists network zones for an environment.

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

# =============================================================================
# Example 1: List all zones in an environment
# =============================================================================

# =============================================================================
# Example 2: List zones filtered by zone type (or another supported filter)
# =============================================================================

data "vayucloud_network_zone_list" "filtered" {
  environment_id = xxxxx

  filter {
    name   = "zone_type"
    values = ["xxxx"]
  }
}

output "network_zones" {
  description = "Network zones for the environment"
  value       = data.vayucloud_network_zone_list.filtered.network_zones
}
