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

variable "engagement_id" {
  type        = number
  description = "Engagement ID whose locations to retrieve."
}

variable "endpoint_display_name_filter" {
  type        = list(string)
  description = "Filter values for endpoint_display_name."
}

variable "endpoint_id_filter" {
  type        = list(string)
  description = "Filter values for endpoint_id."
}

data "vayucloud_account_location" "by_display_name" {
  engagement_id = var.engagement_id

  filter {
    name   = "endpoint_display_name"
    values = var.endpoint_display_name_filter
  }
}

output "locations_by_display_name" {
  description = "Locations matching endpoint display name filter"
  value       = data.vayucloud_account_location.by_display_name.locations
}

data "vayucloud_account_location" "by_id" {
  engagement_id = var.engagement_id

  filter {
    name   = "endpoint_id"
    values = var.endpoint_id_filter
  }
}

output "filtered_by_id" {
  description = "Locations matching endpoint ID filter"
  value       = data.vayucloud_account_location.by_id.locations
}
