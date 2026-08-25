# VayuCloud S3 User Resource Example
#
# User CRUD is synchronous (no audit log / action-state).
# Parent vayucloud_s3_domain must exist and be active first.
# Not supported on AI_STANDARD (DDN OSS) domains — use root + vayucloud_s3_token instead.

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
  insecure = var.insecure
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

variable "insecure" {
  type        = bool
  description = "Skip TLS certificate verification (UAT only)."
  default     = false
}

variable "users" {
  description = "S3 users to create, keyed by Terraform resource name."
  type = map(object({
    domain_id = string
    user_name = string
  }))
}

resource "vayucloud_s3_user" "this" {
  for_each = var.users

  domain_id = each.value.domain_id
  user_name = each.value.user_name
}

output "s3_users" {
  description = "Created S3 user attributes."
  value = {
    for key, user in vayucloud_s3_user.this : key => {
      id        = user.id
      domain_id = user.domain_id
      user_name = user.user_name
    }
  }
}
