package internal

import "os"

var (
	MCPServerAPIGroupName = "mcp.opendatahub.io"
	OperatorNamespace     = getEnvOrDefault("POD_NAMESPACE", "redhat-ods-applications")
	MCPServerConfigMap    = "mcpserver-config"

	MCPServerContainerName = "mcpserver-container"
)

var (
	DeploymentModeAnnotation = MCPServerAPIGroupName + "/deployment-mode"

	MCPServerTemplateAnnotation = MCPServerAPIGroupName + "/mcpservertemplate"

	MCPServerPodLabelKey = MCPServerAPIGroupName + "/" + "mcpserver"
)

const (
	CommonDefaultHttpPort = 80

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
