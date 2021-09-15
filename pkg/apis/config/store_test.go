/*
Copyright 2021 The Tekton Authors

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

package config_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/tektoncd/triggers/pkg/apis/config"
	"github.com/tektoncd/triggers/test"
	logtesting "knative.dev/pkg/logging/testing"
)

func TestStoreLoadWithContext(t *testing.T) {
	defaultConfig := test.ConfigMapFromTestFile(t, "config-defaults-triggers")

	expectedDefaults, _ := config.NewDefaultsFromConfigMap(defaultConfig)

	expectedFeatureFlag, _ := config.NewFeatureFlagsFromConfigMap(defaultConfig)

	expected := &config.Config{
		Defaults:     expectedDefaults,
		FeatureFlags: expectedFeatureFlag,
	}

	store := config.NewStore(logtesting.TestLogger(t))
	store.OnConfigChanged(defaultConfig)

	cfg := config.FromContext(store.ToContext(context.Background()))

	if d := cmp.Diff(cfg, expected); d != "" {
		t.Errorf("Unexpected config %s", fmt.Sprintf("(-want, +got): %s", d))
	}
}

func TestFromContextOrDefaults(t *testing.T) {
	defaultConfigCM := test.ConfigMapFromTestFile(t, "config-defaults-triggers")
	defaults, _ := config.NewDefaultsFromConfigMap(defaultConfigCM)

	featureFlagsCM := test.ConfigMapFromTestFile(t, "feature-flags-triggers")
	featureFlags, _ := config.NewFeatureFlagsFromConfigMap(featureFlagsCM)

	for _, tc := range []struct {
		name string
		in   context.Context
		want *config.Config
	}{{
		name: "sets to default when context has no config",
		in:   context.Background(),
		want: &config.Config{
			Defaults:     defaults,
			FeatureFlags: featureFlags,
		},
	}, {
		name: "uses Config from context if present",
		in: config.ToContext(context.Background(), &config.Config{
			FeatureFlags: featureFlags,
		}),
		want: &config.Config{
			FeatureFlags: featureFlags,
		},
	}} {
		got := config.FromContextOrDefaults(tc.in)
		if diff := cmp.Diff(got, tc.want); diff != "" {
			t.Errorf("unexpected config (-want/+got): %s", diff)
		}
	}
}
