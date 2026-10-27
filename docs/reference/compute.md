---
page_title: "Virtual Machines and Compute"
description: |-
  VayuCloud virtual machine resources and related image, flavor, volume, and security-group data sources.
---

# Virtual Machines and Compute

Create a virtual machine in a network zone using an image and flavor discovered from the platform catalog. Add block storage, power actions, or security-group associations as separate concerns.

## Prerequisites and use cases

Have a valid zone, image, and flavor. Use the catalog data sources when names vary by engagement or endpoint. Use the VM list and single-VM data sources to integrate existing workloads.

## Resources

- [Virtual machine resource](../resources/virtualmachine.md)
- [Virtual machine block storage resource](../resources/virtualmachine_blockstorage.md)
- [Virtual machine state resource](../resources/virtualmachine_state.md)
- [Virtual machine security-group association resource](../resources/virtualmachine_security_group_association.md)

## Data sources

- [Virtual machine](../data-sources/virtualmachine.md)
- [Virtual machine list](../data-sources/virtualmachine_list.md)
- [Virtual machine image](../data-sources/virtualmachine_image.md)
- [Virtual machine flavor](../data-sources/virtualmachine_flavor.md)
- [Virtual machine block storage](../data-sources/virtualmachine_blockstorage.md)
- [Virtual machine security-group rules](../data-sources/virtualmachine_list_security_group_rules.md)

## Behavior notes

VM creation, deletion, resizing, and power actions may be asynchronous. Root and additional VM disks use IOPS tiers **1**, **3**, or **5** (IOPS/GB), not absolute IOPS counts. VM custom credentials are create-time secrets and may not be returned by later reads.
