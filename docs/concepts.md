---
page_title: "Core Concepts and Best Practices"
description: |-
  Core VayuCloud provider concepts, dependency ordering, lifecycle behavior, and Terraform/OpenTofu practices.
---

# Core Concepts and Best Practices

Use this section to understand how the VayuCloud provider maps Terraform or OpenTofu configuration to platform resources.

## Resources and data sources

Resources create, update, and destroy infrastructure. Data sources read infrastructure that already exists and expose its attributes to configuration. A data source never creates or changes the object it reads.

Names ending in `_list` return collections. The corresponding singular data source normally reads one object by identifier. List data sources may support a `filter` block; check the linked reference page for the exact filter names.

## Build from the platform dependencies

Most configurations follow this order:

1. Firewall and account context
2. Business unit and environment
3. Network zone
4. Virtual machine, storage, or network service
5. Optional load balancer, VPN, security group, NAS, or S3 resources
6. Public IP associations and firewall rules

Use data sources when an earlier object already exists. Terraform references create implicit dependencies; use `depends_on` only when an API-side dependency is not represented by an attribute reference.

## Asynchronous operations

The provider waits for platform audit completion for long-running operations. Applies that create firewalls, virtual machines, load balancers, NAS resources, or similar services can take several minutes. Do not add shell sleeps or separate polling resources.

Plan pipeline timeouts and change windows accordingly. A failed audit is reported as a Terraform diagnostic and should be investigated before retrying.

### Status fields

`status` usually describes the last provider-tracked audit operation. Resource-specific fields such as `power_status`, `vpn_status`, or `ci_status` describe the platform object's current state. Do not treat these fields as interchangeable or duplicate them in module outputs without naming the distinction.

### Response codes

`response_code` is returned by the underlying API envelope and is not a universal HTTP status field. Some action-state APIs use `0` for success, while account APIs may return an HTTP-style success code such as `200`. Use the resource or data-source page for the relevant endpoint and rely on provider diagnostics for failures.

## Replacement and in-place updates

Each reference page identifies arguments that force replacement and arguments that can be updated in place. Treat identifiers, parent relationships, and platform identity fields as replacement-sensitive unless the page explicitly says otherwise.

For grow-only storage, increasing a size is supported but shrinking is not. For load balancers and virtual services, pin the assigned VIP before attaching a dependent public IP when the platform or provider version requires a stable address.

## Imports and drift

Import existing objects using the identifier documented on the resource page, then run `plan`. Some platform values are not returned by read APIs and must remain in configuration after import. Review the import notes before accepting a plan.

## Terraform and OpenTofu

The provider uses the same resource and data-source names with both tools. Replace `terraform` with `tofu` in command examples when using OpenTofu.

Recommended workflow:

```shell
terraform fmt
terraform init
terraform validate
terraform plan
terraform apply
```

For OpenTofu:

```shell
tofu fmt
tofu init
tofu validate
tofu plan
tofu apply
```

## Configuration practices

- Pin provider versions in release and production configurations.
- Keep environment-specific values in variables or approved variable files.
- Prefer data sources over copied platform identifiers where discovery is available.
- Keep resource names stable; changing a Terraform address can cause replacement.
- Review generated plans for replacement and public-IP dissociation before applying.
- Do not commit credentials, private keys, or state files.
