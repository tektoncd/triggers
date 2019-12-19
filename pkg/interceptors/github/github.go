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

package github

import (
	"errors"
	"fmt"
	"net/http"

	gh "github.com/google/go-github/github"
	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	"github.com/tektoncd/triggers/pkg/interceptors"
	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes"
)

type Interceptor struct {
	KubeClientSet          kubernetes.Interface
	Logger                 *zap.SugaredLogger
	GitHub                 *triggersv1.GitHubInterceptor
	EventListenerNamespace string
}

func NewInterceptor(gh *triggersv1.GitHubInterceptor, k kubernetes.Interface, ns string, l *zap.SugaredLogger) interceptors.Interceptor {
	return &Interceptor{
		Logger:                 l,
		GitHub:                 gh,
		KubeClientSet:          k,
		EventListenerNamespace: ns,
	}
}

func (w *Interceptor) ExecuteTrigger(payload []byte, request *http.Request, _ *triggersv1.EventListenerTrigger, _ string) ([]byte, error) {
	// Validate secrets first before anything else, if set
	if w.GitHub.SecretRef != nil {
		header := request.Header.Get("X-Hub-Signature")
		if header == "" {
			return nil, errors.New("no X-Hub-Signature header set")
		}

		secretToken, err := interceptors.GetSecretToken(w.KubeClientSet, w.GitHub.SecretRef, w.EventListenerNamespace)
		if err != nil {
			return nil, err
		}
		if err := gh.ValidateSignature(header, payload, secretToken); err != nil {
			return nil, err
		}
	}

	// Next see if the event type is in the allow-list
	if w.GitHub.EventTypes != nil {
		actualEvent := request.Header.Get("X-GitHub-Event")
		isAllowed := false
		for _, allowedEvent := range w.GitHub.EventTypes {
			if actualEvent == allowedEvent {
				isAllowed = true
				break
			}
		}
		if !isAllowed {
			return nil, fmt.Errorf("event type %s is not allowed", actualEvent)
		}
	}

	return payload, nil
}
