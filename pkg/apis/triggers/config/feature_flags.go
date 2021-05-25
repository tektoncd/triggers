/*
Copyright 2020 The Tekton Authors

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

package config

import (
	"fmt"
	"os"
	"strings"

	corev1 "k8s.io/api/core/v1"
)

const (
	StableAPIFields        = "stable"
	AlphaAPIFields         = "alpha"
	enableAPIFields        = "enable-api-fields"
	DefaultEnableAPIFields = StableAPIFields
)

// FeatureFlags holds the features configurations
// +k8s:deepcopy-gen=true
type FeatureFlags struct {
	EnableAPIFields string
}

// GetFeatureFlagsConfigName returns the name of the configmap containing all
// feature flags.
func GetFeatureFlagsConfigName() string {
	if e := os.Getenv("CONFIG_FEATURE_FLAGS_NAME"); e != "" {
		return e
	}
	return "feature-flags-triggers"
}

// NewFeatureFlagsFromMap returns a Config given a map corresponding to a ConfigMap
func NewFeatureFlagsFromMap(cfgMap map[string]string) (*FeatureFlags, error) {
	tc := FeatureFlags{}
	if err := setEnabledAPIFields(cfgMap, DefaultEnableAPIFields, &tc.EnableAPIFields); err != nil {
		return nil, err
	}
	return &tc, nil
}

// setEnabledAPIFields sets the "enable-api-fields" flag based on the content of a given map.
// If the feature gate is invalid or missing then an error is returned.
func setEnabledAPIFields(cfgMap map[string]string, defaultValue string, feature *string) error {
	value := defaultValue
	if cfg, ok := cfgMap[enableAPIFields]; ok {
		value = strings.ToLower(cfg)
	}
	switch value {
	case AlphaAPIFields, StableAPIFields:
		*feature = value
	default:
		return fmt.Errorf("invalid value for feature flag %q: %q", enableAPIFields, value)
	}
	return nil
}

// FIXME: We have to wire this up to a main; probably using a store.
// NewFeatureFlagsFromConfigMap returns a Config for the given configmap
func NewFeatureFlagsFromConfigMap(config *corev1.ConfigMap) (*FeatureFlags, error) {
	return NewFeatureFlagsFromMap(config.Data)
}
