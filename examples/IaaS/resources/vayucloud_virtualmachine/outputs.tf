output "virtual_machines" {
  description = "All virtual machines in this example, keyed by resource name"
  value = {
    for key, vm in vayucloud_virtualmachine.this : key => {
      id                       = vm.id
      name                     = vm.name
      vm_purpose               = vm.vm_purpose
      image_id                 = vm.image_id
      flavor_id                = vm.flavor_id
      zone_id                  = vm.zone_id
      iops                     = vm.iops
      is_kdump_or_page_enabled = vm.is_kdump_or_page_enabled
      usage_type               = vm.usage_type
      pricing_model            = vm.pricing_model
      root_disk_size           = vm.root_disk_size
      root_disk_id             = vm.root_disk_id
      root_disk_partitions     = vm.root_disk_partitions
      additional_disk          = vm.additional_disk
      public_ip                = try(vm.public_ip.ip, null)
      public_ip_config         = try(vm.public_ip, null)
      power_status             = vm.power_status
      audit_id                 = vm.audit_id
      status                   = vm.status
    }
  }
}
