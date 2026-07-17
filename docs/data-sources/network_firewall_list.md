---
page_title: "vayucloud_network_firewall_list Data Source"
subcategory: ""
description: |-
  Lists network firewalls for an engagement and endpoint. Optional filters narrow results client-side.
---

# `vayucloud_network_firewall_list`

Returns all firewalls for a given `engagement_id` and `endpoint_id`. Use [`vayucloud_network_firewall`](network_firewall.md) when you already have a single firewall ID.

## Example Usage

```hcl
data "vayucloud_network_firewall_list" "all" {
  engagement_id = {{engagement_id}}
  endpoint_id   = {{endpoint_id}}
}

output "firewall_ids" {
  value = [for fw in data.vayucloud_network_firewall_list.all.firewalls : fw.network_firewall_id]
}
```

### Filter by hypervisor

```hcl
data "vayucloud_network_firewall_list" "kvm" {
  engagement_id = {{engagement_id}}
  endpoint_id   = {{endpoint_id}}

  filter {
    name   = "hypervisor"
    values = [{{hypervisor}}]
  }
}
```

## Argument Reference

### Required

* `engagement_id` — (Number) Engagement identifier.
* `endpoint_id` — (Number) Endpoint identifier.

### Optional

* `filter` — (Block) Repeatable. Each block requires `name` and `values` (list of strings). Applied after the list is returned.

## Attributes Reference

* `firewalls` — (List of Object) One element per firewall. Typical fields include `network_firewall_id`, `firewall_display_name`, throughput and bandwidth strings, `hypervisor`, and pricing fields as returned by the API.
* `status` — (String) API status.
* `message` — (String) API message.
* `response_code` — (Number) API response code.

Resolve `engagement_id` and `endpoint_id` from [`vayucloud_account_engagement`](account_engagement.md) and [`vayucloud_account_location`](account_location.md) when you do not hard-code values.
