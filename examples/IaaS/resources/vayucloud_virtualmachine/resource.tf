# VayuCloud virtual machine example
#
# Prerequisites: network zone ID in the tenancy. Resolve image_id and flavor_id from the catalog
# using filters (exact, case-insensitive per provider filter semantics).
#
# The resource name vm_with_disks matches examples/IaaS/resources/vayucloud_virtualmachine/import.sh

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

variable "zone_id" {
  type        = number
  description = "Network zone numeric ID where the VM is created."
}

variable "vm_name" {
  type        = string
  description = "Name of the virtual machine (immutable after create)."
  default     = "terraform-example-vm"
}

variable "vm_purpose" {
  type        = string
  description = "Purpose code (e.g. WEB, DB, OTHERS)."
  default     = "WEB"
}

variable "vm_image_name_filter" {
  type        = string
  description = "Image catalog name matched by data.vayucloud_virtualmachine_image filter 'name' (exact, case-insensitive)."
}

variable "vm_flavor_name" {
  type        = string
  description = "Flavor display name matched by data.vayucloud_virtualmachine_flavor filter 'name' (exact, case-insensitive)."
}

variable "vm_flavor_os_model" {
  type        = string
  description = "Flavor OS model matched by filter 'os_model', e.g. ubuntu (exact, case-insensitive)."
}

variable "root_iops" {
  type        = number
  description = "Root disk IOPS (required by API; replaces VM if changed)."
  default     = 1
}

variable "root_disk_size_gb" {
  type        = number
  description = "Root disk size in GB."
  default     = 150
}

data "vayucloud_virtualmachine_image" "os" {
  zone_id = tostring(var.zone_id)

  filter {
    name   = "name"
    values = [var.vm_image_name_filter]
  }
}

data "vayucloud_virtualmachine_flavor" "size" {
  zone_id = tostring(var.zone_id)

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
  name       = var.vm_name
  vm_purpose = var.vm_purpose

  image_id  = data.vayucloud_virtualmachine_image.os.images[0].id
  flavor_id = data.vayucloud_virtualmachine_flavor.size.flavors[0].id
  zone_id   = var.zone_id
  iops      = var.root_iops

  is_kdump_or_page_enabled = "No"

  usage_type    = "ppu"
  pricing_model = "hourly"

  root_disk_size = var.root_disk_size_gb

  # Uncomment to define a layout (partition values must be allowed per schema: /root, /boot, /usr, /swap, /cDrive, /page).
  # root_disk_partitions = [
  #   { partition = "/root", size = 72 },
  #   { partition = "/boot", size = 2 },
  #   { partition = "/usr",  size = 10 },
  #   { partition = "/swap", size = 16 },
  # ]

  additional_disk = [
    { size = 100, iops = 1 },
  ]

  # Uncomment to allocate a public IP at create time.
  # public_ip = {
  #   assign_public_ip               = "yes"
  #   retain_public_ip_on_termination = "no"
  #   public_ip_pricing_model         = "daily"
  # }
}

output "virtual_machine" {
  value = {
    id           = vayucloud_virtualmachine.vm_with_disks.id
    audit_id     = vayucloud_virtualmachine.vm_with_disks.audit_id
    status       = vayucloud_virtualmachine.vm_with_disks.status
    power_status = vayucloud_virtualmachine.vm_with_disks.power_status
  }
}
