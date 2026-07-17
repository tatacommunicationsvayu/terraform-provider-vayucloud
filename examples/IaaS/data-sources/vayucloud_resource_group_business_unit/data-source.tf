# VayuCloud Resource Group Business Unit Data Source Example
#
# Reads one business unit by resource ID.

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

data "vayucloud_resource_group_business_unit" "example" {
  resource_group_business_unit_id = "xxxx"
}

output "business_unit_summary" {
  description = "Business unit fields from read"
  value = {
    business_unit = data.vayucloud_resource_group_business_unit.example.business_unit
    users        = data.vayucloud_resource_group_business_unit.example.users
    status       = data.vayucloud_resource_group_business_unit.example.status
  }
}
