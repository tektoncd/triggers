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
current_example_version=""
yaml_files=""
port_forward_pid=""

info() {
  echo; echo "*** Example ${current_example_version}/${current_example}: $@ ***"; echo;
}

err() {
  echo; echo "ERROR: Example ${current_example_version}/${current_example}: $@"; echo;
}

apply_files() {
  info "Applying Resources"
  folder=${REPO_ROOT_DIR}/examples/${current_example_version}/${current_example}
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
  kubectl wait --for=condition=Ready --timeout=20s eventlisteners/${elName}  || {
    err "eventlistener failed to get in running state"
    exit 1
  }
  kubectl get eventlisteners
}

ignoreExamples=( cron trigger-ref v1alpha1-task )

port_forward_and_curl() {

  if [[ " ${ignoreExamples[@]} " =~ " ${current_example} " ]]; then
    info 'ignoring for port forwarding'
    return
  fi

  info "Port forwarding to execute curl command"

  kubectl port-forward service/el-${current_example}-listener 8080 &
  port_forward_pid=$!

  # Wait a few seconds for forwarding to happen
  sleep 3

  response=$(bash ${REPO_ROOT_DIR}/examples/${current_example_version}/${current_example}/curl.sh)

  echo; echo "response received: $response"; echo

  eventID=$(jq -n "$response" | jq --raw-output '.eventID')

  if [ -z "$eventID" ]
  then
     err "failed to get eventID"
     exit 1
  fi

  tr=$(kubectl get taskruns -A -l triggers.tekton.dev/triggers-eventid=${eventID} -o name)
  pr=$(kubectl get pipelineruns -A -l triggers.tekton.dev/triggers-eventid=${eventID} -o name)

  if [ -z "$tr" ] && [ -z "$pr" ]
  then
     err "failed to create taskrun/pipelinerun"
     exit 1
  fi

  if [ -z "$tr" ]
  then
    echo "PipelineRun created : $pr"
  else
    echo "Taskrun created : $tr"
  fi

  kill $port_forward_pid
}

create_example_pipeline() {
  kubectl apply -f ${REPO_ROOT_DIR}/examples/example-pipeline.yaml || {
      err "failed to apply example pipeline"
      exit 1
  }
}

trap "cleanup" EXIT SIGINT
cleanup() {
  info "Cleaning up resources"
  kill $port_forward_pid || true
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
  examples="bitbucket cron embedded-trigger github gitlab label-selector namespace-selector v1alpha1-task trigger-ref"
  create_example_pipeline
  for v in ${versions}; do
    current_example_version=${v}
    echo "Applying examples for version: ${v}"
    for e in ${examples}; do
      current_example=${e}
      echo "*** Example ${current_example_version}/${current_example} ***";
      apply_files
      check_eventlistener
      port_forward_and_curl
      cleanup
      info "Test Successful"
    done
  done

  echo; echo "*** Completed Examples Test Successfully ***"; echo;
}

main $@
