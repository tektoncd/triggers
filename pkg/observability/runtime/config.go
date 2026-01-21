/*
Copyright 2025 The Knative Authors

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

package runtime

import (
	"fmt"
	"time"
)

const (
	ProfilingEnabled  = "enabled"
	ProfilingDisabled = "disabled"
)

type Config struct {
	Profiling      string        `json:"profiling,omitempty"`
	ExportInterval time.Duration `json:"exportInterval,omitempty"`
}

func (c *Config) Validate() error {
	switch c.Profiling {
	case ProfilingEnabled, ProfilingDisabled:
	default:
		return fmt.Errorf("unsupported profile setting %q", c.Profiling)
	}

	// ExportInterval == 0 => OTel will use a default value
	if c.ExportInterval < 0 {
		return fmt.Errorf("export interval %q should be greater than zero", c.ExportInterval)
	}
	return nil
}

func (c *Config) ProfilingEnabled() bool {
	return c.Profiling == ProfilingEnabled
}

func DefaultConfig() Config {
	return Config{
		Profiling: ProfilingDisabled,
		// same as OTel runtime.DefaultMinimumReadMemStatsInterval
		ExportInterval: 15 * time.Second,
	}
}

func NewFromMap(m map[string]string) (Config, error) {
	c := DefaultConfig()

	if val, ok := m["runtime-profiling"]; ok {
		c.Profiling = val
	}
	if val, ok := m["runtime-export-interval"]; ok {
		if duration, err := time.ParseDuration(val); err != nil {
			return c, fmt.Errorf("invalid duration %q: %w", val, err)
		} else {
			c.ExportInterval = duration
		}
	}

	return c, c.Validate()
}
