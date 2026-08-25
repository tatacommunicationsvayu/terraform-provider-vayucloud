# VayuCloud Account Engagement Data Source Example
#
# Retrieves account engagements for the authenticated user from the VayuCloud API.

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

variable "engagement_name_filter" {
  type        = list(string)
  description = "Filter values for engagement_name."
}

variable "engagement_type_filter" {
  type        = list(string)
  description = "Filter values for engagement_type."
}

variable "engagement_id_filter" {
  type        = list(string)
  description = "Filter values for engagement id."
}

variable "customer_name_filter" {
  type        = list(string)
  description = "Filter values for customer_name."
}

data "vayucloud_account_engagement" "filtered_by_name" {
  filter {
    name   = "engagement_name"
    values = var.engagement_name_filter
  }
}

output "filtered_by_name" {
  description = "Engagements matching engagement name filter"
  value       = data.vayucloud_account_engagement.filtered_by_name.engagements
}

data "vayucloud_account_engagement" "filtered_by_type" {
  filter {
    name   = "engagement_type"
    values = var.engagement_type_filter
  }
}

output "filtered_by_type" {
  description = "Engagements matching engagement type filter"
  value       = data.vayucloud_account_engagement.filtered_by_type.engagements
}

data "vayucloud_account_engagement" "multi_filter" {
  filter {
    name   = "id"
    values = var.engagement_id_filter
  }

  filter {
    name   = "customer_name"
    values = var.customer_name_filter
  }
}

output "multi_filter" {
  description = "Engagements matching all filters"
  value       = data.vayucloud_account_engagement.multi_filter.engagements
}
