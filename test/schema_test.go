//go:build e2e
// +build e2e

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

package test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/openapi"
	knativetest "knative.dev/pkg/test"
)

func TestStructuralCRDsPublished(t *testing.T) {
	cfg, err := knativetest.BuildClientConfig(knativetest.Flags.Kubeconfig, knativetest.Flags.Cluster)
	if err != nil {
		t.Fatal(err)
	}
	dynamicClient, err := dynamic.NewForConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}

	want := []struct {
		name     string
		kind     string
		scope    apiextensionsv1.ResourceScope
		versions []string
	}{
		{"triggers.triggers.tekton.dev", "Trigger", apiextensionsv1.NamespaceScoped, []string{"v1alpha1", "v1beta1"}},
		{"triggerbindings.triggers.tekton.dev", "TriggerBinding", apiextensionsv1.NamespaceScoped, []string{"v1alpha1", "v1beta1"}},
		{"clustertriggerbindings.triggers.tekton.dev", "ClusterTriggerBinding", apiextensionsv1.ClusterScoped, []string{"v1alpha1", "v1beta1"}},
		{"triggertemplates.triggers.tekton.dev", "TriggerTemplate", apiextensionsv1.NamespaceScoped, []string{"v1alpha1", "v1beta1"}},
		{"eventlisteners.triggers.tekton.dev", "EventListener", apiextensionsv1.NamespaceScoped, []string{"v1alpha1", "v1beta1"}},
		{"interceptors.triggers.tekton.dev", "Interceptor", apiextensionsv1.NamespaceScoped, []string{"v1alpha1"}},
		{"clusterinterceptors.triggers.tekton.dev", "ClusterInterceptor", apiextensionsv1.ClusterScoped, []string{"v1alpha1"}},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	crds := make(map[string]*apiextensionsv1.CustomResourceDefinition, len(want))
	for _, expected := range want {
		crdObject, err := dynamicClient.Resource(schema.GroupVersionResource{Group: "apiextensions.k8s.io", Version: "v1", Resource: "customresourcedefinitions"}).Get(ctx, expected.name, metav1.GetOptions{})
		if err != nil {
			t.Fatalf("get CRD %s: %v", expected.name, err)
		}
		crdJSON, err := json.Marshal(crdObject.Object)
		if err != nil {
			t.Fatalf("marshal CRD %s: %v", expected.name, err)
		}
		var crd apiextensionsv1.CustomResourceDefinition
		if err := json.Unmarshal(crdJSON, &crd); err != nil {
			t.Fatalf("decode CRD %s: %v", expected.name, err)
		}
		crds[expected.name] = &crd
	}

	for _, expected := range want {
		crd := crds[expected.name]
		if crd.Spec.Group != "triggers.tekton.dev" || crd.Spec.Names.Kind != expected.kind || crd.Spec.Scope != expected.scope {
			t.Errorf("CRD %s identity = group %q, kind %q, scope %q", expected.name, crd.Spec.Group, crd.Spec.Names.Kind, crd.Spec.Scope)
		}
		if !hasCRDCondition(crd, apiextensionsv1.Established, apiextensionsv1.ConditionTrue) {
			t.Errorf("CRD %s is not Established", expected.name)
		}
		if hasCRDCondition(crd, apiextensionsv1.NonStructuralSchema, apiextensionsv1.ConditionTrue) {
			t.Errorf("CRD %s reports a non-structural schema", expected.name)
		}

		versions := make(map[string]struct{}, len(crd.Spec.Versions))
		for _, version := range crd.Spec.Versions {
			if !version.Served {
				continue
			}
			versions[version.Name] = struct{}{}
			if version.Schema == nil || version.Schema.OpenAPIV3Schema == nil || version.Schema.OpenAPIV3Schema.Type != "object" || len(version.Schema.OpenAPIV3Schema.Properties) == 0 {
				t.Errorf("CRD %s version %s has no structural root schema", expected.name, version.Name)
			}
		}
		for _, version := range expected.versions {
			if _, ok := versions[version]; !ok {
				t.Errorf("CRD %s does not serve %s", expected.name, version)
			}
		}
	}

	paths, err := discoveryClient.OpenAPIV3().Paths()
	if err != nil {
		t.Fatalf("read OpenAPI v3 discovery: %v", err)
	}
	documents := make(map[string]map[string]interface{})
	for _, expected := range want {
		for _, version := range expected.versions {
			document, ok := documents[version]
			if !ok {
				wantPath := "apis/triggers.tekton.dev/" + version
				var groupVersion openapi.GroupVersion
				for path, published := range paths {
					if strings.TrimPrefix(path, "/") == wantPath {
						groupVersion = published
						break
					}
				}
				if groupVersion == nil {
					t.Errorf("OpenAPI v3 discovery does not publish %s", wantPath)
					continue
				}
				data, err := groupVersion.Schema("application/json")
				if err != nil {
					t.Errorf("get OpenAPI v3 schema %s: %v", wantPath, err)
					continue
				}
				if err := json.Unmarshal(data, &document); err != nil {
					t.Errorf("decode OpenAPI v3 schema %s: %v", wantPath, err)
					continue
				}
				documents[version] = document
			}
			if err := validateOpenAPISchema(document, "triggers.tekton.dev", version, expected.kind, openAPIFields(expected.kind, version)); err != nil {
				t.Errorf("OpenAPI v3 schema for %s/%s: %v", expected.kind, version, err)
			}
		}
	}
}

