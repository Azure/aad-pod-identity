---
title: "Changelog"
linkTitle: "Changelog"
type: docs
menu:
  main:
    weight: 10
---

## v1.7.2

### Breaking Change
- If upgrading from versions 1.5.x to 1.7.x of pod-identity, please carefully review this [doc](https://azure.github.io/aad-pod-identity/docs/#v16x-breaking-change) before upgrade.
- Pod Identity is disabled by default for Clusters with Kubenet. Please review this [doc](https://azure.github.io/aad-pod-identity/docs/configure/aad_pod_identity_on_kubenet/) before upgrade.
- Helm chart contains breaking changes. Please review this [doc](https://github.com/Azure/aad-pod-identity/tree/master/charts/aad-pod-identity#300) before upgrade.


### Features

- feat: add arm64 build ([#950](https://github.com/Azure/aad-pod-identity/pull/950))

### Bug Fixes

- fix: fix typos in stats variables ([#919](https://github.com/Azure/aad-pod-identity/pull/919))
- fix: drop all unnecessary root capabilities for NMI ([#940](https://github.com/Azure/aad-pod-identity/pull/940))
- fix: copy response header and status code to http.ResponseWriter ([#946](https://github.com/Azure/aad-pod-identity/pull/946))

### Security

- dockerfile: fix CVE-2020-29362, CVE-2020-29363, CVE-2020-29361 ([#924](https://github.com/Azure/aad-pod-identity/pull/924))
- dockerfile: upgrade debian-iptables to buster-v1.4.0 ([#948](https://github.com/Azure/aad-pod-identity/pull/948))

### Helm

- helm: remove deprecated forceNameSpaced from values.yaml ([#927](https://github.com/Azure/aad-pod-identity/pull/927))
- helm: skip MIC exception installation when using managed mode ([#936](https://github.com/Azure/aad-pod-identity/pull/936))

### Documentation

- docs: document breaking change on `azureIdentities` ([#944](https://github.com/Azure/aad-pod-identity/pull/944))

### Other Improvements

- chore: update github pr template ([#925](https://github.com/Azure/aad-pod-identity/pull/925))
- cleanup: refactor demo code ([#930](https://github.com/Azure/aad-pod-identity/pull/930))
- chore: switch to using golang builder ([#952](https://github.com/Azure/aad-pod-identity/pull/952))

## v1.7.1

### Breaking Change
- If upgrading from versions 1.5.x to 1.7.x of pod-identity, please carefully review this [doc](https://azure.github.io/aad-pod-identity/docs/#v16x-breaking-change) before upgrade.
- Pod Identity is disabled by default for Clusters with Kubenet. Please review this [doc](https://azure.github.io/aad-pod-identity/docs/configure/aad_pod_identity_on_kubenet/) before upgrade.
- Helm chart contains breaking changes. Please review this [doc](https://github.com/Azure/aad-pod-identity/tree/master/charts/aad-pod-identity#300) before upgrade.

### Bug Fixes
- allow overwriting NODE_RESOURCE_GROUP in role-assignment.sh ([#873](https://github.com/Azure/aad-pod-identity/pull/873))

### Other Improvements
- fix CVE-2020-1971 ([#905](https://github.com/Azure/aad-pod-identity/pull/905))
- fix CVE-2020-27350 ([#909](https://github.com/Azure/aad-pod-identity/pull/909))

### Documentation
- add note about specifying which identity to use ([#869](https://github.com/Azure/aad-pod-identity/pull/869))
- fix `|` in markdown table ([#882](https://github.com/Azure/aad-pod-identity/pull/882))
- use `az aks show` for node resource group & more convenient command to run role assignment script ([#879](https://github.com/Azure/aad-pod-identity/pull/879))
- reduce number of role assignments ([#883](https://github.com/Azure/aad-pod-identity/pull/883))
- add spring boot example which interacts with blob storage ([#878](https://github.com/Azure/aad-pod-identity/pull/878))
- add changelog & development section and move java-blob example to website ([#891](https://github.com/Azure/aad-pod-identity/pull/891))
- Added instructions how to mitigate ARP spoofing on kubenet clusters with OPA/Gatekeeper ([#894](https://github.com/Azure/aad-pod-identity/pull/894))
- add warning note to kubenet docs ([#911](https://github.com/Azure/aad-pod-identity/pull/911))

### Helm
- rename forceNameSpaced to forceNamespaced ([#874](https://github.com/Azure/aad-pod-identity/pull/874))
- bump helm chart version to 2.1.0 for aad-pod-identity v1.7.0 ([#884](https://github.com/Azure/aad-pod-identity/pull/884))
- add topologySpreadConstraints and PodDisruptionBudget in helm chart ([#886](https://github.com/Azure/aad-pod-identity/pull/886))
- adding option to configure kubeletConfig ([#906](https://github.com/Azure/aad-pod-identity/pull/906))
- deprecate forceNameSpaced value ([#914](https://github.com/Azure/aad-pod-identity/pull/914))
- add notes ([#916](https://github.com/Azure/aad-pod-identity/pull/916))
- use map for azureIdentities instead of list in helm chart ([#899](https://github.com/Azure/aad-pod-identity/pull/899))

### Test Improvements
- remove getIdentityValidatorArgs ([#910](https://github.com/Azure/aad-pod-identity/pull/910))
- less error-prone identityvalidator ([#901](https://github.com/Azure/aad-pod-identity/pull/901))

## v1.7.0

### Breaking Change
- If upgrading from versions 1.5.x to 1.7.x of pod-identity, please carefully review this [doc](https://azure.github.io/aad-pod-identity/docs/#v16x-breaking-change) before upgrade.
- Pod Identity is disabled by default for Clusters with Kubenet. Please review this [doc](https://azure.github.io/aad-pod-identity/docs/configure/aad_pod_identity_on_kubenet/) before upgrade.

### Features
- support JSON logging format ([#839](https://github.com/Azure/aad-pod-identity/pull/839))
- disable aad-pod-identity by default for kubenet ([#842](https://github.com/Azure/aad-pod-identity/pull/842))
- add auxiliary tenant ids for service principal ([#843](https://github.com/Azure/aad-pod-identity/pull/843))

### Bug Fixes
- account for 150+ identity assignment and unassignment ([#847](https://github.com/Azure/aad-pod-identity/pull/847))

### Other Improvements
-  include image scanning as part of CI & set non-root user in Dockerfile ([#803](https://github.com/Azure/aad-pod-identity/pull/803))

### Documentation
- initial layout for static site ([#801](https://github.com/Azure/aad-pod-identity/pull/801))
- update website theme to docsy ([#828](https://github.com/Azure/aad-pod-identity/pull/828))
- update invalid URLs in website ([#832](https://github.com/Azure/aad-pod-identity/pull/832))
- fix casing of "priorityClassName" parameters in README.md ([#856](https://github.com/Azure/aad-pod-identity/pull/856))
- add docs for various topics ([#858](https://github.com/Azure/aad-pod-identity/pull/858))
- s/cluster resource group/node resource group ([#862](https://github.com/Azure/aad-pod-identity/pull/862))
- add docs for configuring in custom cloud ([#863](https://github.com/Azure/aad-pod-identity/pull/863))
- fix broken links and typo ([#864](https://github.com/Azure/aad-pod-identity/pull/864))

### Helm
- remove extra indentation in crd.yaml ([#833](https://github.com/Azure/aad-pod-identity/pull/833))
- make runAsUser conditional for MIC in helm ([#844](https://github.com/Azure/aad-pod-identity/pull/844))

### Test Improvements
- remove aks cluster version in e2e ([#808](https://github.com/Azure/aad-pod-identity/pull/808))
- decrease length of RG name to allow cluster creation in eastus2euap ([#810](https://github.com/Azure/aad-pod-identity/pull/810))
- health check with podIP from the busybox container ([#840](https://github.com/Azure/aad-pod-identity/pull/840))
- add gosec as part of linting ([#850](https://github.com/Azure/aad-pod-identity/pull/850))
- remove --ignore-unfixed for trivy ([#854](https://github.com/Azure/aad-pod-identity/pull/854))

{{% alert title="Warning" color="warning" %}}
v1.6.0+ contains breaking changes. Please carefully review this [doc](README.md#v16x-breaking-change) before upgrade from 1.x.x versions of pod-identity.
{{% /alert %}}

## v1.6.3

### Breaking Change

v1.6.0+ contains breaking changes. Please carefully review this [doc](https://azure.github.io/aad-pod-identity/docs/#v16x-breaking-change) before upgrading from 1.x.x versions of pod-identity.

### Features

- throttling - honor retry after header ([#742](https://github.com/Azure/aad-pod-identity/pull/742))
- reconcile identity assignment on Azure ([#734](https://github.com/Azure/aad-pod-identity/pull/734))

### Bug Fixes

- add certs volume for non-rbac manifests ([#713](https://github.com/Azure/aad-pod-identity/pull/713))
- Report original error from getPodListRetry ([#762](https://github.com/Azure/aad-pod-identity/pull/762))
- initialize klog flags for NMI ([#767](https://github.com/Azure/aad-pod-identity/pull/767))
- ensure stats collector doesn't aggregate stats from multiple runs ([#750](https://github.com/Azure/aad-pod-identity/pull/750))

### Other Improvements

- add deploy manifests and helm charts to staging dir ([#736](https://github.com/Azure/aad-pod-identity/pull/736))
- fix miscellaneous linting problem in the codebase ([#733](https://github.com/Azure/aad-pod-identity/pull/733))
- remove privileged: true for NMI daemonset ([#745](https://github.com/Azure/aad-pod-identity/pull/745))
- Update to go1.15 ([#751](https://github.com/Azure/aad-pod-identity/pull/751))
- automate role assignments and improve troubleshooting guide ([#754](https://github.com/Azure/aad-pod-identity/pull/754))
- set dnspolicy to clusterfirstwithhostnet for NMI ([#776](https://github.com/Azure/aad-pod-identity/pull/776))
- bump debian-base to v2.1.3 and debian-iptables to v12.1.2 ([#783](https://github.com/Azure/aad-pod-identity/pull/783))
- add logs for ignored pods ([#785](https://github.com/Azure/aad-pod-identity/pull/785))

### Documentation

- docs: fix broken test standard link in GitHub Pull Request template ([#710](https://github.com/Azure/aad-pod-identity/pull/710))
- Fixed typo ([#757](https://github.com/Azure/aad-pod-identity/pull/757))
- Fixed Grammar ([#758](https://github.com/Azure/aad-pod-identity/pull/758))
- add doc for deleting/recreating identity with same name ([#786](https://github.com/Azure/aad-pod-identity/pull/786))
- add best practices documentation ([#779](https://github.com/Azure/aad-pod-identity/pull/779))

### Helm

- add release namespace to chart manifests ([#741](https://github.com/Azure/aad-pod-identity/pull/741))
- Add imagePullSecretes to the Helm chart ([#774](https://github.com/Azure/aad-pod-identity/pull/774))
- Expose metrics port ([#777](https://github.com/Azure/aad-pod-identity/pull/777))
- add user managed identity support to helm charts ([#781](https://github.com/Azure/aad-pod-identity/pull/781))

### Test Improvements

- add e2e test for block-instance-metadata ([#715](https://github.com/Azure/aad-pod-identity/pull/715))
- add aks as part of pr and nightly test ([#717](https://github.com/Azure/aad-pod-identity/pull/717))
- add load test pipeline to nightly job ([#744](https://github.com/Azure/aad-pod-identity/pull/744))
- install aad-pod-identity in kube-system namespace ([#747](https://github.com/Azure/aad-pod-identity/pull/747))
- bump golangci-lint to v1.30.0 ([#759](https://github.com/Azure/aad-pod-identity/pull/759))


## v1.6.2

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


## v1.6.1

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

## v1.6.0

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

## v1.5.5

### Bug Fixes

- Prevent flushing custom iptable rules frequently ([#474](https://github.com/Azure/aad-pod-identity/pull/474))

## v1.5.4

### Features

- Add block-instance-metadata flag ([#396](https://github.com/Azure/aad-pod-identity/pull/396))
- Add metrics ([#429](https://github.com/Azure/aad-pod-identity/pull/429))
- Adding support for whitelisting of user-defined managed identities ([#431](https://github.com/Azure/aad-pod-identity/pull/431))

### Bug Fixes

- Fix glog flag parse error in nmi ([#435](https://github.com/Azure/aad-pod-identity/pull/435))

### Other Improvements

- Add application/json header for all return paths ([#424](https://github.com/Azure/aad-pod-identity/pull/424))
- Update golang used to build binaries ([#426](https://github.com/Azure/aad-pod-identity/pull/426))
- Reduce log verbosity for debug log ([#433](https://github.com/Azure/aad-pod-identity/pull/433))
- Move to latest Alpine 3.10.4 ([#446](https://github.com/Azure/aad-pod-identity/pull/446))
- Validate resource param exists in request ([#450](https://github.com/Azure/aad-pod-identity/pull/450))

## v1.5.3

### Bug Fixes

- Fix concurrent map read and map write while updating stats ([#344](https://github.com/Azure/aad-pod-identity/pull/344))
- Fix list calls to use local cache inorder to reduce api server load ([#358](https://github.com/Azure/aad-pod-identity/pull/358))
- Clean up assigned identities if node not found ([#367](https://github.com/Azure/aad-pod-identity/pull/367))
- Fixes to identity operations on VMSS ([#379](https://github.com/Azure/aad-pod-identity/pull/379))
- Fix namespaced multiple binding/identity handling and verbose logs ([#388](https://github.com/Azure/aad-pod-identity/pull/388))
- Fix panic issues while identity ids is nil ([#403](https://github.com/Azure/aad-pod-identity/pull/403))

### Other Improvements

- Set Content-Type on token response ([#341](https://github.com/Azure/aad-pod-identity/pull/341))
- Redact client id in NMI logs ([#343](https://github.com/Azure/aad-pod-identity/pull/343))
- Add user agent to kube-api calls ([#353](https://github.com/Azure/aad-pod-identity/pull/353))
- Add resource and request limits ([#372](https://github.com/Azure/aad-pod-identity/pull/372))
- Add user agent to ARM calls ([#387](https://github.com/Azure/aad-pod-identity/pull/387))
- Scale and performance improvements ([#408](https://github.com/Azure/aad-pod-identity/pull/408))
- Remove unused GET in CreateOrUpdate ([#411](https://github.com/Azure/aad-pod-identity/pull/411))
- Remove deprecated API Version usages ([#416](https://github.com/Azure/aad-pod-identity/pull/416))

## v1.5.2

### Bug Fixes

- Fix the token backward compat in host based token fetching ([#337](https://github.com/Azure/aad-pod-identity/pull/337))

## v1.5.1

### Bug Fixes

- Append NMI version to the `User-Agent` for adal only once ([#333](https://github.com/Azure/aad-pod-identity/pull/333))

### Other Improvements

- Change 'updateStrategy' for nmi DaemonSet to `RollingUpdate` ([#334](https://github.com/Azure/aad-pod-identity/pull/334))

## v1.5

### Features

- Support aad-pod-identity in init containers ([#191](https://github.com/Azure/aad-pod-identity/pull/191))
- Cleanup iptable chain and rule on uninstall ([#211](https://github.com/Azure/aad-pod-identity/pull/211))
- Remove dependency on azure.json ([#221](https://github.com/Azure/aad-pod-identity/pull/221))
- Add states for AzureAssignedIdentity and improve performance ([#219](https://github.com/Azure/aad-pod-identity/pull/219))
- System MSI cluster support ([#265](https://github.com/Azure/aad-pod-identity/pull/265))
- Leader election in MIC ([#277](https://github.com/Azure/aad-pod-identity/pull/277))
- Liveness probe for MIC and NMI ([#309](https://github.com/Azure/aad-pod-identity/pull/309))
- Application Exception ([#310](https://github.com/Azure/aad-pod-identity/pull/310))

### Bug Fixes

- Fix AzureIdentity with service principal ([#197](https://github.com/Azure/aad-pod-identity/pull/197))
- Determine resource manager endpoint based on cloud name ([#226](https://github.com/Azure/aad-pod-identity/pull/226))
- Fix incorrect resource endpoint with sp ([#251](https://github.com/Azure/aad-pod-identity/pull/251))
- Fix vmss identity deletion for ID in use ([#203](https://github.com/Azure/aad-pod-identity/pull/203))
- Fix removal of user assigned identity from nodes with system assigned ([#259](https://github.com/Azure/aad-pod-identity/pull/259))
- Handle case sensitive id check ([#271](https://github.com/Azure/aad-pod-identity/pull/271))
- Fix assigned id deletion when no identity exists ([#320](https://github.com/Azure/aad-pod-identity/pull/320))

### Other Improvements

- Use go modules ([#179](https://github.com/Azure/aad-pod-identity/pull/179))
- Log binary versions of MIC and NMI in logs ([#216](https://github.com/Azure/aad-pod-identity/pull/216))
- List CRDs via cache and avoid extra work on pod update ([#232](https://github.com/Azure/aad-pod-identity/pull/232))
- Reduce identity assignment times ([#199](https://github.com/Azure/aad-pod-identity/pull/199))
- NMI retries and ticker for periodic sync reconcile ([#272](https://github.com/Azure/aad-pod-identity/pull/272))
- Update error status code based on state ([#292](https://github.com/Azure/aad-pod-identity/pull/292))
- Process identity assignment/removal for nodes in parallel ([#305](https://github.com/Azure/aad-pod-identity/pull/305))
- Update base alpine image to 3.10.1 ([#324](https://github.com/Azure/aad-pod-identity/pull/324))
