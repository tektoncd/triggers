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

package cmd

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/spf13/cobra"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	triggersclientset "github.com/tektoncd/triggers/pkg/client/clientset/versioned"
	dynamicClientset "github.com/tektoncd/triggers/pkg/client/dynamic/clientset"
	"github.com/tektoncd/triggers/pkg/client/dynamic/clientset/tekton"
	"github.com/tektoncd/triggers/pkg/sink"
	"github.com/tektoncd/triggers/pkg/template"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/runtime/serializer/streaming"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/yaml"
)

var (
	kubeconfig string
	rootCmd    = &cobra.Command{
		Use:   "triggers-run",
		Short: "This is the CLI for tekton trigger.",
		Long:  "tkn-trigger will allow users easily test out the Trigger config.",
		RunE:  rootRun,
	}
	action      string
	triggerFile string
	httpPath    string
)

func init() {
	rootCmd.Flags().StringVarP(&action, "show or create", "a", "", "it's to show or create resources")
	rootCmd.Flags().StringVarP(&triggerFile, "triggerFile", "t", "", "Path to trigger yaml file")
	rootCmd.Flags().StringVarP(&httpPath, "httpPath", "r", "", "Path to body event")
	rootCmd.PersistentFlags().StringVarP(&kubeconfig, "kubeconfig", "k", "", "absolute path to the kubeconfig file")
}

func rootRun(cmd *cobra.Command, args []string) error {
	err := trigger(triggerFile, httpPath, action, kubeconfig, os.Stdout)
	if err != nil {
		return fmt.Errorf("fail to call trigger: %v", err)
	}

	return nil
}

func trigger(triggerFile, httpPath, action, kubeconfig string, writer io.Writer) error {
	// Read HTTP request.
	request, body, err := readHTTP(httpPath)
	if err != nil {
		return fmt.Errorf("error reading HTTP txt file: %w", err)
	}

	// Read triggers.
	triggers, err := readTrigger(triggerFile)
	if err != nil {
		return fmt.Errorf("error reading triggers: %w", err)
	}

	kubeClient, triggerClient, err := getKubeClient(kubeconfig)
	if err != nil {
		return fmt.Errorf("fail to get clients: %w", err)
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return fmt.Errorf("Failed to build config from the flags: %w", err)
	}

	logger, _ := zap.NewProduction()
	sugerLogger := logger.Sugar()
	eventID := template.UID()
	r := newSink(config, sugerLogger)
	eventLog := sugerLogger.With(zap.String(triggersv1.EventIDLabelKey, eventID))
	for _, tri := range triggers {
		resources, err := processTriggerSpec(kubeClient, triggerClient, tri,
			request, body, eventID, eventLog, r)
		if err != nil {
			return fmt.Errorf("fail to create resources: %w", err)
		}

		switch action {
		case "show":
			{
				for _, resource := range resources {
					s, err := yaml.Marshal(resource)
					if err != nil {
						return fmt.Errorf("fail to print out the resource: %w", err)
					}
					fmt.Fprintln(writer, "-----------------------------------")
					fmt.Fprintf(writer, "%s", s)
				}
			}
		case "create":
			{
				err := r.CreateResources("", resources, tri.Name, eventID, eventLog)
				if err != nil {
					return fmt.Errorf("fail to create resources: %w", err)
				}
			}
		}
	}

	return nil
}

func readTrigger(path string) ([]*v1alpha1.Trigger, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("error reading trigger file: %w", err)
	}
	defer f.Close()

	var list []*v1alpha1.Trigger
	decoder := streaming.NewDecoder(f, scheme.Codecs.UniversalDecoder())
	b := new(v1alpha1.Trigger)
	for err == nil {
		_, _, err = decoder.Decode(nil, b)
		if err != nil {
			if err != io.EOF {
				return nil, fmt.Errorf("error decoding triggers: %w", err)
			}
			break
		}
		list = append(list, b)
	}
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("error decoding triggers: %w", err)
	}

	return list, nil
}

func readHTTP(path string) (*http.Request, []byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("error opening file: %w", err)
	}
	defer f.Close()

	request, err := http.ReadRequest(bufio.NewReader(f))
	if err != nil {
		return nil, nil, fmt.Errorf("error reading HTTP file: %w", err)
	}

	body, err := ioutil.ReadAll(request.Body)
	defer request.Body.Close()
	if err != nil {
		return nil, nil, fmt.Errorf("error reading HTTP body: %w", err)
	}

	request.Body = ioutil.NopCloser(bytes.NewReader(body))

	return request, body, err
}

func processTriggerSpec(kubeClient kubernetes.Interface, client triggersclientset.Interface, tri *triggersv1.Trigger, request *http.Request, body []byte, eventID string, eventLog *zap.SugaredLogger, r sink.Sink) ([]json.RawMessage, error) {
	if tri == nil {
		return nil, errors.New("trigger is not defined")
	}

	//convert trigger to eventListener
	el, err := triggersv1.ToEventListenerTrigger(tri.Spec)
	if err != nil {
		return nil, fmt.Errorf("fail to convert Trigger to EvenetListener: %w", err)
	}

	log := eventLog.With(zap.String(triggersv1.TriggerLabelKey, el.Name))

	finalPayload, header, err := r.ExecuteInterceptors(&el, request, body, log)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	if tri.Namespace == "" {
		tri.Namespace = "default"
	}

	rt, err := template.ResolveTrigger(el,
		client.TriggersV1alpha1().TriggerBindings(tri.Namespace).Get,
		client.TriggersV1alpha1().ClusterTriggerBindings().Get,
		client.TriggersV1alpha1().TriggerTemplates(tri.Namespace).Get)
	if err != nil {
		log.Error("Failed to resolve Trigger: ", err)
		return nil, err
	}

	params, err := template.ResolveParams(rt, finalPayload, header)
	if err != nil {
		log.Error("Failed to resolve parameters", err)
		return nil, err
	}
	log.Infof("ResolvedParams : %+v", params)

	resources := template.ResolveResources(rt.TriggerTemplate, params)

	return resources, nil
}

func newSink(config *rest.Config, sugerLogger *zap.SugaredLogger) sink.Sink {
	sinkClients, err := sink.ConfigureClients(config)
	if err != nil {
		log.Fatalf("Failed to get the sink client: %v", err)
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		log.Fatalf("Failed to get the dynamic client: %v", err)
	}

	dynamicCS := dynamicClientset.New(tekton.WithClient(dynamicClient))
	kubeClient, _, err := getKubeClient(kubeconfig)
	if err != nil {
		log.Fatal("fail to get clients: %w", err)
	}

	s := sink.Sink{
		KubeClientSet:          kubeClient,
		HTTPClient:             http.DefaultClient,
		Auth:                   sink.DefaultAuthOverride{},
		DiscoveryClient:        sinkClients.DiscoveryClient,
		DynamicClient:          dynamicCS,
		Logger:                 sugerLogger,
		EventListenerNamespace: "default",
	}

	return s
}

func getKubeClient(kubeconfig string) (kubernetes.Interface, triggersclientset.Interface, error) {
	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to build config from the flags: %w", err)
	}

	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to get the Kubernetes client set: %v", err)
	}

	client, err := triggersclientset.NewForConfig(config)
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to get the trigger client set: %v", err)
	}

	return kubeClient, client, nil
}

// Execute runs the command.
func Execute() error {
	return rootCmd.Execute()
}
