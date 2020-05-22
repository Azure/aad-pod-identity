:warning: v1.6.0+ contains breaking changes. Please carefully review this [doc](README.md#v160-breaking-change) before upgrade from 1.x.x versions of pod-identity.

# v1.6.1

### Features
- re-initialize MIC cloud client when cloud config is updated ([#590](https://github.com/Azure/aad-pod-identity/pull/590))
- add finalizer for assigned identity ([#593](https://github.com/Azure/aad-pod-identity/pull/593))
- make update user msi calls retriable ([#601](https://github.com/Azure/aad-pod-identity/pull/601))

### Bug Fixes
- Fix issue that caused failures with long pod name > 63 chars ([#545](https://github.com/Azure/aad-pod-identity/pull/545))
- Fix updating assigned identity when azure identity updated ([#559](https://github.com/Azure/aad-pod-identity/pull/559))

### Other Improvements
- Add linting tools in Makefile ([#551](https://github.com/Azure/aad-pod-identity/pull/551))
- Code clean up and enable linting tools in CI ([#597](https://github.com/Azure/aad-pod-identity/pull/597))
- change to 404 instead if no azure identity found ([#629](https://github.com/Azure/aad-pod-identity/pull/629))

### Documentation
- document required role assignments ([#592](https://github.com/Azure/aad-pod-identity/pull/592))
- add `--subscription` parameter to az cli commands ([#602](https://github.com/Azure/aad-pod-identity/pull/602))
- add mic pod exception to deployment ([#611](https://github.com/Azure/aad-pod-identity/pull/611))
- reduce ambiguity in demo and role assignment docs ([#620](https://github.com/Azure/aad-pod-identity/pull/620))
- add support information to readme ([#623](https://github.com/Azure/aad-pod-identity/pull/623))
- update docs for pod-identity exception ([#624](https://github.com/Azure/aad-pod-identity/pull/624))

### Helm

- make cloud config configurable in helm chart ([#598](https://github.com/Azure/aad-pod-identity/pull/598))
- Support multiple identities in helm chart ([#457](https://github.com/Azure/aad-pod-identity/pull/457))


# v1.6.0

### Features
- Add support for pod-identity managed mode ([#486](https://github.com/Azure/aad-pod-identity/pull/486))
- Deny requests without metadata header to avoid SSRF ([#500](https://github.com/Azure/aad-pod-identity/pull/500))

### Bug Fixes
- Fix issue that caused failures with long pod name > 63 chars ([#545](https://github.com/Azure/aad-pod-identity/pull/545))
- Fix updating assigned identity when azure identity updated ([#559](https://github.com/Azure/aad-pod-identity/pull/559))

### Other Improvements
- Switch to using klog for logging ([#449](https://github.com/Azure/aad-pod-identity/pull/449))
- Create internal API for aadpodidentity ([#459](https://github.com/Azure/aad-pod-identity/pull/459))
- Switch to using PATCH instead of CreateOrUpdate for identities ([#522](https://github.com/Azure/aad-pod-identity/pull/522))
- Update client-go version to v0.17.2 ([#398](https://github.com/Azure/aad-pod-identity/pull/398))
- Update to go1.14 ([#543](https://github.com/Azure/aad-pod-identity/pull/543))
- Add validation for resource id format ([#548](https://github.com/Azure/aad-pod-identity/pull/548))