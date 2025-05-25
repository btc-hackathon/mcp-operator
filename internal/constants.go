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

import "os"

var (
	MCPServerAPIGroupName  = "mcp.opendatahub.io"
	OperatorNamespace      = getEnvOrDefault("POD_NAMESPACE", "redhat-ods-applications")
	MCPServerConfigMap     = "mcp-operator-mcpserver-config"
	MCPServerContainerName = "mcpserver-container"
)

var (
	DeploymentModeAnnotation    = MCPServerAPIGroupName + "/deploymentmode"
	MCPServerTemplateAnnotation = MCPServerAPIGroupName + "/mcpservertemplate"
	MCPServerPodLabelKey        = MCPServerAPIGroupName + "/" + "mcpserver"
)

const (
	CommonDefaultHttpPort    = 80
	MCPServerDefaultHttpPort = "8080"
)

type DeploymentModeType string

const (
	Serverless    DeploymentModeType = "Serverless"
	RawDeployment DeploymentModeType = "RawDeployment"
)

func getEnvOrDefault(key string, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
