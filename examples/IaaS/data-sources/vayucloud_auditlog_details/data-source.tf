# VayuCloud Audit Log Details Data Source Example
#
# Retrieves detailed information for a single audit log entry.

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
# Example: Audit log detail by audit ID
# =============================================================================

data "vayucloud_auditlog_details" "example" {
  audit_id = "xxxx"
}

output "audit_status" {
  description = "Status of the audit log entry"
  value       = data.vayucloud_auditlog_details.example.status
}

output "audit_action" {
  description = "Action performed"
  value       = data.vayucloud_auditlog_details.example.action
}

output "audit_comments" {
  description = "Comments associated with the audit log"
  value       = data.vayucloud_auditlog_details.example.comments
}

output "audit_output" {
  description = "Output of the audit action"
  value       = data.vayucloud_auditlog_details.example.output
}
