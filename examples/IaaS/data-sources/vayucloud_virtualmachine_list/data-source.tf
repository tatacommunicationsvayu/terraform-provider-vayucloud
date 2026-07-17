# VayuCloud Virtual Machine List Data Source Example
#
# Lists virtual machines in a zone with optional filters.

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
# Example 1: All instances in the zone
# =============================================================================

# =============================================================================
# Example 2: Instances with optional nested filters (field names supported by provider)
# =============================================================================

data "vayucloud_virtualmachine_list" "filtered" {
  zone_id = xxxxx

  filter {
    name   = "power_status"
    values = ["xxxx"]
  }

  filter {
    name   = "os_type"
    values = ["xxxx"]
  }
}

output "virtual_machines" {
  description = "Virtual machines returned for the zone"
  value       = data.vayucloud_virtualmachine_list.filtered.virtual_machines
}
