#!/usr/bin/env bash

set -e
source $(dirname $0)/../vendor/github.com/tektoncd/plumbing/scripts/e2e-tests.sh
source $(dirname $0)/resolve-yamls.sh

set -x

readonly API_SERVER=$(oc config view --minify | grep server | awk -F'//' '{print $2}' | awk -F':' '{print $1}')
readonly OPENSHIFT_REGISTRY="${OPENSHIFT_REGISTRY:-"registry.svc.ci.openshift.org"}"
readonly TEST_NAMESPACE=tekton-triggers-tests
readonly TEKTON_TRIGGERS_NAMESPACE=tekton-pipelines
readonly KO_DOCKER_REPO=image-registry.openshift-image-registry.svc:5000/tektoncd-triggers
# Where the CRD will install the triggers
readonly TEKTON_NAMESPACE=tekton-pipelines
# Variable usually set by openshift CI but generating one if not present when running it locally
readonly OPENSHIFT_BUILD_NAMESPACE=${OPENSHIFT_BUILD_NAMESPACE:-tektoncd-build-$$}
# Yaml test skipped due of not being able to run on openshift CI, usually becaus
# of rights.
# test-git-volume: `"gitRepo": gitRepo volumes are not allowed to be used]'
declare -ar SKIP_YAML_TEST=(test-git-volume)

function install_tekton_triggers() {
  header "Installing Tekton Triggers"

  create_triggers

  wait_until_pods_running $TEKTON_TRIGGERS_NAMESPACE || return 1

  header "Tekton Triggers Installed successfully"
}

function create_triggers() {
  resolve_resources config/ tekton-triggers-resolved.yaml "nothing" $OPENSHIFT_REGISTRY/$OPENSHIFT_BUILD_NAMESPACE/stable
  oc apply -f tekton-triggers-resolved.yaml
}

function create_test_namespace() {
  for ns in ${TEKTON_NAMESPACE} ${OPENSHIFT_BUILD_NAMESPACE} ${TEST_NAMESPACE};do
     oc get project ${ns} >/dev/null 2>/dev/null || oc new-project ${ns}
  done

  oc policy add-role-to-group system:image-puller system:serviceaccounts:$TEST_NAMESPACE -n $OPENSHIFT_BUILD_NAMESPACE
}

create_test_namespace

[[ -z ${E2E_DEBUG} ]] && install_tekton_triggers


function run_go_e2e_tests() {
  header "Running Go e2e tests"
  go test -v -count=1 -tags=e2e -timeout=20m ./test --kubeconfig $KUBECONFIG || return 1
}

run_go_e2e_tests || failed=1

((failed)) && exit 1

success
