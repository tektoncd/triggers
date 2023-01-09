source $(dirname $0)/e2e-common.sh

set -x
set -o errexit
set -o pipefail

current_example=""
current_example_version=""

apply_files() {
  echo "Applying Resources"
  yaml_files=$(find ${REPO_ROOT_DIR}/examples/${current_example_version}/${current_example} -type f \( -iname "*.yaml" ! -iname "rbac.yaml" \) | sort) || {
    echo "failed to find files"
    exit 1
  }
  for y in ${yaml_files}; do
    kubectl apply  -f ${y} || {
      echo "failed to apply ${y}"
      exit 1
    }
  done
}

delete_files() {
  echo "Cleaning up resources"
  for y in ${yaml_files}; do
    kubectl delete -f ${y} --ignore-not-found || true
  done
}

main() {
  versions="v1alpha1 v1beta1"
  examples="triggertemplates triggerbindings eventlisteners clustertriggerbindings"
  for v in ${versions}; do
    current_example_version=${v}
    echo "Applying examples for version: ${v}"
    for e in ${examples}; do
      current_example=${e}
      echo "*** Example ${current_example_version}/${current_example} ***";
      apply_files
      delete_files
      echo "Test Successful"
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
}

main $@
