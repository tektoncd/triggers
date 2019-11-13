package github

import (
	"errors"
	"net/http"

	gh "github.com/google/go-github/github"

	"github.com/tektoncd/triggers/pkg/interceptors"

	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"

	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type Interceptor struct {
	KubeClientSet          kubernetes.Interface
	Logger                 *zap.SugaredLogger
	Github                 *triggersv1.GithubInterceptor
	EventListenerNamespace string
}

func NewInterceptor(gh *triggersv1.GithubInterceptor, k kubernetes.Interface, ns string, l *zap.SugaredLogger) interceptors.Interceptor {
	return &Interceptor{
		Logger:                 l,
		Github:                 gh,
		KubeClientSet:          k,
		EventListenerNamespace: ns,
	}
}

func getSecretToken(cs kubernetes.Interface, gh *triggersv1.GithubInterceptor, eventListenerNamespace string) ([]byte, error) {
	ns := gh.SecretRef.Namespace
	if ns == "" {
		ns = eventListenerNamespace
	}
	secret, err := cs.CoreV1().Secrets(ns).Get(gh.SecretRef.SecretName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return secret.Data[gh.SecretRef.SecretKey], nil
}

func (w *Interceptor) ExecuteTrigger(payload []byte, request *http.Request, _ *triggersv1.EventListenerTrigger, _ string) ([]byte, error) {
	// No secret set, just continue
	if w.Github.SecretRef == nil {
		return payload, nil
	}
	header := request.Header.Get("X-Hub-Signature")
	if header == "" {
		return nil, errors.New("no X-Hub-Signature header set")
	}

	secretToken, err := getSecretToken(w.KubeClientSet, w.Github, w.EventListenerNamespace)
	if err != nil {
		return nil, err
	}
	if err := gh.ValidateSignature(header, payload, secretToken); err != nil {
		return nil, err
	}

	return payload, nil
}
