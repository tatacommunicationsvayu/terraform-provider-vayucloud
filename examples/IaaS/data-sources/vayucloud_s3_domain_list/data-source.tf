# VayuCloud S3 Domain List Data Source Example
#
# Lists S3 domains for an IPC engagement and endpoint.

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

variable "endpoint_id" {
  type        = number
  description = "Endpoint ID."
}

variable "firewall_id" {
  type        = number
  description = "Optional firewall ID to include access IPs on list items."
  default     = null
  nullable    = true
}

variable "domain_name_fqdn_filter" {
  type        = list(string)
  description = "Optional domain_name_fqdn filter values. Empty list skips the filtered data source."
  default     = []
}

data "vayucloud_s3_domain_list" "all" {
  engagement_id = var.engagement_id
  endpoint_id   = var.endpoint_id
  firewall_id   = var.firewall_id
}

data "vayucloud_s3_domain_list" "by_name" {
  count         = length(var.domain_name_fqdn_filter) > 0 ? 1 : 0
  engagement_id = var.engagement_id
  endpoint_id   = var.endpoint_id
  firewall_id   = var.firewall_id

  filter {
    name   = "domain_name_fqdn"
    values = var.domain_name_fqdn_filter
  }
}

output "domain_ids" {
  description = "All S3 domain IDs for the engagement and endpoint"
  value       = [for d in data.vayucloud_s3_domain_list.all.domains : d.s3_domain_id]
}

output "filtered_domains" {
  description = "Domains matching the filter"
  value       = length(data.vayucloud_s3_domain_list.by_name) > 0 ? data.vayucloud_s3_domain_list.by_name[0].domains : []
}
