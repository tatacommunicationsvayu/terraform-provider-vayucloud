# VayuCloud Virtual Machine Flavor Data Source Example
#
# Queries the flavor catalog for a zone with optional filters.

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
# Example: Zone catalog with nested filters (names per API-supported fields)
# =============================================================================

data "vayucloud_virtualmachine_flavor" "filtered" {
  zone_id = "xxxx"

  filter {
    name   = "name"
    values = ["xxxx"]
  }

  filter {
    name   = "os_model"
    values = ["xxxx"]
  }
}

output "first_matching_flavor_id" {
  description = "Flavor id of first match (if any)"
  value       = try(data.vayucloud_virtualmachine_flavor.filtered.flavors[0].id, null)
}
