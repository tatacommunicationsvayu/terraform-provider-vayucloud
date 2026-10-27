---
page_title: "Security Groups"
description: |-
  VayuCloud security-group resources and data sources for firewall and virtual-machine access control.
---

# Security Groups

Security groups provide workload-level rules in addition to firewall rules. Create or adopt a group, define rules, and associate it with a virtual machine.

## Prerequisites and use cases

Know the firewall and VM identifiers. Use adoption modes when a platform security group already exists; use the list data sources to inspect existing groups and rules.

## Resources

- [Security group resource](../resources/security_group.md)
- [Virtual machine security-group association](../resources/virtualmachine_security_group_association.md)

## Data sources

- [Security groups](../data-sources/list_security_group.md)
- [Security-group rules](../data-sources/list_security_group_rules.md)
- [VM security-group rules](../data-sources/virtualmachine_list_security_group_rules.md)

## Behavior notes

Group creation, rule changes, association changes, and deletion may be asynchronous. Review adoption and detach behavior before importing or destroying a group used by other workloads.
