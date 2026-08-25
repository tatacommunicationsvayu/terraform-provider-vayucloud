# VayuCloud Network Load Balancer Resource Example
#
# Enable is asynchronous: the provider polls the audit log until completion.
# First-time enable typically takes 20–30 minutes on the platform.

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
  type        = string
  description = "Firewall CI Master ID on which to enable HAProxy."
}

variable "zone_id" {
  type        = number
  description = "Network zone ID on the firewall (required for VS and pool members)."
}

variable "bandwidth" {
  type        = string
  description = "Load balancer bandwidth in Mbps (for example 100 or 100Mbps)."
  default     = "100"
}

variable "pricing_model" {
  type        = string
  description = "LB pricing model (for example daily, monthly)."
  default     = "daily"
}

resource "vayucloud_network_lb" "lb_haproxy" {
  firewall_id   = var.firewall_id
  zone_id       = var.zone_id
  bandwidth     = var.bandwidth
  pricing_model = var.pricing_model
}

output "lb" {
  description = "Load balancer attributes — use id in VS / SSL examples"
  value = {
    id         = vayucloud_network_lb.lb_haproxy.id
    name       = vayucloud_network_lb.lb_haproxy.name
    type       = vayucloud_network_lb.lb_haproxy.type
    zone_id    = vayucloud_network_lb.lb_haproxy.zone_id
    bandwidth  = vayucloud_network_lb.lb_haproxy.bandwidth
    ci_status  = vayucloud_network_lb.lb_haproxy.ci_status
    audit_id   = vayucloud_network_lb.lb_haproxy.audit_id
    status     = vayucloud_network_lb.lb_haproxy.status
  }
}
