:warning: v1.6.0 contains breaking changes. Please carefully review this [doc](README.md#v160-breaking-change) before upgrade from 1.x.x versions of pod-identity.

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