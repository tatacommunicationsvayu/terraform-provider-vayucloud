# VayuCloud S3 Object List Data Source Example

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

variable "prefix" {
  type        = string
  description = "Object key prefix to list."
  default     = null
  nullable    = true
}

data "vayucloud_s3_object_list" "test_prefix" {
  domain_id   = var.domain_id
  bucket_name = var.bucket_name
  prefix      = var.prefix
}

output "s3_objects" {
  value = data.vayucloud_s3_object_list.test_prefix.objects
}
