#!/usr/bin/env bash

# Copyright 2026 The Tekton Authors
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

set -o errexit
set -o nounset
set -o pipefail

readonly REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
readonly CONTROLLER_TOOLS_VERSION="v0.20.1"
readonly YQ_VERSION="v4.45.1"
readonly CRD_FILES=(
  300-trigger.yaml
  300-triggerbinding.yaml
  300-clustertriggerbinding.yaml
  300-triggertemplate.yaml
  300-eventlistener.yaml
  300-interceptor.yaml
  300-clusterinterceptor.yaml
)

check_only=false
if [[ $# -gt 1 ]]; then
  echo "usage: $0 [--check]" >&2
  exit 2
fi
if [[ $# -eq 1 ]]; then
  if [[ "$1" != "--check" ]]; then
    echo "usage: $0 [--check]" >&2
    exit 2
  fi
  check_only=true
fi

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT
mkdir -p "$tmpdir/bin" "$tmpdir/input" "$tmpdir/schema" "$tmpdir/columns"
cd "$REPO_ROOT"

for file in "${CRD_FILES[@]}"; do
  cp "$REPO_ROOT/config/$file" "$tmpdir/input/$file"
done

GOBIN="$tmpdir/bin" GOFLAGS= go install "sigs.k8s.io/controller-tools/cmd/controller-gen@$CONTROLLER_TOOLS_VERSION"
GOBIN="$tmpdir/bin" GOFLAGS= go install "github.com/mikefarah/yq/v4@$YQ_VERSION"

"$tmpdir/bin/controller-gen" \
  schemapatch:manifests="$tmpdir/input",generateEmbeddedObjectMeta=true \
  paths=./pkg/apis/triggers/... \
  output:dir="$tmpdir/schema"
"$tmpdir/bin/controller-gen" \
  crd:crdVersions=v1,maxDescLen=0 \
  paths=./pkg/apis/triggers/... \
  output:dir="$tmpdir/columns"
# schemapatch preserves CRD metadata; copy marker-generated columns by version name.
for file in "${CRD_FILES[@]}"; do
  plural="$("$tmpdir/bin/yq" '.spec.names.plural' "$tmpdir/schema/$file")"
  COLUMNS_FILE="$tmpdir/columns/triggers.tekton.dev_$plural.yaml" "$tmpdir/bin/yq" -i '
    load(strenv(COLUMNS_FILE)).spec.versions as $generated |
    .spec.versions[] |= (.name as $name |
      .additionalPrinterColumns = ($generated[] | select(.name == $name) | .additionalPrinterColumns))
  ' "$tmpdir/schema/$file"
done

# EventListener accepts a partial Pod, including a container without a name.
# Remove nested Pod descriptions to stay below kubectl apply's annotation limit.
"$tmpdir/bin/yq" -i '
  (.spec.versions[].schema.openAPIV3Schema.properties.spec.properties.resources.properties.kubernetesResource.properties.spec.properties.template.properties.spec) |= (
    .required -= ["containers"] |
    .properties.containers.items.required -= ["name"] |
    .properties.containers."x-kubernetes-list-type" = "atomic" |
    del(.properties.containers."x-kubernetes-list-map-keys") |
    del(.properties[] | .. | select(tag == "!!map") | .description)
  )
' "$tmpdir/schema/300-eventlistener.yaml"

if [[ "$check_only" == true ]]; then
  drift=false
  for file in "${CRD_FILES[@]}"; do
    if ! cmp -s "$REPO_ROOT/config/$file" "$tmpdir/schema/$file"; then
      echo "Showing at most the first 40 diff lines for $file." >&2
      diff -u "$REPO_ROOT/config/$file" "$tmpdir/schema/$file" | sed -n '1,40p' >&2 || true
      drift=true
    fi
  done
  if [[ "$drift" == true ]]; then
    echo "CRD schemas are out of date; run hack/update-schemas.sh" >&2
    exit 1
  fi
  echo "CRD schemas are up to date."
  exit 0
fi

for file in "${CRD_FILES[@]}"; do
  cp "$tmpdir/schema/$file" "$REPO_ROOT/config/$file"
done
