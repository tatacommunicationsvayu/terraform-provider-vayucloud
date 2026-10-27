---
page_title: "Account and Resource Groups"
description: |-
  VayuCloud account context, engagements, locations, business units, environments, and audit data sources.
---

# Account and Resource Groups

Use these objects to select the account context and organize infrastructure before creating zones, virtual machines, and services.

## Prerequisites and use cases

You need an authenticated provider. Resolve an engagement and endpoint when the IDs are not already known. Create business units and environments when you want platform-side grouping and lifecycle boundaries.

## Resources

- [Business unit resource](../resources/resource_group_business_unit.md)
- [Environment resource](../resources/resource_group_environment.md)

## Data sources

- [Engagement](../data-sources/account_engagement.md)
- [Location](../data-sources/account_location.md)
- [Business unit](../data-sources/resource_group_business_unit.md)
- [Business unit list](../data-sources/resource_group_business_unit_list.md)
- [Environment](../data-sources/resource_group_environment.md)
- [Environment list](../data-sources/resource_group_environment_list.md)
- [Engagement audit log](../data-sources/account_engagement_auditlog.md)
- [Audit log details](../data-sources/auditlog_details.md)

## Behavior notes

Business units and environments are asynchronous platform resources. Validate parent identifiers before apply and review replacement behavior before changing parent IDs.
