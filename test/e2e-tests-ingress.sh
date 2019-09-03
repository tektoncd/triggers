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

source $(dirname $0)/e2e-common.sh

# Waits until taskrun completed.
# Parameters: $1 - taskrun name
function wait_until_taskrun_completed() {
  echo -e "Waiting until taskrun $1 completed\n"
  for i in {1..150}; do  # timeout after 5 minutes
    taskrun_status="$(kubectl get taskrun $1 -o=jsonpath='{.status.conditions[0]}')"
    match=$(echo $taskrun_status | grep "message:All Steps have completed executing reason:Succeeded status:True type:Succeeded")  || true
    l=${#match}
    if [ 0 -ne $l ];
    then
      return 0
    fi
    sleep 2
  done
  echo -e "\n\nERROR: timeout waiting for taskrun successful completion\n"
  return 1
}

# Waits until pod started.
# Parameters: $1 - pod name prefix
function wait_until_pod_started() {
  echo -e "Waiting until pod started\n"
  for i in {1..150}; do  # timeout after 5 minutes
    pod_status=$(kubectl get pod | grep $1 | grep "Running") || true
    l=${#pod_status}
    if [ 0 -ne $l ];
    then
      return 0
    fi
    sleep 2
  done
  echo -e "\n\nERROR: timeout waiting for pod successful start\n"
  return 1
}

set -o errexit
set -o pipefail

# verify if the yaml file is valid
for op in apply delete;do 
  kubectl ${op} -f ${REPO_ROOT_DIR}/docs/create-webhook.yaml
done

# make sure no remaining resources from the previous run
echo "clean up before start. Ignore (NotFound) errors"
kubectl delete secret secret1 || true
kubectl delete eventlistener listener || true
kubectl delete taskrun create-webhook || true
kubectl delete secret githubsecret || true

# setup
kubectl apply -f ${REPO_ROOT_DIR}/test/ingress/ingress-clusterrole.yaml
kubectl apply -f ${REPO_ROOT_DIR}/test/ingress/ingress-clusterrolebinding.yaml
kubectl apply -f ${REPO_ROOT_DIR}/docs/create-webhook.yaml
kubectl create secret generic githubsecret --from-literal=accessToken=ff7d2c2949844f68cb18a68f4febad4454df2336 --from-literal=userName=tektonuser


# test
kubectl apply -f ${REPO_ROOT_DIR}/docs/create-webhook-run.yaml
wait_until_taskrun_completed create-webhook 

# check certificate
echo -e "Testing certificate"
crt=$(kubectl get secret secret1 -o=jsonpath='{.data.tls\.crt}')
echo $crt | base64 --decode | grep "\-\-\-\-\-BEGIN CERTIFICATE\-\-\-\-\-"
echo $crt | base64 --decode | grep "\-\-\-\-\-END CERTIFICATE\-\-\-\-\-"

key=$(kubectl get secret secret1 -o=jsonpath='{.data.tls\.key}')
echo $key | base64 --decode | grep "\-\-\-\-\-BEGIN RSA PRIVATE KEY\-\-\-\-\-"
echo $key | base64 --decode | grep "\-\-\-\-\-END RSA PRIVATE KEY\-\-\-\-\-"
echo -e "Certificate is OK"

# check ingress
svc=$(kubectl get ingress listener -o=jsonpath='{.spec.rules[0].http.paths[0].backend.serviceName}')
host=$(kubectl get ingress listener -o=jsonpath='{.spec.rules[0].host}')
tlshost=$(kubectl get ingress listener -o=jsonpath='{.spec.tls[0].hosts[0]}')
secret=$(kubectl get ingress listener -o=jsonpath='{.spec.tls[0].secretName}')
if [ $svc != "listener" ] || [ $host != "listener.192.168.0.1.nip.io" ] || [ $tlshost != "listener.192.168.0.1.nip.io" ] || [ $secret != "secret1" ]; then
  echo -e "unexpected values " "wanted: listener; got:" $svc", wanted: listener.192.168.0.1.nip.io; got:" $host", wanted: listener.192.168.0.1.nip.io; got:" $tlshost", wanted: secret1; got:" $secret
  exit 1
fi

# check event listener
listenername=$(kubectl get eventlistener listener -o=jsonpath='{.metadata.name}')
bindingname=$(kubectl get eventlistener listener -o=jsonpath='{.spec.triggers[0].binding.name}')
templatename=$(kubectl get eventlistener listener -o=jsonpath='{.spec.triggers[0].template.name}')
if [ $listenername != "listener" ] || [ $bindingname != "pipeline-binding" ] || [ $templatename != "pipeline-template" ]; then
  echo -e "unexpected values " "wanted: listener; got:" $listenername", wanted: pipeline-binding; got:" $bindingname", wanted: pipeline-template; got:" $templatename
  exit 1
fi

# Checking EventListener log
wait_until_pod_started listener
log=$(kubectl logs $(kubectl get pod | grep listener | cut -f 1 -d " "))
entry=$(echo $log | grep "Listen and serve on port 8082")  || true
ll=${#entry}
if [ 0 -eq $ll ];
then
  echo "Event Listener POD didn't start expectedly"
  echo "POD dump:"
  kubectl get pod listener -o yaml
  exit 1
fi


# clean up
kubectl delete -f ${REPO_ROOT_DIR}/test/ingress/ingress-clusterrole.yaml
kubectl delete -f ${REPO_ROOT_DIR}/test/ingress/ingress-clusterrolebinding.yaml
kubectl delete -f ${REPO_ROOT_DIR}/docs/create-webhook.yaml
kubectl delete -f ${REPO_ROOT_DIR}/docs/create-webhook-run.yaml
kubectl delete secret secret1
kubectl delete eventlistener listener
kubectl delete secret githubsecret


