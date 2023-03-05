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
forwardingPort=8888
elName=""

info() {
  echo; echo "*** Example ${current_example_version}/${current_example}: $@ ***"; echo;
}

err() {
  echo; echo "ERROR: Example ${current_example_version}/${current_example}: $@"; echo;
}

install_knative_serving() {
  # Install Knative by referring https://knative.dev/docs/admin/install/serving/install-serving-with-yaml/#install-the-knative-serving-component
  kubectl apply -f https://github.com/knative/serving/releases/download/knative-v1.6.0/serving-crds.yaml
  kubectl apply -f https://github.com/knative/serving/releases/download/knative-v1.6.0/serving-core.yaml

  # Wait for pods to be running in the namespaces we are deploying to
  wait_until_pods_running knative-serving || fail_test "Knative Serving did not come up"

  kubectl apply -f https://github.com/knative/net-kourier/releases/download/knative-v1.6.0/kourier.yaml

  kubectl patch configmap/config-network \
    --namespace knative-serving \
    --type merge \
    --patch '{"data":{"ingress.class":"kourier.ingress.networking.knative.dev"}}'

  # Wait for pods to be running in the namespaces we are deploying to
  wait_until_pods_running knative-serving || fail_test "Knative Serving & Kourier did not come up"
  # Changing port of kourier service to use 8888 instead of 80 so that port-forward can be done easily
  # Because with 80 its giving permission deneid and in Knative all traffic goes via gateway which is kourier here.
  kubectl -n kourier-system patch service/kourier --type=json -p="[{"op": "add", "path": "/spec/ports/0/port", "value": $forwardingPort}]"
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
  kubectl wait --for=condition=Ready --timeout=60s eventlisteners/${elName}  || {
    err "eventlistener failed to get in running state"
    exit 1
  }
  
  kubectl get eventlisteners
}

ignoreExamples=( cron trigger-ref triggergroups )

port_forward_and_curl() {
sleep 10
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

  # Wait a few seconds for resources to show up
  sleep 10

  tr=$(kubectl get taskruns -A -l triggers.tekton.dev/triggers-eventid=${eventID} -o name)
  pr=$(kubectl get pipelineruns -A -l triggers.tekton.dev/triggers-eventid=${eventID} -o name)

  if [ -z "$tr" ] && [ -z "$pr" ]
  then
     err "failed to create taskrun/pipelinerun"
     kubectl logs -l "eventlistener=${current_example}-listener"
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

curl_knative_service() {
  # port forwarding to access application
  kubectl -n kourier-system port-forward service/kourier $forwardingPort:$forwardingPort > /dev/null 2>&1 &
  sleep 1
  hostURL=$(kubectl get el ${elName} -o=jsonpath='{.status.address.url}')
  host=$(echo $(echo $hostURL | tr "://" "\n") | cut -d' ' -f 2)

  bash ${REPO_ROOT_DIR}/examples/${current_example_version}/${current_example}/curl.sh $forwardingPort $host

  # kill the process which ran eith port-forwarding
  ps -ef | grep $forwardingPort | grep -v grep | awk '{print $2}' | xargs kill
}

kill_process() {
  kill $port_forward_pid || true
}

trap "cleanup" EXIT SIGINT
cleanup() {
  info "Cleaning up resources"
   # kill $port_forward_pid
  for y in ${yaml_files}; do
    kubectl delete -f ${y} --ignore-not-found || true
  done
}

# Assumptions:
# Name of example would be name of directory
# Name of eventlistener must be (exampleName)-listener
main() {
  install_knative_serving

  versions="v1alpha1 v1beta1"
  # List of examples test will run on
  examples_v1alpha1="bitbucket-server cron embedded-trigger github gitlab label-selector namespace-selector trigger-ref"
  examples_v1beta1="${examples_v1alpha1} slack bitbucket-cloud triggergroups github-add-changed-files-pr github-add-changed-files-push-cel github-owners"
  # examples_v1alpha1=""
  # examples_v1beta1="slack"
  create_example_pipeline
  for v in ${versions}; do
    current_example_version=${v}
    examples=examples_$v
    echo "Applying examples for version: ${v}"
    for e in ${!examples}; do
      current_example=${e}
      echo "*** Example ${current_example_version}/${current_example} ***";
      apply_files
      check_eventlistener
      port_forward_and_curl
      kill_process
      cleanup
      info "Test Successful"
    done
    # To test Knative Serving example
    info "Knative Example Test Started"
    current_example="custom-resource"
    echo "*** Example ${current_example_version}/${current_example} ***";
    apply_files
    check_eventlistener
    curl_knative_service
    cleanup
    info "Knative Example Test Successful"
  done

  echo; echo "*** Completed Examples Test Successfully ***"; echo;
}

main $@
