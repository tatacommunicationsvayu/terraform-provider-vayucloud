# VayuCloud end-to-end IaaS example
#
# Chains firewall, resource groups, zone, VM image/flavor lookups, VMs, and a power action.

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

variable "engagement_id" {
  type        = number
  description = "Engagement ID for the firewall (your tenant)."
}

variable "endpoint_id" {
  type        = number
  description = "Endpoint (location) ID for the firewall."
}

variable "firewall_display_name" {
  type        = string
  description = "Display name for the network firewall."
}

variable "firewall_throughput" {
  type        = string
  description = "Firewall throughput (e.g. 15Mbps)."
}

variable "internet_bandwidth" {
  type        = string
  description = "Internet bandwidth (e.g. 15Mbps)."
}

variable "business_unit_name" {
  type        = string
  description = "Business unit name created under the firewall."
}

variable "environment_name" {
  type        = string
  description = "Environment name created under the business unit."
}

variable "network_zone_name" {
  type        = string
  description = "Network zone name created in the environment."
}

variable "network_zone_no_of_ips" {
  type        = number
  description = "IPv4 allocation size for the zone."
  default     = 20
}

variable "vm_image_catalog_name_filter" {
  type        = string
  description = "Exact image name substring for vayucloud_virtualmachine_image filter 'name' (catalog-specific)."
}

variable "vm_flavor_name" {
  type        = string
  description = "Flavor catalog name filter (e.g. B8)."
}

variable "vm_flavor_os_model" {
  type        = string
  description = "Flavor catalog os_model filter (e.g. windows, Rocky Linux)."
}

variable "vm_name_primary" {
  type        = string
  description = "Name for the first example virtual machine."
}

variable "vm_name_secondary" {
  type        = string
  description = "Name for the second example virtual machine."
}

variable "vm_purpose" {
  type        = string
  description = "vm_purpose for both VMs (e.g. WEB)."
  default     = "WEB"
}

variable "power_action" {
  type        = string
  description = "Power action applied to vm_primary via vayucloud_virtualmachine_state (e.g. power_off)."
  default     = "power_off"
}

# =============================================================================
# Create a Firewall
# =============================================================================

resource "vayucloud_network_firewall" "terraform_firewall" {
  engagement_id          = var.engagement_id
  endpoint_id            = var.endpoint_id
  firewall_display_name  = var.firewall_display_name
  firewall_throughput    = var.firewall_throughput
  internet_bandwidth     = var.internet_bandwidth
}

resource "vayucloud_resource_group_business_unit" "terraform_bu" {
  firewall_id   = vayucloud_network_firewall.terraform_firewall.id
  business_unit = var.business_unit_name
}

resource "vayucloud_resource_group_environment" "production_2" {
  firewall_id      = vayucloud_network_firewall.terraform_firewall.id
  environment      = var.environment_name
  business_unit_id = vayucloud_resource_group_business_unit.terraform_bu.id
}

resource "vayucloud_network_zone" "auto_ipam_zone" {
  name           = var.network_zone_name
  environment_id = vayucloud_resource_group_environment.production_2.id
  firewall_id    = vayucloud_network_firewall.terraform_firewall.id
  no_of_ips      = var.network_zone_no_of_ips
}

data "vayucloud_virtualmachine_image" "standard_images" {
  zone_id = vayucloud_network_zone.auto_ipam_zone.id

  filter {
    name   = "name"
    values = [var.vm_image_catalog_name_filter]
  }
}

data "vayucloud_virtualmachine_flavor" "selected_flavor" {
  zone_id = vayucloud_network_zone.auto_ipam_zone.id

  filter {
    name   = "name"
    values = [var.vm_flavor_name]
  }

  filter {
    name   = "os_model"
    values = [var.vm_flavor_os_model]
  }
}

