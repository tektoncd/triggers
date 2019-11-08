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

package sink

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"golang.org/x/xerrors"
	corev1 "k8s.io/api/core/v1"
)

// GetURI retrieves the ObjectReference to URI.
func GetURI(objRef *corev1.ObjectReference, ns string) (*url.URL, error) {
	// TODO: This should work for any Adressable.
	// Use something like https://github.com/knative/eventing-contrib/blob/7c0fc5cfa8bd44da0767d9e7b250264ea6eb7d8d/pkg/controller/sinks/sinks.go#L32
	if objRef.Kind == "Service" && objRef.APIVersion == "v1" {
		// TODO: Also assuming port 80 and http here. Use DNS/or the env vars?
		if objRef.Namespace != "" {
			ns = objRef.Namespace
		}
		return url.Parse(fmt.Sprintf("http://%s.%s.svc/", objRef.Name, ns))
	}
	return nil, xerrors.New("Invalid objRef")
}

func createOutgoingRequest(ctx context.Context, original *http.Request, url *url.URL, payload []byte, headerParams []pipelinev1.Param) *http.Request {
	r := original.Clone(ctx)
	r.RequestURI = "" // RequestURI cannot be set in outgoing requests
	r.URL = url
	r.Body = ioutil.NopCloser(bytes.NewBuffer(payload))
	addInterceptorHeaders(r.Header, headerParams)
	return r
}

func makeRequest(client *http.Client, request *http.Request) ([]byte, error) {
	resp, err := client.Do(request)
	if err != nil {
		// TODO: Add Error types - ValidationError and other General ProcessingErrors
		return nil, xerrors.Errorf("Failed to proxy request to interceptor: %w", err.Error())
	}
	defer resp.Body.Close()

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, xerrors.Errorf("Failed to parse response body")
	}

	// Wrap error and return
	if resp.StatusCode != http.StatusOK {
		return nil, xerrors.Errorf("Request rejected; status: %s; message: %s", resp.Status, respBody)
	}
	return respBody, nil
}

func addInterceptorHeaders(header http.Header, headerParams []pipelinev1.Param) {
	// This clobbers any matching headers
	for _, param := range headerParams {
		if param.Value.Type == pipelinev1.ParamTypeString {
			header[param.Name] = []string{param.Value.StringVal}
		} else {
			header[param.Name] = param.Value.ArrayVal
		}
	}
}
