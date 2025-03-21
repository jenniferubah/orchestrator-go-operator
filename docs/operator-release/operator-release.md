# Releasing the GO Operator
This outlines the process of releasing a version of the go operator.


## Preparing the code for releasing

1. Pull a fresh copy of the repository. Alternatively pull the latest from main on your existing repository and ensure that the HEAD matches the upstream's HEAD commit hash.
1. Create a new branch, example `release-1.5.0-rc1`.
1. Update the version tag field in Makefile to the version you want it to be.
1. To validate all files are synced, run `make generate manifest`. If there are changes to any files, halt release and verify those changes.
1. Run `make bundle`. This updates the contents of the `/bundle` with the new version tag.
1. Review the changes and if satisfied, push the commit.
1. Create a new PR against main, unless the changes are targeting a specific release.
1. Get the PR reviewed by another team member. Two more pair of eyes are always welcome for these kind of things.
1. Merge the PR.

At this point releasing the go operator can branch into 2 scenarios:
* Manual release for local consumption. This kind of releases are only meant to be used for local development or early QE testing, not for general consumption in the RH catalog.
* Konflux managed release for staging and production environments. It uses the Konflux pipelines to bundle the images to the Red Hat Operator Ecosystems Catalog.

## Konflux release (for downstream)

Follow the [Konflux release documentation](docs/konflux/release_operator_with_konflux.md) for staging and production releases using Konflux.

## Manual release (for upstream only)
1. Switch to the main branch and pull the changes so that your fork and upstream are in sync and contain the new additions.
1. Run the following commands in an AMD64 environment. 
These commands will build the controller image, push it to its `quay.io/orchestrator/orchestrator-go-operator` [repository](https://quay.io/repository/orchestrator/orchestrator-go-operator?tab=tags),\
build the bundle (update the contents of `/bundle` based on the information in `/config`), build the bundle image and push it to its [repository](https://quay.io/repository/orchestrator/orchestrator-go-operator-bundle?tab=tags),\
and finally build the catalog container image and push it to its [repository](https://quay.io/repository/orchestrator/orchestrator-go-operator-catalog?tab=tags).

```shell
make docker-build docker-push bundle bundle-build bundle-push catalog-build catalog-push
```

1. Navigate to the [catalog repository](https://quay.io/repository/orchestrator/orchestrator-go-operator-catalog?tab=tags) and locate the latest build image. \
The last modified value should give it away but worth checking just in case the push failed (e.g. podman could not authenticate against quay.io because credentials have expired). \ 
In these cases, retry pushing the images manually.
1. Retrieve the SHA256 digest (e.g. `sha256:370334e33390a79dc514ab762a89b3d5fe10eb39d9a60831c8774659f37e5c56` ) and create a new catalog source manifest that points to the new image:
```yaml
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
  image: quay.io/orchestrator/orchestrator-go-operator-catalog@sha256:370334e33390a79dc514ab762a89b3d5fe10eb39d9a60831c8774659f37e5c56
  updateStrategy:
    registryPoll:
      interval: 10m
```
1. Deploy the `catalogsource` on a cluster and ensure that the latest version in the OLM menu for the orchestrator go operator matches with the new version of the operator.
1. Install the operator and ensure the controller pod is running without errors and ready to receive a CR.
2. Optionally, create and apply sample CR. Validate the CR deploys successfully by checking its status.
3. Optionally, you can take it further a notch and validate the related objects also successfully deploy.
1. Crate a Jira ticket and add attach the catalogsoure to it. In the description box, summarize the changes contained in the release. \
Example of Jira ticket: https://issues.redhat.com/browse/FLPATH-2107
1. Share the new manifest and Jira ticket in the development channel to announce the new release. Tag the QE team so that they are aware and can take action as soon as they are able.

