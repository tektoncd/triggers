module github.com/tektoncd/triggers

go 1.15

require (
	github.com/GoogleCloudPlatform/cloud-builders/gcs-fetcher v0.0.0-20191203181535-308b93ad1f39
	github.com/gobuffalo/envy v1.9.0 // indirect
	github.com/golang/protobuf v1.4.3
	github.com/google/cel-go v0.6.0
	github.com/google/go-cmp v0.5.4
	github.com/google/go-github/v31 v31.0.0
	github.com/google/uuid v1.2.0
	github.com/gorilla/mux v1.7.4
	github.com/sirupsen/logrus v1.7.0
	github.com/spf13/cobra v1.0.0
	github.com/tektoncd/pipeline v0.20.1-0.20210203144343-1b7a37f0d21d
	github.com/tektoncd/plumbing v0.0.0-20201021153918-6b7e894737b5
	github.com/tidwall/gjson v1.3.5 // indirect
	github.com/tidwall/sjson v1.0.4
	go.uber.org/zap v1.16.0
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1
	google.golang.org/genproto v0.0.0-20201211151036-40ec1c210f7a
	google.golang.org/grpc v1.34.0
	google.golang.org/protobuf v1.25.0
	k8s.io/api v0.19.7
	k8s.io/apimachinery v0.19.7
	k8s.io/client-go v0.19.7
	k8s.io/code-generator v0.19.7
	k8s.io/kube-openapi v0.0.0-20210113233702-8566a335510f
	knative.dev/pkg v0.0.0-20210130001831-ca02ef752ac6
	sigs.k8s.io/yaml v1.2.0
)

replace (
	contrib.go.opencensus.io/exporter/stackdriver => contrib.go.opencensus.io/exporter/stackdriver v0.13.4
	github.com/Azure/azure-sdk-for-go => github.com/Azure/azure-sdk-for-go v38.2.0+incompatible
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v14.2.0+incompatible
)
