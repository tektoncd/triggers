package tekton

import (
	"github.com/tektoncd/pipeline/pkg/apis/pipeline"
	"github.com/tektoncd/triggers/pkg/apis/triggers"
	"github.com/tektoncd/triggers/pkg/client/dynamic/clientset"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

var (
	allowedPipelineTypes = map[string][]string{
		"v1alpha1": {"pipelineresources", "pipelineruns", "taskruns", "pipelines", "clustertasks", "tasks", "conditions", "runs"},
		"v1beta1":  {"pipelineruns", "taskruns", "pipelines", "clustertasks", "tasks"},
	}
	allowedTriggersTypes = map[string][]string{
		"v1alpha1": {"clusterinterceptors"},
		"v1beta1":  {"clustertriggerbindings", "eventlisteners", "triggerbindings", "triggers", "triggertemplates"},
	}
)

// WithClient adds Tekton related clients to the Dynamic client.
func WithClient(client dynamic.Interface) clientset.Option {
	return func(cs *clientset.Clientset) {
		for version, resources := range allowedPipelineTypes {
			for _, resource := range resources {
				r := schema.GroupVersionResource{
					Group:    pipeline.GroupName,
					Version:  version,
					Resource: resource,
				}
				cs.Add(r, client)
			}
		}
		for version, resources := range allowedTriggersTypes {
			for _, resource := range resources {
				r := schema.GroupVersionResource{
					Group:    triggers.GroupName,
					Version:  version,
					Resource: resource,
				}
				cs.Add(r, client)
			}
		}
	}
}
