# VayuCloud S3 User Data Source Example
#
# Reads one user by domain_id + user_name.

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

variable "user_name" {
  type        = string
  description = "S3 user name to read."
}

data "vayucloud_s3_user" "example" {
  domain_id = var.domain_id
  user_name = var.user_name
}

output "s3_user" {
  value = {
    id        = data.vayucloud_s3_user.example.id
    domain_id = data.vayucloud_s3_user.example.domain_id
    user_name = data.vayucloud_s3_user.example.user_name
  }
}
