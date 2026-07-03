# Security Policy

## Supported Versions

Security fixes are applied to the latest released version and the `main` branch.
Older versions are not maintained — if you are on an older release, please upgrade
to pick up security fixes.

## Secure by Default

helium's parser is **secure by default**: `NewParser()` is safe to point at
untrusted XML with no extra configuration — external entity and DTD loading is
blocked (XXE-safe), the filesystem is deny-all, network access is forbidden,
nesting depth is bounded, and entity-expansion amplification is guarded. The
XInclude processor and the XSD schema compiler are likewise deny-all by default.

See the [Security section of the README](README.md#security) for the full posture,
how to opt into loading external resources from *trusted* sources, and the
resource-budget responsibilities (maximum document size, context deadlines) that
remain with the caller.

## Scope

The following are **not** treated as vulnerabilities:

- Insecure behavior that results from deliberately relaxing the safe defaults —
  e.g. `BlockXXE(false)`, `LoadExternalDTD(true)`, `SubstituteEntities(true)`,
  `FS(helium.PermissiveFS())`, or removing the entity-amplification, name-length,
  or content-model-depth guards. Opting out of a protection is the caller's
  responsibility.
- The documented incomplete-sandbox caveat for permissive or directory-rooted
  `fs.FS` values (see the README Security section); rely on the deny-all default
  for confinement.

The `xmldsig1` (XML Digital Signatures) and `xmlenc1` (XML Encryption) packages are
**experimental** and fall outside the security-support boundary until they
stabilize. Reports are welcome, but these packages should not be relied on inside a
security or compliance boundary yet.

## Reporting a Vulnerability

If you think you found a vulnerability, please report it via
[GitHub Security Advisory](https://github.com/lestrrat-go/helium/security/advisories/new).
Please include explicit steps to reproduce the security issue.

We will do our best to respond in a timely manner, but please also be aware that
this project is maintained by a very limited number of people. Please help us with
test code and such.
