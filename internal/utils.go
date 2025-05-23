package internal

import (
	"fmt"
	mcpv1alpha1 "github.com/opendatahub-io/mcp-operator/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
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
	mergedContainer.Args = append(append([]string{}, templateContainer.Args...), mcpServerContainer.Args...)

	return &mergedContainer, nil
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
