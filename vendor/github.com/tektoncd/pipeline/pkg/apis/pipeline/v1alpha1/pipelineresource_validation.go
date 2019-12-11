/*
Copyright 2019 The Tekton Authors

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

package v1alpha1

import (
	"context"
	"net/url"
	"strconv"
	"strings"

	"github.com/tektoncd/pipeline/pkg/apis/validate"
	"k8s.io/apimachinery/pkg/api/equality"
	"knative.dev/pkg/apis"
	logging "knative.dev/pkg/logging"
)

var _ apis.Validatable = (*PipelineResource)(nil)

func (r *PipelineResource) Validate(ctx context.Context) *apis.FieldError {
	if err := validate.ObjectMetadata(r.GetObjectMeta()); err != nil {
		return err.ViaField("metadata")
	}

	return r.Spec.Validate(ctx)
}

func (rs *PipelineResourceSpec) Validate(ctx context.Context) *apis.FieldError {
	if equality.Semantic.DeepEqual(rs, &PipelineResourceSpec{}) {
		return apis.ErrMissingField(apis.CurrentField)
	}
	if rs.Type == PipelineResourceTypeCluster {
		var authFound, cadataFound, nameFound, isInsecure bool
		for _, param := range rs.Params {
			switch {
			case strings.EqualFold(param.Name, "URL"):
				if err := validateURL(param.Value, "URL"); err != nil {
					return err
				}
			case strings.EqualFold(param.Name, "Username"):
				authFound = true
			case strings.EqualFold(param.Name, "CAData"):
				authFound = true
				cadataFound = true
			case strings.EqualFold(param.Name, "Token"):
				authFound = true
			case strings.EqualFold(param.Name, "name"):
				nameFound = true
			case strings.EqualFold(param.Name, "insecure"):
				b, _ := strconv.ParseBool(param.Value)
				isInsecure = b
			}
		}

		for _, secret := range rs.SecretParams {
			switch {
			case strings.EqualFold(secret.FieldName, "Username"):
				authFound = true
			case strings.EqualFold(secret.FieldName, "CAData"):
				authFound = true
				cadataFound = true
			}
		}

		if nameFound {
			logging.FromContext(ctx).Warn(
				"The name parameter on the cluster resource is deprecated. Support will be removed in a future release")
		}
		// One auth method must be supplied
		if !(authFound) {
			return apis.ErrMissingField("username or CAData  or token param")
		}
		if !cadataFound && !isInsecure {
			return apis.ErrMissingField("CAData param")
		}
	}
	if rs.Type == PipelineResourceTypeStorage {
		foundTypeParam := false
		var location string
		for _, param := range rs.Params {
			switch {
			case strings.EqualFold(param.Name, "type"):
				if !AllowedStorageType(param.Value) {
					return apis.ErrInvalidValue(param.Value, "spec.params.type")
				}
				foundTypeParam = true
			case strings.EqualFold(param.Name, "Location"):
				location = param.Value
			}
		}

		if !foundTypeParam {
			return apis.ErrMissingField("spec.params.type")
		}
		if location == "" {
			return apis.ErrMissingField("spec.params.location")
		}
	}

	for _, allowedType := range AllResourceTypes {
		if allowedType == rs.Type {
			return nil
		}
	}

	return apis.ErrInvalidValue("spec.type", string(rs.Type))
}

func AllowedStorageType(gotType string) bool {
	switch gotType {
	case string(PipelineResourceTypeGCS):
		return true
	case string(PipelineResourceTypeBuildGCS):
		return true
	}
	return false
}

func validateURL(u, path string) *apis.FieldError {
	if u == "" {
		return nil
	}
	_, err := url.ParseRequestURI(u)
	if err != nil {
		return apis.ErrInvalidValue(u, path)
	}
	return nil
}
