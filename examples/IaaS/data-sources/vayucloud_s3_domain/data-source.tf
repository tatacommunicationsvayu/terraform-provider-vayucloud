# VayuCloud S3 Domain Data Source Example
#
# Reads one S3 domain by resource ID (action-state module=domain, action=read).

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

variable "engagement_id" {
  type        = number
  description = "Engagement ID."
}

variable "s3_domain_id" {
  type        = string
  description = "S3 domain resource ID."
}

variable "firewall_id" {
  type        = number
  description = "Firewall ID required on S3 domain read."
}

data "vayucloud_s3_domain" "example" {
  s3_domain_id  = var.s3_domain_id
  engagement_id = var.engagement_id
  firewall_id   = var.firewall_id
}

output "s3_domain" {
  description = "Key attributes of the S3 domain"
  value = {
    s3_domain_id            = data.vayucloud_s3_domain.example.s3_domain_id
    domain_name_fqdn        = data.vayucloud_s3_domain.example.domain_name_fqdn
    quota                   = data.vayucloud_s3_domain.example.quota
    quota_unit              = data.vayucloud_s3_domain.example.quota_unit
    storage_class           = data.vayucloud_s3_domain.example.storage_class
    variant                 = data.vayucloud_s3_domain.example.variant
    engagement_id           = data.vayucloud_s3_domain.example.engagement_id
    endpoint_id             = data.vayucloud_s3_domain.example.endpoint_id
    firewall_id             = data.vayucloud_s3_domain.example.firewall_id
    domain_access_ip        = data.vayucloud_s3_domain.example.domain_access_ip
    domain_access_public_ip = data.vayucloud_s3_domain.example.domain_access_public_ip
  }
}
