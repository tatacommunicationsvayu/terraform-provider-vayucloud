# VayuCloud Keypair Data Source Example
#
# Lists keypairs for an engagement.

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

data "vayucloud_keypair" "all" {
  engagement_id = "xxxx"
}

output "keypairs" {
  description = "Keypairs for the engagement"
  value       = data.vayucloud_keypair.all.keypairs
}
