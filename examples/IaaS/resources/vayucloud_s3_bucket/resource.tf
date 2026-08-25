# VayuCloud S3 Bucket Resource Example
#
# Bucket CRUD is synchronous (no audit log / action-state).
# Parent vayucloud_s3_domain must exist and be active first.

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

variable "buckets" {
  description = "S3 buckets to create, keyed by Terraform resource name."
  type = map(object({
    domain_id   = string
    bucket_name = string
  }))
}

resource "vayucloud_s3_bucket" "this" {
  for_each = var.buckets

  domain_id   = each.value.domain_id
  bucket_name = each.value.bucket_name
}

output "s3_buckets" {
  description = "Created S3 bucket attributes."
  value = {
    for key, bucket in vayucloud_s3_bucket.this : key => {
      id                 = bucket.id
      domain_id          = bucket.domain_id
      bucket_name        = bucket.bucket_name
      versioning_enabled = bucket.versioning_enabled
      versioning_status  = bucket.versioning_status
      created_at         = bucket.created_at
      storage_class      = bucket.storage_class
    }
  }
}
