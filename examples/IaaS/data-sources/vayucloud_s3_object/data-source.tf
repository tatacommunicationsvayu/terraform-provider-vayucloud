# VayuCloud S3 Object Data Source Example

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

variable "domain_id" {
  type        = string
  description = "S3 domain ID."
}

variable "bucket_name" {
  type        = string
  description = "Bucket name."
}

variable "key" {
  type        = string
  description = "Object key to read."
}

data "vayucloud_s3_object" "hello" {
  domain_id   = var.domain_id
  bucket_name = var.bucket_name
  key         = var.key
}

output "s3_object" {
  value = {
    id            = data.vayucloud_s3_object.hello.id
    etag          = data.vayucloud_s3_object.hello.etag
    size          = data.vayucloud_s3_object.hello.size
    content_type  = data.vayucloud_s3_object.hello.content_type
    last_modified = data.vayucloud_s3_object.hello.last_modified
  }
}
