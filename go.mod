module github.com/tektoncd/triggers

go 1.15

require (
	github.com/GoogleCloudPlatform/cloud-builders/gcs-fetcher v0.0.0-20191203181535-308b93ad1f39
	github.com/golang/protobuf v1.5.2
	github.com/google/cel-go v0.7.3
	github.com/google/go-cmp v0.5.6
	github.com/google/go-containerregistry v0.5.2-0.20210709161016-b448abac9a70 // indirect
	github.com/google/go-github/v31 v31.0.0
	github.com/google/uuid v1.3.0
	github.com/gorilla/mux v1.7.4
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/cobra v1.2.1
	github.com/tektoncd/pipeline v0.27.1
	github.com/tektoncd/plumbing v0.0.0-20210514044347-f8a9689d5bd5
	github.com/tidwall/gjson v1.6.5 // indirect
	github.com/tidwall/sjson v1.0.4
	go.opencensus.io v0.23.0
	go.uber.org/zap v1.18.1
	golang.org/x/crypto v0.0.0-20210817164053-32db794688a5 // indirect
	golang.org/x/mod v0.5.0 // indirect
	golang.org/x/net v0.0.0-20210825183410-e898025ed96a // indirect
	golang.org/x/sys v0.0.0-20210823070655-63515b42dcdf // indirect
	golang.org/x/time v0.0.0-20210611083556-38a9dc6acbc6 // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1
	google.golang.org/api v0.50.0 // indirect
	google.golang.org/genproto v0.0.0-20210624195500-8bfb893ecb84
	google.golang.org/grpc v1.39.0
	google.golang.org/protobuf v1.27.1
	k8s.io/api v0.20.7
	k8s.io/apiextensions-apiserver v0.20.7
	k8s.io/apimachinery v0.20.7
	k8s.io/client-go v0.20.7
	k8s.io/code-generator v0.20.7
	k8s.io/kube-openapi v0.0.0-20210113233702-8566a335510f
	knative.dev/pkg v0.0.0-20210730172132-bb4aaf09c430
	knative.dev/serving v0.24.0
	sigs.k8s.io/yaml v1.2.0
)

replace (
	contrib.go.opencensus.io/exporter/stackdriver => contrib.go.opencensus.io/exporter/stackdriver v0.13.4
	github.com/Azure/azure-sdk-for-go => github.com/Azure/azure-sdk-for-go v38.2.0+incompatible
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v14.2.0+incompatible
)
