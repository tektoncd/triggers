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

package gitlab

import (
	"crypto/subtle"
	"errors"
	"net/http"

	"github.com/tektoncd/triggers/pkg/interceptors"

	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"

	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes"
)

type Interceptor struct {
	KubeClientSet          kubernetes.Interface
	Logger                 *zap.SugaredLogger
	Gitlab                 *triggersv1.GitlabInterceptor
	EventListenerNamespace string
}

func NewInterceptor(gl *triggersv1.GitlabInterceptor, k kubernetes.Interface, ns string, l *zap.SugaredLogger) interceptors.Interceptor {
	return &Interceptor{
		Logger:                 l,
		Gitlab:                 gl,
		KubeClientSet:          k,
		EventListenerNamespace: ns,
	}
}

func (w *Interceptor) ExecuteTrigger(payload []byte, request *http.Request, _ *triggersv1.EventListenerTrigger, _ string) ([]byte, error) {
	// No secret set, just continue
	if w.Gitlab.SecretRef == nil {
		return payload, nil
	}
	header := request.Header.Get("X-Gitlab-Token")
	if header == "" {
		return nil, errors.New("no X-Gitlab-Token header set")
	}

	secretToken, err := interceptors.GetSecretToken(w.KubeClientSet, w.Gitlab.SecretRef, w.EventListenerNamespace)
	if err != nil {
		return nil, err
	}

	// Make sure to use a constant time comparison here.
	if subtle.ConstantTimeCompare([]byte(header), secretToken) == 0 {
		return nil, errors.New("Invalid X-Gitlab-Token")
	}

	return payload, nil
}