type openAPIFieldAssertion struct {
	path        []string
	typeName    string
	description bool
}

func openAPIFields(kind, version string) []openAPIFieldAssertion {
	fields := []openAPIFieldAssertion{{path: []string{"spec"}, typeName: "object", description: true}}
	switch kind {
	case "Trigger":
		fields = append(fields, openAPIFieldAssertion{path: []string{"spec", "template"}, typeName: "object", description: true})
	case "TriggerBinding", "ClusterTriggerBinding":
		fields = append(fields, openAPIFieldAssertion{path: []string{"spec", "params"}, typeName: "array"})
	case "TriggerTemplate":
		fields = append(fields, openAPIFieldAssertion{path: []string{"spec", "resourcetemplates"}, typeName: "array", description: true})
	case "EventListener":
		fields = append(fields, openAPIFieldAssertion{path: []string{"spec", "resources"}, typeName: "object"})
		if version == "v1beta1" {
			fields = append(fields, openAPIFieldAssertion{path: []string{"spec", "triggerGroups"}, typeName: "array", description: true})
		}
	case "Interceptor", "ClusterInterceptor":
		fields = append(fields, openAPIFieldAssertion{path: []string{"spec", "clientConfig", "url"}, typeName: "string"})
	}
	return fields
}

func validateOpenAPISchema(document interface{}, group, version, kind string, fields []openAPIFieldAssertion) error {
	root, ok := findOpenAPISchemaForGVK(document, group, version, kind)
	if !ok {
		return fmt.Errorf("no component schema declares GVK %s/%s, Kind=%s", group, version, kind)
	}
	for _, assertion := range fields {
		property, ok := openAPIProperty(root, assertion.path...)
		if !ok {
			return fmt.Errorf("schema is missing properties.%s", strings.Join(assertion.path, ".properties."))
		}
		if got, _ := property["type"].(string); got != assertion.typeName {
			return fmt.Errorf("properties.%s has type %q, want %q", strings.Join(assertion.path, ".properties."), got, assertion.typeName)
		}
		if assertion.description {
			description, _ := property["description"].(string)
			if strings.TrimSpace(description) == "" {
				return fmt.Errorf("properties.%s has no description", strings.Join(assertion.path, ".properties."))
			}
		}
	}
	return nil
}

func findOpenAPISchemaForGVK(document interface{}, group, version, kind string) (map[string]interface{}, bool) {
	documentMap, ok := document.(map[string]interface{})
	if !ok {
		return nil, false
	}
	components, ok := documentMap["components"].(map[string]interface{})
	if !ok {
		return nil, false
	}
	schemas, ok := components["schemas"].(map[string]interface{})
	if !ok {
		return nil, false
	}
	for _, value := range schemas {
		candidate, ok := value.(map[string]interface{})
		if !ok {
			continue
		}
		gvks, ok := candidate["x-kubernetes-group-version-kind"].([]interface{})
		if !ok {
			continue
		}
		for _, value := range gvks {
			gvk, ok := value.(map[string]interface{})
			if ok && gvk["group"] == group && gvk["version"] == version && gvk["kind"] == kind {
				return candidate, true
			}
		}
	}
	return nil, false
}

func openAPIProperty(root map[string]interface{}, path ...string) (map[string]interface{}, bool) {
	current := root
	for _, field := range path {
		properties, ok := current["properties"].(map[string]interface{})
		if !ok {
			return nil, false
		}
		child, ok := properties[field].(map[string]interface{})
		if !ok {
			return nil, false
		}
		current = child
	}
	return current, true
}

