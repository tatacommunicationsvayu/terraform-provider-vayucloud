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

# =============================================================================
# Example 1: Retrieve all audit logs for an engagement
# =============================================================================

# =============================================================================
# Example 2: Filter audit logs using server-side request body filters
# =============================================================================
data "vayucloud_account_engagement_auditlog" "by_status" {
  engagement_id     = xxxxx
  request_status    = "Completed"
  resource_category = "xxxx"
}

output "completed_audit_logs" {
  description = "Audit logs matching server-side filters"
  value       = data.vayucloud_account_engagement_auditlog.by_status.audit_logs
}

# =============================================================================
# Example 3: Use client-side filter blocks for additional filtering
# =============================================================================

data "vayucloud_account_engagement_auditlog" "filtered" {
  engagement_id = xxxxx

  filter {
    name   = "action"
    values = ["xxxx"]
  }

  filter {
    name   = "status"
    values = ["xxxx"]
  }
}

output "filtered_audit_logs" {
  description = "Audit logs matching client-side filters"
  value       = data.vayucloud_account_engagement_auditlog.filtered.audit_logs
}
