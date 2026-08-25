---
page_title: "vayucloud_network_lb_ssl_profile Resource"
subcategory: ""
description: |-
  Uploads a client SSL certificate profile on a VayuCloud Load Balancer for HTTPS virtual services. Create and delete operations are asynchronous where the platform requires audit polling.
---

# `vayucloud_network_lb_ssl_profile`

Uploads a client SSL certificate and private key to an HAProxy load balancer. This is a prerequisite for HTTPS virtual services.

For the full module walkthrough, see the [Load balancer guide](../guides/load_balancer.md).

Example configuration: [`examples/IaaS/resources/vayucloud_network_lb_ssl_profile`](../../examples/IaaS/resources/vayucloud_network_lb_ssl_profile/resource.tf). Generate PEM files per [`CERTIFICATE.txt`](../../examples/IaaS/resources/vayucloud_network_lb_ssl_profile/CERTIFICATE.txt).

Provide the certificate upload name and PEM file contents (same as uploading cert and key in the UI). The upload name may omit `.pem`; HAProxy stores it as `{name}.pem`. Use that stored name as `certificate_name` on [`vayucloud_network_lb_virtualservice`](network_lb_virtualservice.md) (for example `my-cert.pem`).

## What you provide

You only configure **local file paths** in Terraform. The provider reads the files and uploads the PEM contents — there is no separate file upload step outside Terraform.

| Argument | You provide | Example |
|----------|-------------|---------|
| `certificate_name` | Text label for the cert on the LB | `"my-cert"` |
| `certificate` | Path via `file()` | `file("certs/server.crt")` |
| `private_key` | Path via `file()` | `file("certs/server.key")` |

**File format:** PEM-encoded text inside the file. Local extensions such as `.pem`, `.crt`, or `.cert` are all acceptable if the content is PEM. Binary `.pfx` / `.p12` files must be converted to PEM first.

**On the HTTPS virtual service**, set `certificate_name` to the **storage name on the load balancer** (for example `my-cert.pem`), not a filesystem path.

Upload the SSL profile **before** creating an HTTPS virtual service, or reference an existing cert from [`vayucloud_network_lb_ssl_profiles`](../data-sources/network_lb_ssl_profiles.md).

**Update is not supported.** To change certificate content or name, remove and recreate the resource.

Changing **`load_balancer_id`** or **`certificate_name`** forces replacement.

## Example Usage

```hcl
resource "vayucloud_network_lb_ssl_profile" "client_cert" {
  load_balancer_id = {{load_balancer_id}}
  certificate_name = "my-cert"
  certificate      = file("cert.pem")
  private_key      = file("key.pem")
}
```

## Argument Reference

### Required

* `load_balancer_id` — (String) Parent Load Balancer CI Master ID. Forces replacement if changed.
* `certificate_name` — (String) Certificate upload name (UI label). May omit `.pem`; use `{name}.pem` on HTTPS virtual services. Forces replacement if changed.
* `certificate` — (String, Sensitive) Certificate content in PEM format. Set with `file("<path>")` — any local filename (`.pem`, `.crt`, `.cert`) works if the file contains PEM text. Forces replacement if changed.
* `private_key` — (String, Sensitive) Private key content in PEM format. Set with `file("<path>")`. Forces replacement if changed.

## Attributes Reference

* `id` — (String) Composite ID `{load_balancer_id}/{name}.pem` (for example `<lb_id>/my-cert.pem`).
* `valid_from` — (String) Certificate validity start date.
* `valid_to` — (String) Certificate validity end date.
* `ci_status` — (String) Parent load balancer CI status after refresh.
* `audit_id` — (String) Audit identifier for the latest operation.
* `status` — (String) Audit status.
* `last_updated` — (String) Timestamp of the last update.

## Import

Import using the composite ID `{load_balancer_id}/{certificate_name}`:

```shell
terraform import vayucloud_network_lb_ssl_profile.example <load_balancer_id>/<certificate_name>
```

Example:

```shell
terraform import vayucloud_network_lb_ssl_profile.example <lb_id>/my-cert.pem
```

OpenTofu: use the same syntax with `tofu import`.

Imported resources do not retain PEM contents in state; you may need to set `certificate` and `private_key` in configuration for a fully managed resource.

## See also

* [Load balancer guide](../guides/load_balancer.md) — End-to-end HAProxy walkthrough.
* [`vayucloud_network_lb`](network_lb.md) — Parent load balancer (enable first with `zone_id`).
* [`vayucloud_network_lb_ssl_profiles`](../data-sources/network_lb_ssl_profiles.md) — List uploaded SSL profiles.
* [`vayucloud_network_lb_virtualservice`](network_lb_virtualservice.md) — Attach the certificate to an HTTPS listener.
