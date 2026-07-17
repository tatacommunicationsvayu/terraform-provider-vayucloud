# VayuCloud resource group business unit example
#
# Creates a business unit on an existing firewall.

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
  description = "Existing firewall to attach this business unit to."
}

variable "business_unit_name" {
  type        = string
  description = "Business unit label."
}

resource "vayucloud_resource_group_business_unit" "example" {
  firewall_id   = var.firewall_id
  business_unit = var.business_unit_name
}

output "business_unit" {
  value = {
    id       = vayucloud_resource_group_business_unit.example.id
    audit_id = vayucloud_resource_group_business_unit.example.audit_id
    status   = vayucloud_resource_group_business_unit.example.status
  }
}
