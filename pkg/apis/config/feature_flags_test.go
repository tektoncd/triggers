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
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/tektoncd/triggers/pkg/apis/config"
	"github.com/tektoncd/triggers/test"
)

func TestNewFeatureFlagsFromConfigMap(t *testing.T) {
	type testCase struct {
		expectedConfig *config.FeatureFlags
		fileName       string
	}

	testCases := []testCase{{
		expectedConfig: &config.FeatureFlags{
			EnableAPIFields: "stable",
		},
		fileName: config.GetFeatureFlagsConfigName(),
	}, {
		expectedConfig: &config.FeatureFlags{
			EnableAPIFields:                              "alpha",
			LabelsExclusionPattern:                       "^abc-",
			InterceptorsGitHubUseEnterpriseHostAllowlist: true,
		},
		fileName: "feature-flags-all-flags-set",
	}, {
		expectedConfig: &config.FeatureFlags{
			EnableAPIFields: "stable",
		},
		fileName: "feature-flags-upper-case",
	}}

	for _, tc := range testCases {
		fileName := tc.fileName
		expectedConfig := tc.expectedConfig
		t.Run(fileName, func(t *testing.T) {
			verifyConfigFileWithExpectedFeatureFlagsConfig(t, fileName, expectedConfig)
		})
	}
}

func TestNewFeatureFlagsFromEmptyConfigMap(t *testing.T) {
	FeatureFlagsConfigEmptyName := "feature-flags-empty"
	expectedConfig := &config.FeatureFlags{
		EnableAPIFields: "stable",
	}
	verifyConfigFileWithExpectedFeatureFlagsConfig(t, FeatureFlagsConfigEmptyName, expectedConfig)
}

func TestGetFeatureFlagsConfigName(t *testing.T) {
	for _, tc := range []struct {
		description         string
		featureFlagEnvValue string
		expected            string
	}{{
		description:         "Feature flags config value not set",
		featureFlagEnvValue: "",
		expected:            "feature-flags-triggers",
	}, {
		description:         "Feature flags config value set",
		featureFlagEnvValue: "feature-flags-test",
		expected:            "feature-flags-test",
	}} {
		t.Run(tc.description, func(t *testing.T) {
			if tc.featureFlagEnvValue != "" {
				t.Setenv("CONFIG_FEATURE_FLAGS_NAME", tc.featureFlagEnvValue)
			}
			got := config.GetFeatureFlagsConfigName()
			want := tc.expected
			if got != want {
				t.Errorf("GetFeatureFlagsConfigName() = %s, want %s", got, want)
			}
		})
	}
}

func TestNewFeatureFlagsConfigMapErrors(t *testing.T) {
	for _, tc := range []struct {
		fileName string
	}{{
		fileName: "feature-flags-invalid-enable-api-fields",
	}, {
		fileName: "feature-flags-invalid-exclusion-pattern-fields",
	}} {
		t.Run(tc.fileName, func(t *testing.T) {
			cm := test.ConfigMapFromTestFile(t, tc.fileName)
			if _, err := config.NewFeatureFlagsFromConfigMap(cm); err == nil {
				t.Error("expected error but received nil")
			}
		})
	}
}

func TestNewFeatureFlagsFromMap_EnterpriseHostAllowlist(t *testing.T) {
	for _, tc := range []struct {
		name string
		data map[string]string
		want bool
	}{{
		name: "key absent defaults to false",
		data: map[string]string{},
		want: false,
	}, {
		name: "explicit true",
		data: map[string]string{
			"interceptors.github.use-enterprise-host-allowlist": "true",
		},
		want: true,
	}, {
		name: "explicit false",
		data: map[string]string{
			"interceptors.github.use-enterprise-host-allowlist": "false",
		},
		want: false,
	}, {
		name: "case insensitive True",
		data: map[string]string{
			"interceptors.github.use-enterprise-host-allowlist": "True",
		},
		want: true,
	}} {
		t.Run(tc.name, func(t *testing.T) {
			flags, err := config.NewFeatureFlagsFromMap(tc.data)
			if err != nil {
				t.Fatalf("NewFeatureFlagsFromMap() error = %v", err)
			}
			if flags.InterceptorsGitHubUseEnterpriseHostAllowlist != tc.want {
				t.Errorf("InterceptorsGitHubUseEnterpriseHostAllowlist = %v, want %v",
					flags.InterceptorsGitHubUseEnterpriseHostAllowlist, tc.want)
			}
		})
	}
}

func verifyConfigFileWithExpectedFeatureFlagsConfig(t *testing.T, fileName string, expectedConfig *config.FeatureFlags) {
	cm := test.ConfigMapFromTestFile(t, fileName)
	if flags, err := config.NewFeatureFlagsFromConfigMap(cm); err == nil {
		if d := cmp.Diff(expectedConfig, flags); d != "" {
			t.Errorf("Diff(-want/+got):\n%s", d)
		}
	} else {
		t.Errorf("NewFeatureFlagsFromConfigMap(actual) = %v", err)
	}
}
