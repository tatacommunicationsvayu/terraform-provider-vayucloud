# VayuCloud Account Engagement Data Source Example
#
# Retrieves account engagements for the authenticated user from the VayuCloud API.

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

# =============================================================================
# Example 1: Retrieve all account engagements for the authenticated user
# =============================================================================

# =============================================================================
# Example 2: Filter by engagement name (substring match)
# =============================================================================
data "vayucloud_account_engagement" "filtered_by_name" {
  filter {
    name   = "engagement_name"
    values = ["xxxx"]
  }
}

output "filtered_by_name" {
  description = "Engagements matching engagement name filter"
  value       = data.vayucloud_account_engagement.filtered_by_name.engagements
}

# =============================================================================
# Example 3: Filter by engagement type
# =============================================================================

data "vayucloud_account_engagement" "filtered_by_type" {
  filter {
    name   = "engagement_type"
    values = ["xxxx"]
  }
}

output "filtered_by_type" {
  description = "Engagements matching engagement type filter"
  value       = data.vayucloud_account_engagement.filtered_by_type.engagements
}

# =============================================================================
# Example 4: Multiple filters (AND logic)
# =============================================================================

data "vayucloud_account_engagement" "multi_filter" {
  filter {
    name   = "id"
    values = ["xxxx"]
  }

  filter {
    name   = "customer_name"
    values = ["xxxx"]
  }
}

output "multi_filter" {
  description = "Engagements matching all filters"
  value       = data.vayucloud_account_engagement.multi_filter.engagements
}
