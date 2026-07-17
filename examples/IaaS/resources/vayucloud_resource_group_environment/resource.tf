# VayuCloud resource group environment example
#
# Creates an environment under an existing firewall and business unit.

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
  description = "Existing firewall scope for the resource group."
}

variable "business_unit_id" {
  type        = number
  description = "Existing resource_group_business_unit id."
}

variable "environment_name" {
  type        = string
  description = "Display name for the new environment."
}

resource "vayucloud_resource_group_environment" "example" {
  firewall_id      = var.firewall_id
  business_unit_id = var.business_unit_id
  environment      = var.environment_name
}

output "environment" {
  value = {
    id        = vayucloud_resource_group_environment.example.id
    audit_id  = vayucloud_resource_group_environment.example.audit_id
    status    = vayucloud_resource_group_environment.example.status
  }
}
