# VayuCloud Resource Group Business Unit List Data Source Example
#
# Lists business units for a firewall.

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
# Example 1: All business units for a firewall
# =============================================================================

# =============================================================================
# Example 2: Same list with optional name filter
# =============================================================================

data "vayucloud_resource_group_business_unit_list" "filtered" {
  firewall_id = xxxxx

  filter {
    name   = "business_unit"
    values = ["xxxx"]
  }
}

output "business_units" {
  description = "Business units returned for the firewall"
  value       = data.vayucloud_resource_group_business_unit_list.filtered.business_units
}
