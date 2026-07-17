# VayuCloud Virtual Machine Image Data Source Example
#
# Queries the VM image/template catalog for a zone.

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
# Example 1: All images/templates for the zone (no nested filters)
# =============================================================================

# =============================================================================
# Example 2: Narrow with client-side filters
# =============================================================================

data "vayucloud_virtualmachine_image" "filtered" {
  zone_id = "xxxx"

  filter {
    name   = "name"
    values = ["xxxx"]
  }
}

output "images" {
  description = "Images returned for the zone and filters"
  value       = data.vayucloud_virtualmachine_image.filtered.images
}

output "first_image_id" {
  description = "ID of first image when present"
  value       = try(data.vayucloud_virtualmachine_image.filtered.images[0].id, null)
}
