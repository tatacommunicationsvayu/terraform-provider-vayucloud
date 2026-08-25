# VayuCloud S3 Bucket Data Source Example
#
# Reads one bucket by domain_id + bucket_name.

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
  description = "Bucket name to read."
}

data "vayucloud_s3_bucket" "example" {
  domain_id   = var.domain_id
  bucket_name = var.bucket_name
}

output "s3_bucket" {
  value = {
    id                = data.vayucloud_s3_bucket.example.id
    versioning_status = data.vayucloud_s3_bucket.example.versioning_status
    created_at        = data.vayucloud_s3_bucket.example.created_at
    storage_class     = data.vayucloud_s3_bucket.example.storage_class
  }
}
