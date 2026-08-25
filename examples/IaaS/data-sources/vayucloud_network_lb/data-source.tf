# VayuCloud Load Balancer discovery (data sources only — separate state)

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
  description = "Firewall CI Master ID — input for LB list filtering"
  type        = string
}

variable "lb_id" {
  description = "Fallback LB CI id for read-by-id when list-by-firewall finds none"
  type        = string
  default     = null
}

data "vayucloud_network_firewall" "fw" {
  network_firewall_id = var.firewall_id
}

data "vayucloud_network_lb_list" "all" {
  engagement_id = data.vayucloud_network_firewall.fw.engagement_id
  endpoint_id   = data.vayucloud_network_firewall.fw.endpoint_id
}

locals {
  lbs_on_firewall = [
    for lb in data.vayucloud_network_lb_list.all.load_balancers :
    lb if lb.firewall_id == var.firewall_id
  ]

  haproxy_lbs_on_firewall = [
    for lb in local.lbs_on_firewall :
    lb if lower(lb.lb_type) == "haproxy"
  ]

  lb_id_for_read = length(local.lbs_on_firewall) > 0 ? local.lbs_on_firewall[0].id : var.lb_id
}

output "lbs_on_firewall" {
  description = "Load balancers attached to firewall_id"
  value       = local.lbs_on_firewall
}

output "lb_id" {
  description = "First LB on this firewall (use as load_balancer_id in VS resource)"
  value       = length(local.lbs_on_firewall) > 0 ? local.lbs_on_firewall[0].id : null
}

output "haproxy_lbs_on_firewall" {
  description = "HAProxy LBs on this firewall only"
  value       = local.haproxy_lbs_on_firewall
}

data "vayucloud_network_lb" "by_id" {
  count = local.lb_id_for_read != null ? 1 : 0
  id    = local.lb_id_for_read
}

output "lb_by_id" {
  value = length(data.vayucloud_network_lb.by_id) > 0 ? {
    name        = data.vayucloud_network_lb.by_id[0].name
    type        = data.vayucloud_network_lb.by_id[0].type
    firewall_id = data.vayucloud_network_lb.by_id[0].firewall_id
    ci_status   = data.vayucloud_network_lb.by_id[0].ci_status
  } : null
}
