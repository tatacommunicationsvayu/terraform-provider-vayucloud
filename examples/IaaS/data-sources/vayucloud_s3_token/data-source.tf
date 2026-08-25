# VayuCloud S3 Token Data Source Example
#
# Reads one token by domain_id + user_name + access_key_id (metadata only; no secret).

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
  description = "S3 user name."
}

variable "access_key_id" {
  type        = string
  description = "Access key ID of the token to read."
  sensitive   = true
}

data "vayucloud_s3_token" "example" {
  domain_id     = var.domain_id
  user_name     = var.user_name
  access_key_id = var.access_key_id
}

output "s3_token" {
  sensitive = true
  value = {
    id                = data.vayucloud_s3_token.example.id
    access_key_id     = data.vayucloud_s3_token.example.access_key_id
    token_expiry_date = data.vayucloud_s3_token.example.token_expiry_date
  }
}
