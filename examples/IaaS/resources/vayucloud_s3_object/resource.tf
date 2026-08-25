# VayuCloud S3 Object Resource Example
#
# Object CRUD is synchronous (no audit log / action-state).
# Parent vayucloud_s3_bucket must exist first.

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

variable "objects" {
  description = "S3 objects to create, keyed by Terraform resource name. Set either source or content."
  type = map(object({
    domain_id    = string
    bucket_name  = string
    key          = string
    source       = optional(string)
    content      = optional(string)
    content_type = optional(string)
  }))
}

resource "vayucloud_s3_object" "this" {
  for_each = var.objects

  domain_id    = each.value.domain_id
  bucket_name  = each.value.bucket_name
  key          = each.value.key
  source       = each.value.source
  content      = each.value.content
  content_type = each.value.content_type
}

output "s3_objects" {
  description = "Uploaded object attributes."
  value = {
    for key, object in vayucloud_s3_object.this : key => {
      id            = object.id
      key           = object.key
      etag          = object.etag
      size          = object.size
      last_modified = object.last_modified
    }
  }
}
