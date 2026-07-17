# VayuCloud Resource Group Environment List Data Source Example
#
# Lists environments in a business unit.

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
# Example 1: All environments in the business unit
# =============================================================================

# =============================================================================
# Example 2: Filtered environments (adjust filter attributes as needed)
# =============================================================================

data "vayucloud_resource_group_environment_list" "filtered" {
  business_unit_id = xxxxx

  filter {
    name   = "status"
    values = ["xxxx"]
  }
}

output "environments" {
  description = "Environments returned for the business unit"
  value       = data.vayucloud_resource_group_environment_list.filtered.environments
}
