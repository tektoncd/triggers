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

package config

import (
	"fmt"
	"os"
	"strconv"

	corev1 "k8s.io/api/core/v1"
)

const (
	defaultServiceAccountKey   = "default-service-account"
	defaultRunAsUserKey        = "default-run-as-user"
	defaultRunAsGroupKey       = "default-run-as-group"
	defaultRunAsNonRootKey     = "default-run-as-non-root"
	DefaultServiceAccountValue = "default"
	defaultRunAsUserValue      = "65532"
	defaultRunAsGroupValue     = "65532"
	defaultRunAsNonRootValue   = true
)

// Defaults holds the default configurations
// +k8s:deepcopy-gen=true
type Defaults struct {
	DefaultServiceAccount string
	DefaultRunAsUser      string
	DefaultRunAsGroup     string
	DefaultRunAsNonRoot   bool
}

// GetDefaultsConfigName returns the name of the configmap containing all
// defined defaults.
func GetDefaultsConfigName() string {
	if e := os.Getenv("CONFIG_DEFAULTS_NAME"); e != "" {
		return e
	}
	return "config-defaults-triggers"
}

// Equals returns true if two Configs are identical
func (cfg *Defaults) Equals(other *Defaults) bool {
	if cfg == nil && other == nil {
		return true
	}

	if cfg == nil || other == nil {
		return false
	}

	return other.DefaultServiceAccount == cfg.DefaultServiceAccount &&
		other.DefaultRunAsUser == cfg.DefaultRunAsUser &&
		other.DefaultRunAsGroup == cfg.DefaultRunAsGroup &&
		other.DefaultRunAsNonRoot == cfg.DefaultRunAsNonRoot
}

// NewDefaultsFromMap returns a Config given a map corresponding to a ConfigMap
func NewDefaultsFromMap(cfgMap map[string]string) (*Defaults, error) {
	tc := Defaults{
		DefaultServiceAccount: DefaultServiceAccountValue,
		DefaultRunAsUser:      defaultRunAsUserValue,
		DefaultRunAsGroup:     defaultRunAsGroupValue,
		DefaultRunAsNonRoot:   defaultRunAsNonRootValue,
	}

	if defaultServiceAccount, ok := cfgMap[defaultServiceAccountKey]; ok {
		tc.DefaultServiceAccount = defaultServiceAccount
	}

	if defaultRunAsUser, ok := cfgMap[defaultRunAsUserKey]; ok {
		tc.DefaultRunAsUser = defaultRunAsUser
	}

	if defaultRunAsGroup, ok := cfgMap[defaultRunAsGroupKey]; ok {
		tc.DefaultRunAsGroup = defaultRunAsGroup
	}

	if defaultRunAsNonRoot, ok := cfgMap[defaultRunAsNonRootKey]; ok {
		if defaultRunAsNonRoot != "" {
			runAsNonRoot, err := strconv.ParseBool(defaultRunAsNonRoot)
			if err != nil {
				return nil, fmt.Errorf("failed parsing runAsGroup config %v", defaultRunAsNonRoot)
			}
			tc.DefaultRunAsNonRoot = runAsNonRoot
		} else {
			// if "" value is provided via configmap set back to default value which is true
			tc.DefaultRunAsNonRoot = defaultRunAsNonRootValue
		}

	}

	return &tc, nil
}

// NewDefaultsFromConfigMap returns a Config for the given configmap
func NewDefaultsFromConfigMap(config *corev1.ConfigMap) (*Defaults, error) {
	return NewDefaultsFromMap(config.Data)
}
