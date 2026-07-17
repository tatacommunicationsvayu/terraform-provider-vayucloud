# VayuCloud keypair resource example
#
# Creates a keypair in the platform (mode=create generates a PEM private key).

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

variable "keypair_name" {
  type        = string
  description = "Logical name for the new keypair in VayuCloud."
}

variable "engagement_id" {
  type        = string
  description = "Engagement ID owning the keypair."
}

resource "vayucloud_keypair" "created_keypair" {
  name                    = var.keypair_name
  engagement_id           = var.engagement_id
  mode                    = "create"
  keypair_type            = "rsa"
  private_key_file_format = "pem"
}

output "created_keypair_id" {
  value = vayucloud_keypair.created_keypair.id
}

output "created_keypair_status" {
  value = vayucloud_keypair.created_keypair.status
}

output "created_keypair_private_key" {
  value       = vayucloud_keypair.created_keypair.private_key_content
  sensitive   = true
}