func TestValidateOpenAPISchemaRequiresExactGVKAndPaths(t *testing.T) {
	trigger := map[string]interface{}{
		"x-kubernetes-group-version-kind": []interface{}{map[string]interface{}{
			"group": "triggers.tekton.dev", "version": "v1beta1", "kind": "Trigger",
		}},
		"properties": map[string]interface{}{
			"spec": map[string]interface{}{
				"type": "object", "description": "Trigger spec",
				"properties": map[string]interface{}{
					"template": map[string]interface{}{"type": "object", "description": "Template reference or inline spec"},
				},
			},
		},
	}
	eventListener := map[string]interface{}{
		"x-kubernetes-group-version-kind": []interface{}{map[string]interface{}{
			"group": "triggers.tekton.dev", "version": "v1beta1", "kind": "EventListener",
		}},
		"properties": map[string]interface{}{
			"spec": map[string]interface{}{
				"type": "object", "description": "EventListener spec",
				"properties": map[string]interface{}{
					"resources":     map[string]interface{}{"type": "object"},
					"triggerGroups": map[string]interface{}{"type": "array", "description": "Shared interceptor groups"},
				},
			},
		},
	}
	document := map[string]interface{}{"components": map[string]interface{}{"schemas": map[string]interface{}{
		"trigger": trigger, "eventListener": eventListener,
	}}}

	t.Run("valid exact GVK", func(t *testing.T) {
		if err := validateOpenAPISchema(document, "triggers.tekton.dev", "v1beta1", "Trigger", openAPIFields("Trigger", "v1beta1")); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("another Kind does not satisfy the request", func(t *testing.T) {
		if err := validateOpenAPISchema(document, "triggers.tekton.dev", "v1beta1", "ClusterTriggerBinding", openAPIFields("ClusterTriggerBinding", "v1beta1")); err == nil {
			t.Fatal("schema for another Kind satisfied the request")
		}
	})
	t.Run("another version does not satisfy the request", func(t *testing.T) {
		if err := validateOpenAPISchema(document, "triggers.tekton.dev", "v1alpha1", "Trigger", openAPIFields("Trigger", "v1alpha1")); err == nil {
			t.Fatal("schema for another version satisfied the request")
		}
	})
	t.Run("wrong root field type does not satisfy the request", func(t *testing.T) {
		wrongType := map[string]interface{}{"components": map[string]interface{}{"schemas": map[string]interface{}{
			"trigger": map[string]interface{}{
				"x-kubernetes-group-version-kind": trigger["x-kubernetes-group-version-kind"],
				"properties":                      map[string]interface{}{"spec": map[string]interface{}{"type": "array", "description": "decoy"}},
			},
		}}}
		if err := validateOpenAPISchema(wrongType, "triggers.tekton.dev", "v1beta1", "Trigger", openAPIFields("Trigger", "v1beta1")); err == nil {
			t.Fatal("wrongly typed spec satisfied the request")
		}
	})
	t.Run("nested spec does not satisfy the root path", func(t *testing.T) {
		withoutRootSpec := map[string]interface{}{"components": map[string]interface{}{"schemas": map[string]interface{}{
			"trigger": map[string]interface{}{
				"x-kubernetes-group-version-kind": trigger["x-kubernetes-group-version-kind"],
				"properties": map[string]interface{}{"metadata": map[string]interface{}{
					"properties": map[string]interface{}{"spec": map[string]interface{}{"type": "object", "description": "nested decoy"}},
				}},
			},
		}}}
		if err := validateOpenAPISchema(withoutRootSpec, "triggers.tekton.dev", "v1beta1", "Trigger", openAPIFields("Trigger", "v1beta1")); err == nil {
			t.Fatal("nested decoy spec satisfied the root spec requirement")
		}
	})
	t.Run("beta EventListener exposes described triggerGroups", func(t *testing.T) {
		if err := validateOpenAPISchema(document, "triggers.tekton.dev", "v1beta1", "EventListener", openAPIFields("EventListener", "v1beta1")); err != nil {
			t.Fatal(err)
		}
	})
}

func TestTriggerTemplateSchemaStrictAndPreservation(t *testing.T) {
	clients, namespace := setup(t)
	defer tearDown(t, clients, namespace)

	cfg, err := knativetest.BuildClientConfig(knativetest.Flags.Kubeconfig, knativetest.Flags.Cluster)
	if err != nil {
		t.Fatal(err)
	}
	dynamicClient, err := dynamic.NewForConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	resource := dynamicClient.Resource(schema.GroupVersionResource{Group: "triggers.tekton.dev", Version: "v1beta1", Resource: "triggertemplates"}).Namespace(namespace)
	ctx := context.Background()

	wantResource := map[string]interface{}{
		"apiVersion": "example.dev/v1",
		"kind":       "Widget",
		"metadata": map[string]interface{}{
			"name": "$(tt.params.name)",
			"labels": map[string]interface{}{
				"source": "template",
			},
		},
		"spec": map[string]interface{}{
			"nested":   map[string]interface{}{"value": "$(tt.params.value)"},
			"items":    []interface{}{"first", "second"},
			"nullable": nil,
		},
	}
	strict := metav1.CreateOptions{FieldValidation: "Strict"}
	wantJSON, err := json.Marshal(wantResource)
	if err != nil {
		t.Fatal(err)
	}
	newTemplate := func(version, name string) *unstructured.Unstructured {
		return &unstructured.Unstructured{Object: map[string]interface{}{
			"apiVersion": "triggers.tekton.dev/" + version,
			"kind":       "TriggerTemplate",
			"metadata":   map[string]interface{}{"name": name},
			"spec": map[string]interface{}{
				"params": []interface{}{
					map[string]interface{}{"name": "name"},
					map[string]interface{}{"name": "value"},
				},
				"resourcetemplates": []interface{}{wantResource},
			},
		}}
	}
	assertResourcePreserved := func(version string, object *unstructured.Unstructured) {
		t.Helper()
		resources, found, err := unstructured.NestedSlice(object.Object, "spec", "resourcetemplates")
		if err != nil || !found || len(resources) != 1 {
			t.Fatalf("%s stored resourcetemplates = %v (found=%t, err=%v), want one resource", version, resources, found, err)
		}
		gotJSON, err := json.Marshal(resources[0])
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(gotJSON, wantJSON) {
			t.Fatalf("%s stored resource template changed:\n got: %s\nwant: %s", version, gotJSON, wantJSON)
		}
	}

	for _, versions := range []struct {
		write string
		read  string
		name  string
	}{
		{write: "v1alpha1", read: "v1beta1", name: "schema-alpha-roundtrip"},
		{write: "v1beta1", read: "v1alpha1", name: "schema-beta-roundtrip"},
	} {
		writer := dynamicClient.Resource(schema.GroupVersionResource{Group: "triggers.tekton.dev", Version: versions.write, Resource: "triggertemplates"}).Namespace(namespace)
		reader := dynamicClient.Resource(schema.GroupVersionResource{Group: "triggers.tekton.dev", Version: versions.read, Resource: "triggertemplates"}).Namespace(namespace)
		object := newTemplate(versions.write, versions.name)
		if _, err := writer.Create(ctx, object, strict); err != nil {
			t.Fatalf("create %s TriggerTemplate with a free-form resource template: %v", versions.write, err)
		}
		stored, err := reader.Get(ctx, object.GetName(), metav1.GetOptions{})
		if err != nil {
			t.Fatalf("get %s TriggerTemplate through %s: %v", versions.write, versions.read, err)
		}
		assertResourcePreserved(versions.read, stored)

		stored.SetAnnotations(map[string]string{"schema-test": "metadata-only-update"})
		if _, err := reader.Update(ctx, stored, metav1.UpdateOptions{}); err != nil {
			t.Fatalf("metadata-only update through %s: %v", versions.read, err)
		}
		stored, err = writer.Get(ctx, object.GetName(), metav1.GetOptions{})
		if err != nil {
			t.Fatalf("get %s TriggerTemplate after cross-version update: %v", versions.write, err)
		}
		assertResourcePreserved(versions.write, stored)
	}

	invalid := newTemplate("v1beta1", "schema-strict-reject")
	spec, _, err := unstructured.NestedMap(invalid.Object, "spec")
	if err != nil {
		t.Fatal(err)
	}
	spec["resourceTemplte"] = []interface{}{wantResource}
	if err := unstructured.SetNestedMap(invalid.Object, spec, "spec"); err != nil {
		t.Fatal(err)
	}
	if _, err := resource.Create(ctx, invalid, strict); !apierrors.IsBadRequest(err) {
		t.Fatalf("strict create with a misspelled structural field error = %v, want BadRequest", err)
	}
}

func TestInterceptorParamNullPreserved(t *testing.T) {
	clients, namespace := setup(t)
	defer tearDown(t, clients, namespace)

	cfg, err := knativetest.BuildClientConfig(knativetest.Flags.Kubeconfig, knativetest.Flags.Cluster)
	if err != nil {
		t.Fatal(err)
	}
	dynamicClient, err := dynamic.NewForConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	paramInterceptor := func() map[string]interface{} {
		return map[string]interface{}{
			"ref": map[string]interface{}{"name": "custom"},
			"params": []interface{}{
				map[string]interface{}{"name": "nullable", "value": nil},
				map[string]interface{}{"name": "mixed", "value": map[string]interface{}{
					"string": "preserved", "integer": 7, "fraction": 1.25, "boolean": true,
					"array":  []interface{}{nil, "item", 3, false},
					"nested": map[string]interface{}{"null": nil},
				}},
			},
		}
	}
	createAndAssert := func(version, resourceName, name string, object map[string]interface{}, valuePath ...string) {
		t.Helper()
		resource := dynamicClient.Resource(schema.GroupVersionResource{Group: "triggers.tekton.dev", Version: version, Resource: resourceName}).Namespace(namespace)
		unstructuredObject := &unstructured.Unstructured{Object: object}
		if _, err := resource.Create(ctx, unstructuredObject, metav1.CreateOptions{FieldValidation: "Strict"}); err != nil {
			t.Fatalf("create %s %s with null interceptor param: %v", version, resourceName, err)
		}
		stored, err := resource.Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			t.Fatalf("get %s %s: %v", version, resourceName, err)
		}
		assertStoredNull(t, stored.Object, valuePath...)
		want := map[string]interface{}{
			"string": "preserved", "integer": 7, "fraction": 1.25, "boolean": true,
			"array":  []interface{}{nil, "item", 3, false},
			"nested": map[string]interface{}{"null": nil},
		}
		mixedPath := append([]string(nil), valuePath...)
		mixedPath[len(mixedPath)-2] = "1"
		got, found := valueAtPath(t, stored.Object, mixedPath...)
		if !found {
			t.Fatalf("stored mixed JSON value at %s was not found", strings.Join(mixedPath, "."))
		}
		gotJSON, err := json.Marshal(got)
		if err != nil {
			t.Fatal(err)
		}
		wantJSON, err := json.Marshal(want)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(gotJSON, wantJSON) {
			t.Fatalf("stored mixed JSON value changed: got %s, want %s", gotJSON, wantJSON)
		}
	}

	for _, version := range []string{"v1alpha1", "v1beta1"} {
		name := "schema-null-trigger-" + version
		createAndAssert(version, "triggers", name, map[string]interface{}{
			"apiVersion": "triggers.tekton.dev/" + version,
			"kind":       "Trigger",
			"metadata":   map[string]interface{}{"name": name, "namespace": namespace},
			"spec": map[string]interface{}{
				"template":     map[string]interface{}{"ref": "unused-template"},
				"interceptors": []interface{}{paramInterceptor()},
			},
		}, "spec", "interceptors", "0", "params", "0", "value")

		name = "schema-null-eventlistener-" + version
		listenerSpec := map[string]interface{}{
			"triggers": []interface{}{map[string]interface{}{
				"template":     map[string]interface{}{"ref": "unused-template"},
				"interceptors": []interface{}{paramInterceptor()},
			}},
		}
		if version == "v1beta1" {
			listenerSpec["triggerGroups"] = []interface{}{map[string]interface{}{
				"name":         "null-param-group",
				"interceptors": []interface{}{paramInterceptor()},
				"triggerSelector": map[string]interface{}{
					"namespaceSelector": map[string]interface{}{"matchNames": []interface{}{namespace}},
				},
			}}
		}
		createAndAssert(version, "eventlisteners", name, map[string]interface{}{
			"apiVersion": "triggers.tekton.dev/" + version,
			"kind":       "EventListener",
			"metadata":   map[string]interface{}{"name": name, "namespace": namespace},
			"spec":       listenerSpec,
		}, "spec", "triggers", "0", "interceptors", "0", "params", "0", "value")
		if version == "v1beta1" {
			listener, err := dynamicClient.Resource(schema.GroupVersionResource{Group: "triggers.tekton.dev", Version: version, Resource: "eventlisteners"}).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
			if err != nil {
				t.Fatal(err)
			}
			assertStoredNull(t, listener.Object, "spec", "triggerGroups", "0", "interceptors", "0", "params", "0", "value")
		}
	}
}

func valueAtPath(t *testing.T, object map[string]interface{}, path ...string) (interface{}, bool) {
	t.Helper()
	var current interface{} = object
	for _, field := range path {
		switch value := current.(type) {
		case map[string]interface{}:
			var found bool
			current, found = value[field]
			if !found {
				return nil, false
			}
		case []interface{}:
			index, err := strconv.Atoi(field)
			if err != nil || index < 0 || index >= len(value) {
				return nil, false
			}
			current = value[index]
		default:
			return nil, false
		}
	}
	return current, true
}

func assertStoredNull(t *testing.T, object map[string]interface{}, path ...string) {
	t.Helper()
	current, found := valueAtPath(t, object, path...)
	if !found {
		t.Fatalf("stored object is missing or cannot traverse %s", strings.Join(path, "."))
	}
	if current != nil {
		t.Fatalf("stored value at %s = %#v, want null", strings.Join(path, "."), current)
	}
}

func TestEventListenerPartialPodServerSideApply(t *testing.T) {
	clients, namespace := setup(t)
	defer tearDown(t, clients, namespace)

	cfg, err := knativetest.BuildClientConfig(knativetest.Flags.Kubeconfig, knativetest.Flags.Cluster)
	if err != nil {
		t.Fatal(err)
	}
	dynamicClient, err := dynamic.NewForConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	listeners := dynamicClient.Resource(schema.GroupVersionResource{Group: "triggers.tekton.dev", Version: "v1beta1", Resource: "eventlisteners"}).Namespace(namespace)
	ctx := context.Background()
	const name = "schema-partial-pod"

	apply := func(cpu string) []byte {
		podSpec := map[string]interface{}{"nodeSelector": map[string]interface{}{"disk": "ssd"}}
		if cpu != "" {
			podSpec["containers"] = []interface{}{map[string]interface{}{
				"resources": map[string]interface{}{"limits": map[string]interface{}{"cpu": cpu}},
			}}
		}
		object := map[string]interface{}{
			"apiVersion": "triggers.tekton.dev/v1beta1",
			"kind":       "EventListener",
			"metadata":   map[string]interface{}{"name": name, "namespace": namespace},
			"spec": map[string]interface{}{
				"namespaceSelector": map[string]interface{}{"matchNames": []interface{}{namespace}},
				"resources": map[string]interface{}{
					"kubernetesResource": map[string]interface{}{
						"spec": map[string]interface{}{
							"template": map[string]interface{}{
								"spec": podSpec,
							},
						},
					},
				},
			},
		}
		data, err := json.Marshal(object)
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	options := metav1.PatchOptions{FieldManager: "triggers-schema-e2e"}
	if _, err := listeners.Patch(ctx, name, types.ApplyPatchType, apply(""), options); err != nil {
		t.Fatalf("server-side apply EventListener with a partial Pod and no containers: %v", err)
	}
	if _, err := listeners.Patch(ctx, name, types.ApplyPatchType, apply("100m"), options); err != nil {
		t.Fatalf("server-side apply EventListener with partial Pod and unnamed container: %v", err)
	}
	updated, err := listeners.Patch(ctx, name, types.ApplyPatchType, apply("200m"), options)
	if err != nil {
		t.Fatalf("reapply EventListener with an atomic unnamed-container list: %v", err)
	}
	containers, found, err := unstructured.NestedSlice(updated.Object, "spec", "resources", "kubernetesResource", "spec", "template", "spec", "containers")
	if err != nil || !found || len(containers) != 1 {
		t.Fatalf("stored partial Pod containers = %v (found=%t, err=%v), want one", containers, found, err)
	}
	container, ok := containers[0].(map[string]interface{})
	if !ok {
		t.Fatalf("stored partial Pod container has type %T, want object", containers[0])
	}
	if _, hasName := container["name"]; hasName {
		t.Fatalf("server added a required container name to the partial Pod: %v", containers[0])
	}
	limits, found, err := unstructured.NestedStringMap(container, "resources", "limits")
	if err != nil || !found || limits["cpu"] != "200m" {
		t.Fatalf("stored partial Pod limits = %v (found=%t, err=%v), want cpu=200m", limits, found, err)
	}
}

func hasCRDCondition(crd *apiextensionsv1.CustomResourceDefinition, conditionType apiextensionsv1.CustomResourceDefinitionConditionType, status apiextensionsv1.ConditionStatus) bool {
	for _, condition := range crd.Status.Conditions {
		if condition.Type == conditionType && condition.Status == status {
			return true
		}
	}
	return false
}
