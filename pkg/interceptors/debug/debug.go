package debug

import (
	"context"
	"encoding/json"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"k8s.io/client-go/kubernetes"

	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
)

var _ triggersv1.InterceptorInterface = (*Interceptor)(nil)

type Interceptor struct {
	KubeClientSet kubernetes.Interface
	Logger        *zap.SugaredLogger
}

func NewInterceptor(k kubernetes.Interface, l *zap.SugaredLogger) *Interceptor {
	return &Interceptor{
		Logger:        l,
		KubeClientSet: k,
	}
}

func (w *Interceptor) Process(ctx context.Context, r *triggersv1.InterceptorRequest) *triggersv1.InterceptorResponse {
	reqContext, err := json.Marshal(r.Context)
	if err != nil {
		w.Logger.Errorf("failed to marshal request context for debug output: %s", err)
		return &triggersv1.InterceptorResponse{
			Continue: false,
			Status: triggersv1.Status{
				Message: err.Error(),
				Code:    codes.FailedPrecondition,
			},
		}
	}
	w.Logger.Infow("Debug Interceptor invoked", "headers", r.Header, "body", r.Body, "extensions", r.Extensions, "context", string(reqContext))
	return &triggersv1.InterceptorResponse{
		Continue: true,
	}
}
