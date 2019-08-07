source $(dirname $0)/e2e-common.sh

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

    # Apply EventListeners
    for file in $(find ${REPO_ROOT_DIR}/examples/eventlisteners/ -name *.yaml | sort); do
    kubectl ${op} -f ${file}
    done
done