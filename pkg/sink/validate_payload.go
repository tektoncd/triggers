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

package sink

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

func (r Sink) IsValidPayload(eventHandler http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		payload, err := ioutil.ReadAll(request.Body)
		request.Body = ioutil.NopCloser(bytes.NewBuffer(payload))
		if err != nil {
			r.Logger.Errorf("Error reading event body: %s", err)
			response.WriteHeader(http.StatusInternalServerError)
			return
		}
		var event map[string]interface{}
		if err := json.Unmarshal([]byte(payload), &event); err != nil {
			errMsg := fmt.Sprintf("Invalid event body format format: %s", err)
			r.Logger.Error(errMsg)
			response.WriteHeader(http.StatusBadRequest)
			response.Header().Set("Content-Type", "application/json")
			body := Response{
				EventListener: r.EventListenerName,
				Namespace:     r.EventListenerNamespace,
				ErrorMessage:  errMsg,
			}
			if err := json.NewEncoder(response).Encode(body); err != nil {
				r.Logger.Errorf("failed to write back sink response: %v", err)
			}
			return
		}
		eventHandler.ServeHTTP(response, request)
	})
}
