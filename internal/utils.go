package internal

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
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

// MergeRuntimeContainers Merge the MCPServer Container struct with the Template Container struct, allowing users
// to override runtime template settings from the mcp server spec.
func MergeRuntimeContainers(templateContainer *v1.Container, mcpServerContainer *v1.Container) (*v1.Container, error) {
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
