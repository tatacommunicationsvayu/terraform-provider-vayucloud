---
page_title: "Load Balancer Guide"
subcategory: "Guides"
description: |-
  End-to-end guide for HAProxy load balancers on VayuCloud: enable LB, SSL profiles, virtual services, pool members, public IP, imports, and operational best practices.
---

# Load Balancer Guide

This guide walks through the **HAProxy load balancer** module in the VayuCloud provider: enable a load balancer on a firewall, upload SSL certificates, create virtual services with backend pools, and optionally expose the listener with a public IP.

For per-attribute documentation, open the linked resource and data source pages. For the full platform dependency chain (firewall → zone → VM → LB), see [Resources and data sources overview](../resourcesAndDatasource.md).

## Conventions

| Kind | Behavior |
|------|----------|
| **Resources** | Create, update, and destroy LB infrastructure. Long-running operations poll audit logs until completion. |
| **Data sources** | Read-only discovery (wizard options, VM lists, existing SSL profiles). |
| **`zone_id` on `vayucloud_network_lb`** | **Required.** Network zone on the firewall; validated against `firewall_id` before enable. Reuse on virtual services and `vayucloud_virtualmachine_list`. |
| **Bandwidth** | Always **Mbps** (for example `100` or `100Mbps`). Maximum **1000** Mbps. |

### Required before enable

| ID | Purpose |
|----|---------|
| `firewall_id` | Firewall on which to enable HAProxy |
| `zone_id` | Network zone on that firewall for virtual services and pool member VMs |

Both are required. If `zone_id` is missing or does not belong to `firewall_id`, Terraform fails at **plan** with a clear error.

### What you configure

