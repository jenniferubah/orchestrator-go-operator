# Release the operator (for OCP) with Konflux

## Table of contents
* [Introduction](#introduction)
* [Audience](#audience)
* [Prerequisites](#prerequisites)
* [Releasing](#releasing)
  * [Staging](#staging-release)
      * [Releasing the container images to the staging
        registry](#releasing-the-container-images-to-the-staging-registry)
      * [Releasing a new FBC index to
        staging](#releasing-a-new-fbc-index-to-staging)
  * [Production](#production-release)
      * [Releasing the container images to the production
        registry](#releasing-the-container-images-to-the-production-registry)
      * [Releasing a new FBC index to
        production](#releasing-a-new-fbc-index-to-production)
* [Troubleshooting](#troubleshooting)
* [Command tips](#command-tips)

## Introduction
This document captures the steps required to release a new version of the
Orchestrator Operator interacting with Konflux's resources using the CLI, as
opposed to the Konflux UI which can also achieve the same goals. The purpose of
this document is to allow users to benefit from a CLI approach on releasing the
operator, with the end goal of achieving as much automation as possible to
reduce the manual effort required to release it with Konflux. Note that
everything that is described here can be achieved as well using the [Konflux
UI](https://console.redhat.com/application-pipeline/workspaces).

## Audience
This document is aimed for those who need to release a new version of the
Orchestrator operator using Konflux pipelines. This guide assumes that the user
has some knowledge on Konflux and its types of resources and that the release
process has already been introduced by someone else with understanding,
otherwise the document might seem confusing and not clear on the goals.

For further reading on Konflux, visit their [documentation
website](https://konflux-ci.dev/docs/advanced-how-tos/releasing/) to get
acquainted with it and understand the release process as described in it.

## Prerequisites:
To be able to release the operator, you will need first to have access to the
orchestrator-releng workspace in Konflux via the [Red Hat
Console](https://console.redhat.com/application-pipeline/workspaces/orchestrator-releng/applications).
If you don't, please reach out to @jordigilh, @masayag, @rgolangh or
@pkliczewski to request access. You'll also need to be able to create PRs to the
[orchestrator-go-operator](https://github.com/rhdhorchestrator/orchestrator-go-operator)
and [orchestrator-fbc](https://github.com/rhdhorchestrator/orchestrator-fbc)
repositories.

Accessing the Konflux cluster via `oc` CLI requires an auth token from the OCP.
Once you've been added to the `orchestrator-releng` workspace, head to [this
URL](https://oauth-openshift.apps.stone-prd-rh01.pg1f.p1.openshiftapps.com/oauth/token/request)
to obtain a new token and login to the Konflux cluster.


### Preparation for Konflux Release

Make sure you have already manually released the orchestrator go operator
before starting a Konflux release.\
Follow the steps outlined in this
[guide](https://github.com/rhdhorchestrator/orchestrator-go-operator/tree/main?tab=readme-ov-file#preparing-the-code-for-releasing).


**It's crucial to determine the _appropriate releasing scenario_:**
-   **Manual Release:** Suitable for local development or early QE testing.
    Instructions can be found
    [here](https://github.com/rhdhorchestrator/orchestrator-go-operator/tree/main?tab=readme-ov-file#manual-release-for-upstream-only).
-   **Konflux Managed Release:** Designed for staging and production
    environments. Proceed with the instructions in the [Releasing](#releasing)
    section.


Once QE is done with early testing and gives the green light, follow the outlines 
process below carefully to add the configurations/manifests in order to prepare 
for konflux release. 

These configurations are added in the [konflux releae data repo](https://gitlab.cee.redhat.com/releng/konflux-release-data).
To get access to the repo as a CODEOWNER, please reach out to  @jordigilh, @masayag, 
@gciavarr, or @jubah. Once you have the right access, create a branch from main.


#### Create Go Operator Config For New Release
* Navigate to [orchestrator tenant config](https://gitlab.cee.redhat.com/releng/konflux-release-data/-/tree/main/tenants-config/cluster/stone-prd-rh01/tenants/orchestrator-releng-tenant/operator?ref_type=heads)
* Add a new config file `operator-1.x.yaml` for new release.
This should contain the Application, Component, ReleasePlan, ImageRepository, IntegrationTestScenario.
Example for 1.5 release `operator-1-5.yaml`
```yaml
---
apiVersion: appstudio.redhat.com/v1alpha1
kind: Application
metadata:
  name: operator-1-5
  namespace: orchestrator-releng-tenant
spec:
  displayName: operator (release-1-5)
---
apiVersion: appstudio.redhat.com/v1alpha1
kind: Component
metadata:
  name: controller-rhel9-operator-1-5
  namespace: orchestrator-releng-tenant
spec:
  application: operator-1-5
  build-nudges-ref:
    - orchestrator-operator-bundle-1-5
  componentName: controller-rhel9-operator-1-5
  containerImage: quay.io/redhat-user-workloads/orchestrator-releng-tenant/controller-rhel9-operator
  source:
    git:
      dockerfileUrl: Dockerfile
      revision: main
      url: https://github.com/rhdhorchestrator/orchestrator-go-operator.git
---
apiVersion: appstudio.redhat.com/v1alpha1
kind: Component
metadata:
  name: orchestrator-operator-bundle-1-5
  namespace: orchestrator-releng-tenant
spec:
  application: operator-1-5
  componentName: orchestrator-operator-bundle-1-5
  containerImage: quay.io/redhat-user-workloads/orchestrator-releng-tenant/orchestrator-operator-bundle
  source:
    git:
      dockerfileUrl: bundle.konflux.Dockerfile
      revision: main
      url: https://github.com/rhdhorchestrator/orchestrator-go-operator.git
---
apiVersion: appstudio.redhat.com/v1alpha1
kind: ImageRepository
metadata:
  annotations:
    image-controller.appstudio.redhat.com/update-component-image: "true"
  name: imagerepository-for-go-operator-1-5-controller-rhel9-operator-1-5
  namespace: orchestrator-releng-tenant
  labels:
    appstudio.redhat.com/application: operator-1-5
    appstudio.redhat.com/component: controller-rhel9-operator-1-5
spec:
  image:
    visibility: public
    name: orchestrator-releng-tenant/controller-rhel9-operator
  notifications:
    - config:
        url: https://bombino.api.redhat.com/v1/sbom/quay/push
      event: repo_push
      method: webhook
      title: SBOM-event-to-Bombino
---
apiVersion: appstudio.redhat.com/v1alpha1
kind: ImageRepository
metadata:
  annotations:
    image-controller.appstudio.redhat.com/update-component-image: "true"
  name: imagerepository-for-go-operator-1-5-orchestrator-operator-bundle-1-5
  namespace: orchestrator-releng-tenant
  labels:
    appstudio.redhat.com/application: operator-1-5
    appstudio.redhat.com/component: orchestrator-operator-bundle-1-5
spec:
  image:
    visibility: public
    name: orchestrator-releng-tenant/orchestrator-operator-bundle
  notifications:
    - config:
        url: https://bombino.api.redhat.com/v1/sbom/quay/push
      event: repo_push
      method: webhook
      title: SBOM-event-to-Bombino
---
apiVersion: appstudio.redhat.com/v1alpha1
kind: ReleasePlan
metadata:
  name: operator-staging-1-5
  labels:
    release.appstudio.openshift.io/auto-release: "false"
    release.appstudio.openshift.io/releasePlanAdmission: operator-staging-1-5
    release.appstudio.openshift.io/standing-attribution: "true"
spec:
  application: operator-1-5
  target: rhtap-releng-tenant
---
apiVersion: appstudio.redhat.com/v1alpha1
kind: ReleasePlan
metadata:
  name: operator-prod-1-5
  labels:
    release.appstudio.openshift.io/auto-release: "false"
    release.appstudio.openshift.io/releasePlanAdmission: operator-prod-1-5
    release.appstudio.openshift.io/standing-attribution: "true"
spec:
  application: operator-1-5
  target: rhtap-releng-tenant
---
apiVersion: appstudio.redhat.com/v1beta2
kind: IntegrationTestScenario
metadata:
  name: operator-enterprise-contract-1-5
  namespace: orchestrator-releng-tenant
spec:
  params:
    - name: POLICY_CONFIGURATION
      value: rhtap-releng-tenant/registry-orchestrator-releng
    - name: SINGLE_COMPONENT
      value: "true"
  application: operator-1-5
  contexts:
    - description: Application testing
      name: application
  resolverRef:
    params:
      - name: url
        value: "https://github.com/konflux-ci/build-definitions"
      - name: revision
        value: main
      - name: pathInRepo
        value: pipelines/enterprise-contract.yaml
    resolver: git
```
* Update the [kustomization file](https://gitlab.cee.redhat.com/releng/konflux-release-data/-/blob/main/tenants-config/cluster/stone-prd-rh01/tenants/orchestrator-releng-tenant/operator/kustomization.yaml?ref_type=heads#L4). 
Add the new `operator-1-x.yaml` to the resource list.
Example for 1.5 release `operator-1-5.yaml`
```yaml
---
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  - helm-operator-1-3.yaml
  - helm-operator-1-4.yaml
  - operator-1-5.yaml
```


#### Create FBC Config For New OCP Version
* Navigate to [orchestrator tenant config](https://gitlab.cee.redhat.com/releng/konflux-release-data/-/tree/main/tenants-config/cluster/stone-prd-rh01/tenants/orchestrator-releng-tenant?ref_type=heads)
* Add a new folder for the new FBC `fbc-v4-17` for referencing the newly supported OCP version.
Example folder for [v4.17](https://gitlab.cee.redhat.com/releng/konflux-release-data/-/blob/main/tenants-config/cluster/stone-prd-rh01/tenants/orchestrator-releng-tenant/fbc-v4-17/kustomization.yaml?ref_type=heads)
* Add a new config file `fbc-v4-17.yaml` for new OCP version.
This should contain the Application, Component, ReleasePlan, ImageRepository, IntegrationTestScenario.
Example for `v4.17` OCP version `fbc-v4-17.yaml`
```yaml
---
apiVersion: appstudio.redhat.com/v1alpha1
kind: Application
metadata:
  name: fbc-v4-17
  namespace: orchestrator-releng-tenant
spec:
  displayName: FBC v4.17
---
apiVersion: appstudio.redhat.com/v1alpha1
kind: Component
metadata:
  name: fbc-v4-17
  namespace: orchestrator-releng-tenant
spec:
  application: fbc-v4-17
  componentName: fbc-v4-17
  containerImage: quay.io/redhat-user-workloads/orchestrator-releng-tenant/fbc-v4-17
  source:
    git:
      context: v4.17
      dockerfileUrl: catalog.Dockerfile
      revision: main
      url: https://github.com/rhdhorchestrator/orchestrator-fbc.git
---
apiVersion: appstudio.redhat.com/v1alpha1
kind: ImageRepository
metadata:
  annotations:
    image-controller.appstudio.redhat.com/update-component-image: "true"
  name: imagerepository-for-fbc-v4-17
  namespace: orchestrator-releng-tenant
  labels:
    appstudio.redhat.com/application: fbc-v4-17
    appstudio.redhat.com/component: fbc-v4-17
spec:
  image:
    name: orchestrator-releng-tenant/fbc-v4-17
    visibility: public
  notifications:
    - config:
        url: https://bombino.api.redhat.com/v1/sbom/quay/push
      event: repo_push
      method: webhook
      title: SBOM-event-to-Bombino
---
apiVersion: appstudio.redhat.com/v1alpha1
kind: ReleasePlan
metadata:
  labels:
    release.appstudio.openshift.io/auto-release: "false"
    release.appstudio.openshift.io/releasePlanAdmission: orchestrator-fbc-prod-index-v4-15-plus
    release.appstudio.openshift.io/standing-attribution: "true"
  name: fbc-v4-17-release-as-production-fbc
  namespace: orchestrator-releng-tenant
spec:
  application: fbc-v4-17
  releaseGracePeriodDays: 7
  target: rhtap-releng-tenant
---
apiVersion: appstudio.redhat.com/v1alpha1
kind: ReleasePlan
metadata:
  name: fbc-v4-17-release-as-staging-fbc
  namespace: orchestrator-releng-tenant
  labels:
    release.appstudio.openshift.io/auto-release: "false"
    release.appstudio.openshift.io/releasePlanAdmission: orchestrator-fbc-staging-index-v4-15-plus
    release.appstudio.openshift.io/standing-attribution: "true"
spec:
  application: fbc-v4-17
  target: rhtap-releng-tenant
---
apiVersion: appstudio.redhat.com/v1beta2
kind: IntegrationTestScenario
metadata:
  name: fbc-v4-17-enterprise-contract
  namespace: orchestrator-releng-tenant
spec:
  application: fbc-v4-17
  params:
    - name: POLICY_CONFIGURATION
      value: rhtap-releng-tenant/fbc-stage
  resolverRef:
    params:
      - name: url
        value: "https://github.com/konflux-ci/build-definitions"
      - name: revision
        value: main
      - name: pathInRepo
        value: pipelines/enterprise-contract.yaml
    resolver: git
```
* Add the `kustomization.yaml` file under the same folder.
Example for `v4.17`
```yaml
---
kind: Kustomization
apiVersion: kustomize.config.k8s.io/v1beta1
# Naming: <API_GROUP>/<KIND_PLURAL>/<METADATA_NAME>
resources:
  - fbc-v4-17.yaml
```

#### Run Build Manifest Script
* Run the `build-manifests.sh` script (found [here](https://gitlab.cee.redhat.com/releng/konflux-release-data/-/tree/main/tenants-config?ref_type=heads)).
This will add and/update the manifests under the [auto-generated folder](https://gitlab.cee.redhat.com/releng/konflux-release-data/-/tree/main/tenants-config/auto-generated/cluster/stone-prd-rh01/tenants/orchestrator-releng-tenant?ref_type=heads). 
Commit these change in addition to any other relevant additions.

#### Add ReleasePlanAdmission (RPA) For New Release
* Navigate to [orchestrator RPA config folder](https://gitlab.cee.redhat.com/releng/konflux-release-data/-/tree/main/config/stone-prd-rh01.pg1f.p1/product/ReleasePlanAdmission/orchestrator-releng?ref_type=heads)
* Add a new RPA for the staging go operator and follow naming convention `operator-staging-1.x.yaml.`\
Example of staging RPA manifest for 1.5 release.
```yaml
---
apiVersion: appstudio.redhat.com/v1alpha1
kind: ReleasePlanAdmission
metadata:
  labels:
    release.appstudio.openshift.io/auto-release: "true"
    pp.engineering.redhat.com/business-unit: application-developer
  name: operator-staging-1-5
  namespace: rhtap-releng-tenant
spec:
  applications:
    - operator-1-5
  origin: orchestrator-releng-tenant
  policy: registry-orchestrator-releng
  data:
    releaseNotes:
      product_id: 851
      product_name: RHDH
      product_version: "1.5"
      type: "RHBA"
      synopsis: "Red Hat Developer Hub Orchestrator"
      topic: |
        The developer preview release of Red Hat Developer Hub Orchestrator.
      description: |
        Red Hat Developer Hub Orchestrator is a plugin that enables serverless asynchronous workflows to Backstage.
        This plugin is a development preview release.
      solution: |
        RHDH Orchestrator introduces serverless asynchronous workflows to Backstage, with a focus on facilitating the
        transition of applications to the cloud, onboarding developers, and enabling users to create workflows for
        backstage actions or external systems.
      references:
        - https://www.redhat.com/en/technologies/cloud-computing/developer-hub
        - https://rhdhorchestrator.io
    sign:
      configMapName: "hacbs-signing-pipeline-config-staging-redhatbeta2"
      cosignSecretName: konflux-cosign-signing-stage
    mapping:
      components:
        - name: controller-rhel9-operator-1-5
          repository: "registry.stage.redhat.io/rhdh-orchestrator-dev-preview-beta/controller-rhel9-operator"
        - name: orchestrator-operator-bundle-1-5
          repository: "registry.stage.redhat.io/rhdh-orchestrator-dev-preview-beta/orchestrator-operator-bundle"
      defaults:
        tags:
          - "1.5"
          - "1.5-{{ timestamp }}"
          - "{{ git_sha }}"
          - "{{ git_short_sha }}"
        pushSourceContainer: true
    pyxis:
      secret: pyxis-staging-secret
      server: stage
  pipeline:
    pipelineRef:
      resolver: git
      params:
        - name: url
          value: https://github.com/konflux-ci/release-service-catalog.git
        - name: revision
          value: production
        - name: pathInRepo
          value: "pipelines/managed/rh-advisories/rh-advisories.yaml"
    serviceAccountName: release-registry-staging
    timeouts:
      pipeline: "01h0m0s"
      tasks: 01h0m0s
```
* Add a new RPA for the production go operator and follow naming convention `operator-prod-1.x.yaml.`\
Example of production RPA manifest for 1.5 release.
```yaml
---
apiVersion: appstudio.redhat.com/v1alpha1
kind: ReleasePlanAdmission
metadata:
  labels:
    release.appstudio.openshift.io/auto-release: "true"
    pp.engineering.redhat.com/business-unit: application-developer
  name: operator-prod-1-5
  namespace: rhtap-releng-tenant
spec:
  applications:
    - operator-1-5
  origin: orchestrator-releng-tenant
  policy: registry-orchestrator-releng
  data:
    releaseNotes:
      product_id: 851
      product_name: RHDH
      product_version: "1.5"
      type: "RHBA"
      synopsis: "Red Hat Developer Hub Orchestrator"
      topic: |
        The developer preview release of Red Hat Developer Hub Orchestrator.
      description: |
        Red Hat Developer Hub Orchestrator is a plugin that enables serverless asynchronous workflows to Backstage.
        This plugin is a development preview release.
      solution: |
        RHDH Orchestrator introduces serverless asynchronous workflows to Backstage, with a focus on facilitating the
        transition of applications to the cloud, onboarding developers, and enabling users to create workflows for
        backstage actions or external systems.
      references:
        - https://www.redhat.com/en/technologies/cloud-computing/developer-hub
        - https://rhdhorchestrator.io
    sign:
      configMapName: "hacbs-signing-pipeline-config-redhatbeta2"
      cosignSecretName: konflux-cosign-signing-stage
    mapping:
      components:
        - name: controller-rhel9-operator-1-5
          repository: "registry.redhat.io/rhdh-orchestrator-dev-preview-beta/controller-rhel9-operator"
        - name: orchestrator-operator-bundle-1-5
          repository: "registry.redhat.io/rhdh-orchestrator-dev-preview-beta/orchestrator-operator-bundle"
      defaults:
        tags:
          - "1.5"
          - "1.5-{{ timestamp }}"
        pushSourceContainer: true
    pyxis:
      secret: pyxis-prod-secret
      server: production
  pipeline:
    pipelineRef:
      resolver: git
      params:
        - name: url
          value: "https://github.com/konflux-ci/release-service-catalog.git"
        - name: revision
          value: production
        - name: pathInRepo
          value: "pipelines/managed/rh-advisories/rh-advisories.yaml"
    serviceAccountName: release-registry-prod
    timeouts:
      pipeline: "01h0m0s"
      tasks: 01h0m0s
```
* Update the staging RPA for FBC Index.
  If necessary, update the RPA for the FBC index when we want to support a new OCP version.
  Update the existing `orchestrator-fbc-staging-index-v4-15-plus` by adding the new FBC under
  the `spec.applications` list.
```yaml
---
apiVersion: appstudio.redhat.com/v1alpha1
kind: ReleasePlanAdmission
metadata:
  labels:
    release.appstudio.openshift.io/auto-release: "true"
    pp.engineering.redhat.com/business-unit: application-developer
  name: orchestrator-fbc-staging-index-v4-15-plus
  namespace: rhtap-releng-tenant
spec:
  applications:
    - fbc-v4-15
    - fbc-v4-16
    - fbc-v4-17
  data:
    releaseNotes:
      product_id: 851
      product_name: RHDH
      product_version: fbc
      ...
  origin: orchestrator-releng-tenant
  policy: fbc-stage
```

* Update the production RPA for FBC Index.
If necessary, update the RPA for the FBC index when we want to support a new OCP version.
Update the existing `orchestrator-fbc-prod-index-v4-15-plus` by adding the new FBC under 
the `spec.applications` list.

```yaml
---
apiVersion: appstudio.redhat.com/v1alpha1
kind: ReleasePlanAdmission
metadata:
  labels:
    release.appstudio.openshift.io/auto-release: "true"
    pp.engineering.redhat.com/business-unit: application-developer
  name: orchestrator-fbc-prod-index-v4-15-plus
  namespace: rhtap-releng-tenant
spec:
  applications:
    - fbc-v4-15
    - fbc-v4-16
    - fbc-v4-17
  data:
    releaseNotes:
      product_id: 851
      product_name: RHDH
      product_version: fbc
      ...
  origin: orchestrator-releng-tenant
  policy: registry-orchestrator-fbc-prod-with-weekends
```

#### Create Merge Request
* After pushing the changes, create a merge request, have it reviewed approved. 
* After the prep branch is merged to main, ArgoCD will apply those change
and you should see the components in Konflux UI.


#### Prep Go Orchestrator Repo For Release
Once development is complete and QE gives the green light, create a branch from main
* Navigate to the [.tekton folder](https://github.com/rhdhorchestrator/orchestrator-go-operator/tree/main/.tekton) 
and update the pipeline files names to suffix with `xxx-1.x.yaml`
Example for 1.5 release branch
```console
- controller-rhel9-operator-on-pull-request-1-5.yaml
- controller-rhel9-operator-on-push-1-5.yaml
- orchestrator-operator-bundle-on-pull-request-1-5.yaml
- orchestrator-operator-bundle-on-push-1-5.yaml
```
* In each of the pipeline config file listed above, in the `pipelinesascode.tekton.dev/on-cel-expression`, 
update the `target_branch` from main to `release-1.x`
* Update the labels, name, component and any relevant change in each pipeline config.

Example of 1.5 `controller-rhel9-operator-on-pull-request-1-5.yaml`

```yaml
apiVersion: tekton.dev/v1
kind: PipelineRun
metadata:
  annotations:
    build.appstudio.openshift.io/repo: https://github.com/rhdhorchestrator/orchestrator-go-operator?rev={{revision}}
    build.appstudio.redhat.com/commit_sha: '{{revision}}'
    build.appstudio.redhat.com/pull_request_number: '{{pull_request_number}}'
    build.appstudio.redhat.com/target_branch: '{{target_branch}}'
    pipelinesascode.tekton.dev/max-keep-runs: "3"
    pipelinesascode.tekton.dev/on-cel-expression: event == "pull_request" && target_branch == "main" && ("Makefile".pathChanged() || "Dockerfile".pathChanged() || "config/***".pathChanged() || "internal/***".pathChanged() || ".tekton/controller-rhel9-operator-on-pull-request-1-5.yaml".pathChanged())
  creationTimestamp: null
  labels:
    appstudio.openshift.io/application: operator-1-5
    appstudio.openshift.io/component: controller-rhel9-operator-1-5
    pipelines.appstudio.openshift.io/type: build
  name: controller-rhel9-operator-on-pull-request-1-5
  namespace: orchestrator-releng-tenant
.....
```


These changes should be done via PR branch and merged into `release-1-5`


#### Update Main branch Via PR
* Navigate to the [.tekton folder](https://github.com/rhdhorchestrator/orchestrator-go-operator/tree/main/.tekton)
  and update the pipeline files names suffixed with the incremental (next) release `xxx-1.x.yaml`
Example assuming 1.6 is the next release.
```console
- controller-rhel9-operator-on-pull-request-1-6.yaml
- controller-rhel9-operator-on-push-1-6.yaml
- orchestrator-operator-bundle-on-pull-request-1-6.yaml
- orchestrator-operator-bundle-on-push-1-6.yaml
```
* In each of the pipeline config file listed above, in the `pipelinesascode.tekton.dev/on-cel-expression`,
  ensure the `target_branch` points to main.
* Update the labels, name, component and any relevant change in each pipeline config.

These changes should be done via PR branch and merged into `main` branch


## Releasing
Releasing the operator is a 3 stage operation:
* Build the container images using Konflux's pipelines as part of a PR merge.
  The bundle image needs to be built with the latest controller image so that
  they are matched. This is usually handled via Konflux's nudges that trigger
  PRs with an updated digest of the controller to the bundle after a successful
  controller PR is merged.
* Release these images to the staging repository. The images are basically
  inspected based on predefined rules by Konflux and deposited to the staging
  repository upon success.
* Release the
  [FBC](https://docs.openshift.com/container-platform/4.17/extensions/catalogs/fbc.html)
  (File Based Catalog) fragment to the RH catalog staging index using the
  pullspec of the images pushed to the staging registry.

Production releases builds on top of the staging releases to do more or less the
same, except that in this case it goes through a more detailed scrutiny, ending
up in the production registry. Past this step, it's the same FBC graph
generation using the image pullspec in production.

* Release these images to the production repository based on the same snapshot
  used to release to staging.
* Release the
  [FBC](https://docs.openshift.com/container-platform/4.17/extensions/catalogs/fbc.html)
  (File Based Catalog) fragment to the RH catalog index in production after
  updating the FBC graph file to include the new fragment like it was done in
  staging, but using the production image pullspec.

### Preparing the environment
The release of the orchestrator includes support for multiple versions within
the same repository. The git repository for the go operator defines branches
as `release-X.Y` as the location of the semantic X and Y version. In turn,
Konflux supports different versions by defining their own uniquely named
application, with each containing their own set of components, also unique in
their name, such as `controller-rhel9-operator-1-2` or
`controller-rhel9-operator-1-3`.

Thus, to make it easy and reusable, the release process defined in this page
needs to parametrize the names of the components so that the process can be
reused as much as possible. We will start the process by defining an environment
variable that contains the name of the application that holds the controller and
bundle. The next command lists all the applications in the workspace:
```
oc get application -ojsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}'
```
Example:
```console
fbc-v4-13
fbc-v4-14
fbc-v4-15
operator
operator-1-5
serverless-workflows
```
For our case, we'll use the `operator-1-5` application:
```console
applicationName=operator-1-5
releaseVersion=$(echo $applicationName| sed  's/operator//g')
```

Retrieve the names of the components that match the controller and bundle based
on the prefixed component names as we know them `controller-rhel9-operator` and
`orchestrator-operator-bundle`:
```console
controller_rhel9_operator=$(oc get components -ojsonpath='{range .items[?(@.spec.application=="'$applicationName'")]}{.metadata.name}{"\n"}{end}}'|grep controller-rhel9-operator)
orchestrator_operator_bundle=$(oc get components -ojsonpath='{range .items[?(@.spec.application=="'$applicationName'")]}{.metadata.name}{"\n"}{end}}'|grep orchestrator-operator-bundle)
echo "Controller component registered as $controller_rhel9_operator"
echo "Bundle component registered as $orchestrator_operator_bundle"
```

### Staging release
Initiating a staging release requires both a successful build and subsequent
integration tests for all components of the application. The result of this is a
snapshot object generated by Konflux. Think of a snapshot as a tag in git, but
in Konflux it is an object that contains the pullspect of all components of an
application, in our case the controller and bundle.

#### Releasing the container images to the staging registry
* Filter the latest snapshot. Keep in mind that we need to filter based on the
  bundle push event since that will most probably contain the nudged update from
  the controller. But first, let's capture the component names based on the
  application, since each release has its own component name associated to it.


Now we're ready to retrieve the snapshots and filter by those that were
triggered by a nudge. The following commands sorts all the snapshots for the
go operator application that were created as a result of a nudge, by timestamp
in ascending order and displays the name, integration tests success status and
the merge PR number and remote branch used in the commit.
```console
oc get snapshots --sort-by .metadata.creationTimestamp -l pac.test.appstudio.openshift.io/event-type=push,appstudio.openshift.io/component=$orchestrator_operator_bundle -ojsonpath='{range .items[*]}{@.metadata.name}{"\t"}{@.status.conditions[?(@.type=="AppStudioTestSucceeded")].status}{"\t"}{@.metadata.annotations.pac\.test\.appstudio\.openshift\.io/sha-title}{"\n"}{end}' | grep rhdhorchestrator/konflux/component-updates
```
Example:
```console
operator-1-5-n9n6h	True	Merge pull request #262 from rhdhorchestrator/konflux/component-updates/controller-rhel9-operator-1-5
operator-1-5-cf7qp	True	Merge pull request #267 from rhdhorchestrator/konflux/component-updates/controller-rhel9-operator-1-5
```

If you're releasing from a controller's update nudge, which is the most probable
case, check the last snapshot that has passed the integration tests:
```console
operator-1-5-cf7qp	True	Merge pull request #267 from rhdhorchestrator/konflux/component-updates/controller-rhel9-operator-1-5
```
Capture the snapshot in an environment variable:
```console
snapshot=operator-1-5-cf7qp
```

* Ensure that the bundle's controller pullspec matches the one in the snapshot.
  The bundle's container image contains a label with the image pullspec of the
  controller used in the bundle. Use the following commands to extract the
  controller from the bundle of the snapshot `operator-1-5-cf7qp`:
```console
bundle=$(oc get snapshot $snapshot -ojsonpath='{.spec.components[?(@.name=="'$orchestrator_operator_bundle'")].containerImage}')
controllerInBundle=$(skopeo inspect docker://$bundle --format "{{.Labels.controller}}")
controllerSHAInBundle=$(awk -F'@' '{print $2}' <<< "$controllerInBundle")
controllerInSnapshot=$(oc get snapshot $snapshot -ojsonpath='{.spec.components[?(@.name=="'$controller_rhel9_operator'")].containerImage}')
controllerSHAInSnapshot=$(awk -F'@' '{print $2}' <<< "$controllerInSnapshot")
if [ -n "$controllerInBundle" ] && [ "$controllerSHAInBundle" = "$controllerSHAInSnapshot" ]; then echo "controller image pullspec matches";else echo "controller image pullspec does not match. This snapshot is not a good candidate for release";fi
```

* Verify that the bundle and controller release label also matches. Run the
  following command to extract and compare the release label for the bundle and
  controller images.
```console
releaseLabelInBundle=$(skopeo inspect docker://$bundle --format "{{.Labels.release}}")
releaseLabelInController=$(skopeo inspect docker://$controllerInSnapshot --format "{{.Labels.release}}")
if [ -n "$controllerInBundle" ] && [ "$releaseLabelInBundle" = "$releaseLabelInController" ]; then echo "bundle and controller release label matches";else echo "bundle and controller release label does not match. This snapshot is not a good candidate for release";fi
```

* Create a new Release manifest for staging
```console
releaseName=$(bash -c "oc create -f - <<EOF
apiVersion: appstudio.redhat.com/v1alpha1
kind: Release
metadata:
  generateName: operator-staging$releaseVersion-
  namespace: orchestrator-releng-tenant
spec:
  releasePlan: operator-staging$releaseVersion
  snapshot: $snapshot
EOF" | awk '{print $1}')
```

* Wait for the release to be validated:
```console
while [ "$(oc get $releaseName -ojsonpath='{.status.conditions[?(@.type=="Released")].reason}')" == "Progressing" ];do sleep 5;done
[[ "$(oc get $releaseName -ojsonpath='{.status.conditions[?(@.type=="Released")].reason}')" == "Failed" ]] && echo Release failed: $(oc get $releaseName -ojsonpath='{.status.conditions[?(@.type=="Released")].message}') || echo "Release successful"
```

If the release fails, follow the [troubleshooting](#release-pipeline-failed)
guide to find out the root cause.

* Validate that the container images are in the staging registry by inspecting
  them with Skopeo. The pullspec of the images are defined in the advisory
  manifest:

```console
advisoryURL=$(oc get $releaseName -ojsonpath='{.status.artifacts.advisory.url}' | sed  's/blob/raw/')
controllerPullSpec=$(curl $advisoryURL | yq '.spec.content[] | with_entries(select(.value.repository | test("controller-rhel9-operator$") ))| .[].containerImage')
bundlePullSpec=$(curl $advisoryURL | yq '.spec.content[] | with_entries(select(.value.repository | test("orchestrator-operator-bundle$") ))| .[].containerImage')
skopeo inspect docker://$controllerPullSpec >/dev/null && echo "Controller image found in $controllerPullSpec" || echo "Controller image not found in $controllerPullSpec"
skopeo inspect docker://$bundlePullSpec >/dev/null && echo "Bundle image found in $bundlePullSpec" || echo "Controller image not found in $bundlePullSpec"
```

At this point we can confirm that we have a successful release of the images and
that they are ready to be used to release a new FBC version.

#### Releasing a new FBC index to staging
To release a new version of the operator in the Red Hat Catalog, you'll have to
release an updated FBC graph of the IIB catalog. For staging, this means you'll
end up having to add a new IIB catalog source to your cluster so that your new
operator is available for consumption. In production however, this is not
necessary as the release is published directly to the production index.

Note, if you haven't yet released the operator in production, you'll need to
follow this
[documentation](https://gitlab.cee.redhat.com/konflux/docs/users/-/blob/main/topics/releasing/preparing-for-release.md#publishing-a-fbc-graph)
to request your operator to be added to the production index. It is not added by
default.


* Clone the orchestrator-fbc repository:

```console
git clone https://github.com/rhdhorchestrator/orchestrator-fbc.git
```

* Update the graph.yaml in the OCP version following the FBC documentation to
  ensure that each version published has an upgrade path. Check [this
  page](https://docs.openshift.com/container-platform/4.17/extensions/catalogs/fbc.html#olm-channel-schema_fbc)
  to understand the different options when updating the fragment. The most
  common case is when updating the [z-stream
  version](https://github.com/rhdhorchestrator/orchestrator-fbc/pull/92), in
  which case you will have to amend the original fragment (graph.yaml) and
  define the linkage between releases, so that the newest one is marked as a
  replacement to the previous one, and so on. So if we wanted to add the new
  release as `1.2.0-rc11` to the current graph.yaml, we'd be adding a value in
  the `entries:` section, and another pair for the `image` and `schema` with the
  pullspec of the bundle. Note that you should have the digest of the bundle
  image in `$bundlePullSpec`.

```console
---
defaultChannel: alpha
icon:
  base64data: PD94bW....
name: orchestrator-operator
schema: olm.package
---
entries:
- name: orchestrator-operator.v1.2.0-rc11
  replaces: orchestrator-operator.v1.2.0-rc10
- name: orchestrator-operator.v1.2.0-rc10
  replaces: orchestrator-operator.v1.2.0-rc9
- name: orchestrator-operator.v1.2.0-rc9
name: alpha
package: orchestrator-operator
schema: olm.channel
---
image: registry.redhat.io/rhdh-orchestrator-dev-preview-beta/orchestrator-operator-bundle@sha256:e8196126d48692ab2f451ad5ef8033ffc14c89fd9b139615fe5a8a75166b1405
schema: olm.bundle
# orchestrator-operator v.1.5.0-rc9
---
image: registry.redhat.io/rhdh-orchestrator-dev-preview-beta/orchestrator-operator-bundle@sha256:0f109419f233bf3a27e50ef9d1bc8f3bee5ce61b391014cbb52070a90606e08f
schema: olm.bundle
# orchestrator-operator v.1.5.0-rc10
---
image: registry.redhat.io/rhdh-orchestrator-dev-preview-beta/orchestrator-operator-bundle@sha256:5ee318302c87a7ee36c3d620f9c01ac2288c5d59e63ae95fde47c0d172fa13ea
schema: olm.bundle
# orchestrator-operator v.1.5.0-rc11
```


* Run the `generate-fbc.sh --render <OCP version>` command to generate the new
  catalog and then update the resulting catalog manifest to ensure that it
  references the staging repository for the controller. Review the changes and
  revert any reference to the
  `quay.io/redhat-user-workloads/orchestrator-releng-tenant/operator`
  pullspec in the generated images, if any. This is a leftover from the first
  publishes of the catalog where the initial bundle was referencing this
  pullspec instead of staging or production.

* Create a PR with the changes and merge it once it's green. Ensure that the
  on-push and ECP pipelines finish before proceeding. You'll need the snapshot
  generated from the ECP pipeline to release the FBC fragment to the index.

* Follow these steps for each OCP version:

  * Identify the snapshot that contains the PR you just merged:

  ```console
  applicationName=fbc-v4-14
  oc get snapshots --sort-by .metadata.creationTimestamp -l pac.test.appstudio.openshift.io/event-type=push,appstudio.openshift.io/application=$applicationName -ojsonpath='{range .items[*]}{@.metadata.name}{"\t"}{@.status.conditions[?(@.type=="AppStudioTestSucceeded")].status}{"\t"}{@.metadata.annotations.pac\.test\.appstudio\.openshift\.io/sha-title}{"\n"}{end}'
  ```

  Example:

  ```console
  ...
  ...
  fbc-v4-14-5p7m9	True	Merge pull request #81 from jordigilh/release/1.2.0-rc6
  fbc-v4-14-jv6f8	True	Merge pull request #83 from rhdhorchestrator/konflux/references/main
  fbc-v4-14-dhxqb	True	Merge pull request #82 from rhdhorchestrator/konflux/component-updates/operator-bundle
  fbc-v4-14-bdx8p	True	Merge pull request #85 from rhdhorchestrator/konflux/component-updates/operator-bundle
  fbc-v4-14-hftq5	True	Merge pull request #84 from rhdhorchestrator/konflux/references/main
  fbc-v4-14-g6b2z	True	Merge pull request #86 from jordigilh/release/1.2.0-rc9
  fbc-v4-14-kttjb	True	Merge pull request #87 from jordigilh/release/ocp_4.14_rc9
  fbc-v4-14-mcncx	True	Merge pull request #88 from jordigilh/release/orchestrator-rc9_ocp_prod
  fbc-v4-14-hr78w	True	Merge pull request #90 from jordigilh/release/1.2.0-rc10
  fbc-v4-14-6lhrt	True	Merge pull request #91 from jordigilh/4.15/fix_dockerfile_path
  fbc-v4-14-rjwkj	True	Merge pull request #92 from jordigilh/release/stage/1.2.0-rc11
  ```

  The last one matches the source branch for the PR and its Integration Tests
  are successful.

  ```console
  fbc-v4-14-rjwkj	True	Merge pull request #92 from jordigilh/release/stage/1.2.0-rc11
  ```

  * Create a new Release manifest for staging
  ```console
  snapshot=fbc-v4-14-rjwkj

  releaseName=$(bash -c "oc create -f - <<EOF
  apiVersion: appstudio.redhat.com/v1alpha1
  kind: Release
  metadata:
    generateName: $applicationName-
    namespace: orchestrator-releng-tenant
  spec:
    releasePlan: $applicationName-release-as-staging-fbc
    snapshot: $snapshot
  EOF" | awk '{print $1}')
  ```

  * Wait for the release to be validated:
  ```console
  while [ "$(oc get $releaseName -ojsonpath='{.status.conditions[?(@.type=="Released")].reason}')" == "Progressing" ];do sleep 5;done
  [[ "$(oc get $releaseName -ojsonpath='{.status.conditions[?(@.type=="Released")].reason}')" == "Failed" ]] && echo Release failed: $(oc get $releaseName -ojsonpath='{.status.conditions[?(@.type=="Released")].message}') || echo "Release successful"
  ```
  If the release fails, follow the [troubleshooting](#release-pipeline-failed)
  guide to find out the root cause.

  * Extract the catalog IIB container image pullspec digest from the `status` of
    the release.

  ```console
  imagePullSpec=$(oc get $releaseName -ojsonpath={.status.artifacts.index_image.index_image_resolved})
  ```

  * With the retrieved container image pullspec stored in `$imagePullSpec`, run
    the following command to generate a new `catalogsource` that references the
    staging catalog and deploy it in your cluster:

  ```console
  oc create -f - <<EOF
  apiVersion: operators.coreos.com/v1alpha1
  kind: CatalogSource
  metadata:
    name: orchestrator-operator
    namespace: openshift-marketplace
  spec:
    displayName: Orchestrator Operator
    publisher: Red Hat
    sourceType: grpc
    grpcPodConfig:
      securityContextConfig: restricted
    image: $imagePullSpec
    updateStrategy:
      registryPoll:
        interval: 10m
  EOF
  ```

  * To deploy the operator, using the CLI, deploy a subscription that references
    the `orchestrator-operator` as the `source` or use the OLM UI in your OCP
    environment:

  ```console
  apiVersion: operators.coreos.com/v1alpha1
  kind: Subscription
  metadata:
    name: orchestrator-operator
    namespace: openshift-operators
  spec:
    channel: alpha
    installPlanApproval: Automatic
    name: orchestrator-operator
    source: orchestrator-operator
    sourceNamespace: openshift-marketplace
  ```


### Production release
A prerequisite to release to production is for the controller and bundle images
to have been released to the staging registry. There is no shortcut that
bypasses staging for this. However, releasing the image to the production
registry does not deviate much from releasing to staging, even though its
required to release to staging before production.

#### Releasing the container images to the production registry
Releasing to production requires the images to be processed in staging first.
Once that's successful, the process resolves in taking the staging snapshots
from the go operator application and creating a new release using the
production RPA. The FBC follows a similar step in that it needs a release aiming
at the production RPA for each OCP release using the same snapshot. Let's start
with the go operator application and then move on to the FBC release:

* Identify the snapshot used in the stage release. List all the releases for
  staging and extract the snapshot used for that release. We will be using this
  snapshot for the production release. The next command will list all releases
  in staging that were successful sorted by `creationTimestamp` in ascending
  order (latest are the newest releases).

```console
oc get release --sort-by .metadata.creationTimestamp | grep operator-staging$releaseVersion |grep Succeeded
```

Example:
```console
operator-staging-1-5-1df4m         operator-1-5-hrv2d   operator-staging-1-5          Succeeded        12h
operator-staging-1-5-sx9c8         operator-1-5-gsxpz   operator-staging-1-5          Succeeded        20m
```

We'll use the snapshot `operator-1-5-gsxpz` for the production release,
being the last successful release.

```console
snapshot=operator-1-5-gsxpz
```
* Create a new release manifest for Production:
```console
releaseName=$(bash -c "oc create -f - <<EOF
apiVersion: appstudio.redhat.com/v1alpha1
kind: Release
metadata:
  generateName: operator-prod$releaseVersion-
  namespace: orchestrator-releng-tenant
spec:
  releasePlan: operator-prod$releaseVersion
  snapshot: $snapshot
EOF" | awk '{print $1}')
```

* Wait for the release to be validated:

You can also use the
[UI](https://console.redhat.com/application-pipeline/workspaces/orchestrator-releng/applications/)
to view the status of the release as it is being processed in the pipeline (for
example here the [1.5 release](https://console.redhat.com/application-pipeline/workspaces/orchestrator-releng/applications/operator-1-5/releases)).

```console
while [ "$(oc get $releaseName -ojsonpath='{.status.conditions[?(@.type=="Released")].reason}')" == "Progressing" ];do sleep 5;done
[[ "$(oc get $releaseName -ojsonpath='{.status.conditions[?(@.type=="Released")].reason}')" == "Failed" ]] && echo Release failed: $(oc get $releaseName -ojsonpath='{.status.conditions[?(@.type=="Released")].message}') || echo "Release successful"
```

If the release fails, follow the [troubleshooting](#release-pipeline-failed)
guide to find out the root cause.

* Validate that the container images are in the production registry by
  inspecting them with Skopeo. The pullspec of the images are defined in the
  advisory manifest:

```console
advisoryURL=$(oc get $releaseName -ojsonpath='{.status.artifacts.advisory.url}' | sed  's/blob/raw/')
controllerPullSpec=$(curl $advisoryURL | yq '.spec.content[] | with_entries(select(.value.repository | test("controller-rhel9-operator$") ))| .[].containerImage')
bundlePullSpec=$(curl $advisoryURL | yq '.spec.content[] | with_entries(select(.value.repository | test("orchestrator-operator-bundle$") ))| .[].containerImage')
skopeo inspect docker://$controllerPullSpec >/dev/null && echo "Controller image found in $controllerPullSpec" || echo "Controller image not found in $controllerPullSpec"
skopeo inspect docker://$bundlePullSpec >/dev/null && echo "Bundle image found in $bundlePullSpec" || echo "Controller image not found in $bundlePullSpec"
```

At this point, the container images have been pushed to the production registry.
What's left is to update the FBC graph to aim for production registry, with no
changes to the digest.


#### Releasing a new FBC index to production
Releasing the fragment is a simple step to update the FBC graph manifest to
point to the production registry and run the command to generate the catalog.
The latest commit in the repo should reflect the bundle's container image
pullspec being the same as the one in the snapshot we retrieved from the stage
build.

```console
---
defaultChannel: alpha
icon:
  base64data: PD94bW....
name: orchestrator-operator
schema: olm.package
---
entries:
- name: orchestrator-operator.v1.2.0-rc11
  replaces: orchestrator-operator.v1.2.0-rc10
- name: orchestrator-operator.v1.2.0-rc10
  replaces: orchestrator-operator.v1.2.0-rc9
- name: orchestrator-operator.v1.2.0-rc9
name: alpha
package: orchestrator-operator
schema: olm.channel
---
image: registry.redhat.io/rhdh-orchestrator-dev-preview-beta/orchestrator-operator-bundle@sha256:e8196126d48692ab2f451ad5ef8033ffc14c89fd9b139615fe5a8a75166b1405
schema: olm.bundle
# orchestrator-operator v.1.5.0-rc9
---
image: registry.redhat.io/rhdh-orchestrator-dev-preview-beta/orchestrator-operator-bundle@sha256:0f109419f233bf3a27e50ef9d1bc8f3bee5ce61b391014cbb52070a90606e08f
schema: olm.bundle
# orchestrator-operator v.1.5.0-rc10
---
image: registry.redhat.io/rhdh-orchestrator-dev-preview-beta/orchestrator-operator-bundle@sha256:5ee318302c87a7ee36c3d620f9c01ac2288c5d59e63ae95fde47c0d172fa13ea
schema: olm.bundle
# orchestrator-operator v.1.5.0-rc11
```

* Run the `generate-fbc.sh --render <OCP version>` command to generate the new
  catalog and then update the resulting catalog manifest to ensure that it
  references the production repository for the controller. Review the changes
  and revert any reference to the
  `quay.io/redhat-user-workloads/orchestrator-releng-tenant/operator`
  pullspec in the generated images, if any. This is a leftover from the first
  publishes of the catalog where the initial bundle was referencing this
  pullspec instead of staging or production.

* Create a PR with the changes and merge it once it's green. Ensure that the
  on-push and ECP pipelines finish before proceeding. You'll need the snapshot
  generated from the ECP pipeline to add the FBC fragment to the production
  index.

* Follow these steps for each OCP version:

  * Identify the snapshot that contains the PR you just merged:

  ```console
  applicationName=fbc-v4-14
  oc get snapshots --sort-by .metadata.creationTimestamp -l pac.test.appstudio.openshift.io/event-type=push,appstudio.openshift.io/application=$applicationName -ojsonpath='{range .items[*]}{@.metadata.name}{"\t"}{@.status.conditions[?(@.type=="AppStudioTestSucceeded")].status}{"\t"}{@.metadata.annotations.pac\.test\.appstudio\.openshift\.io/sha-title}{"\n"}{end}'
  ```

  Example:

  ```console
  ...
  ...
  fbc-v4-14-5p7m9	True	Merge pull request #81 from jordigilh/release/1.2.0-rc6
  fbc-v4-14-jv6f8	True	Merge pull request #83 from rhdhorchestrator/konflux/references/main
  fbc-v4-14-dhxqb	True	Merge pull request #82 from rhdhorchestrator/konflux/component-updates/operator-bundle
  fbc-v4-14-bdx8p	True	Merge pull request #85 from rhdhorchestrator/konflux/component-updates/operator-bundle
  fbc-v4-14-hftq5	True	Merge pull request #84 from rhdhorchestrator/konflux/references/main
  fbc-v4-14-g6b2z	True	Merge pull request #86 from jordigilh/release/1.2.0-rc9
  fbc-v4-14-kttjb	True	Merge pull request #87 from jordigilh/release/ocp_4.14_rc9
  fbc-v4-14-mcncx	True	Merge pull request #88 from jordigilh/release/orchestrator-rc9_ocp_prod
  fbc-v4-14-hr78w	True	Merge pull request #90 from jordigilh/release/1.2.0-rc10
  fbc-v4-14-6lhrt	True	Merge pull request #91 from jordigilh/4.15/fix_dockerfile_path
  fbc-v4-14-rjwkj	True	Merge pull request #92 from jordigilh/release/stage/1.2.0-rc11
  fbc-v4-14-k3ksj	True	Merge pull request #93 from jordigilh/release/prod/1.2.0-rc11
  ```

  The last one matches the source branch for the PR and its Integration Tests
  are successful.

  ```console
  fbc-v4-14-k3ksj	True	Merge pull request #93 from jordigilh/release/prod/1.2.0-rc11
  ```

  * Create a new Release manifest for production
  ```console
  snapshot=fbc-v4-14-k3ksj

  releaseName=$(bash -c "oc create -f - <<EOF
  apiVersion: appstudio.redhat.com/v1alpha1
  kind: Release
  metadata:
    generateName: $applicationName-
    namespace: orchestrator-releng-tenant
  spec:
    releasePlan: $applicationName-release-as-production-fbc
    snapshot: $snapshot
  EOF" | awk '{print $1}')
  ```

  * Wait for the release to be validated:

  You can also use the
  [UI](https://console.redhat.com/application-pipeline/workspaces/orchestrator-releng/applications)
  to view the status of the release as it is being processed in the pipeline 
  (for example here the [1.5 release](https://console.redhat.com/application-pipeline/workspaces/orchestrator-releng/applications/operator-1-5/releases)).
  ```console
  while [ "$(oc get $releaseName -ojsonpath='{.status.conditions[?(@.type=="Released")].reason}')" == "Progressing" ];do sleep 5;done
  [[ "$(oc get $releaseName -ojsonpath='{.status.conditions[?(@.type=="Released")].reason}')" == "Failed" ]] && echo Release failed: $(oc get $releaseName -ojsonpath='{.status.conditions[?(@.type=="Released")].message}') || echo "Release successful"
  ```

  If the release fails, follow the [troubleshooting](#release-pipeline-failed)
  guide to find out the root cause.

  And that's all: the operator FBC's fragment has been added to the Red Hat
  Catalog. It will take some minutes for clusters to pull the new image and make
  the operator available.

## Troubleshooting
This section is meant to grow as more experience is gained in Konflux. For now,
the main goal is to describe the steps to identify which tasks and capture the
error message generated by Konflux. If you need help on Konflux, open an "Ask
for support" ticket in #konflux-users.

### Release pipeline failed

If the release fails, you'll need to identify which task failed and why. This
gets a bit tricky as you'll have to jump over different objects until you get
the information you seek. First, you'll need to get the pipelinerun from the
status in the release. The following command will list the failed tasksrun
objects for the pipelinerun

```console
pipelineRunName=$(oc get $releaseName -ojsonpath='{.status.processing.pipelineRun}{"\n"}' | awk -F'/' '{print $2}')
oc get taskrun -n rhtap-releng-tenant -l tekton.dev/pipelineRun=$pipelineRunName -ojsonpath='{range .items[*]}{.status.conditions[?(@.type=="Succeeded")].status}{"\t"}{.metadata.name}{"\n"}{end}' | awk '{ if ($1=="False") print $2 }'
```

For each failed task, follow these steps to determine the problem:

* Retrieve the pod. Remember to define `failedTask` with the name of the task
  from the previous command.
```console
taskRunPodName=$(oc get taskrun $failedTask -n rhtap-releng-tenant -ojsonpath='{.status.podName}')
```

* Retrieve the logs from the pod. Each task has different containers, so in some
  cases you'll have to dig in to find out which is the container that has the
  logs. For instance, the `verify-enteprise-contract` task has the logs in the
  `step-validate` container. The `rh-sign-image` has only one container, so
  there is no need to specify any.

```console
oc logs $taskRunPodName -n rhtap-releng-tenant -c step-validate
```

* Analyze the logs and determine the cause of the failure This is an example of
  a failed enterprise contract taskrun:

```console
Success: false
Result: FAILURE
Violations: 1, Warnings: 15, Successes: 196

Components:
- Name: orchestrator-operator-bundle
  ImageRef: quay.io/redhat-user-workloads/orchestrator-releng-tenant/operator/orchestrator-operator-bundle@sha256:0...
  Violations: 1, Warnings: 7, Successes: 98

- Name: controller-rhel9-operator
  ImageRef: quay.io/redhat-user-workloads/orchestrator-releng-tenant/operator/controller-rhel9-operator@sha256:f...
  Violations: 0, Warnings: 8, Successes: 98

Results:
✕ [Violation] olm.allowed_registries
  ImageRef: quay.io/redhat-user-workloads/orchestrator-releng-tenant/operator/orchestrator-operator-bundle@sha256:0...
  Reason: The
  "quay.io/redhat-user-workloads/orchestrator-releng-tenant/operator/controller-rhel9-operator@sha256:f...
  CSV image reference is not from an allowed registry.
  Title: Images referenced by OLM bundle are from allowed registries
  Description: Each image referenced by the OLM bundle should match an entry in the list of prefixes defined by the rule data key
  `allowed_registry_prefixes` in your policy configuration. To exclude this rule add
  "olm.allowed_registries:quay.io/redhat-user-workloads/orchestrator-releng-tenant/operator/controller-rhel9-operator" to the
  `exclude` section of the policy configuration.
  Solution: Use image from an allowed registry, or modify your xref:ec-cli:ROOT:configuration.adoc#_data_sources[policy
  configuration] to include additional registry prefixes.
```

### Cluster fails to pull images from production registry (registry.redhat.io) because images are only in staging
Deploy this [ImageDigestMirrorSet](imagedigestmirrorset.yaml) to your cluster to
configure the cluster to use the staging registry as mirror to the production
registry.

## Command tips
* Retrieve the components for a given application:
```console
oc get components -ojsonpath='{range .items[?(@.spec.application=="operator-1-5")]}{.metadata.name}{"\n"}{end}'
```
