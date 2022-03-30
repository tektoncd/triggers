module github.com/tektoncd/triggers

go 1.16

require (
	cloud.google.com/go/iam v0.2.0 // indirect
	github.com/GoogleCloudPlatform/cloud-builders/gcs-fetcher v0.0.0-20191203181535-308b93ad1f39
	github.com/ahmetb/gen-crd-api-reference-docs v0.3.1-0.20210609063737-0067dc6dcea2
	github.com/cloudevents/sdk-go/v2 v2.5.0
	github.com/golang/protobuf v1.5.2
	github.com/google/cel-go v0.11.2
	github.com/google/go-cmp v0.5.7
	github.com/google/go-github/v31 v31.0.0
	github.com/google/uuid v1.3.0
	github.com/gorilla/mux v1.8.0
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/cobra v1.4.0
	github.com/tektoncd/pipeline v0.33.0
	github.com/tektoncd/plumbing v0.0.0-20211012143332-c7cc43d9bc0c
	github.com/tidwall/sjson v1.2.3
	go.opencensus.io v0.23.0
	go.uber.org/zap v1.21.0
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1
	google.golang.org/genproto v0.0.0-20220310185008-1973136f34c6
	google.golang.org/grpc v1.44.0
	google.golang.org/protobuf v1.27.1
	k8s.io/api v0.23.5
	k8s.io/apiextensions-apiserver v0.23.4
	k8s.io/apimachinery v0.23.5
	k8s.io/client-go v0.23.5
	k8s.io/code-generator v0.23.5
	k8s.io/klog/v2 v2.60.1-0.20220317184644-43cc75f9ae89
	k8s.io/kube-openapi v0.0.0-20220114203427-a0453230fd26
	knative.dev/eventing v0.25.0
	knative.dev/pkg v0.0.0-20220329144915-0a1ec2e0d46c
	knative.dev/serving v0.30.1-0.20220330132846-c136562da27f
	sigs.k8s.io/yaml v1.3.0
)
