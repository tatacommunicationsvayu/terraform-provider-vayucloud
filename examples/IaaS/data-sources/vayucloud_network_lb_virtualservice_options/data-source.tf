# VayuCloud Virtual Service discovery (data sources only — separate state)

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
  description = "Load Balancer CI Master ID"
  type        = string
}

variable "monitor_type" {
  description = "Monitor type used when querying VS options (for example http or tcp)."
  type        = string
}

variable "zone_name" {
  description = "Optional zone name filter for vs_options and pool VM discovery"
  type        = string
  default     = ""
}

data "vayucloud_network_lb_virtualservice_options" "all_options" {
  load_balancer_id = var.load_balancer_id
  monitor_type     = var.monitor_type
}

output "protocols" {
  value = data.vayucloud_network_lb_virtualservice_options.all_options.protocols
}

output "algorithms" {
  value = data.vayucloud_network_lb_virtualservice_options.all_options.algorithms
}

output "monitors" {
  value = data.vayucloud_network_lb_virtualservice_options.all_options.monitors
}

output "zones" {
  description = "Zones available for virtual service placement"
  value       = data.vayucloud_network_lb_virtualservice_options.all_options.zones
}

locals {
  zone_selected = trimspace(var.zone_name) != ""
}

data "vayucloud_network_lb_virtualservice_options" "zone_options" {
  count            = local.zone_selected ? 1 : 0
  load_balancer_id = var.load_balancer_id
  monitor_type     = var.monitor_type

  filter {
    name   = "name"
    values = [var.zone_name]
  }
}

data "vayucloud_virtualmachine_list" "pool_vms" {
  count   = local.zone_selected ? 1 : 0
  zone_id = data.vayucloud_network_lb_virtualservice_options.zone_options[0].zones[0].id
}

output "selected_zone" {
  value = local.zone_selected ? data.vayucloud_network_lb_virtualservice_options.zone_options[0].zones[0] : null
}

output "pool_vms" {
  value = local.zone_selected ? data.vayucloud_virtualmachine_list.pool_vms[0].virtual_machines : []
}
