# VayuCloud virtual machine example
#
# Prerequisites: network zone ID in the tenancy. Resolve image_id and flavor_id from the catalog
# using filters (exact, case-insensitive per provider filter semantics).
#
# Supply virtual_machines via terraform.tfvars. Keys become resource addresses:
#   vayucloud_virtualmachine.this["<key>"]
#
# Import an existing VM with:
#   INSTANCE_ID=<id> terraform import 'vayucloud_virtualmachine.this["<key>"]' "$INSTANCE_ID"

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

variable "virtual_machines" {
  description = "Virtual machines to create, keyed by Terraform resource name."
  type = map(object({
    name                     = string
    vm_purpose               = string
    image_id                 = number
    flavor_id                = number
    zone_id                  = number
    iops                     = number
    is_kdump_or_page_enabled = string
    usage_type               = string
    pricing_model            = string
    root_disk_size           = number
    additional_disk = optional(list(object({
      size = number
      iops = number
    })), [])
    public_ip = optional(object({
      assign_public_ip                = string
      retain_public_ip_on_termination = string
      public_ip_pricing_model         = string
    }))
  }))
}

resource "vayucloud_virtualmachine" "this" {
  for_each = var.virtual_machines

  name                     = each.value.name
  vm_purpose               = each.value.vm_purpose
  image_id                 = each.value.image_id
  flavor_id                = each.value.flavor_id
  zone_id                  = each.value.zone_id
  iops                     = each.value.iops
  is_kdump_or_page_enabled = each.value.is_kdump_or_page_enabled
  usage_type               = each.value.usage_type
  pricing_model            = each.value.pricing_model
  root_disk_size           = each.value.root_disk_size
  additional_disk          = each.value.additional_disk
  public_ip                = each.value.public_ip
}
