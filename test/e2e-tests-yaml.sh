source $(dirname $0)/e2e-common.sh

set -x
set -o errexit
set -o pipefail

for op in apply delete;do
    # Apply TriggerTemplates
    for file in $(find ${REPO_ROOT_DIR}/examples/triggertemplates/ -name *.yaml | sort); do
        kubectl ${op} -f ${file}
    done

    # Apply TriggerBindings
    for file in $(find ${REPO_ROOT_DIR}/examples/triggerbindings/ -name *.yaml | sort); do
        kubectl ${op} -f ${file}
    done

    # Apply EventInterceptors
    for file in $(find ${REPO_ROOT_DIR}/examples/event-interceptors/ -name *.yaml | sort); do
        kubectl ${op} -f ${file}
    done

    # Apply EventListeners
    for file in $(find ${REPO_ROOT_DIR}/examples/eventlisteners/ -name *.yaml | sort); do
        kubectl ${op} -f ${file}
    done

    # Apply ClusterTriggerBindings
    for file in $(find ${REPO_ROOT_DIR}/examples/clustertriggerbindings/ -name *.yaml | sort); do
        kubectl ${op} -f ${file}
    done

done

# Test getting-started examples that use the `getting-started` namespace
kubectl create namespace getting-started
for op in apply delete;do
  for file in $(find ${REPO_ROOT_DIR}/docs/getting-started -name *.yaml | sort); do
      kubectl ${op} -f ${file}
  done
done
kubectl delete namespace getting-started
