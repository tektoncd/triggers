#!/usr/bin/env bash

# Copyright 2021 The Tekton Authors
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

# This script calls out to scripts in tektoncd/plumbing to setup a cluster
# and deploy Tekton Pipelines to it for running integration tests.

source $(dirname $0)/e2e-common.sh
# Script entry point.

set -u -e -o pipefail -x

current_example=""
yaml_files=""

info() {
  echo; echo "*** Example ${current_example}: $@ ***"; echo;
}

err() {
  echo; echo "ERROR: Example ${current_example}: $@"; echo;
}

apply_files() {
  info "Applying Resources"
  local version=$1
  folder=${REPO_ROOT_DIR}/examples/${version}/${current_example}
  yaml_files=$(find ${folder}/ -name *.yaml | sort) || {
    err "failed to find files"
    exit 1
  }
  for y in ${yaml_files}; do
    kubectl apply  -f ${y} || {
      err "failed to apply ${y}"
      exit 1
    }
  done
}

check_eventlistener() {
  info "Waiting for EventListener to be available"

  elName=${current_example}-listener
  kubectl wait --for=condition=Ready --timeout=20s eventlisteners/${elName} || {
    err "eventlistener failed to get in running state"
    exit 1
  }
  kubectl get eventlisteners
}

trap "cleanup" EXIT SIGINT
cleanup() {
  info "Cleaning up resources"

  for y in ${yaml_files}; do
    kubectl delete -f ${y} --ignore-not-found || true
  done
}

# Assumptions:
# Name of example would be name of directory
# Name of eventlistener must be (exampleName)-listener
main() {
  versions="v1alpha1 v1beta1"
  # List of examples test will run on
  examples="bitbucket cron embedded-trigger github gitlab label-selector namespace-selector trigger-ref v1alpha1-task"

  for v in ${versions}; do
    echo "Applying examples for version: ${v}"
    for e in ${examples}; do
      current_example=${e}
      echo "*** Example ${current_example} ***";
      apply_files ${v}
      check_eventlistener
      cleanup
      info "Test Successful"
    done
  done
}

main $@

