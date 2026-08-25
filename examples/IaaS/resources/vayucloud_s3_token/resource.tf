# VayuCloud S3 Token Resource Example
#
# Token CRUD is synchronous. Parent vayucloud_s3_user must exist first
# (except AI_STANDARD domains where user_name = "root" is pre-provisioned).
#
# secret_access_key is only available at create time.

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

variable "tokens" {
  description = "S3 tokens to create, keyed by Terraform resource name."
  type = map(object({
    domain_id         = string
    user_name         = string
    token_expiry_date = string
    token_description = optional(string)
  }))
}

resource "vayucloud_s3_token" "this" {
  for_each = var.tokens

  domain_id         = each.value.domain_id
  user_name         = each.value.user_name
  token_expiry_date = each.value.token_expiry_date
  token_description = each.value.token_description
}

output "s3_tokens" {
  description = "Created S3 token attributes (secret is sensitive)."
  sensitive   = true
  value = {
    for key, token in vayucloud_s3_token.this : key => {
      id                = token.id
      domain_id         = token.domain_id
      user_name         = token.user_name
      access_key_id     = token.access_key_id
      token_expiry_date = token.token_expiry_date
    }
  }
}
