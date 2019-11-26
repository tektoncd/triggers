package cel

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"reflect"
	"testing"

	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/tektoncd/pipeline/pkg/logging"
	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	fakekubeclient "knative.dev/pkg/client/injection/kube/client/fake"
	rtesting "knative.dev/pkg/reconciler/testing"
)

// Allow configuration via a config map

func TestInterceptor_ExecuteTrigger(t *testing.T) {
	type args struct {
		payload []byte
	}
	tests := []struct {
		name    string
		CEL     *triggersv1.CELInterceptor
		args    args
		want    []byte
		wantErr bool
	}{
		{
			name: "simple body check with matching body",
			CEL: &triggersv1.CELInterceptor{
				Filter: "body.value == 'testing'",
			},
			args: args{
				payload: []byte(`{"value":"testing"}`),
			},
			want:    []byte(`{"value":"testing"}`),
			wantErr: false,
		},
		{
			name: "simple body check with non-matching body",
			CEL: &triggersv1.CELInterceptor{
				Filter: "body.value == 'test'",
			},
			args: args{
				payload: []byte(`{"value":"testing"}`),
			},
			wantErr: true,
		},
		{
			name: "simple header check with matching header",
			CEL: &triggersv1.CELInterceptor{
				Filter: "header['X-Test'][0] == 'test-value'",
			},
			args: args{
				payload: []byte(`{}`),
			},
			want:    []byte(`{}`),
			wantErr: false,
		},
		{
			name: "simple header check with non matching header",
			CEL: &triggersv1.CELInterceptor{
				Filter: "header['X-Test'][0] == 'unknown'",
			},
			args: args{
				payload: []byte(`{}`),
			},
			wantErr: true,
		},
		{
			name: "overloaded header check with case insensitive failed match",
			CEL: &triggersv1.CELInterceptor{
				Filter: "header.match('x-test', 'no-match')",
			},
			args: args{
				payload: []byte(`{}`),
			},
			wantErr: true,
		},
		{
			name: "overloaded header check with case insensitive matching",
			CEL: &triggersv1.CELInterceptor{
				Filter: "header.match('x-test', 'test-value')",
			},
			args: args{
				payload: []byte(`{}`),
			},
			want:    []byte(`{}`),
			wantErr: false,
		},
		{
			name: "body and header check",
			CEL: &triggersv1.CELInterceptor{
				Filter: "header.match('x-test', 'test-value') && body.value == 'test'",
			},
			args: args{
				payload: []byte(`{"value":"test"}`),
			},
			want:    []byte(`{"value":"test"}`),
			wantErr: false,
		},
		{
			name: "unable to parse the expression",
			CEL: &triggersv1.CELInterceptor{
				Filter: "header['X-Test",
			},
			args: args{
				payload: []byte(`{"value":"test"}`),
			},
			wantErr: true,
		},
		{
			name: "unable to parse the JSON body",
			CEL: &triggersv1.CELInterceptor{
				Filter: "body.value == 'test'",
			},
			args: args{
				payload: []byte(`{]`),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, _ := rtesting.SetupFakeContext(t)
			logger, _ := logging.NewLogger("", "")
			kubeClient := fakekubeclient.Get(ctx)
			w := &Interceptor{
				KubeClientSet: kubeClient,
				CEL:           tt.CEL,
				Logger:        logger,
			}
			request := &http.Request{
				Body: ioutil.NopCloser(bytes.NewReader(tt.args.payload)),
				Header: http.Header{
					"Content-Type": []string{"application/json"},
					"X-Test":       []string{"test-value"},
				},
			}
			got, err := w.ExecuteTrigger(tt.args.payload, request, nil, "")
			if (err != nil) != tt.wantErr {
				t.Errorf("Interceptor.ExecuteTrigger() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Interceptor.ExecuteTrigger() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestFilterEvaluation(t *testing.T) {
	jsonMap := map[string]interface{}{
		"value": "testing",
		"sha":   "ec26c3e57ca3a959ca5aad62de7213c562f8c821",
	}
	header := http.Header{}
	evalEnv := map[string]interface{}{"body": jsonMap, "header": header}
	env, err := makeCelEnv()
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name string
		expr string
		want ref.Val
	}{
		{
			name: "simple body value",
			expr: "body.value",
			want: types.String("testing"),
		},
		{
			name: "boolean body value",
			expr: "body.value == 'testing'",
			want: types.Bool(true),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := evaluate(tt.expr, env, evalEnv)
			if err != nil {
				t.Errorf("evaluate() got an error %s", err)
				return
			}
			_, ok := got.(*types.Err)
			if ok {
				t.Errorf("error evaluating expression: %s", got)
				return
			}

			if !got.Equal(tt.want).(types.Bool) {
				t.Errorf("evaluate() = %s, want %s", got, tt.want)
			}
		})
	}
}
