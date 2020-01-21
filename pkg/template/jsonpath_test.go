package template

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

var objects = `{"a":"v\r\n烈","c":{"d":"e"},"empty": "","null": null, "number": 42}`
var arrays = `[{"a": "b"}, {"c": "d"}, {"e": "f"}]`

// Checks that we print JSON strings when the JSONPath selects
// an array or map value and regular values otherwise
func TestParseJSONPath(t *testing.T) {
	var objectBody = fmt.Sprintf(`{"body":%s}`, objects)
	var arrayBody = fmt.Sprintf(`{"body":%s}`, arrays)
	tests := []struct {
		name string
		expr string
		in   string
		want string
	}{{
		name: "objects",
		in:   objectBody,
		expr: "$(body)",
		// TODO: Do we need to escape backslashes for backwards compat?
		want: objects,
	}, {
		name: "array of objects",
		in:   arrayBody,
		expr: "$(body)",
		want: arrays,
	}, {
		name: "array of values",
		in:   `{"body": ["a", "b", "c"]}`,
		expr: "$(body)",
		want: `["a", "b", "c"]`,
	}, {
		name: "string values",
		in:   objectBody,
		expr: "$(body.a)",
		want: "v\\r\\n烈",
	}, {
		name: "empty string",
		in:   objectBody,
		expr: "$(body.empty)",
		want: "",
	}, {
		name: "numbers",
		in:   objectBody,
		expr: "$(body.number)",
		want: "42",
	}, {
		name: "booleans",
		in:   `{"body": {"bool": true}}`,
		expr: "$(body.bool)",
		want: "true",
	}, {
		name: "null values",
		in:   objectBody,
		expr: "$(body.null)",
		want: "null",
	}, {
		name: "multiple results",
		in:   arrayBody,
		expr: "$(body[:2])",
		want: `[{"a": "b"}, {"c": "d"}]`,
	}, {
		name: "multiple results with empty string",
		in:   `{"body":["", "some", "thing"]}`,
		expr: "$(body[:2])",
		want: `["", "some"]`,
	}, {
		name: "multiple results newlines/special chars",
		in:   `{"body":["", "v\r\n烈", "thing"]}`,
		expr: "$(body[:2])",
		want: `["", "v\r\n烈"]`,
	}, {
		name: "multiple results with null",
		in:   `{"body":["", null, "thing"]}`,
		expr: "$(body[:2])",
		want: `["", null]`,
	}, {
		name: "Array filters",
		in:   `{"body":{"taskRun":{"apiVersion":"tekton.dev/v1alpha1","kind":"TaskRun","metadata":{"annotations":{"kubectl.kubernetes.io/last-applied-configuration":"{\"apiVersion\":\"tekton.dev/v1alpha1\",\"kind\":\"Task\",\"metadata\":{\"annotations\":{},\"name\":\"publish-tekton-pipelines\",\"namespace\":\"default\"},\"spec\":{\"inputs\":{\"params\":[{\"description\":\"The vX.Y.Z version that the artifacts should be tagged with (including v)\",\"name\":\"versionTag\"},{\"description\":\"TODO(#569) This is a hack to make it easy for folks to switch the registry being used by the many many image outputs\",\"name\":\"imageRegistry\"},{\"description\":\"The path to the folder in the go/src dir that contains the project, which is used by ko to name the resulting images\",\"name\":\"pathToProject\"}],\"resources\":[{\"name\":\"source\",\"targetPath\":\"go/src/github.com/tektoncd/pipeline\",\"type\":\"git\"},{\"name\":\"bucket\",\"type\":\"storage\"}]},\"outputs\":{\"resources\":[{\"name\":\"bucket\",\"type\":\"storage\"},{\"name\":\"builtBaseImage\",\"type\":\"image\"},{\"name\":\"builtEntrypointImage\",\"type\":\"image\"},{\"name\":\"builtKubeconfigWriterImage\",\"type\":\"image\"},{\"name\":\"builtCredsInitImage\",\"type\":\"image\"},{\"name\":\"builtGitInitImage\",\"type\":\"image\"},{\"name\":\"builtControllerImage\",\"type\":\"image\"},{\"name\":\"builtWebhookImage\",\"type\":\"image\"},{\"name\":\"builtDigestExporterImage\",\"type\":\"image\"},{\"name\":\"builtPullRequestInitImage\",\"type\":\"image\"},{\"name\":\"builtGcsFetcherImage\",\"type\":\"image\"},{\"name\":\"notification\",\"type\":\"cloudEvent\"}]},\"steps\":[{\"args\":[\"--dockerfile=/workspace/go/src/github.com/tektoncd/pipeline/images/Dockerfile\",\"--destination=$(inputs.params.imageRegistry)/$(inputs.params.pathToProject)/$(outputs.resources.builtBaseImage.url)\",\"--context=/workspace/go/src/github.com/tektoncd/pipeline\"],\"command\":[\"/kaniko/executor\"],\"env\":[{\"name\":\"GOOGLE_APPLICATION_CREDENTIALS\",\"value\":\"/secret/release.json\"}],\"image\":\"gcr.io/kaniko-project/executor:v0.9.0\",\"name\":\"build-push-base-images\",\"volumeMounts\":[{\"mountPath\":\"/secret\",\"name\":\"gcp-secret\"}]},{\"image\":\"busybox\",\"name\":\"create-ko-yaml\",\"script\":\"#!/bin/sh\\nset -ex\\n\\ncat \\u003c\\u003cEOF \\u003e /workspace/go/src/github.com/tektoncd/pipeline/.ko.yaml\\n# By default ko will build images on top of distroless\\nbaseImageOverrides:\\n  $(inputs.params.pathToProject)/$(outputs.resources.builtCredsInitImage.url): $(inputs.params.imageRegistry)/$(inputs.params.pathToProject)/build-base:latest\\n  $(inputs.params.pathToProject)/$(outputs.resources.builtGitInitImage.url): $(inputs.params.imageRegistry)/$(inputs.params.pathToProject)/build-base:latest\\n  $(inputs.params.pathToProject)/$(outputs.resources.builtEntrypointImage.url): busybox # image should have shell in $PATH\\nbaseBuildOverrides:\\n  $(inputs.params.pathToProject)/$(outputs.resources.builtControllerImage.url):\\n    env:\\n      - name: CGO_ENABLED\\n        value: 1\\n    flags:\\n      - name: ldflags\\n        value: \\\"-X $(inputs.params.pathToProject)/pkg/version.PipelineVersion=$(inputs.params.versionTag)\\\"\\nEOF\\n\\ncat /workspace/go/src/github.com/tektoncd/pipeline/.ko.yaml\\n\"},{\"args\":[\"-r\",\"/workspace/bucket\",\"/workspace/output/\"],\"command\":[\"cp\"],\"image\":\"busybox\",\"name\":\"link-input-bucket-to-output\"},{\"args\":[\"-p\",\"/workspace/output/bucket/latest/\",\"/workspace/output/bucket/previous/\"],\"command\":[\"mkdir\"],\"image\":\"busybox\",\"name\":\"ensure-release-dirs-exist\"},{\"env\":[{\"name\":\"KO_DOCKER_REPO\",\"value\":\"$(inputs.params.imageRegistry)\"},{\"name\":\"GOPATH\",\"value\":\"/workspace/go\"},{\"name\":\"GO111MODULE\",\"value\":\"off\"},{\"name\":\"CLOUDSDK_AUTH_CREDENTIAL_FILE_OVERRIDE\",\"value\":\"/secret/release.json\"}],\"image\":\"gcr.io/tekton-releases/dogfooding/ko:gcloud-latest\",\"name\":\"run-ko\",\"script\":\"#!/usr/bin/env bash\\nset -ex\\n\\n# Auth with CLOUDSDK_AUTH_CREDENTIAL_FILE_OVERRIDE\\ngcloud auth configure-docker\\n\\n# ko requires this variable to be set in order to set image creation timestamps correctly https://github.com/google/go-containerregistry/pull/146\\nexport SOURCE_DATE_EPOCH=date +%s\\n\\n# Change to directory with our .ko.yaml\\ncd /workspace/go/src/github.com/tektoncd/pipeline\\n\\n# For each cmd/* directory, include a full gzipped tar of all source in\\n# vendor/. This is overkill. Some deps' licenses require the source to be\\n# included in the container image when they're used as a dependency.\\n# Rather than trying to determine which deps have this requirement (and\\n# probably get it wrong), we'll just targz up the whole vendor tree and\\n# include it. As of 9/20/2019, this amounts to about 11MB of additional\\n# data in each image.\\nTMPDIR=$(mktemp -d)\\ntar cvfz ${TMPDIR}/source.tar.gz vendor/\\nfor d in cmd/*; do\\n  ln -s ${TMPDIR}/source.tar.gz ${d}/kodata/\\ndone\\n\\n# Publish images and create release.yaml\\nko resolve --preserve-import-paths -t $(inputs.params.versionTag) -f /workspace/go/src/github.com/tektoncd/pipeline/config/ \\u003e /workspace/output/bucket/latest/release.yaml\\n\",\"volumeMounts\":[{\"mountPath\":\"/secret\",\"name\":\"gcp-secret\"}]},{\"image\":\"busybox\",\"name\":\"copy-to-tagged-bucket\",\"script\":\"#!/bin/sh\\nset -ex\\n\\nmkdir -p /workspace/output/bucket/previous/$(inputs.params.versionTag)/\\ncp /workspace/output/bucket/latest/release.yaml /workspace/output/bucket/previous/$(inputs.params.versionTag)/release.yaml\\n\",\"workingDir\":\"/workspace/output/bucket\"},{\"image\":\"google/cloud-sdk\",\"name\":\"tag-images\",\"script\":\"#!/usr/bin/env bash\\nset -ex\\n\\nREGIONS=(us eu asia)\\nIMAGES=(\\n  $(inputs.params.imageRegistry)/$(inputs.params.pathToProject)/$(outputs.resources.builtEntrypointImage.url):$(inputs.params.versionTag)\\n  $(inputs.params.imageRegistry)/$(inputs.params.pathToProject)/$(outputs.resources.builtKubeconfigWriterImage.url):$(inputs.params.versionTag)\\n  $(inputs.params.imageRegistry)/$(inputs.params.pathToProject)/$(outputs.resources.builtCredsInitImage.url):$(inputs.params.versionTag)\\n  $(inputs.params.imageRegistry)/$(inputs.params.pathToProject)/$(outputs.resources.builtGitInitImage.url):$(inputs.params.versionTag)\\n  $(inputs.params.imageRegistry)/$(inputs.params.pathToProject)/$(outputs.resources.builtControllerImage.url):$(inputs.params.versionTag)\\n  $(inputs.params.imageRegistry)/$(inputs.params.pathToProject)/$(outputs.resources.builtWebhookImage.url):$(inputs.params.versionTag)\\n  $(inputs.params.imageRegistry)/$(inputs.params.pathToProject)/$(outputs.resources.builtDigestExporterImage.url):$(inputs.params.versionTag)\\n  $(inputs.params.imageRegistry)/$(inputs.params.pathToProject)/$(outputs.resources.builtPullRequestInitImage.url):$(inputs.params.versionTag)\\n  $(inputs.params.imageRegistry)/$(inputs.params.pathToProject)/$(outputs.resources.builtGcsFetcherImage.url):$(inputs.params.versionTag)\\n)\\n# Parse the built images from the release.yaml generated by ko\\nBUILT_IMAGES=( $(/workspace/go/src/github.com/tektoncd/pipeline/tekton/koparse/koparse.py --path /workspace/output/bucket/latest/release.yaml --base $(inputs.params.imageRegistry)/$(inputs.params.pathToProject) --images ${IMAGES[@]}) )\\n\\n# Auth with account credentials\\ngcloud auth activate-service-account --key-file=/secret/release.json\\n\\n# Tag the images and put them in all the regions\\nfor IMAGE in \\\"${BUILT_IMAGES[@]}\\\"\\ndo\\n  IMAGE_WITHOUT_SHA=${IMAGE%%@*}\\n  IMAGE_WITHOUT_SHA_AND_TAG=${IMAGE_WITHOUT_SHA%%:*}\\n  IMAGE_WITH_SHA=${IMAGE_WITHOUT_SHA_AND_TAG}@${IMAGE##*@}\\n  gcloud -q container images add-tag ${IMAGE_WITH_SHA} ${IMAGE_WITHOUT_SHA_AND_TAG}:latest\\n  for REGION in \\\"${REGIONS[@]}\\\"\\n  do\\n    for TAG in \\\"latest\\\" $(inputs.params.versionTag)\\n    do\\n      gcloud -q container images add-tag ${IMAGE_WITH_SHA} ${REGION}.${IMAGE_WITHOUT_SHA_AND_TAG}:$TAG\\n    done\\n  done\\ndone\\n\",\"volumeMounts\":[{\"mountPath\":\"/secret\",\"name\":\"gcp-secret\"}]}],\"volumes\":[{\"name\":\"gcp-secret\",\"secret\":{\"secretName\":\"release-secret\"}}]}}\n"},"creationTimestamp":"2020-01-21T02:06:34Z","generation":1,"labels":{"tekton.dev/eventlistener":"pipeline-nightly-release-cron","tekton.dev/pipeline":"pipeline-release-nightly","tekton.dev/pipelineRun":"pipeline-release-nightly-bqstn-js2wf","tekton.dev/pipelineTask":"publish-images","tekton.dev/task":"publish-tekton-pipelines","tekton.dev/trigger":"pipeline-nightly-release-cron-trigger","tekton.dev/triggers-eventid":"qqckv"},"name":"pipeline-release-nightly-bqstn-js2wf-publish-images-f6r95","namespace":"default","ownerReferences":[{"apiVersion":"tekton.dev/v1alpha1","blockOwnerDeletion":true,"controller":true,"kind":"PipelineRun","name":"pipeline-release-nightly-bqstn-js2wf","uid":"c1e82300-3bf1-11ea-b66a-42010a8000ba"}],"resourceVersion":"39102061","selfLink":"/apis/tekton.dev/v1alpha1/namespaces/default/taskruns/pipeline-release-nightly-bqstn-js2wf-publish-images-f6r95","uid":"a60976de-3bf2-11ea-b66a-42010a8000ba"},"spec":{"inputs":{"params":[{"name":"pathToProject","value":"github.com/tektoncd/pipeline"},{"name":"versionTag","value":"v20200121-ab43a6a96a"},{"name":"imageRegistry","value":"gcr.io/tekton-nightly"}],"resources":[{"name":"source","resourceRef":{"name":"git-source-bqstn"}},{"name":"bucket","resourceRef":{"name":"tekton-bucket-nightly-bqstn"}}]},"outputs":{"resources":[{"name":"builtEntrypointImage","paths":["/pvc/publish-images/builtEntrypointImage"],"resourceRef":{"apiVersion":"tekton.dev/v1alpha1","name":"entrypoint-image"}},{"name":"builtKubeconfigWriterImage","paths":["/pvc/publish-images/builtKubeconfigWriterImage"],"resourceRef":{"apiVersion":"tekton.dev/v1alpha1","name":"kubeconfigwriter-image"}},{"name":"builtControllerImage","paths":["/pvc/publish-images/builtControllerImage"],"resourceRef":{"apiVersion":"tekton.dev/v1alpha1","name":"controller-image"}},{"name":"builtPullRequestInitImage","paths":["/pvc/publish-images/builtPullRequestInitImage"],"resourceRef":{"apiVersion":"tekton.dev/v1alpha1","name":"pull-request-init-image"}},{"name":"builtGcsFetcherImage","paths":["/pvc/publish-images/builtGcsFetcherImage"],"resourceRef":{"apiVersion":"tekton.dev/v1alpha1","name":"gcs-fetcher-image"}},{"name":"bucket","paths":["/pvc/publish-images/bucket"],"resourceRef":{"name":"tekton-bucket-nightly-bqstn"}},{"name":"builtBaseImage","paths":["/pvc/publish-images/builtBaseImage"],"resourceRef":{"apiVersion":"tekton.dev/v1alpha1","name":"base-image"}},{"name":"builtCredsInitImage","paths":["/pvc/publish-images/builtCredsInitImage"],"resourceRef":{"apiVersion":"tekton.dev/v1alpha1","name":"creds-init-image"}},{"name":"builtGitInitImage","paths":["/pvc/publish-images/builtGitInitImage"],"resourceRef":{"apiVersion":"tekton.dev/v1alpha1","name":"git-init-image"}},{"name":"builtWebhookImage","paths":["/pvc/publish-images/builtWebhookImage"],"resourceRef":{"apiVersion":"tekton.dev/v1alpha1","name":"webhook-image"}},{"name":"builtDigestExporterImage","paths":["/pvc/publish-images/builtDigestExporterImage"],"resourceRef":{"apiVersion":"tekton.dev/v1alpha1","name":"digest-exporter-image"}},{"name":"notification","paths":["/pvc/publish-images/notification"],"resourceRef":{"apiVersion":"tekton.dev/v1alpha1","name":"post-nightly-release-trigger"}}]},"podTemplate":{},"serviceAccountName":"default","taskRef":{"kind":"Task","name":"publish-tekton-pipelines"},"timeout":"1h0m0s"},"status":{"cloudEvents":[{"status":{"condition":"Sent","message":"","retryCount":1,"sentAt":"2020-01-21T02:14:52Z"},"target":"http://el-pipeline-release-post-processing.default:8080"}],"completionTime":"2020-01-21T02:14:52Z","conditions":[{"lastTransitionTime":"2020-01-21T02:14:52Z","message":"All Steps have completed executing","reason":"Succeeded","status":"True","type":"Succeeded"}],"podName":"pipeline-release-nightly-bqstn-js2wf-publish-images-f6r95-pod-b5129f","resourcesResult":[{"digest":"","key":"commit","name":"","resourceRef":{},"value":"ab43a6a96a245f62c43884ab6f97590f6b7379f6"}],"startTime":"2020-01-21T02:06:34Z","steps":[{"container":"step-build-push-base-images","imageID":"docker-pullable://gcr.io/kaniko-project/executor@sha256:d9fe474f80b73808dc12b54f45f5fc90f7856d9fc699d4a5e79d968a1aef1a72","name":"build-push-base-images","terminated":{"containerID":"docker://8fe34f47c5fe49a30c11c00c92d0ba7c116e6afc5279b3047829dd325ab7b13f","exitCode":0,"finishedAt":"2020-01-21T02:07:22Z","reason":"Completed","startedAt":"2020-01-21T02:06:53Z"}},{"container":"step-create-dir-builtcredsinitimage-qztlv","imageID":"docker-pullable://busybox@sha256:edafc0a0fb057813850d1ba44014914ca02d671ae247107ca70c94db686e7de6","name":"create-dir-builtcredsinitimage-qztlv","terminated":{"containerID":"docker://a9028ff319ac5cd15df27fb295bec2dd4ed3462bb59b8edf1b26273481410880","exitCode":0,"finishedAt":"2020-01-21T02:07:00Z","reason":"Completed","startedAt":"2020-01-21T02:06:50Z"}},{"container":"step-link-input-bucket-to-output","imageID":"docker-pullable://busybox@sha256:edafc0a0fb057813850d1ba44014914ca02d671ae247107ca70c94db686e7de6","name":"link-input-bucket-to-output","terminated":{"containerID":"docker://6333be22edf4ecd30084eed36cc1712628e178be56cc4bd825429b908d7edd72","exitCode":0,"finishedAt":"2020-01-21T02:07:22Z","reason":"Completed","startedAt":"2020-01-21T02:06:53Z"}},{"container":"step-ensure-release-dirs-exist","imageID":"docker-pullable://busybox@sha256:edafc0a0fb057813850d1ba44014914ca02d671ae247107ca70c94db686e7de6","name":"ensure-release-dirs-exist","terminated":{"containerID":"docker://c9f5b54892114fc19b0d8e6173f0440f5a235d111c6e05584e823fade3ac6486","exitCode":0,"finishedAt":"2020-01-21T02:07:22Z","reason":"Completed","startedAt":"2020-01-21T02:06:53Z"}},{"container":"step-create-dir-builtcontrollerimage-vf6zm","imageID":"docker-pullable://busybox@sha256:edafc0a0fb057813850d1ba44014914ca02d671ae247107ca70c94db686e7de6","name":"create-dir-builtcontrollerimage-vf6zm","terminated":{"containerID":"docker://4f94a48adc3d5c64a6e0787261861116d33cc16b68914a29911052523a6f963d","exitCode":0,"finishedAt":"2020-01-21T02:07:00Z","reason":"Completed","startedAt":"2020-01-21T02:06:50Z"}},{"container":"step-copy-to-tagged-bucket","imageID":"docker-pullable://busybox@sha256:edafc0a0fb057813850d1ba44014914ca02d671ae247107ca70c94db686e7de6","name":"copy-to-tagged-bucket","terminated":{"containerID":"docker://6b5daacfa4501f5ba523ad35566db6203faf39820c781810a70c87c1d6b4a06f","exitCode":0,"finishedAt":"2020-01-21T02:10:45Z","reason":"Completed","startedAt":"2020-01-21T02:06:54Z"}},{"container":"step-tag-images","imageID":"docker-pullable://google/cloud-sdk@sha256:b35a8a6e344714684e8a7ab4d074d16caf95f338522327adfbf56ff229f011a5","name":"tag-images","terminated":{"containerID":"docker://b34c6ca4eee0566bb699a458984165f0f2d1b508cc0a8fca143348f2903002f3","exitCode":0,"finishedAt":"2020-01-21T02:14:38Z","reason":"Completed","startedAt":"2020-01-21T02:06:54Z"}},{"container":"step-create-dir-builtentrypointimage-52k9x","imageID":"docker-pullable://busybox@sha256:edafc0a0fb057813850d1ba44014914ca02d671ae247107ca70c94db686e7de6","name":"create-dir-builtentrypointimage-52k9x","terminated":{"containerID":"docker://11e91160f5197501bc1f4a758707d5bf00607c52c58825e37d863d8d6f683e19","exitCode":0,"finishedAt":"2020-01-21T02:07:01Z","reason":"Completed","startedAt":"2020-01-21T02:06:51Z"}},{"container":"step-create-dir-builtgcsfetcherimage-7z4fk","imageID":"docker-pullable://busybox@sha256:edafc0a0fb057813850d1ba44014914ca02d671ae247107ca70c94db686e7de6","name":"create-dir-builtgcsfetcherimage-7z4fk","terminated":{"containerID":"docker://b168698731ca1a9105dcc1de42ace42f1661c326f8e2f5fc77cfd3e6997545b7","exitCode":0,"finishedAt":"2020-01-21T02:06:58Z","reason":"Completed","startedAt":"2020-01-21T02:06:48Z"}},{"container":"step-create-dir-builtgitinitimage-bg252","imageID":"docker-pullable://busybox@sha256:edafc0a0fb057813850d1ba44014914ca02d671ae247107ca70c94db686e7de6","name":"create-dir-builtgitinitimage-bg252","terminated":{"containerID":"docker://688e07623e2bae06656541ecc6d93731c825a05acfc1e91ef2095c57de22fbc5","exitCode":0,"finishedAt":"2020-01-21T02:07:00Z","reason":"Completed","startedAt":"2020-01-21T02:06:50Z"}},{"container":"step-create-dir-builtkubeconfigwriterimage-29wbc","imageID":"docker-pullable://busybox@sha256:edafc0a0fb057813850d1ba44014914ca02d671ae247107ca70c94db686e7de6","name":"create-dir-builtkubeconfigwriterimage-29wbc","terminated":{"containerID":"docker://737a3691f5ab7e81aff0ce728c6e314863822db77311b8077be0e0e2e84acbea","exitCode":0,"finishedAt":"2020-01-21T02:07:01Z","reason":"Completed","startedAt":"2020-01-21T02:06:51Z"}},{"container":"step-create-dir-builtpullrequestinitimage-pqbbf","imageID":"docker-pullable://busybox@sha256:edafc0a0fb057813850d1ba44014914ca02d671ae247107ca70c94db686e7de6","name":"create-dir-builtpullrequestinitimage-pqbbf","terminated":{"containerID":"docker://44960e65e683b45831e631130953e60a77da2e34736107b17952ef65a9b0f44a","exitCode":0,"finishedAt":"2020-01-21T02:06:59Z","reason":"Completed","startedAt":"2020-01-21T02:06:49Z"}},{"container":"step-create-dir-builtwebhookimage-r2wvq","imageID":"docker-pullable://busybox@sha256:edafc0a0fb057813850d1ba44014914ca02d671ae247107ca70c94db686e7de6","name":"create-dir-builtwebhookimage-r2wvq","terminated":{"containerID":"docker://78cd2216a126c94f84e0e5a262f535a2924ef776551693ea69599b038cadfc3e","exitCode":0,"finishedAt":"2020-01-21T02:06:59Z","reason":"Completed","startedAt":"2020-01-21T02:06:49Z"}},{"container":"step-create-dir-notification-brwb8","imageID":"docker-pullable://busybox@sha256:edafc0a0fb057813850d1ba44014914ca02d671ae247107ca70c94db686e7de6","name":"create-dir-notification-brwb8","terminated":{"containerID":"docker://6cff4af31c3ad59f161fe6f54f4999d2d8bd05efabd289dd96655c2af304ab34","exitCode":0,"finishedAt":"2020-01-21T02:06:58Z","reason":"Completed","startedAt":"2020-01-21T02:06:48Z"}},{"container":"step-create-dir-tekton-bucket-nightly-bqstn-lf7cv","imageID":"docker-pullable://busybox@sha256:edafc0a0fb057813850d1ba44014914ca02d671ae247107ca70c94db686e7de6","name":"create-dir-tekton-bucket-nightly-bqstn-lf7cv","terminated":{"containerID":"docker://2d99ac65f8d373087feba6a3a86a3a23d29ddae9053f1bad04227c019d03eae5","exitCode":0,"finishedAt":"2020-01-21T02:07:05Z","reason":"Completed","startedAt":"2020-01-21T02:06:52Z"}},{"container":"step-create-ko-yaml","imageID":"docker-pullable://busybox@sha256:edafc0a0fb057813850d1ba44014914ca02d671ae247107ca70c94db686e7de6","name":"create-ko-yaml","terminated":{"containerID":"docker://d54ed455fccd3974b008def4303179b1208bf6e9b6df70a6cf51c655c153f7d4","exitCode":0,"finishedAt":"2020-01-21T02:07:22Z","reason":"Completed","startedAt":"2020-01-21T02:06:53Z"}},{"container":"step-create-dir-builtbaseimage-svlpz","imageID":"docker-pullable://busybox@sha256:edafc0a0fb057813850d1ba44014914ca02d671ae247107ca70c94db686e7de6","name":"create-dir-builtbaseimage-svlpz","terminated":{"containerID":"docker://b9d7ff6ae364a310fa741d355308cc573bdb56fde3777457caef3f02d98910fe","exitCode":0,"finishedAt":"2020-01-21T02:07:01Z","reason":"Completed","startedAt":"2020-01-21T02:06:51Z"}},{"container":"step-fetch-tekton-bucket-nightly-bqstn-brhl8","imageID":"docker-pullable://google/cloud-sdk@sha256:b35a8a6e344714684e8a7ab4d074d16caf95f338522327adfbf56ff229f011a5","name":"fetch-tekton-bucket-nightly-bqstn-brhl8","terminated":{"containerID":"docker://9de33ca9cea20a1781f511c6435b505ca5117883475439da061a6f9a54ec4fc7","exitCode":0,"finishedAt":"2020-01-21T02:07:13Z","reason":"Completed","startedAt":"2020-01-21T02:06:52Z"}},{"container":"step-git-source-git-source-bqstn-b8gj7","imageID":"docker-pullable://gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init@sha256:ce917a4a8f41a811c485fafec0f92774df3e09316da1d35e5a01e9e95a313f1e","name":"git-source-git-source-bqstn-b8gj7","terminated":{"containerID":"docker://e622513b3e8c5564086e641c6ff52aadb2123767f09bd07dea4cb45aac1ca801","exitCode":0,"finishedAt":"2020-01-21T02:07:04Z","message":"[{\"name\":\"\",\"digest\":\"\",\"key\":\"commit\",\"value\":\"ab43a6a96a245f62c43884ab6f97590f6b7379f6\",\"resourceRef\":{}}]","reason":"Completed","startedAt":"2020-01-21T02:06:52Z"}},{"container":"step-image-digest-exporter-nct97","imageID":"docker-pullable://gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/imagedigestexporter@sha256:10cc6e64fbb28ad87c1a95d7300caa4545fba7903996128198077cd65ca45f0e","name":"image-digest-exporter-nct97","terminated":{"containerID":"docker://50bb62e65aa083d256a113ca32a7e6123cd3daf5bb960852d4e74290f44e58b8","exitCode":0,"finishedAt":"2020-01-21T02:14:38Z","message":"[]","reason":"Completed","startedAt":"2020-01-21T02:06:55Z"}},{"container":"step-create-dir-bucket-54ftm","imageID":"docker-pullable://busybox@sha256:edafc0a0fb057813850d1ba44014914ca02d671ae247107ca70c94db686e7de6","name":"create-dir-bucket-54ftm","terminated":{"containerID":"docker://865b4026f769dca191b15d1398ea5a8e802f44744789f2f4960196de7622d531","exitCode":0,"finishedAt":"2020-01-21T02:07:01Z","reason":"Completed","startedAt":"2020-01-21T02:06:51Z"}},{"container":"step-run-ko","imageID":"docker-pullable://gcr.io/tekton-releases/dogfooding/ko@sha256:26abf7c6b0f205dda7eda4efd3235cd8b45c2d836c179ba5ff42e2dd0d43ea1f","name":"run-ko","terminated":{"containerID":"docker://a03141f97909e3bf0e3c4ed29e915303ee35c6cbd359abcce7b67e40ddaa01a3","exitCode":0,"finishedAt":"2020-01-21T02:10:44Z","reason":"Completed","startedAt":"2020-01-21T02:06:54Z"}},{"container":"step-source-copy-tekton-bucket-nightly-bqstn-wcvm9","imageID":"docker-pullable://busybox@sha256:edafc0a0fb057813850d1ba44014914ca02d671ae247107ca70c94db686e7de6","name":"source-copy-tekton-bucket-nightly-bqstn-wcvm9","terminated":{"containerID":"docker://4b0848dad139f0cba269b04d5cc1921d3a04e8d63d9cf5abc97383bd78ebe68a","exitCode":0,"finishedAt":"2020-01-21T02:14:39Z","reason":"Completed","startedAt":"2020-01-21T02:06:55Z"}},{"container":"step-source-mkdir-tekton-bucket-nightly-bqstn-d7l7c","imageID":"docker-pullable://busybox@sha256:edafc0a0fb057813850d1ba44014914ca02d671ae247107ca70c94db686e7de6","name":"source-mkdir-tekton-bucket-nightly-bqstn-d7l7c","terminated":{"containerID":"docker://3d1b76c2cd829a09982e1ba4a1e6a697a2f462bb9944b994fbba9f6f283aeb7d","exitCode":0,"finishedAt":"2020-01-21T02:14:39Z","reason":"Completed","startedAt":"2020-01-21T02:06:55Z"}},{"container":"step-create-dir-builtdigestexporterimage-hwc2c","imageID":"docker-pullable://busybox@sha256:edafc0a0fb057813850d1ba44014914ca02d671ae247107ca70c94db686e7de6","name":"create-dir-builtdigestexporterimage-hwc2c","terminated":{"containerID":"docker://d844176d05534883c4b74c8f190873ebfb784ce9af9bf944eac49d45f9f1a51a","exitCode":0,"finishedAt":"2020-01-21T02:06:59Z","reason":"Completed","startedAt":"2020-01-21T02:06:49Z"}},{"container":"step-upload-tekton-bucket-nightly-bqstn-k257n","imageID":"docker-pullable://google/cloud-sdk@sha256:b35a8a6e344714684e8a7ab4d074d16caf95f338522327adfbf56ff229f011a5","name":"upload-tekton-bucket-nightly-bqstn-k257n","terminated":{"containerID":"docker://3b630b72082a5e569a92c482110c689ed4e387cbd753aacca5a23a9738bf911d","exitCode":0,"finishedAt":"2020-01-21T02:14:50Z","reason":"Completed","startedAt":"2020-01-21T02:06:56Z"}}]}}}}`,
		expr: "$(body.taskRun.spec.outputs.resources[?(@.name == 'bucket')].resourceRef.name)",
		want: "tekton-bucket-nightly-bqstn",
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var data interface{}
			err := json.Unmarshal([]byte(tt.in), &data)
			if err != nil {
				t.Fatalf("Could not unmarshall body : %q", err)
			}
			got, err := ParseJSONPath(data, tt.expr)
			if err != nil {
				t.Fatalf("ParseJSONPath() error = %v", err)
			}
			if diff := cmp.Diff(strings.Replace(tt.want, " ", "", -1), got); diff != "" {
				t.Errorf("ParseJSONPath() -want,+got: %s", diff)
			}
		})
	}
}

