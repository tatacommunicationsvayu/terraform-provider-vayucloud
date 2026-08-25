# VayuCloud Network LB Virtual Service Resource Example
#
# Prerequisites: enabled load balancer (with zone_id), backend VMs in that zone.
# For HTTPS: upload vayucloud_network_lb_ssl_profile first.

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

variable "load_balancer_id" {
  type        = string
  description = "Load Balancer CI Master ID."
}

variable "zone_id" {
  type        = number
  description = "Network zone ID — use vayucloud_network_lb.<name>.zone_id."
}

variable "vs_name" {
  type        = string
  description = "Virtual service name (unique per load balancer)."
  default     = "app-vs-http"
}

variable "vs_port" {
  type        = string
  description = "Listener port."
  default     = "80"
}

variable "vs_protocol" {
  type        = string
  description = "http, https, or tcp."
  default     = "http"
}

variable "pool_algorithm" {
  type        = string
  description = "Pool algorithm (for example roundrobin, leastconn)."
  default     = "roundrobin"
}

variable "certificate_name" {
  type        = string
  description = "SSL storage name on the LB when vs_protocol is https (for example my-cert.pem)."
  default     = ""
}

variable "vip_ip" {
  type        = string
  description = "Private VIP. Omit on first create; pin after terraform output vip_ip."
  default     = null
}

variable "pool_members" {
  description = "Pool members — resolve IPs from vayucloud_virtualmachine_list in zone_id."
  type = list(object({
    name       = string
    ip_address = string
    port       = number
  }))
}

locals {
  monitor_type = lower(var.vs_protocol) == "tcp" ? "tcp" : "http"
}

resource "vayucloud_network_lb_virtualservice" "app_vs" {
  load_balancer_id = var.load_balancer_id
  name             = var.vs_name
  zone_id          = var.zone_id
  port             = var.vs_port
  protocol         = var.vs_protocol
  pool_algorithm   = var.pool_algorithm
  monitor          = [local.monitor_type == "tcp" ? "tcp-check" : "httpchk"]
  vip_ip           = var.vip_ip
  certificate_name = lower(var.vs_protocol) == "https" ? (
    endswith(var.certificate_name, ".pem") ? var.certificate_name : "${var.certificate_name}.pem"
  ) : null

  dynamic "pool_member" {
    for_each = var.pool_members
    content {
      name       = pool_member.value.name
      ip_address = pool_member.value.ip_address
      port       = pool_member.value.port
    }
  }
}

output "virtual_service" {
  value = {
    id         = vayucloud_network_lb_virtualservice.app_vs.id
    name       = vayucloud_network_lb_virtualservice.app_vs.name
    vip_ip     = vayucloud_network_lb_virtualservice.app_vs.vip_ip
    protocol   = vayucloud_network_lb_virtualservice.app_vs.protocol
    port       = vayucloud_network_lb_virtualservice.app_vs.port
    ci_status  = vayucloud_network_lb_virtualservice.app_vs.ci_status
    audit_id   = vayucloud_network_lb_virtualservice.app_vs.audit_id
  }
}
