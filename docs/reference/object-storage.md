---
page_title: "S3 and VCS Object Storage"
description: |-
  VayuCloud S3-compatible VCS domains, buckets, users, tokens, and object data sources.
---

# S3 and VCS Object Storage

Create a VCS domain on a firewall, then manage buckets and service credentials inside that domain. Use object data sources for inventory and metadata; object bodies are not returned by the metadata data source.

## Prerequisites and use cases

Have a firewall and account context. Use domains to establish the VCS service boundary, buckets for objects, users and tokens for application access, and firewall/public-IP resources for controlled external access.

## Resources

- [S3 domain resource](../resources/s3_domain.md)
- [S3 bucket resource](../resources/s3_bucket.md)
- [S3 user resource](../resources/s3_user.md)
- [S3 token resource](../resources/s3_token.md)

## Data sources

- [S3 domain](../data-sources/s3_domain.md)
- [S3 domain list](../data-sources/s3_domain_list.md)
- [S3 bucket](../data-sources/s3_bucket.md)
- [S3 bucket list](../data-sources/s3_bucket_list.md)
- [S3 object](../data-sources/s3_object.md)
- [S3 object list](../data-sources/s3_object_list.md)
- [S3 user](../data-sources/s3_user.md)
- [S3 user list](../data-sources/s3_user_list.md)
- [S3 token](../data-sources/s3_token.md)
- [S3 token list](../data-sources/s3_token_list.md)

## Behavior notes

Domain creation may be asynchronous. Bucket deletion can be blocked while objects remain. Tokens and passwords are secrets; capture them only through approved sensitive-value workflows and never commit them to configuration.
