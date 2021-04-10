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

package main

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	triggersclientset "github.com/tektoncd/triggers/pkg/client/clientset/versioned"
	"github.com/tektoncd/triggers/pkg/interceptors"
	"github.com/tektoncd/triggers/pkg/interceptors/server"
	"go.uber.org/zap"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeclientset "k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/rest"
	kubeclient "knative.dev/pkg/client/injection/kube/client"
	secretInformer "knative.dev/pkg/client/injection/kube/informers/core/v1/secret"
	"knative.dev/pkg/injection"
	"knative.dev/pkg/logging"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	"github.com/tektoncd/triggers/pkg/interceptors/bitbucket"
	"github.com/tektoncd/triggers/pkg/interceptors/cel"
	"github.com/tektoncd/triggers/pkg/interceptors/github"
	"github.com/tektoncd/triggers/pkg/interceptors/gitlab"
	"github.com/tektoncd/triggers/pkg/interceptors/sdk"
	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes"
	"knative.dev/pkg/signals"
	certresources "knative.dev/pkg/webhook/certificates/resources"
)

func main() {

	// Set up a signal context with our interceptor options
	ctx := sdk.WithOptions(signals.NewContext(), sdk.Options{
		Port: 8082,
	})

	// FIXME: call getCerts
	sdk.InterceptorMainWithConfig(ctx, "interceptors", map[string]func(kubernetes.Interface, *zap.SugaredLogger) v1alpha1.InterceptorInterface{
		"bitbucket": bitbucket.NewInterceptor,
		"cel":       cel.NewInterceptor,
		"github":    github.NewInterceptor,
		"gitlab":    gitlab.NewInterceptor,
	})
}

// getCerts uses Knative pkg to generate certs for clusterinterceptor to run as https
func getCerts(ctx context.Context, secretLister v1.SecretLister, kubeClient *kubeclientset.Clientset, logger *zap.SugaredLogger) (string, string, []byte, error) {
	interceptorSvcName := os.Getenv("INTERCEPTOR_TLS_SVC_NAME")
	interceptorSecretName := os.Getenv("INTERCEPTOR_TLS_SECRET_NAME")
	namespace := os.Getenv("SYSTEM_NAMESPACE")

	secret, err := secretLister.Secrets(namespace).Get(interceptorSecretName)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// The secret should be created explicitly by a higher-level system
			// that's responsible for install/updates.  We simply populate the
			// secret information.
			logger.Infof("secret %s is missing", interceptorSecretName)
			return "", "", []byte{}, err
		}
		logger.Infof("error accessing certificate secret %q: %v", interceptorSecretName, err)
		return "", "", []byte{}, err
	}

	var (
		serverKey, serverCert, caCert []byte
		createCerts                   bool
	)
	serverKey, haskey := secret.Data[certresources.ServerKey]
	if !haskey {
		logger.Infof("secret %q is missing key %q", secret.Name, certresources.ServerKey)
		createCerts = true
	}
	serverCert, haskey = secret.Data[certresources.ServerCert]
	if !haskey {
		logger.Infof("secret %q is missing key %q", secret.Name, certresources.ServerCert)
		createCerts = true
	}
	caCert, haskey = secret.Data[certresources.CACert]
	if !haskey {
		logger.Infof("secret %q is missing key %q", secret.Name, certresources.CACert)
		createCerts = true
	}

	// TODO: Certification validation and rotation is pending

	if createCerts {
		serverKey, serverCert, caCert, err = certresources.CreateCerts(ctx, interceptorSvcName, namespace, time.Now().Add(decade))
		if err != nil {
			logger.Errorf("failed to create certs : %v", err)
			return "", "", []byte{}, err
		}

		secret.Data = map[string][]byte{
			certresources.ServerKey:  serverKey,
			certresources.ServerCert: serverCert,
			certresources.CACert:     caCert,
		}
		if _, err = kubeClient.CoreV1().Secrets(namespace).Update(ctx, secret, metav1.UpdateOptions{}); err != nil {
			logger.Errorf("failed to update secret : %v", err)
			return "", "", []byte{}, err
		}
	}

	// write serverKey to file so that it can be passed while running https server.
	if err = ioutil.WriteFile(keyFile, serverKey, 0600); err != nil {
		logger.Errorf("failed to write serverKey file %v", err)
		return "", "", []byte{}, err
	}

	// write serverCert to file so that it can be passed while running https server.
	if err = ioutil.WriteFile(certFile, serverCert, 0600); err != nil {
		logger.Errorf("failed to write serverCert file %v", err)
		return "", "", []byte{}, err
	}
	return keyFile, certFile, caCert, nil
}
