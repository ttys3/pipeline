// +build e2e

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

package test

import (
	"context"
	"fmt"
	"testing"

	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"

	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	knativetest "knative.dev/pkg/test"
)

const (
	embedTaskName    = "helloworld"
	embedTaskRunName = "helloworld-run"

	// TODO(#127) Currently not reliable to retrieve this output
	taskOutput = "do you want to build a snowman"
)

// TestTaskRun_EmbeddedResource is an integration test that will verify a very simple "hello world" TaskRun can be
// executed with an embedded resource spec.
func TestTaskRun_EmbeddedResource(t *testing.T) {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	c, namespace := setup(ctx, t)
	t.Parallel()

	knativetest.CleanupOnInterrupt(func() { tearDown(ctx, t, c, namespace) }, t.Logf)
	defer tearDown(ctx, t, c, namespace)

	t.Logf("Creating Task and TaskRun in namespace %s", namespace)
	if _, err := c.TaskClient.Create(ctx, getEmbeddedTask([]string{"/bin/sh", "-c", fmt.Sprintf("echo %s", taskOutput)}), metav1.CreateOptions{}); err != nil {
		t.Fatalf("Failed to create Task `%s`: %s", embedTaskName, err)
	}
	if _, err := c.TaskRunClient.Create(ctx, getEmbeddedTaskRun(namespace), metav1.CreateOptions{}); err != nil {
		t.Fatalf("Failed to create TaskRun `%s`: %s", embedTaskRunName, err)
	}

	t.Logf("Waiting for TaskRun %s in namespace %s to complete", embedTaskRunName, namespace)
	if err := WaitForTaskRunState(ctx, c, embedTaskRunName, TaskRunSucceed(embedTaskRunName), "TaskRunSuccess"); err != nil {
		t.Errorf("Error waiting for TaskRun %s to finish: %s", embedTaskRunName, err)
	}

	// TODO(#127) Currently we have no reliable access to logs from the TaskRun so we'll assume successful
	// completion of the TaskRun means the TaskRun did what it was intended.
}

func getEmbeddedTask(args []string) *v1alpha1.Task {
	return &v1alpha1.Task{
		ObjectMeta: metav1.ObjectMeta{
			Name: embedTaskName,
		},
		Spec: v1alpha1.TaskSpec{
			Inputs: &v1alpha1.Inputs{
				Resources: []v1alpha1.TaskResource{{
					ResourceDeclaration: v1alpha1.ResourceDeclaration{
						Name: "docs",
						Type: v1alpha1.PipelineResourceTypeGit,
					},
				}},
			},
			TaskSpec: v1beta1.TaskSpec{
				Steps: []v1alpha1.Step{
					{
						Container: corev1.Container{
							Image:   "ubuntu",
							Command: []string{"/bin/bash"},
							Args:    []string{"-c", "cat /workspace/docs/LICENSE"},
						},
					},
					{
						Container: corev1.Container{
							Image:   "busybox",
							Command: args,
						},
					},
				},
			},
		},
	}
}

func getEmbeddedTaskRun(namespace string) *v1alpha1.TaskRun {
	testSpec := &v1alpha1.PipelineResourceSpec{
		Type: v1alpha1.PipelineResourceTypeGit,
		Params: []v1alpha1.ResourceParam{{
			Name:  "URL",
			Value: "https://github.com/knative/docs",
		}},
	}
	return &v1alpha1.TaskRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      embedTaskRunName,
			Namespace: namespace,
		},
		Spec: v1alpha1.TaskRunSpec{
			TaskRef: &v1alpha1.TaskRef{
				Name: embedTaskName,
			},
			Inputs: &v1alpha1.TaskRunInputs{
				Resources: []v1alpha1.TaskResourceBinding{{
					PipelineResourceBinding: v1alpha1.PipelineResourceBinding{
						Name:         "docs",
						ResourceSpec: testSpec,
					},
				}},
			},
		},
	}
}
