# VayuCloud Resource Group Environment Data Source Example
#
# Reads one environment by resource ID.

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

variable "resource_group_environment_id" {
  type        = string
  description = "Environment resource ID."
}

data "vayucloud_resource_group_environment" "example" {
  resource_group_environment_id = var.resource_group_environment_id
}

output "environment_summary" {
  description = "Environment fields from read"
  value = {
    environment      = data.vayucloud_resource_group_environment.example.environment
    business_unit_id = data.vayucloud_resource_group_environment.example.business_unit_id
    status           = data.vayucloud_resource_group_environment.example.status
  }
}
