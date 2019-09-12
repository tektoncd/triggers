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

package resources

import (
	"encoding/json"
	"flag"

	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/names"
	"golang.org/x/xerrors"
	corev1 "k8s.io/api/core/v1"
)

const TerminationMessagePath = "/builder/home/image-outputs/termination-log"

var (
	imageDigestExporterImage = flag.String("imagedigest-exporter-image", "override-with-imagedigest-exporter-image:latest", "The container image containing our image digest exporter binary.")
)

// AddOutputImageDigestExporter add a step to check the index.json for all output images
func AddOutputImageDigestExporter(
	tr *v1alpha1.TaskRun,
	taskSpec *v1alpha1.TaskSpec,
	gr GetResource,
) error {

	output := []*v1alpha1.ImageResource{}
	if len(tr.Spec.Outputs.Resources) > 0 {
		for _, trb := range tr.Spec.Outputs.Resources {
			boundResource, err := getBoundResource(trb.Name, tr.Spec.Outputs.Resources)
			if err != nil {
				return xerrors.Errorf("Failed to get bound resource: %w while adding output image digest exporter", err)
			}

			resource, err := getResource(boundResource, gr)
			if err != nil {
				return xerrors.Errorf("Failed to get output pipeline Resource for taskRun %q resource %v; error: %w while adding output image digest exporter", tr.Name, boundResource, err)
			}
			if resource.Spec.Type == v1alpha1.PipelineResourceTypeImage {
				imageResource, err := v1alpha1.NewImageResource(resource)
				if err != nil {
					return xerrors.Errorf("Invalid Image Resource for taskRun %q resource %v; error: %w", tr.Name, boundResource, err)
				}
				for _, o := range taskSpec.Outputs.Resources {
					if o.Name == boundResource.Name {
						if o.OutputImageDir != "" {
							imageResource.OutputImageDir = o.OutputImageDir
							break
						}
					}
				}
				output = append(output, imageResource)
			}
		}

		if len(output) > 0 {
			augmentedSteps := []v1alpha1.Step{}
			imagesJSON, err := json.Marshal(output)
			if err != nil {
				return xerrors.Errorf("Failed to format image resource data for output image exporter: %w", err)
			}

			augmentedSteps = append(augmentedSteps, taskSpec.Steps...)
			augmentedSteps = append(augmentedSteps, imageDigestExporterStep(imagesJSON))

			taskSpec.Steps = augmentedSteps
		}

	}

	return nil
}

// UpdateTaskRunStatusWithResourceResult if there is an update to the outout image resource, add to taskrun status result
func UpdateTaskRunStatusWithResourceResult(taskRun *v1alpha1.TaskRun, logContent []byte) error {
	err := json.Unmarshal(logContent, &taskRun.Status.ResourcesResult)
	if err != nil {
		return xerrors.Errorf("Failed to unmarshal output image exporter JSON output: %w", err)
	}
	return nil
}

func imageDigestExporterStep(imagesJSON []byte) v1alpha1.Step {
	return v1alpha1.Step{Container: corev1.Container{
		Name:    names.SimpleNameGenerator.RestrictLengthWithRandomSuffix("image-digest-exporter"),
		Image:   *imageDigestExporterImage,
		Command: []string{"/ko-app/imagedigestexporter"},
		Args: []string{
			"-images", string(imagesJSON),
			"-terminationMessagePath", TerminationMessagePath,
		},
		TerminationMessagePath:   TerminationMessagePath,
		TerminationMessagePolicy: corev1.TerminationMessageFallbackToLogsOnError,
	}}
}

// TaskRunHasOutputImageResource return true if the task has any output resources of type image
func TaskRunHasOutputImageResource(gr GetResource, taskRun *v1alpha1.TaskRun) bool {
	if len(taskRun.Spec.Outputs.Resources) > 0 {
		for _, r := range taskRun.Spec.Outputs.Resources {
			resource, err := gr(r.ResourceRef.Name)
			if err != nil {
				return false
			}
			if resource.Spec.Type == v1alpha1.PipelineResourceTypeImage {
				return true
			}
		}
	}
	return false
}
