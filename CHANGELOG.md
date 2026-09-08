## 0.3.0

NOTES:
* TLS certificate verification is now enabled by default. Set `ca_cert_path` to your host's CA certificate (PEM), or `insecure = true` to skip it (dev/testing only).
* All connections to the XenServer host now use HTTPS.

BUG FIXES:
* Fix name-based network resolution on XenServer 9 hosts.

## 0.2.2

NOTES:
* Update dependencies; rebuilt with the XenServer Go SDK 25.3.0.

## 0.2.1

FEATURES:
* Support creating a VM with a full disk copy.

BUG FIXES:
* Small bug fixes.

## 0.2.0

FEATURES:
* Add Pool resource support (join/eject).
* Add NFS and SMB ISO storage repository support.

## 0.1.2

NOTES:
* Rebuilt with the XenServer Go SDK 24.32.0 to resolve a compatibility issue. No provider code changes.

## 0.1.1

NOTES:
* Documentation updates.

## 0.1.0

FEATURES:
* Initial release.
