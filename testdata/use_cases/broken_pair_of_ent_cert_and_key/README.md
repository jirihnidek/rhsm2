Broken Pair of Entitlement Certificate and Key
==============================================

When more than one entitlement certificate is installed, the client picks a
cert/key pair to authenticate against the CDN. If the selected certificate and
key do not belong together (e.g. a pair left behind by a failed certificate
reissue), loading them as a TLS key pair fails with:

    tls: private key does not match public key

The client must skip such a broken pair and use a valid one instead of refusing
to start.

This use case contains such a broken pair:

* `1234567890123456789.pem` - an entitlement certificate (with `CERTIFICATE`,
  `ENTITLEMENT DATA` and `RSA SIGNATURE` blocks).
* `1234567890123456789-key.pem` - an unrelated private key that does **not**
  match the certificate above.

The files in this directory are used only for testing purposes. The certificate
may look like a real entitlement certificate, but the organization ID is not
valid and its private key is intentionally missing, so it cannot be used to
authenticate anywhere.
