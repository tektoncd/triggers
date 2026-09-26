/*
Copyright 2026 The Tekton Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package triggers

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/yaml"
)

const maxSchemaCRDJSONSize = 256 * 1024

var schemaCRDs = []struct {
	file     string
	name     string
	kind     string
	scope    apiextensionsv1.ResourceScope
	versions []string
	storage  string
}{
	{"300-trigger.yaml", "triggers.triggers.tekton.dev", "Trigger", apiextensionsv1.NamespaceScoped, []string{"v1beta1", "v1alpha1"}, "v1beta1"},
	{"300-triggerbinding.yaml", "triggerbindings.triggers.tekton.dev", "TriggerBinding", apiextensionsv1.NamespaceScoped, []string{"v1beta1", "v1alpha1"}, "v1beta1"},
	{"300-clustertriggerbinding.yaml", "clustertriggerbindings.triggers.tekton.dev", "ClusterTriggerBinding", apiextensionsv1.ClusterScoped, []string{"v1beta1", "v1alpha1"}, "v1beta1"},
	{"300-triggertemplate.yaml", "triggertemplates.triggers.tekton.dev", "TriggerTemplate", apiextensionsv1.NamespaceScoped, []string{"v1beta1", "v1alpha1"}, "v1beta1"},
	{"300-eventlistener.yaml", "eventlisteners.triggers.tekton.dev", "EventListener", apiextensionsv1.NamespaceScoped, []string{"v1beta1", "v1alpha1"}, "v1beta1"},
	{"300-interceptor.yaml", "interceptors.triggers.tekton.dev", "Interceptor", apiextensionsv1.NamespaceScoped, []string{"v1alpha1"}, "v1alpha1"},
	{"300-clusterinterceptor.yaml", "clusterinterceptors.triggers.tekton.dev", "ClusterInterceptor", apiextensionsv1.ClusterScoped, []string{"v1alpha1"}, "v1alpha1"},
}

var preservedSchemaPaths = map[string][]string{
	"Trigger": {
		"spec.interceptors[].params[].value",
		"spec.interceptors[].webhook.header[].value",
		"spec.template.spec.resourcetemplates[]",
	},
	"TriggerTemplate": {"spec.resourcetemplates[]"},
	"EventListener": {
		"spec.resources.customResource",
		"spec.triggerGroups[].interceptors[].params[].value",
		"spec.triggerGroups[].interceptors[].webhook.header[].value",
		"spec.triggers[].interceptors[].params[].value",
		"spec.triggers[].interceptors[].webhook.header[].value",
		"spec.triggers[].template.spec.resourcetemplates[]",
	},
}

var preservedSchemaPathsByVersion = map[string][]string{
	"EventListener/v1alpha1": {
		"spec.resources.customResource",
		"spec.triggers[].interceptors[].params[].value",
		"spec.triggers[].interceptors[].webhook.header[].value",
		"spec.triggers[].template.spec.resourcetemplates[]",
	},
}

func TestCRDSchemas(t *testing.T) {
	crds := make(map[string]*apiextensionsv1.CustomResourceDefinition, len(schemaCRDs))
	for _, expected := range schemaCRDs {
		t.Run(expected.kind, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join("..", "..", "..", "config", expected.file))
			if err != nil {
				t.Fatal(err)
			}
			var crd apiextensionsv1.CustomResourceDefinition
			if err := yaml.Unmarshal(data, &crd); err != nil {
				t.Fatal(err)
			}
			if crd.Name != expected.name || crd.Spec.Group != "triggers.tekton.dev" || crd.Spec.Names.Kind != expected.kind || crd.Spec.Scope != expected.scope {
				t.Fatalf("unexpected CRD identity: name=%q group=%q kind=%q scope=%q", crd.Name, crd.Spec.Group, crd.Spec.Names.Kind, crd.Spec.Scope)
			}
			if len(crd.Spec.Versions) != len(expected.versions) {
				t.Fatalf("got %d versions, want %v", len(crd.Spec.Versions), expected.versions)
			}
			for i, version := range crd.Spec.Versions {
				if version.Name != expected.versions[i] || !version.Served || version.Storage != (version.Name == expected.storage) {
					t.Fatalf("unexpected version at %d: name=%q served=%t storage=%t", i, version.Name, version.Served, version.Storage)
				}
				if version.Schema == nil || version.Schema.OpenAPIV3Schema == nil {
					t.Fatalf("%s has no OpenAPI schema", version.Name)
				}
				root := version.Schema.OpenAPIV3Schema
				if root.Type != "object" || len(root.Properties) == 0 || (root.XPreserveUnknownFields != nil && *root.XPreserveUnknownFields) {
					t.Fatalf("%s root must be a typed object without blanket preserve-unknown", version.Name)
				}
				var preservePaths []string
				collectPreserveUnknownPaths(root, "", &preservePaths)
				sort.Strings(preservePaths)
				wantPreservePaths := preservedSchemaPaths[expected.kind]
				if versionPaths, ok := preservedSchemaPathsByVersion[expected.kind+"/"+version.Name]; ok {
					wantPreservePaths = versionPaths
				}
				wantPreservePaths = append([]string(nil), wantPreservePaths...)
				sort.Strings(wantPreservePaths)
				if !reflect.DeepEqual(preservePaths, wantPreservePaths) {
					t.Fatalf("preserve-unknown paths = %v, want %v", preservePaths, wantPreservePaths)
				}
				if metadata, ok := root.Properties["metadata"]; !ok || metadata.Type != "object" {
					t.Fatalf("%s metadata schema = %#v, want object", version.Name, metadata)
				}
				if spec, ok := root.Properties["spec"]; !ok || spec.Description == "" {
					t.Fatalf("%s spec is missing its explain description", version.Name)
				}
				assertStructuralShape(t, root, expected.kind+"/"+version.Name)
				if len(version.AdditionalPrinterColumns) == 0 {
					t.Fatalf("%s has no printer columns", version.Name)
				}
				if expected.kind == "EventListener" {
					want := []apiextensionsv1.CustomResourceColumnDefinition{
						{Name: "Address", Type: "string", JSONPath: ".status.address.url"},
						{Name: "Available", Type: "string", JSONPath: ".status.conditions[?(@.type=='Available')].status"},
						{Name: "Reason", Type: "string", JSONPath: ".status.conditions[?(@.type=='Available')].reason"},
						{Name: "Ready", Type: "string", JSONPath: ".status.conditions[?(@.type=='Ready')].status"},
						{Name: "Reason", Type: "string", JSONPath: ".status.conditions[?(@.type=='Ready')].reason"},
					}
					if !reflect.DeepEqual(version.AdditionalPrinterColumns, want) {
						t.Fatalf("printer columns = %#v, want %#v", version.AdditionalPrinterColumns, want)
					}
				} else if len(version.AdditionalPrinterColumns) != 1 || version.AdditionalPrinterColumns[0].Name != "Age" || version.AdditionalPrinterColumns[0].Type != "date" || version.AdditionalPrinterColumns[0].JSONPath != ".metadata.creationTimestamp" {
					t.Fatalf("unexpected printer columns: %#v", version.AdditionalPrinterColumns)
				}
			}
			jsonData, err := json.Marshal(&crd)
			if err != nil {
				t.Fatal(err)
			}
			if len(jsonData) > maxSchemaCRDJSONSize {
				t.Fatalf("serialized CRD is %d bytes, exceeds %d-byte client-side apply annotation limit", len(jsonData), maxSchemaCRDJSONSize)
			}
			t.Logf("serialized CRD is %d bytes", len(jsonData))
			crds[expected.kind] = &crd
		})
	}
	if _, ok := crds["Interceptor"]; !ok {
		t.Fatal("missing Interceptor schema")
	}
	if crd := crds["Interceptor"]; len(crd.Spec.Versions) != 1 || crd.Spec.Versions[0].Name != "v1alpha1" {
		t.Fatalf("unexpected Interceptor versions: %#v", crd.Spec.Versions)
	}
}

func collectPreserveUnknownPaths(schema *apiextensionsv1.JSONSchemaProps, path string, paths *[]string) {
	if schema.XPreserveUnknownFields != nil && *schema.XPreserveUnknownFields {
		*paths = append(*paths, path)
	}
	for name, child := range schema.Properties {
		collectPreserveUnknownPaths(&child, joinSchemaPath(path, name), paths)
	}
	for name, child := range schema.PatternProperties {
		collectPreserveUnknownPaths(&child, joinSchemaPath(path, name), paths)
	}
	if schema.Items != nil {
		if schema.Items.Schema != nil {
			collectPreserveUnknownPaths(schema.Items.Schema, path+"[]", paths)
		}
		for i := range schema.Items.JSONSchemas {
			collectPreserveUnknownPaths(&schema.Items.JSONSchemas[i], path+"[]", paths)
		}
	}
	if schema.AdditionalProperties != nil && schema.AdditionalProperties.Schema != nil {
		collectPreserveUnknownPaths(schema.AdditionalProperties.Schema, path+"{}", paths)
	}
	if schema.AdditionalItems != nil && schema.AdditionalItems.Schema != nil {
		collectPreserveUnknownPaths(schema.AdditionalItems.Schema, path+"[]", paths)
	}
	for i := range schema.AllOf {
		collectPreserveUnknownPaths(&schema.AllOf[i], path+"&", paths)
	}
	for i := range schema.OneOf {
		collectPreserveUnknownPaths(&schema.OneOf[i], path+"|", paths)
	}
	for i := range schema.AnyOf {
		collectPreserveUnknownPaths(&schema.AnyOf[i], path+"?", paths)
	}
	if schema.Not != nil {
		collectPreserveUnknownPaths(schema.Not, path+"!", paths)
	}
}

func joinSchemaPath(parent, child string) string {
	if parent == "" {
		return child
	}
	return parent + "." + child
}

func TestSchemaCompatibilityBoundaries(t *testing.T) {
	trigger := readSchemaCRD(t, "300-trigger.yaml", "v1beta1")
	triggerRoot := trigger.Schema.OpenAPIV3Schema
	triggerSpec := mustSchemaAt(t, triggerRoot, "spec")
	if !hasRequired(triggerSpec.Required, "template") || hasRequired(triggerSpec.Required, "bindings") {
		t.Fatalf("TriggerSpec required fields = %v, want template and optional bindings", triggerSpec.Required)
	}
	resourceTemplate := mustSchemaAt(t, triggerRoot, "spec", "template", "spec", "resourcetemplates", "items")
	assertPreserveUnknown(t, resourceTemplate, "inline TriggerTemplate resource")
	jsonValue := mustSchemaAt(t, triggerRoot, "spec", "interceptors", "items", "params", "items", "value")
	assertPreserveUnknown(t, jsonValue, "interceptor params.value")
	if jsonValue.Type != "" {
		t.Fatalf("interceptor params.value has restrictive type %q", jsonValue.Type)
	}
	headerValue := mustSchemaAt(t, triggerRoot, "spec", "interceptors", "items", "webhook", "header", "items", "value")
	assertPreserveUnknown(t, headerValue, "webhook header.value")
	if headerValue.Type != "" {
		t.Fatalf("webhook header.value has restrictive type %q", headerValue.Type)
	}
	if ref := mustSchemaAt(t, triggerRoot, "spec", "interceptors", "items"); hasRequired(ref.Required, "ref") {
		t.Fatalf("TriggerInterceptor unexpectedly requires ref: %v", ref.Required)
	}
	for _, file := range []string{"300-triggerbinding.yaml", "300-clustertriggerbinding.yaml"} {
		binding := readSchemaCRD(t, file, "v1beta1")
		params := mustSchemaAt(t, binding.Schema.OpenAPIV3Schema, "spec", "params", "items")
		if hasRequired(params.Required, "name") || hasRequired(params.Required, "value") {
			t.Fatalf("%s Param required fields = %v, want name/value optional", file, params.Required)
		}
	}

	template := readSchemaCRD(t, "300-triggertemplate.yaml", "v1beta1")
	templateSpec := mustSchemaAt(t, template.Schema.OpenAPIV3Schema, "spec")
	if !hasRequired(templateSpec.Required, "resourcetemplates") {
		t.Fatalf("TriggerTemplate required fields = %v, want resourcetemplates", templateSpec.Required)
	}
	resourceTemplates := mustSchemaAt(t, template.Schema.OpenAPIV3Schema, "spec", "resourcetemplates")
	if resourceTemplates.MinItems == nil || *resourceTemplates.MinItems != 1 {
		t.Fatalf("resourcetemplates minItems = %v, want 1", resourceTemplates.MinItems)
	}
	assertPreserveUnknown(t, mustSchemaAt(t, template.Schema.OpenAPIV3Schema, "spec", "resourcetemplates", "items"), "TriggerTemplate resource")
	params := mustSchemaAt(t, template.Schema.OpenAPIV3Schema, "spec", "params", "items")
	if hasRequired(params.Required, "name") {
		t.Fatalf("TriggerTemplate ParamSpec unexpectedly requires name: %v", params.Required)
	}

	eventListener := readSchemaCRD(t, "300-eventlistener.yaml", "v1beta1")
	elRoot := eventListener.Schema.OpenAPIV3Schema
	customResource := mustSchemaAt(t, elRoot, "spec", "resources", "customResource")
	assertPreserveUnknown(t, customResource, "EventListener customResource")
	podSpec := mustSchemaAt(t, elRoot, "spec", "resources", "kubernetesResource", "spec", "template", "spec")
	if hasRequired(podSpec.Required, "containers") {
		t.Fatal("partial EventListener Pod spec requires containers")
	}
	containers := mustSchemaAt(t, elRoot, "spec", "resources", "kubernetesResource", "spec", "template", "spec", "containers")
	if containers.XListType == nil || *containers.XListType != "atomic" || len(containers.XListMapKeys) != 0 {
		t.Fatalf("partial Pod containers topology = %v/%v, want atomic", containers.XListType, containers.XListMapKeys)
	}
	if hasRequired(containers.Items.Schema.Required, "name") {
		t.Fatal("partial EventListener containers require a name")
	}
	env := mustSchemaAt(t, elRoot, "spec", "resources", "kubernetesResource", "spec", "template", "spec", "containers", "items", "env", "items")
	if !hasRequired(env.Required, "name") {
		t.Fatalf("container env required fields = %v, want name preserved", env.Required)
	}
	groups := mustSchemaAt(t, elRoot, "spec", "triggerGroups", "items")
	if !hasRequired(groups.Required, "interceptors") || !hasRequired(groups.Required, "triggerSelector") || hasRequired(groups.Required, "name") {
		t.Fatalf("TriggerGroup required fields = %v, want interceptors and triggerSelector only", groups.Required)
	}
	groupInterceptors := mustSchemaAt(t, elRoot, "spec", "triggerGroups", "items", "interceptors")
	if groupInterceptors.MinItems == nil || *groupInterceptors.MinItems != 1 {
		t.Fatalf("TriggerGroup interceptors minItems = %v, want 1", groupInterceptors.MinItems)
	}
	alphaEventListener := readSchemaCRD(t, "300-eventlistener.yaml", "v1alpha1")
	alphaSpec := mustSchemaAt(t, alphaEventListener.Schema.OpenAPIV3Schema, "spec")
	if hasRequired(alphaSpec.Required, "triggers") {
		t.Fatalf("alpha EventListener unexpectedly requires triggers with selectors: %v", alphaSpec.Required)
	}

	interceptor := readSchemaCRD(t, "300-interceptor.yaml", "v1alpha1")
	if hasRequired(interceptor.Schema.OpenAPIV3Schema.Required, "spec") || hasRequired(mustSchemaAt(t, interceptor.Schema.OpenAPIV3Schema, "spec").Required, "clientConfig") {
		t.Fatal("Interceptor spec/clientConfig became required")
	}
	url := mustSchemaAt(t, interceptor.Schema.OpenAPIV3Schema, "spec", "clientConfig", "url")
	if url.Type != "string" {
		t.Fatalf("interceptor URL schema type = %q, want string", url.Type)
	}
	port := mustSchemaAt(t, interceptor.Schema.OpenAPIV3Schema, "spec", "clientConfig", "service", "port")
	if port.Type != "integer" || port.Format != "int32" {
		t.Fatalf("service port schema = %s/%s, want integer/int32", port.Type, port.Format)
	}
}

func TestInterceptorParamValuesPreserveNull(t *testing.T) {
	for _, test := range []struct {
		kind    string
		file    string
		version string
		path    []string
	}{
		{"Trigger", "300-trigger.yaml", "v1alpha1", []string{"spec", "interceptors", "items", "params", "items", "value"}},
		{"Trigger", "300-trigger.yaml", "v1beta1", []string{"spec", "interceptors", "items", "params", "items", "value"}},
		{"EventListener", "300-eventlistener.yaml", "v1alpha1", []string{"spec", "triggers", "items", "interceptors", "items", "params", "items", "value"}},
		{"EventListener", "300-eventlistener.yaml", "v1beta1", []string{"spec", "triggers", "items", "interceptors", "items", "params", "items", "value"}},
		{"EventListener", "300-eventlistener.yaml", "v1beta1", []string{"spec", "triggerGroups", "items", "interceptors", "items", "params", "items", "value"}},
	} {
		t.Run(test.kind+"/"+test.version+"/"+strings.Join(test.path, "."), func(t *testing.T) {
			version := readSchemaCRD(t, test.file, test.version)
			value := mustSchemaAt(t, version.Schema.OpenAPIV3Schema, test.path...)
			assertPreserveUnknown(t, value, "interceptor params.value")
			if !value.Nullable {
				t.Fatal("interceptor params.value is not nullable")
			}
			if value.Type != "" {
				t.Fatalf("interceptor params.value is restricted to type %q", value.Type)
			}
		})
	}
}

func assertStructuralShape(t *testing.T, schema *apiextensionsv1.JSONSchemaProps, path string) {
	t.Helper()
	if schema.Type == "" && (schema.XPreserveUnknownFields == nil || !*schema.XPreserveUnknownFields) && !schema.XIntOrString {
		t.Errorf("%s has no structural type or preserve-unknown marker", path)
	}
	if len(schema.Properties) > 0 && schema.Type != "object" {
		t.Errorf("%s has properties with type %q", path, schema.Type)
	}
	if schema.Items != nil && schema.Type != "array" {
		t.Errorf("%s has items with type %q", path, schema.Type)
	}
	for _, required := range schema.Required {
		if _, ok := schema.Properties[required]; !ok {
			t.Errorf("%s requires missing property %q", path, required)
		}
	}
	for name, child := range schema.Properties {
		assertStructuralShape(t, &child, path+"."+name)
	}
	if schema.Items != nil {
		if schema.Items.Schema != nil {
			assertStructuralShape(t, schema.Items.Schema, path+"[]")
		}
		for i := range schema.Items.JSONSchemas {
			assertStructuralShape(t, &schema.Items.JSONSchemas[i], path+"[]")
		}
	}
	if schema.AdditionalProperties != nil && schema.AdditionalProperties.Schema != nil {
		assertStructuralShape(t, schema.AdditionalProperties.Schema, path+"{}")
	}
}

func readSchemaCRD(t *testing.T, file, versionName string) *apiextensionsv1.CustomResourceDefinitionVersion {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "..", "config", file))
	if err != nil {
		t.Fatal(err)
	}
	var crd apiextensionsv1.CustomResourceDefinition
	if err := yaml.Unmarshal(data, &crd); err != nil {
		t.Fatal(err)
	}
	for i := range crd.Spec.Versions {
		if crd.Spec.Versions[i].Name == versionName {
			return &crd.Spec.Versions[i]
		}
	}
	t.Fatalf("%s is missing version %s", file, versionName)
	return nil
}

func mustSchemaAt(t *testing.T, root *apiextensionsv1.JSONSchemaProps, path ...string) *apiextensionsv1.JSONSchemaProps {
	t.Helper()
	current := root
	for _, field := range path {
		if field == "items" {
			if current.Items == nil || current.Items.Schema == nil {
				t.Fatalf("schema is missing items schema at path %v", path)
			}
			current = current.Items.Schema
			continue
		}
		child, ok := current.Properties[field]
		if !ok {
			t.Fatalf("schema is missing property %q at path %v", field, path)
		}
		current = &child
	}
	return current
}

func assertPreserveUnknown(t *testing.T, schema *apiextensionsv1.JSONSchemaProps, path string) {
	t.Helper()
	if schema.XPreserveUnknownFields == nil || !*schema.XPreserveUnknownFields {
		t.Fatalf("%s does not preserve unknown fields", path)
	}
}

func hasRequired(fields []string, wanted string) bool {
	for _, field := range fields {
		if field == wanted {
			return true
		}
	}
	return false
}
