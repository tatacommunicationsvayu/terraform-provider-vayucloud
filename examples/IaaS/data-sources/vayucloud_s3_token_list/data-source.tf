# VayuCloud S3 Token List Data Source Example
#
# Lists tokens for a user (or domain-wide when user_name is omitted).

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
  description = "S3 user name. Omit (null) to list domain-wide tokens."
  default     = null
  nullable    = true
}

data "vayucloud_s3_token_list" "example" {
  domain_id = var.domain_id
  user_name = var.user_name
}

output "s3_tokens" {
  sensitive = true
  value     = data.vayucloud_s3_token_list.example.tokens
}
