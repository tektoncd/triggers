module github.com/tektoncd/triggers

go 1.14

require (
	github.com/GoogleCloudPlatform/cloud-builders/gcs-fetcher v0.0.0-20191203181535-308b93ad1f39
	github.com/gobuffalo/envy v1.9.0 // indirect
	github.com/golang/protobuf v1.4.1
	github.com/google/cel-go v0.5.1
	github.com/google/go-cmp v0.5.0
	github.com/google/go-github/v31 v31.0.0
	github.com/gorilla/mux v1.7.3
	github.com/grpc-ecosystem/grpc-gateway v1.13.0 // indirect
	github.com/spf13/cobra v0.0.6
	github.com/tektoncd/pipeline v0.14.2
	github.com/tektoncd/plumbing v0.0.0-20200430135134-e53521e1d887
	github.com/tidwall/gjson v1.3.5 // indirect
	github.com/tidwall/sjson v1.0.4
	go.uber.org/zap v1.15.0
	golang.org/x/text v0.3.3 // indirect
	golang.org/x/xerrors v0.0.0-20191204190536-9bdfabe68543
	google.golang.org/genproto v0.0.0-20200526211855-cb27e3aa2013
	google.golang.org/protobuf v1.25.0
	gopkg.in/yaml.v2 v2.3.0
	k8s.io/api v0.18.2
	k8s.io/apimachinery v0.18.2
	k8s.io/client-go v11.0.1-0.20190805182717-6502b5e7b1b5+incompatible
	k8s.io/code-generator v0.18.0
	k8s.io/klog v1.0.0
	k8s.io/kube-openapi v0.0.0-20200410145947-bcb3869e6f29
	k8s.io/utils v0.0.0-20200324210504-a9aa75ae1b89 // indirect
	knative.dev/caching v0.0.0-20200521155757-e78d17bc250e
	knative.dev/pkg v0.0.0-20200625173728-dfb81cf04a7c
	sigs.k8s.io/yaml v1.2.0
)

// Knative deps (release-0.15)
replace (
	contrib.go.opencensus.io/exporter/stackdriver => contrib.go.opencensus.io/exporter/stackdriver v0.12.9-0.20191108183826-59d068f8d8ff
	github.com/Azure/azure-sdk-for-go => github.com/Azure/azure-sdk-for-go v38.2.0+incompatible
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v13.4.0+incompatible
)

// Pin k8s deps to 1.16.5
replace (
	k8s.io/api => k8s.io/api v0.16.5
	k8s.io/apimachinery => k8s.io/apimachinery v0.16.5
	k8s.io/client-go => k8s.io/client-go v0.16.5
	k8s.io/code-generator => k8s.io/code-generator v0.16.5
)