func TestParseJSONPath_Error(t *testing.T) {
	testJSON := `{"body": {"key": "val"}}`
	invalidExprs := []string{
		"$({.hello)",
		"$(+12.3.0)",
		"$([1)",
		"$(body",
		"body)",
		"body",
		"$(body.missing)",
		"$(body.key[0])",
	}
	var data interface{}
	err := json.Unmarshal([]byte(testJSON), &data)
	if err != nil {
		t.Fatalf("Could not unmarshall body : %q", err)
	}

	for _, expr := range invalidExprs {
		t.Run(expr, func(t *testing.T) {
			got, err := ParseJSONPath(data, expr)
			if err == nil {
				t.Errorf("ParseJSONPath() did not return expected error; got = %v", got)
			}
		})
	}
}

func TestTektonJSONPathExpression(t *testing.T) {
	tests := []struct {
		expr string
		want string
	}{
		{"$(metadata.name)", "{.metadata.name}"},
		{"$(.metadata.name)", "{.metadata.name}"},
		{"$({.metadata.name})", "{.metadata.name}"},
		{"$()", ""},
	}
	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			got, err := TektonJSONPathExpression(tt.expr)
			if err != nil {
				t.Errorf("TektonJSONPathExpression() unexpected error = %v,  got = %v", err, got)
			}
			if got != tt.want {
				t.Errorf("TektonJSONPathExpression() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTektonJSONPathExpression_Error(t *testing.T) {
	tests := []string{
		"{.metadata.name}", // not wrapped in $()
		"",
		"$({asd)",
		"$({)",
		"$({foo.bar)",
		"$(foo.bar})",
		"$({foo.bar}})",
		"$({{foo.bar)",
	}
	for _, expr := range tests {
		t.Run(expr, func(t *testing.T) {
			_, err := TektonJSONPathExpression(expr)
			if err == nil {
				t.Errorf("TektonJSONPathExpression() did not get expected error for expression = %s", expr)
			}
		})
	}
}

func TestRelaxedJSONPathExpression(t *testing.T) {
	tests := []struct {
		expr string
		want string
	}{
		{"metadata.name", "{.metadata.name}"},
		{".metadata.name", "{.metadata.name}"},
		{"{.metadata.name}", "{.metadata.name}"},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			got, err := relaxedJSONPathExpression(tt.expr)
			if err != nil {
				t.Errorf("TektonJSONPathExpression() unexpected error = %v,  got = %v", err, got)
			}
			if got != tt.want {
				t.Errorf("TektonJSONPathExpression() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRelaxedJSONPathExpression_Error(t *testing.T) {
	tests := []string{
		"{foo.bar",
		"foo.bar}",
		"{foo.bar}}",
		"{{foo.bar}",
	}
	for _, expr := range tests {
		t.Run(expr, func(t *testing.T) {
			got, err := relaxedJSONPathExpression(expr)
			if err == nil {
				t.Errorf("TektonJSONPathExpression() did not get expected error = %v,  got = %v", err, got)
			}
		})
	}
}
