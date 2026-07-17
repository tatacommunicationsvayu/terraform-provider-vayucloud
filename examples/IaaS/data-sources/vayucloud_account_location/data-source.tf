# VayuCloud Account Location Data Source Example
#
# This example demonstrates how to retrieve the list of locations (endpoints)
# for an engagement from the VayuCloud API.

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
# Example 1: Retrieve all locations for an engagement
# =============================================================================

# =============================================================================
# Example 2: Filter by endpoint display name (substring match)
# =============================================================================
data "vayucloud_account_location" "mumbai" {
  engagement_id = xxxxx

  filter {
    name   = "endpoint_display_name"
    values = ["xxxx"]
  }
}

output "mumbai_locations" {
  description = "Locations matching endpoint display name filter"
  value       = data.vayucloud_account_location.mumbai.locations
}

# =============================================================================
# Example 3: Filter by endpoint ID
# =============================================================================

data "vayucloud_account_location" "by_id" {
  engagement_id = xxxxx

  filter {
    name   = "endpoint_id"
    values = ["xxxx"]
  }
}

output "filtered_by_id" {
  description = "Locations matching endpoint ID filter"
  value       = data.vayucloud_account_location.by_id.locations
}
