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
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/spf13/cobra"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime/serializer/streaming"
	"k8s.io/client-go/kubernetes/scheme"
)

var (
	rootCmd = &cobra.Command{
		Use:   "triggers-run",
		Short: "Tekton Trigger test",
		Run:   rootRun,
	}

	triggerFile string
	httpPath    string
)

func init() {
	rootCmd.Flags().StringVarP(&triggerFile, "triggerFile", "t", "", "Path to trigger yaml file")
	rootCmd.Flags().StringVarP(&httpPath, "httpPath", "r", "", "Path to body event")
}

func rootRun(cmd *cobra.Command, args []string) {
	// TODO: Not implemented
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

func readHTTP(path string) (*http.Request, error) {
	f, err := os.Open(path)

	if err != nil {
		return nil, fmt.Errorf("error opening file: %w", err)
	}
	defer f.Close()

	return http.ReadRequest(bufio.NewReader(f))
}

// Execute runs the command.
func Execute() error {
	return rootCmd.Execute()
}