resource "vayucloud_virtualmachine" "vm_with_disks" {
  name                     = var.vm_name_primary
  vm_purpose               = var.vm_purpose
  image_id                 = data.vayucloud_virtualmachine_image.standard_images.images[0].id
  flavor_id                = data.vayucloud_virtualmachine_flavor.selected_flavor.flavors[0].id
  zone_id                  = vayucloud_network_zone.auto_ipam_zone.id
  iops                     = 1
  is_kdump_or_page_enabled = "No"
  usage_type               = "ppu"
  pricing_model            = "hourly"

  root_disk_partitions = [
    { partition = "/root", size = 47 },
    { partition = "/boot", size = 2 },
  ]

  additional_disk = [
    { size = 100, iops = 1 },
  ]
}

resource "vayucloud_virtualmachine" "vm_with_disks_2" {
  name                     = var.vm_name_secondary
  vm_purpose               = var.vm_purpose
  image_id                 = data.vayucloud_virtualmachine_image.standard_images.images[0].id
  flavor_id                = data.vayucloud_virtualmachine_flavor.selected_flavor.flavors[0].id
  zone_id                  = vayucloud_network_zone.auto_ipam_zone.id
  iops                     = 1
  is_kdump_or_page_enabled = "No"
  usage_type               = "ppu"
  pricing_model            = "hourly"
  root_disk_size           = 100

  root_disk_partitions = [
    { partition = "/root", size = 97 },
    { partition = "/boot", size = 2 },
  ]

  additional_disk = [
    { size = 100, iops = 1 },
  ]
}

resource "vayucloud_virtualmachine_state" "example" {
  instance_id = vayucloud_virtualmachine.vm_with_disks.id
  action      = var.power_action
}

# =============================================================================
# Outputs
# =============================================================================

output "firewall" {
  value = {
    id        = vayucloud_network_firewall.terraform_firewall.id
    audit_id  = vayucloud_network_firewall.terraform_firewall.audit_id
    status    = vayucloud_network_firewall.terraform_firewall.status
  }
}

output "business_unit" {
  value = {
    id     = vayucloud_resource_group_business_unit.terraform_bu.id
    status = vayucloud_resource_group_business_unit.terraform_bu.status
  }
}

output "environment" {
  value = {
    id     = vayucloud_resource_group_environment.production_2.id
    status = vayucloud_resource_group_environment.production_2.status
  }
}

output "network_zone" {
  value = {
    id     = vayucloud_network_zone.auto_ipam_zone.id
    status = vayucloud_network_zone.auto_ipam_zone.status
  }
}

output "selected_image" {
  value = {
    id   = data.vayucloud_virtualmachine_image.standard_images.images[0].id
    name = data.vayucloud_virtualmachine_image.standard_images.images[0].name
  }
}

output "selected_flavor" {
  value = {
    id   = data.vayucloud_virtualmachine_flavor.selected_flavor.flavors[0].id
    name = data.vayucloud_virtualmachine_flavor.selected_flavor.flavors[0].name
    vcpu = data.vayucloud_virtualmachine_flavor.selected_flavor.flavors[0].vcpu
    vram = data.vayucloud_virtualmachine_flavor.selected_flavor.flavors[0].vram
  }
}

output "vm_primary" {
  value = {
    id               = vayucloud_virtualmachine.vm_with_disks.id
    audit_id         = vayucloud_virtualmachine.vm_with_disks.audit_id
    status           = vayucloud_virtualmachine.vm_with_disks.status
    root_disk_id     = vayucloud_virtualmachine.vm_with_disks.root_disk_id
    root_disk_size   = vayucloud_virtualmachine.vm_with_disks.root_disk_size
    power_status     = vayucloud_virtualmachine.vm_with_disks.power_status
    power_audit_id   = vayucloud_virtualmachine_state.example.audit_id
    power_action_status = vayucloud_virtualmachine_state.example.status
  }
}

output "vm_secondary" {
  value = {
    id       = vayucloud_virtualmachine.vm_with_disks_2.id
    audit_id = vayucloud_virtualmachine.vm_with_disks_2.audit_id
    status   = vayucloud_virtualmachine.vm_with_disks_2.status
  }
}
