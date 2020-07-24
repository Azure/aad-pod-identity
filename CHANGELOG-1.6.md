:warning: v1.6.0+ contains breaking changes. Please carefully review this [doc](README.md#v160-breaking-change) before upgrade from 1.x.x versions of pod-identity.

# v1.6.2

### Features

- Acquire an token with the certificate of service principal ([#517](https://github.com/Azure/aad-pod-identity/pull/517))
- Handle MSI auth requests by ResourceID ([#540](https://github.com/Azure/aad-pod-identity/pull/540))
- make NMI listen only on localhost ([#658](https://github.com/Azure/aad-pod-identity/pull/658))
- trigger MIC sync when a pod label changes ([#682](https://github.com/Azure/aad-pod-identity/pull/682))

### Bug Fixes

- check iptable rules match expected ([#663](https://github.com/Azure/aad-pod-identity/pull/663))

### Other Improvements

- update base image with debian base ([#641](https://github.com/Azure/aad-pod-identity/pull/641))
- update node selector label to kubernetes.io/os ([#652](https://github.com/Azure/aad-pod-identity/pull/652))
- better error messages and handling ([#666](https://github.com/Azure/aad-pod-identity/pull/666))
- add default known types to scheme ([#668](https://github.com/Azure/aad-pod-identity/pull/668))
- Remove unused cert volumes from mic deployment ([#670](https://github.com/Azure/aad-pod-identity/pull/670))

### Documentation

- update typed namespacedname case for sp example ([#649](https://github.com/Azure/aad-pod-identity/pull/649))
- list components prometheus enpoints ([#660](https://github.com/Azure/aad-pod-identity/pull/660))
- add helm upgrade guide and known issues ([#683](https://github.com/Azure/aad-pod-identity/pull/683))
- add requirements to PR template and test standard to CONTRIBUTING.md ([#706](https://github.com/Azure/aad-pod-identity/pull/706))

### Helm

- add aks add-on exception in kube-system ([#634](https://github.com/Azure/aad-pod-identity/pull/634))
- disable crd-install when using Helm 3 ([#642](https://github.com/Azure/aad-pod-identity/pull/642))
- update default http probe port at deploy to 8085 ([#708](https://github.com/Azure/aad-pod-identity/pull/708))

### Test Improvements

- new test framework for aad-pod-identity ([#640](https://github.com/Azure/aad-pod-identity/pull/640))
- convert e2e test cases from old to new framework ([#650](https://github.com/Azure/aad-pod-identity/pull/650)), ([#656](https://github.com/Azure/aad-pod-identity/pull/656)), ([#662](https://github.com/Azure/aad-pod-identity/pull/662)), ([#664](https://github.com/Azure/aad-pod-identity/pull/664)), ([#667](https://github.com/Azure/aad-pod-identity/pull/667)), ([#680](https://github.com/Azure/aad-pod-identity/pull/680))
- add soak testing as part of nightly build & test and remove Jenkinsfile ([#687](https://github.com/Azure/aad-pod-identity/pull/687))
- update e2e suite to remove flakes ([#693](https://github.com/Azure/aad-pod-identity/pull/693)), ([#695](https://github.com/Azure/aad-pod-identity/pull/695)), ([#697](https://github.com/Azure/aad-pod-identity/pull/697)), ([#699](https://github.com/Azure/aad-pod-identity/pull/699)), ([#701](https://github.com/Azure/aad-pod-identity/pull/701))
- add e2e tests with resource id ([#696](https://github.com/Azure/aad-pod-identity/pull/696))
- add code coverage as part of CI ([#705](https://github.com/Azure/aad-pod-identity/pull/705))


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