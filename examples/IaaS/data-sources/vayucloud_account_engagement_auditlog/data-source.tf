# VayuCloud Account Engagement Audit Log Data Source Example
#
# Retrieves audit logs for an engagement from the VayuCloud API.

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
  description = "Engagement ID whose audit logs to retrieve."
}

variable "request_status" {
  type        = string
  description = "Optional server-side request status filter (for example Completed)."
  default     = null
  nullable    = true
}

variable "resource_category" {
  type        = string
  description = "Optional server-side resource category filter."
  default     = null
  nullable    = true
}

variable "action_filter" {
  type        = list(string)
  description = "Client-side filter values for action."
  default     = []
}

variable "status_filter" {
  type        = list(string)
  description = "Client-side filter values for status."
  default     = []
}

data "vayucloud_account_engagement_auditlog" "by_status" {
  engagement_id     = var.engagement_id
  request_status    = var.request_status
  resource_category = var.resource_category
}

output "completed_audit_logs" {
  description = "Audit logs matching server-side filters"
  value       = data.vayucloud_account_engagement_auditlog.by_status.audit_logs
}

data "vayucloud_account_engagement_auditlog" "filtered" {
  engagement_id = var.engagement_id

  dynamic "filter" {
    for_each = length(var.action_filter) > 0 ? [var.action_filter] : []
    content {
      name   = "action"
      values = filter.value
    }
  }

  dynamic "filter" {
    for_each = length(var.status_filter) > 0 ? [var.status_filter] : []
    content {
      name   = "status"
      values = filter.value
    }
  }
}

output "filtered_audit_logs" {
  description = "Audit logs matching client-side filters"
  value       = data.vayucloud_account_engagement_auditlog.filtered.audit_logs
}
