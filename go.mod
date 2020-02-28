module github.com/tektoncd/triggers

go 1.13

require (
	github.com/golang/protobuf v1.3.4
	github.com/google/cel-go v0.3.2
	github.com/google/go-cmp v0.4.0
	github.com/google/go-github v17.0.0+incompatible
	github.com/google/go-querystring v1.0.0 // indirect
	github.com/gorilla/mux v1.7.4
	github.com/tektoncd/pipeline v0.10.1
	github.com/tidwall/gjson v1.5.0 // indirect
	github.com/tidwall/sjson v1.0.4
	go.uber.org/zap v1.14.0
	golang.org/x/xerrors v0.0.0-20191204190536-9bdfabe68543
	gomodules.xyz/jsonpatch/v2 v2.1.0 // indirect
	google.golang.org/genproto v0.0.0-20200227132054-3f1135a288c9
	gopkg.in/yaml.v2 v2.2.8
	k8s.io/api v0.17.3
	k8s.io/apiextensions-apiserver v0.17.3 // indirect
	k8s.io/apimachinery v0.17.3
	k8s.io/client-go v11.0.0+incompatible
	k8s.io/klog v1.0.0
	knative.dev/caching v0.0.0-20200227184451-e2e5784288b6
	knative.dev/pkg v0.0.0-20200227193851-2fe8db300072
)
