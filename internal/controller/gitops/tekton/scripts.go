package tekton

const gitCLITaskScript = `
#!/usr/bin/env sh
set -eu

if [ "${PARAM_VERBOSE}" = "true" ] ; then
  set -x
fi

if [ "${WORKSPACE_BASIC_AUTH_DIRECTORY_BOUND}" = "true" ] ; then
  cp "${WORKSPACE_BASIC_AUTH_DIRECTORY_PATH}/.git-credentials" "${PARAM_USER_HOME}/.git-credentials"
  cp "${WORKSPACE_BASIC_AUTH_DIRECTORY_PATH}/.gitconfig" "${PARAM_USER_HOME}/.gitconfig"
  chmod 400 "${PARAM_USER_HOME}/.git-credentials"
  chmod 400 "${PARAM_USER_HOME}/.gitconfig"
fi

if [ "${WORKSPACE_SSH_DIRECTORY_BOUND}" = "true" ] ; then
  cp -R "${WORKSPACE_SSH_DIRECTORY_PATH}" "${PARAM_USER_HOME}"/.ssh
  chmod 700 "${PARAM_USER_HOME}"/.ssh
  chmod -R 400 "${PARAM_USER_HOME}"/.ssh/*
fi

# Setting up the config for the git.
git config --global user.email "$(params.GIT_USER_EMAIL)"
git config --global user.name "$(params.GIT_USER_NAME)"

eval '$(params.GIT_SCRIPT)'

RESULT_SHA="$(git rev-parse HEAD | tr -d '\n')"
EXIT_CODE="$?"
if [ "$EXIT_CODE" != 0 ]
then
  exit $EXIT_CODE
fi
# Make sure we don't add a trailing newline to the result!
printf "%s" "$RESULT_SHA" > "$(results.commit.path)"
`
const flattenerTaskScript = `
ROOT=/workspace/workflow
TARGET=flat
mkdir -p flat

if [ -d "workflow/$(params.workflowId)" ]; then
  cp -r workflow/$(params.workflowId)/src/main/resources flat/$(params.workflowId)
  cp workflow/$(params.workflowId)/LICENSE flat/$(params.workflowId)
else
  cp -r workflow/src/main/resources flat/$(params.workflowId)
  cp workflow/LICENSE flat/$(params.workflowId)
fi

if [ "$(params.convertToFlat)" == "false" ]; then
  rm -rf workflow/src/main/resources
  mv workflow/src flat/$(params.workflowId)/
fi

ls flat/$(params.workflowId)

curl -L https://raw.githubusercontent.com/parodos-dev/serverless-workflows/v1.3.x/pipeline/workflow-builder.Dockerfile -o flat/workflow-builder.Dockerfile
`

const buildManifestTaskScript = `
microdnf install -y tar gzip
KN_CLI_URL="https://developers.redhat.com/content-gateway/file/pub/cgw/serverless-logic/1.33.0/kn-workflow-linux-amd64.tar.gz"
curl -L "$KN_CLI_URL" | tar -xz --no-same-owner && chmod +x kn-workflow-linux-amd64 && mv kn-workflow-linux-amd64 kn-workflow
./kn-workflow gen-manifest --namespace ""
`

const buildGitOpsTaskScript = `
cp $(workspaces.workflow-source.path)/flat/$(params.workflowId)/manifests/* kustomize/base
microdnf install -y findutils && microdnf clean all
cd kustomize
./updater.sh $(params.workflowId) $(params.imageTag)
`