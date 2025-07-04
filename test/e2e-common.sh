#!/usr/bin/env bash

# Copyright 2019 The Tekton Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# This script runs the presubmit tests; it is started by prow for each PR.
# For convenience, it can also be executed manually.
# Running the script without parameters, or with the --all-tests
# flag, causes all tests to be executed, in the right order.
# Use the flags --build-tests, --unit-tests and --integration-tests
# to run a specific set of tests.

# Helper functions for E2E tests.

# Check if we have a specific RELEASE_YAML global environment variable to use
# instead of detecting the latest released one from tektoncd/pipeline releases
RELEASE_YAML=${RELEASE_YAML:-}
SKIP_SECURITY_CTX=${SKIP_SECURITY_CTX:="false"}
source $(dirname $0)/../vendor/github.com/tektoncd/plumbing/scripts/e2e-tests.sh

function install_pipeline_crd() {
  local latestreleaseyaml
  echo ">> Deploying Tekton Pipelines"
  if [[ -n ${RELEASE_YAML} ]];then
	  latestreleaseyaml=${RELEASE_YAML}
  else
    latestreleaseyaml="https://storage.googleapis.com/tekton-releases/pipeline/latest/release.yaml"
  fi
  [[ -z ${latestreleaseyaml} ]] && fail_test "Could not get latest released release.yaml"
  kubectl apply -f ${latestreleaseyaml} ||
    fail_test "Tekton pipeline installation failed"

  # Make sure that eveything is cleaned up in the current namespace.
  for res in tasks pipelines taskruns pipelineruns; do
    kubectl delete --ignore-not-found=true ${res}.tekton.dev --all
  done

  # Wait for pods to be running in the namespaces we are deploying to
  wait_until_pods_running tekton-pipelines || fail_test "Tekton Pipeline did not come up"
}

function install_triggers_crd() {
  echo ">> Deploying Tekton Triggers"
  rel=$(mktemp)
  release=$(mktemp)
  ko resolve -f config/ > "${rel}" || fail_test "Tekton Triggers build failed"

  if [ "${SKIP_SECURITY_CTX}" == "true" ]; then
      yq 'del(.spec.template.spec.containers[]?.securityContext.runAsUser, .spec.template.spec.containers[]?.securityContext.runAsGroup)' "${rel}" > "${release}"
  else
      cat "${rel}" > "${release}"
  fi

  kubectl apply -f "${release}" || fail_test "Tekton Triggers installation failed"

  # Wait for the Interceptors CRD to be available before adding the core-interceptors
  kubectl wait --for=condition=Established --timeout=30s crds/clusterinterceptors.triggers.tekton.dev
  ko resolve -f config/interceptors > "${rel}" || fail_test "Core interceptors build failed"

  if [ "${SKIP_SECURITY_CTX}" == "true" ]; then
      kubectl patch configmap config-defaults-triggers -n tekton-pipelines --type='merge' -p='{"data":{"default-run-as-user":"","default-fs-group":"", "default-run-as-group":""}}'
      yq 'del(.spec.template.spec.containers[]?.securityContext.runAsUser, .spec.template.spec.containers[]?.securityContext.runAsGroup)' "${rel}" > "${release}"
  else
      cat "${rel}" > "${release}"
  fi

  kubectl apply -f "${release}" || fail_test "Core interceptors installation failed"

  # Make sure that eveything is cleaned up in the current namespace.
  for res in eventlistener triggertemplate triggerbinding clustertriggerbinding; do
    kubectl delete --ignore-not-found=true ${res}.triggers.tekton.dev --all
  done

  rm -f "${rel}" "${release}"
  # Wait for pods to be running in the namespaces we are deploying to
  wait_until_pods_running tekton-pipelines || fail_test "Tekton Triggers did not come up"
}
