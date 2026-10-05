---
page_title: "S3 Operations Guide"
subcategory: "Guides"
description: |-
  End-to-end guide for VayuCloud S3 (VCS): domain on a firewall, buckets, users, access tokens, and optional public access via public IP and firewall rules.
---

# S3 Operations Guide

This guide walks through **S3 object storage (VCS)** in the VayuCloud provider: create a domain on a firewall, add buckets and service users, issue access tokens, and optionally expose the domain with a **public IP** and **`ill_to_vcs`** firewall rule. Upload and manage object data with an S3-compatible client using credentials from `vayucloud_s3_token`; use data sources to read object metadata in Terraform.

For per-attribute documentation, open the linked resource and data source pages. For perimeter and VM context, see [Firewall to VM guide](firewall_to_vm.md). For the full dependency chain, see [Resources and data sources overview](../resourcesAndDatasource.md).

## Conventions

| Kind | Behavior |
|------|----------|
| **Resources** | Domain create/delete is async (~30 minutes typical). Buckets, users, and tokens are synchronous. |
| **Data sources** | Read-only discovery (domains, buckets, users, tokens, object metadata). |
| **`firewall_id` on domain** | Required; `engagement_id` and `endpoint_id` are computed from the firewall. |
| **Secrets** | `secret_access_key` on `vayucloud_s3_token` is only available at **create**; store it securely. |
| **Provider login** | `VAYU_USERNAME` / `VAYU_PASSWORD` environment variables, or `username` / `password` in the provider block |

### Required before bucket create

| ID | Purpose |
|----|---------|
| `firewall_id` | IPC firewall with S3 mapping |
| Active `vayucloud_s3_domain` | Parent domain must complete async provisioning first |

Child resources should `depends_on = [vayucloud_s3_domain.<name>]` until domain `status` is complete.

### What you configure

| Item | How you provide it |
|------|-------------------|
| Domain | `vayucloud_s3_domain` — `domain_name`, `quota`, `storage_class`, `variant`, `pricing_model` |
| Bucket | `vayucloud_s3_bucket` — unique name within domain |
| Service user | `vayucloud_s3_user` — not on `AI_STANDARD` domains |
| API credentials | `vayucloud_s3_token` — expiry and description |
| Object data | S3-compatible client (not a provider resource) |
| Object metadata in IaC | `vayucloud_s3_object` / `vayucloud_s3_object_list` data sources |
| Public access | `vayucloud_network_public_ip` on `domain_access_ip` + `ill_to_vcs` rule |

## Resources in this module

| Resource | Description | Async | In-place updates | Documentation |
|----------|-------------|:-----:|------------------|----------------|
| `vayucloud_s3_domain` | S3 domain on firewall | Yes | **`quota`** only | [S3 domain](../resources/s3_domain.md) |
| `vayucloud_s3_bucket` | Bucket in domain | No | `versioning_enabled` | [S3 bucket](../resources/s3_bucket.md) |
| `vayucloud_s3_user` | Service user | No | None (replace to rename) | [S3 user](../resources/s3_user.md) |
| `vayucloud_s3_token` | Access key / secret | No | None (replace to rotate) | [S3 token](../resources/s3_token.md) |
| `vayucloud_network_public_ip` | NAT to domain private IP | Yes | `retain_on_dissociate` only | [Network public IP](../resources/network_public_ip.md) |
| `vayucloud_network_firewall_rule` | Internet → VCS (`ill_to_vcs`) | Yes | Addresses, services | [Network firewall rule](../resources/network_firewall_rule.md) |

**Replace-only fields (common pitfalls):**

| Resource | Changing these forces replacement |
|----------|-----------------------------------|
| `vayucloud_s3_domain` | `domain_name`, `storage_class`, `variant`, `firewall_id`, pricing (except quota) |
| `vayucloud_s3_bucket` | `domain_id`, `bucket_name` |
| `vayucloud_s3_user` | `domain_id`, `user_name` |
| `vayucloud_s3_token` | `domain_id`, `user_name`, expiry, description |
| `vayucloud_network_public_ip` | `resource_type`, `resource_id`, `private_ip`, pricing model |

## Data sources in this module

