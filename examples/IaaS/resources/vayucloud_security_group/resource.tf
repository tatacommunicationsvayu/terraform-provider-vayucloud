# VayuCloud Security Group Resource Example
#
# Three modes (use exactly one):
#   1) Create a new SG with name (+ optional description), then create rules
#   2) Adopt an existing SG by id, then create rules
#   3) Look up a system SG by type + resource_id (SG_Zone_*, SG_VM_*, SG_TR_*), then create rules

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

variable "firewall_id" {
  type        = number
  description = "Firewall ID that owns the security group."
}

variable "name" {
  type        = string
  description = "Security group name (mode 1: create)."
}

variable "description" {
  type        = string
  description = "Security group description."
  default     = null
  nullable    = true
}

variable "rules" {
  description = "Security group rules to create."
  type = list(object({
    protocol              = string
    ether_type            = string
    direction             = string
    port_range            = optional(number)
    remote_ip_prefix      = optional(list(string))
    remote_security_group = optional(string)
  }))
}

resource "vayucloud_security_group" "created" {
  firewall_id = var.firewall_id
  name        = var.name
  description = var.description

  dynamic "rule" {
    for_each = var.rules
    content {
      protocol              = rule.value.protocol
      ether_type            = rule.value.ether_type
      direction             = rule.value.direction
      port_range            = rule.value.port_range
      remote_ip_prefix      = rule.value.remote_ip_prefix
      remote_security_group = rule.value.remote_security_group
    }
  }
}

# --- Mode 2: adopt existing SG by id (uncomment and set) ---
# resource "vayucloud_security_group" "adopted_by_id" {
#   firewall_id = var.firewall_id
#   id          = var.existing_security_group_id
#
#   dynamic "rule" {
#     for_each = var.rules
#     content {
#       protocol              = rule.value.protocol
#       ether_type            = rule.value.ether_type
#       direction             = rule.value.direction
#       port_range            = rule.value.port_range
#       remote_ip_prefix      = rule.value.remote_ip_prefix
#       remote_security_group = rule.value.remote_security_group
#     }
#   }
# }

# --- Mode 3: look up system SG by type + resource_id (uncomment and set) ---
# resource "vayucloud_security_group" "adopted_by_type" {
#   firewall_id = var.firewall_id
#   type        = var.system_sg_type
#   resource_id = var.system_sg_resource_id
#
#   dynamic "rule" {
#     for_each = var.rules
#     content {
#       protocol              = rule.value.protocol
#       ether_type            = rule.value.ether_type
#       direction             = rule.value.direction
#       port_range            = rule.value.port_range
#       remote_ip_prefix      = rule.value.remote_ip_prefix
#       remote_security_group = rule.value.remote_security_group
#     }
#   }
# }

output "security_group" {
  value = {
    id                     = vayucloud_security_group.created.id
    name                   = vayucloud_security_group.created.name
    managed_security_group = vayucloud_security_group.created.managed_security_group
    rule_ids               = [for r in vayucloud_security_group.created.rule : r.id]
  }
}
