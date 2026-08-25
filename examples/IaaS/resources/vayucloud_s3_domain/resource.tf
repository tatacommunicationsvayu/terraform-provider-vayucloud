# VayuCloud S3 Domain Resource Example
#
# Create is asynchronous: polls audit log until provisioned (~30 min; may take
# longer while firewall mapping completes).
#
# Change domain_name to a unique short name before each create test.
# Only firewall_id is required; engagement_id and endpoint_id are computed from it.

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

variable "domains" {
  description = "S3 domains to create, keyed by Terraform resource name."
  type = map(object({
    domain_name   = string
    quota         = number
    storage_class = string
    variant       = string
    firewall_id   = number
  }))
}

resource "vayucloud_s3_domain" "this" {
  for_each = var.domains

  domain_name   = each.value.domain_name
  quota         = each.value.quota
  storage_class = each.value.storage_class
  variant       = each.value.variant
  firewall_id   = each.value.firewall_id
}

output "s3_domains" {
  description = "Core identity and connectivity details for the provisioned S3 domains."
  value = {
    for key, domain in vayucloud_s3_domain.this : key => merge(
      {
        id               = domain.id
        audit_id         = domain.audit_id
        engagement_id    = domain.engagement_id
        endpoint_id      = domain.endpoint_id
        firewall_id      = domain.firewall_id
        domain_name_fqdn = domain.domain_name_fqdn
        domain_access_ip = domain.domain_access_ip
        quota            = domain.quota
        quota_unit       = domain.quota_unit
        storage_class    = domain.storage_class
        variant          = domain.variant
      },
      domain.domain_access_public_ip != null ? {
        domain_access_public_ip = domain.domain_access_public_ip
      } : {}
    )
  }
}