| Data source | Use for | Documentation |
|-------------|---------|----------------|
| `vayucloud_s3_domain` / `vayucloud_s3_domain_list` | Existing domains | [S3 domain](../data-sources/s3_domain.md) |
| `vayucloud_s3_bucket` / `vayucloud_s3_bucket_list` | Buckets in a domain | [S3 bucket](../data-sources/s3_bucket.md) |
| `vayucloud_s3_user` / `vayucloud_s3_user_list` | Users in a domain | [S3 user](../data-sources/s3_user.md) |
| `vayucloud_s3_token` / `vayucloud_s3_token_list` | Token metadata (no secret) | [S3 token](../data-sources/s3_token.md) |
| `vayucloud_s3_object` / `vayucloud_s3_object_list` | Object metadata (no body) | [S3 object](../data-sources/s3_object.md) |

## Dependency order

S3 stacks on an existing firewall. Domain provisioning is the long pole; everything else is fast once the domain is active.

| Step | What | Terraform |
|------|------|-----------|
| 1 | **Network firewall** | Existing `firewall_id` with S3 mapping |
| 2 | **Create S3 domain** | `vayucloud_s3_domain` — wait for async complete |
| 3 | **Create bucket** | `vayucloud_s3_bucket` |
| 4 | **Create user** | `vayucloud_s3_user` *(STANDARD domains)* |
| 5 | **Create token** | `vayucloud_s3_token` — capture `secret_access_key` |
| 6 | **Upload objects** *(out of band)* | S3-compatible client with token credentials |
| 7 | **Public IP** *(optional)* | `vayucloud_network_public_ip` — `private_ip = domain_access_ip` |
| 8 | **Firewall rule** *(optional)* | `ill_to_vcs` — internet → `vcs`, destination public IP `/32` |

```text
network_firewall
        │
        ▼
   vayucloud_s3_domain  (~30 min async)
        │
   ┌────┴────┬──────────┐
   ▼         ▼          ▼
s3_bucket  s3_user   (domain_access_ip)
   │         │
   │         ▼
   │     s3_token ──► S3 client (objects)
   │
   ▼ (optional)
network_public_ip ──► network_firewall_rule (ill_to_vcs)
```

## Complete example (STANDARD domain)

Repository examples:

| Path | Purpose |
|------|---------|
| `examples/IaaS/resources/vayucloud_s3_domain/` | Create domain |
| `examples/IaaS/resources/vayucloud_s3_bucket/` | Create bucket |
| `examples/IaaS/resources/vayucloud_s3_user/` | Create service user |
| `examples/IaaS/resources/vayucloud_s3_token/` | Create access token |
| `examples/IaaS/resources/vayucloud_network_public_ip/` | Public IP patterns |
| `examples/IaaS/resources/vayucloud_network_firewall_rule/` | `ill_to_vcs` rule |

```hcl
resource "vayucloud_s3_domain" "app" {
  domain_name   = "my-vcs-domain"
  quota         = 10
  storage_class = "STANDARD"
  variant       = "value"
  firewall_id   = tonumber(vayucloud_network_firewall.perimeter.id)
  pricing_model = "daily"
}

resource "vayucloud_s3_bucket" "data" {
  domain_id          = vayucloud_s3_domain.app.id
  bucket_name        = "app-data"
  versioning_enabled = true

  depends_on = [vayucloud_s3_domain.app]
}

resource "vayucloud_s3_user" "service" {
  domain_id = vayucloud_s3_domain.app.id
  user_name = "app-service-user"

  depends_on = [vayucloud_s3_domain.app]
}

resource "vayucloud_s3_token" "service" {
  domain_id         = vayucloud_s3_domain.app.id
  user_name         = vayucloud_s3_user.service.user_name
  token_expiry_date = "2027-12-31"
  token_description = "Terraform-managed token"

  depends_on = [vayucloud_s3_user.service]
}

resource "vayucloud_network_public_ip" "vcs" {
  resource_type           = "firewall"
  resource_id             = tonumber(vayucloud_network_firewall.perimeter.id)
  private_ip              = vayucloud_s3_domain.app.domain_access_ip
  public_ip_pricing_model = "daily"
  retain_on_dissociate    = true

  depends_on = [vayucloud_s3_domain.app]
}

resource "vayucloud_network_firewall_rule" "ill_to_vcs" {
  rule_name   = "ill_to_vcs"
  firewall_id = tonumber(vayucloud_network_firewall.perimeter.id)
  source      = "internet"
  destination = "vcs"
  action      = "allow"
  services    = ["HTTP", "HTTPS", "ALL_ICMP"]

  source_addresses      = ["0.0.0.0/0"]
  destination_addresses = ["${vayucloud_network_public_ip.vcs.public_ip}/32"]

  depends_on = [vayucloud_network_public_ip.vcs]
}
```