| Item | How you provide it |
|------|-------------------|
| `firewall_id` | Terraform variable or literal in `vayucloud_network_lb` |
| `zone_id` | Same — must be a zone on that firewall |
| Pool VM names + ports | `pool_member` blocks; IPs from `vayucloud_virtualmachine_list` |
| HTTPS certificate + key | **Local file paths** in `file()` on `vayucloud_network_lb_ssl_profile` (see [TLS certificates](#tls-certificates-https)) |
| Provider login | `VAYU_USERNAME` / `VAYU_PASSWORD` environment variables, or `username` / `password` in the provider block |

You do **not** upload certificate files separately in the UI when Terraform manages `vayucloud_network_lb_ssl_profile`. Terraform reads the files from the paths you configure and sends the PEM contents to the platform.

## Resources in this module

| Resource | Description | Async | In-place updates | Documentation |
|----------|-------------|:-----:|------------------|----------------|
| `vayucloud_network_lb` | Enable HAProxy on a firewall; requires `zone_id` + `bandwidth` | Yes | **`bandwidth`** only | [Network load balancer](../resources/network_lb.md) |
| `vayucloud_network_lb_ssl_profile` | Upload client cert/key for HTTPS | Yes | None (recreate to change) | [Network LB SSL profile](../resources/network_lb_ssl_profile.md) |
| `vayucloud_network_lb_virtualservice` | Listener, pool, health monitors | Yes | Port, protocol, pool, monitors, VIP, persistence, cert | [Network LB virtual service](../resources/network_lb_virtualservice.md) |
| `vayucloud_network_public_ip` | NAT public IP to VS private VIP | Yes | `retain_on_dissociate` only | [Network public IP](../resources/network_public_ip.md) |

**Replace-only fields (common pitfalls):**

| Resource | Changing these forces replacement |
|----------|-----------------------------------|
| `vayucloud_network_lb` | `firewall_id` |
| `vayucloud_network_lb_virtualservice` | `load_balancer_id`, `name` |
| `vayucloud_network_lb_ssl_profile` | `load_balancer_id`, `certificate_name`, PEM contents |
| `vayucloud_network_public_ip` | `resource_type`, `resource_id`, `private_ip`, `public_ip_pricing_model` |

## Data sources in this module

| Data source | Use for | Documentation |
|-------------|---------|----------------|
| `vayucloud_network_lb_virtualservice_options` | Protocols, algorithms, monitors, persistence; optional zone list for validation | [Virtual service options](../data-sources/network_lb_virtualservice_options.md) |
| `vayucloud_virtualmachine_list` | Pool member VM **names and IPs** in the LB zone | [Virtual machine list](../data-sources/virtualmachine_list.md) |
| `vayucloud_network_lb_ssl_profiles` | Certificate names already on the LB | [Network LB SSL profiles](../data-sources/network_lb_ssl_profiles.md) |
| `vayucloud_network_zone` / `vayucloud_network_zone_list` | Resolve `zone_id` for a firewall | [Network zone](../data-sources/network_zone.md) |
| `vayucloud_network_lb` / `vayucloud_network_lb_list` | Read existing load balancers | [Network load balancer](../data-sources/network_lb.md) |

## Dependency order

Build from prerequisites inward. Steps 1–3 are usually already in place before LB work.

| Step | What | Terraform |
|------|------|-----------|
| 1 | Firewall with connectivity | `vayucloud_network_firewall` or existing firewall ID |
| 2 | Network zone on that firewall | `vayucloud_network_zone` or [`network_zone_list`](../data-sources/network_zone_list.md) |
| 3 | Backend VMs in the zone | `vayucloud_virtualmachine` or pre-existing VMs |
| 4 | **Enable load balancer** | `vayucloud_network_lb` — set `firewall_id`, **`zone_id`**, `bandwidth` |
| 5 | **Upload SSL cert** *(HTTPS)* | `vayucloud_network_lb_ssl_profile` |
| 6 | **Discover options** | `vayucloud_network_lb_virtualservice_options` |
| 7 | **Discover pool VMs** | `vayucloud_virtualmachine_list` with `zone_id = vayucloud_network_lb.<name>.zone_id` |
| 8 | **Create virtual service** | `vayucloud_network_lb_virtualservice` — `zone_id` from LB |
| 9 | **Public IP** *(optional)* | `vayucloud_network_public_ip` — `private_ip = <vs>.vip_ip` |

```text
firewall ──► zone (zone_id on LB) ──► VMs in zone
                    │
                    ▼
            vayucloud_network_lb
                    │
        ┌───────────┼───────────┐
        ▼           ▼           ▼
   ssl_profile   vs_options   vm_list
        │           │           │
        └─────► virtualservice ◄┘
                    │
                    ▼ (optional)
              network_public_ip
```

## Complete example (HTTPS)

Repository examples (same variable pattern as other IaaS modules):

| Path | Purpose |
|------|---------|
| `examples/IaaS/complete/lb_and_vs/` | End-to-end LB → SSL → HTTPS VS → public IP |
| `examples/IaaS/resources/vayucloud_network_lb/` | Enable load balancer only |
| `examples/IaaS/resources/vayucloud_network_lb_ssl_profile/` | Upload certificate |
| `examples/IaaS/resources/vayucloud_network_lb_virtualservice/` | Virtual service only |
| `examples/IaaS/data-sources/vayucloud_network_lb/` | Discover LBs on a firewall |
| `examples/IaaS/data-sources/vayucloud_network_lb_virtualservice_options/` | VS wizard options + pool VMs |

Copy `examples/IaaS/complete/lb_and_vs/terraform.tfvars.example` to `terraform.tfvars` and set `firewall_id`, `zone_id`, and `pool_members` for your environment.

The configuration below matches the complete example:

```hcl
resource "vayucloud_network_lb" "lb" {
  firewall_id   = {{firewall_id}}
  zone_id       = {{zone_id}}
  bandwidth     = "100"
  pricing_model = "daily"
}

data "vayucloud_network_lb_virtualservice_options" "opts" {
  load_balancer_id = vayucloud_network_lb.lb.id
  monitor_type     = "http"
}

data "vayucloud_virtualmachine_list" "pool_vms" {
  zone_id = vayucloud_network_lb.lb.zone_id
}

locals {
  pool_vms_by_name = {
    for vm in data.vayucloud_virtualmachine_list.pool_vms.virtual_machines : vm.name => vm.ip
  }
  pool_members = [
    for member in var.pool_members : {
      name       = member.name
      ip_address = local.pool_vms_by_name[member.name]
      port       = member.port
    }
  ]
}

resource "vayucloud_network_lb_ssl_profile" "cert" {
  load_balancer_id = vayucloud_network_lb.lb.id
  certificate_name = "my-cert"
  certificate      = file("cert.pem")
  private_key      = file("key.pem")
}

resource "vayucloud_network_lb_virtualservice" "https" {
  load_balancer_id = vayucloud_network_lb.lb.id
  name             = "app-vs-https"
  zone_id          = vayucloud_network_lb.lb.zone_id
  port             = "443"
  protocol         = "https"
  pool_algorithm   = "roundrobin"
  monitor          = ["httpchk"]
  certificate_name = "my-cert.pem"
  # vip_ip = "10.x.x.x"  # omit on first create; pin after output vip_ip

  dynamic "pool_member" {
    for_each = local.pool_members
    content {
      name       = pool_member.value.name
      ip_address = pool_member.value.ip_address
      port       = pool_member.value.port
    }
  }

  depends_on = [vayucloud_network_lb_ssl_profile.cert]
}

resource "vayucloud_network_public_ip" "vs_vip" {
  resource_type           = "loadbalancer"
  resource_id             = tonumber(vayucloud_network_lb.lb.id)
  private_ip              = vayucloud_network_lb_virtualservice.https.vip_ip
  public_ip_pricing_model = "daily"
  retain_on_dissociate    = true

  depends_on = [vayucloud_network_lb_virtualservice.https]
}
```

Variable `pool_members` example:

```hcl
variable "pool_members" {
  type = list(object({
    name = string
    port = number
  }))
  default = [
    { name = "web-vm-1", port = 8080 },
    { name = "web-vm-2", port = 8080 },
  ]
}
```

## Operational notes

### Pin the VIP after first create

1. Apply with **`vip_ip` omitted** on the virtual service.
2. Read assigned VIP: `terraform output -raw vip_ip`
3. Set `vip_ip` in configuration and apply again.

Without a pinned VIP, VS edits may allocate a **new** private IP. Because [`vayucloud_network_public_ip`](../resources/network_public_ip.md) binds to `private_ip`, a VIP change can **replace** the public IP association.

### HTTPS protocol

Set `protocol = "https"` in Terraform. The provider maps this to HAProxy API mode `http` while keeping `https` in plan and state.

### TLS certificates (HTTPS)

For HTTPS you need **two PEM-encoded files** on the machine that runs `terraform apply`:

| File | Terraform attribute | Notes |
|------|---------------------|--------|
| Certificate | `certificate` | PEM text (`-----BEGIN CERTIFICATE-----`) |
| Private key | `private_key` | PEM text (`-----BEGIN PRIVATE KEY-----` or `RSA PRIVATE KEY`) |

Reference them with **`file()`** — only the **path** goes in configuration; Terraform reads the file contents:

```hcl
resource "vayucloud_network_lb_ssl_profile" "cert" {
  load_balancer_id = vayucloud_network_lb.lb.id
  certificate_name = "my-cert"
  certificate      = file("certs/server.crt")   # .pem, .crt, or .cert — any name if PEM inside
  private_key      = file("certs/server.key")
}
```

**Local filename vs platform name**

| Setting | Example | Meaning |
|---------|---------|---------|
| `certificate_name` on SSL profile | `"my-cert"` | Upload label in the UI (`.pem` optional) |
| Stored name on the load balancer | `my-cert.pem` | HAProxy storage name |
| `certificate_name` on virtual service | `"my-cert.pem"` | References the cert **already on the LB** — **not** a file path |

**Format rules**

- Content must be **PEM** (text). Binary DER or `.pfx` / `.p12` must be converted first.
- The local extension (`.pem`, `.crt`, `.cert`) does not matter — only the PEM content inside the file.
- Do **not** commit private keys to version control.

**Generate a self-signed certificate** (non-production only):

```shell
openssl req -x509 -newkey rsa:2048 -keyout key.pem -out cert.pem -days 365 -nodes -subj "/CN=my-cert"
```

Then point `file()` at `cert.pem` and `key.pem`. In production, use your organization's CA-issued certificate and key.

### Certificate name

Use the **`.pem` storage name** on the virtual service (for example `my-cert.pem`). The provider resolves the full HAProxy path from the LB SSL profile list. Upload certs with [`vayucloud_network_lb_ssl_profile`](../resources/network_lb_ssl_profile.md) before creating the HTTPS listener.

### Enable timing (load balancer create)

Enabling a load balancer is a **long-running platform operation**. The provider polls the audit log until completion; do not interrupt `terraform apply`.

| Operation | Typical duration | What the platform does |
|-----------|------------------|-------------------------|
| **`vayucloud_network_lb` create (enable)** | **20–30 minutes** | Provisions HAProxy infrastructure: internal business unit / environment / zone resources, internal VMs, and VIP capacity on the firewall |
| SSL profile upload | Minutes | Uploads certificate to the LB |
| Virtual service create | Minutes | Creates listener, pool, and assigns a VIP |
| Public IP association | Minutes | NAT mapping to the VS VIP |
| Bandwidth update / VS update | Minutes | In-place changes (when VIP is pinned) |

Plan CI/CD pipelines and change windows accordingly. A single `terraform apply` that creates LB + SSL + VS + public IP may run **30 minutes or longer** end to end.

### Backend health checks

Pool members must respond to the selected monitor (`httpchk` sends HTTP/OPTIONS). Plain `python -m http.server` returns **501 on OPTIONS** and marks backends DOWN. Use a small HTTP server that handles OPTIONS, or `tcp-check` for TCP virtual services.

### LB bandwidth updates

Only `bandwidth` changes in place on `vayucloud_network_lb`. Virtual service and SSL resources are unaffected when bandwidth is updated correctly and VIP is pinned.

### Import existing infrastructure

| Resource | Import ID | After import |
|----------|-----------|--------------|
| `vayucloud_network_lb` | `<lb_id>` or `<lb_id>:<engagement_id>` | Set **`zone_id`** in config (platform read does not return it) |
| `vayucloud_network_lb_virtualservice` | `<lb_id>/<vs_name>` | Confirm `zone_id`, pool members, `vip_ip` in config |
| `vayucloud_network_lb_ssl_profile` | `<lb_id>/<cert_name.pem>` | Set `certificate` and `private_key` if managing PEM in Terraform |

## Asynchronous operations

| Resource | Polling scope |
|----------|----------------|
| `vayucloud_network_lb` | Create, update (bandwidth), delete |
| `vayucloud_network_lb_virtualservice` | Create, update, delete |
| `vayucloud_network_lb_ssl_profile` | Create, delete |
| `vayucloud_network_public_ip` | Create, destroy (dissociate) |

No separate wait logic is required in configuration; the provider polls audit logs until each operation completes.

## See also

- [VayuCloud provider](../index.md) — Authentication and installation.
- [Resources and data sources overview](../resourcesAndDatasource.md) — Full platform dependency order.
- [Network load balancer](../resources/network_lb.md) — LB resource reference.
- [Network LB virtual service](../resources/network_lb_virtualservice.md) — Virtual service reference.
- [Network LB SSL profile](../resources/network_lb_ssl_profile.md) — Certificate upload reference.
- [Virtual service options](../data-sources/network_lb_virtualservice_options.md) — Wizard discovery.
- [Virtual machine list](../data-sources/virtualmachine_list.md) — Pool member discovery.
- [Network public IP](../resources/network_public_ip.md) — Public access to VS VIP.
