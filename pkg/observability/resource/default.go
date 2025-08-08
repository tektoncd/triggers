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

package resource

import (
	"os"
)

const otelServiceNameKey = "OTEL_SERVICE_NAME"

// Default returns a default service name for OpenTelemetry resource.
//
// It will return:
// - The provided service name, or
// - OTEL_SERVICE_NAME environment variable if set
func Default(serviceName string) string {
	// If the OTEL_SERVICE_NAME is set then let this override
	// our own serviceName
	if name := os.Getenv(otelServiceNameKey); name != "" {
		return name
	}
	return serviceName
}