Use S3-compatible clients with `domain_name_fqdn`, `access_key_id`, and `secret_access_key` from the token resource. Endpoint URL and path-style vs virtual-host-style access follow platform documentation for your tenant.

## Operational notes

### Domain create timing

Domain creation is **long-running** (often **~30 minutes**; longer when firewall mapping is required). Do not interrupt `terraform apply` on the domain resource. Schedule CI/CD and change windows accordingly.

Only **`quota`** updates in place on `vayucloud_s3_domain`. Changing `domain_name`, `storage_class`, `variant`, or `firewall_id` forces replacement.

### AI_STANDARD domains

On `storage_class = "AI_STANDARD"` (DDN OSS):

* **`vayucloud_s3_user` create/delete is blocked** — use the pre-provisioned `root` user.
* **`vayucloud_s3_token`** — only `user_name = "root"` on create; token delete is blocked (rotate by creating new tokens on STANDARD domains instead).

### Token secrets

| Attribute | When available |
|-----------|----------------|
| `access_key_id` | Create and read |
| `secret_access_key` | **Create only** — preserved in Terraform state |

After import, `secret_access_key` is unknown. Create a replacement token if you need a new secret in state. Mark token outputs `sensitive = true`.

### Bucket delete order

Delete fails with *Bucket Not Empty* if objects remain. Empty the bucket out of band before destroying the bucket.

Deleting `vayucloud_s3_user` **cascades all tokens** for that user.

### Public access pattern

1. Read `domain_access_ip` from the domain after create completes.
2. Associate `vayucloud_network_public_ip` with that private IP (`resource_type = "firewall"`, `resource_id = firewall_id`).
3. Add `internet` → `vcs` rule with `destination_addresses = ["<public_ip>/32"]` and services `HTTP`, `HTTPS`, and optionally `ALL_ICMP`.

Changing `domain_access_ip` or dissociating the public IP requires updating the firewall rule destination.

### Geo-resilient variant

For `variant = "geoResilient"`, set `secondary_endpoint_id` on the domain. See [S3 domain](../resources/s3_domain.md) for argument details.

### Import existing infrastructure

| Resource | Import ID | After import |
|----------|-----------|--------------|
| `vayucloud_s3_domain` | `<domain_id>` | Set `firewall_id` and required arguments |
| `vayucloud_s3_bucket` | `<domain_id>:<bucket_name>` | Confirm versioning |
| `vayucloud_s3_user` | `<domain_id>:<user_name>` | — |
| `vayucloud_s3_token` | `<domain_id>:<user_name>:<access_key_id>` | Secret unknown — rotate if needed |

## Asynchronous operations

| Resource | Polling scope |
|----------|----------------|
| `vayucloud_s3_domain` | Create, update (quota), delete |
| `vayucloud_network_public_ip` | Create, destroy |
| `vayucloud_network_firewall_rule` | Create, update, delete |

Synchronous (no audit polling): `vayucloud_s3_bucket`, `vayucloud_s3_user`, `vayucloud_s3_token`.

## See also

- [VayuCloud provider](../index.md) — Authentication and installation.
- [Resources and data sources overview](../resourcesAndDatasource.md) — Full platform dependency order.
- [Firewall to VM guide](firewall_to_vm.md) — Firewall and public IP foundations.
- [S3 domain](../resources/s3_domain.md) — Domain variants and attributes.
- [S3 bucket](../resources/s3_bucket.md) — Versioning and delete constraints.
- [S3 token](../resources/s3_token.md) — Secret handling and AI_STANDARD rules.
- [S3 object](../data-sources/s3_object.md) — Object metadata data source.
- [Network firewall rule](../resources/network_firewall_rule.md) — `ill_to_vcs` example.
- [Network public IP](../resources/network_public_ip.md) — Association to `domain_access_ip`.
