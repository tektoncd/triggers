module github.com/tektoncd/triggers

go 1.15

require (
	github.com/GoogleCloudPlatform/cloud-builders/gcs-fetcher v0.0.0-20191203181535-308b93ad1f39
	github.com/ahmetb/gen-crd-api-reference-docs v0.3.1-0.20210609063737-0067dc6dcea2
	github.com/cloudevents/sdk-go/v2 v2.5.0
	github.com/golang/protobuf v1.5.2
	github.com/google/cel-go v0.7.3
	github.com/google/go-cmp v0.5.6
	github.com/google/go-github/v31 v31.0.0
	github.com/google/uuid v1.3.0
	github.com/gorilla/mux v1.8.0
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/cobra v1.2.1
	github.com/tektoncd/pipeline v0.31.1-0.20220105002759-3e137645be61
	github.com/tektoncd/plumbing v0.0.0-20211012143332-c7cc43d9bc0c
	github.com/tidwall/sjson v1.2.3
	go.opencensus.io v0.23.0
	go.uber.org/zap v1.19.1
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1
	google.golang.org/genproto v0.0.0-20211129164237-f09f9a12af12
	google.golang.org/grpc v1.42.0
	google.golang.org/protobuf v1.27.1
	k8s.io/api v0.22.5
	k8s.io/apiextensions-apiserver v0.22.5
	k8s.io/apimachinery v0.22.5
	k8s.io/client-go v0.22.5
	k8s.io/code-generator v0.22.5
	k8s.io/klog/v2 v2.40.1
	k8s.io/kube-openapi v0.0.0-20211109043538-20434351676c
	knative.dev/eventing v0.25.0
	knative.dev/pkg v0.0.0-20220131144930-f4b57aef0006
	knative.dev/serving v0.25.0
	sigs.k8s.io/yaml v1.3.0
)
