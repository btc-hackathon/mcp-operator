/*
Copyright 2025.

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

package internal

import (
	"fmt"
	mcpv1alpha1 "github.com/opendatahub-io/mcp-operator/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"reflect"
)

func GetDeploymentMode(annotations map[string]string, mcpServerConfig *MCPServerConfig) DeploymentModeType {

	// Second priority, if the status doesn't have the deploymentMode recorded, is explicit annotations
	deploymentMode, ok := annotations[DeploymentModeAnnotation]
	if ok && (deploymentMode == string(RawDeployment) || deploymentMode == string(Serverless)) {
		return DeploymentModeType(deploymentMode)
	}

	// Finally, if an InferenceService is being created and does not explicitly specify a DeploymentMode
	return DeploymentModeType(mcpServerConfig.DefaultDeploymentMode)
}

func GetNetworkVisibility(annotations map[string]string, mcpServerConfig *MCPServerConfig) NetworkVisibility {

	// Second priority, if the status doesn't have the networkVisibility recorded, is explicit annotations
	networkVisibility, ok := annotations[NetworkVisibilityAnnotation]
	if ok && (networkVisibility == string(Exposed) || networkVisibility == string(Hidden)) {
		return NetworkVisibility(networkVisibility)
	}

	// Finally, if an MCPServer is being created and does not explicitly specify a NetworkVisibility
	return NetworkVisibility(mcpServerConfig.DefaultVisibility)
}

func GetUnifiedMCPServerContainer(mcpServerTemplate *mcpv1alpha1.MCPServerTemplate, mcpServer *mcpv1alpha1.MCPServer) (*v1.Container, error) {

	mcpServerContainer, err := FetchMSPServerContainer(mcpServerTemplate.Spec.Containers)
	if err != nil {
		return nil, err
	}

	return MergeContainers(mcpServerContainer, &mcpServer.Spec.Container)
}

func FetchMSPServerContainer(containers []v1.Container) (*v1.Container, error) {
	var mcpServerContainer v1.Container
	containerFound := false

	for _, container := range containers {
		if container.Name == MCPServerContainerName {
			mcpServerContainer = container
			containerFound = true
			break
		}
	}
	if !containerFound {
		return nil, fmt.Errorf("no container found with name %s ", MCPServerContainerName)
	}
	return &mcpServerContainer, nil
}

// MergeContainers Merge the MCPServer Container struct with the Template Container struct, allowing users
// to override runtime template settings from the mcp server spec.
func MergeContainers(templateContainer *v1.Container, mcpServerContainer *v1.Container) (*v1.Container, error) {
	// Save template container name, as the name can be overridden as empty string during the Unmarshal below
	// since the Name field does not have the 'omitempty' struct tag.
	templateContainerName := templateContainer.Name

	// Use JSON Marshal/Unmarshal to merge Container structs using strategic merge patch
	templateContainerJson, err := json.Marshal(templateContainer)
	if err != nil {
		return nil, err
	}

	overrides, err := json.Marshal(mcpServerContainer)
	if err != nil {
		return nil, err
	}

	mergedContainer := v1.Container{}
	jsonResult, err := strategicpatch.StrategicMergePatch(templateContainerJson, overrides, mergedContainer)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(jsonResult, &mergedContainer); err != nil {
		return nil, err
	}

	if mergedContainer.Name == "" {
		mergedContainer.Name = templateContainerName
	}

	// Strategic merge patch will replace args but more useful behaviour here is to concatenate
	//mergedContainer.Args = append(append([]string{}, templateContainer.Args...), mcpServerContainer.Args...)

	return &mergedContainer, nil
}

func GetCommonMeta(mcpServer *mcpv1alpha1.MCPServer, mcpServerTemplate *mcpv1alpha1.MCPServerTemplate) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:      mcpServer.Name,
		Namespace: mcpServer.Namespace,
		Labels: Union(
			mcpServerTemplate.Labels,
			mcpServer.Labels,
			map[string]string{
				MCPServerLabelKey: mcpServer.Name,
			},
		),
		Annotations: Union(
			mcpServerTemplate.Annotations,
			mcpServer.Annotations,
		),
	}
}

func SetDefaultPodSpec(podSpec *v1.PodSpec) {
	if podSpec.DNSPolicy == "" {
		podSpec.DNSPolicy = v1.DNSClusterFirst
	}
	if podSpec.RestartPolicy == "" {
		podSpec.RestartPolicy = v1.RestartPolicyAlways
	}
	if podSpec.TerminationGracePeriodSeconds == nil {
		TerminationGracePeriodSeconds := int64(v1.DefaultTerminationGracePeriodSeconds)
		podSpec.TerminationGracePeriodSeconds = &TerminationGracePeriodSeconds
	}
	if podSpec.SecurityContext == nil {
		podSpec.SecurityContext = &v1.PodSecurityContext{}
	}
	if podSpec.SchedulerName == "" {
		podSpec.SchedulerName = v1.DefaultSchedulerName
	}
	for i := range podSpec.Containers {
		container := &podSpec.Containers[i]
		if container.TerminationMessagePath == "" {
			container.TerminationMessagePath = "/dev/termination-log"
		}
		if container.TerminationMessagePolicy == "" {
			container.TerminationMessagePolicy = v1.TerminationMessageReadFile
		}
		if container.ImagePullPolicy == "" {
			container.ImagePullPolicy = v1.PullIfNotPresent
		}
		// generate default readiness probe for mcp server container
		//if container.Name == MCPServerContainerName {
		//	if container.ReadinessProbe == nil {
		//		if len(container.Ports) == 0 {
		//			container.ReadinessProbe = &corev1.Probe{
		//				ProbeHandler: corev1.ProbeHandler{
		//					TCPSocket: &corev1.TCPSocketAction{
		//						Port: intstr.IntOrString{
		//							IntVal: 8080,
		//						},
		//					},
		//				},
		//				TimeoutSeconds:   1,
		//				PeriodSeconds:    10,
		//				SuccessThreshold: 1,
		//				FailureThreshold: 3,
		//			}
		//		} else {
		//			container.ReadinessProbe = &corev1.Probe{
		//				ProbeHandler: corev1.ProbeHandler{
		//					TCPSocket: &corev1.TCPSocketAction{
		//						Port: intstr.IntOrString{
		//							IntVal: container.Ports[0].ContainerPort,
		//						},
		//					},
		//				},
		//				TimeoutSeconds:   1,
		//				PeriodSeconds:    10,
		//				SuccessThreshold: 1,
		//				FailureThreshold: 3,
		//			}
		//		}
		//	}
		//}
	}
}

func GetCommonPodSpec(mcpServer *mcpv1alpha1.MCPServer, mcpServerTemplate *mcpv1alpha1.MCPServerTemplate) (*v1.PodSpec, error) {
	mergedContainer, err := GetUnifiedMCPServerContainer(mcpServerTemplate, mcpServer)
	if err != nil {
		return nil, err
	}
	var newPodSpecContainers []v1.Container
	for _, container := range mcpServerTemplate.Spec.Containers {
		if container.Name == MCPServerContainerName {
			newPodSpecContainers = append(newPodSpecContainers, *mergedContainer)
		} else {
			newPodSpecContainers = append(newPodSpecContainers, container)
		}
	}

	podSpec := &v1.PodSpec{
		Containers:         newPodSpecContainers,
		ImagePullSecrets:   append(mcpServerTemplate.Spec.ImagePullSecrets, mcpServer.Spec.ImagePullSecrets...),
		ServiceAccountName: mcpServer.Spec.ServiceAccountName,
	}
	return podSpec, nil
}

func IsNil(i any) bool {
	return reflect.ValueOf(i).IsNil()
}

func IsNotNil(i any) bool {
	return !IsNil(i)
}

func Union(maps ...map[string]string) map[string]string {
	result := make(map[string]string)
	for _, m := range maps {
		for k, v := range m {
			result[k] = v
		}
	}
	return result
}
