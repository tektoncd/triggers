module github.com/tektoncd/triggers

go 1.13

require (
	contrib.go.opencensus.io/exporter/prometheus v0.1.0 // indirect
	contrib.go.opencensus.io/exporter/stackdriver v0.12.4 // indirect
	github.com/evanphx/json-patch v4.5.0+incompatible // indirect
	github.com/gobuffalo/envy v1.7.0 // indirect
	github.com/gogo/protobuf v1.2.1 // indirect
	github.com/golang/groupcache v0.0.0-20190702054246-869f871628b6 // indirect
	github.com/google/go-cmp v0.3.1
	github.com/google/go-github/v28 v28.1.1
	github.com/google/uuid v1.1.1 // indirect
	github.com/googleapis/gnostic v0.3.0 // indirect
	github.com/gregjones/httpcache v0.0.0-20190611155906-901d90724c79 // indirect
	github.com/imdario/mergo v0.3.7 // indirect
	github.com/markbates/inflect v1.0.4 // indirect
	github.com/mattbaird/jsonpatch v0.0.0-20171005235357-81af80346b1a // indirect
	github.com/peterbourgon/diskv v2.0.1+incompatible // indirect
	github.com/prometheus/client_golang v1.1.0 // indirect
	github.com/rogpeppe/go-internal v1.3.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/tektoncd/pipeline v0.9.1
	github.com/tektoncd/plumbing v0.0.0-20191223144933-a403a048e00c
	go.uber.org/atomic v1.4.0 // indirect
	go.uber.org/multierr v1.1.0 // indirect
	go.uber.org/zap v1.9.2-0.20180814183419-67bc79d13d15
	golang.org/x/crypto v0.0.0-20191117063200-497ca9f6d64f // indirect
	golang.org/x/net v0.0.0-20191119073136-fc4aabc6c914 // indirect
	golang.org/x/sys v0.0.0-20191119060738-e882bf8e40c2 // indirect
	golang.org/x/tools v0.0.0-20191118222007-07fc4c7f2b98 // indirect
	golang.org/x/xerrors v0.0.0-20190717185122-a985d3407aa7
	google.golang.org/api v0.7.1-0.20190805211801-b7b1a549a9ef // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/yaml.v2 v2.2.2
	k8s.io/api v0.0.0-20191004102255-dacd7df5a50b
	k8s.io/apimachinery v0.0.0-20191004074956-01f8b7d1121a
	k8s.io/client-go v0.0.0-20191004102537-eb5b9a8cfde7
	k8s.io/code-generator v0.0.0-00010101000000-000000000000
	k8s.io/klog v0.2.0
	k8s.io/kube-openapi v0.0.0-20190722073852-5e22f3d471e6
	knative.dev/caching v0.0.0-20190719140829-2032732871ff
	knative.dev/pkg v0.0.0-20190909195211-528ad1c1dd62
	sigs.k8s.io/yaml v1.1.0 // indirect
)

// Knative deps
replace (
	contrib.go.opencensus.io/exporter/stackdriver => contrib.go.opencensus.io/exporter/stackdriver v0.12.5
	github.com/google/go-containerregistry => github.com/google/go-containerregistry v0.0.0-20190320210540-8d4083db9aa0
	knative.dev/pkg => knative.dev/pkg v0.0.0-20190909195211-528ad1c1dd62
	knative.dev/pkg/vendor/github.com/spf13/pflag => github.com/spf13/pflag v1.0.5
)

// Pin k8s deps to 1.12.9
replace (
	k8s.io/api => k8s.io/api v0.0.0-20191004102255-dacd7df5a50b
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.0.0-20191004105443-a7d558db75c6
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20191004074956-01f8b7d1121a
	k8s.io/apiserver => k8s.io/apiserver v0.0.0-20191004103531-b568748c9b85
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.0.0-20191004110054-fe9b9282443f
	k8s.io/client-go => k8s.io/client-go v0.0.0-20191004102537-eb5b9a8cfde7
	k8s.io/code-generator => k8s.io/code-generator v0.0.0-20181117043124-c2090bec4d9b
	k8s.io/gengo => k8s.io/gengo v0.0.0-20190327210449-e17681d19d3a
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.0.0-20191004103911-2797d0dcf14b
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.0.0-20191016015407-72acd948ffff
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.0.0-20191016015246-999188f3eff6
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.0.0-20191016015341-7be46aeada42
	k8s.io/kubelet => k8s.io/kubelet v0.0.0-20191016015314-e7fc4f69fc2c
	k8s.io/kubernetes => k8s.io/kubernetes v1.13.12
	k8s.io/metrics => k8s.io/metrics v0.0.0-20191004105814-56635b1b5a0c
)
