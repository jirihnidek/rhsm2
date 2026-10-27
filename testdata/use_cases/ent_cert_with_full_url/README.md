Entitlement Certificate with Full URLs
======================================

Content definitions in entitlement certificates usually contain relative paths.
In that case, the `baseurl` of each repository in the generated `redhat.repo`
file is created by appending the relative path to the `baseurl` option from
the `[rhsm]` section of `rhsm.conf`.

This use case contains an entitlement certificate whose content definitions
contain full URLs instead of relative paths. This is typically the case when
content templates are used. Such URLs are written to `redhat.repo` as they are,
without any modification.

The .pem file in this directory is used only for testing purposes. It may look
like a real entitlement certificate, but the organization ID is not valid.