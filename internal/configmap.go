package internal

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	DefaultConfigDataKey = "config.json"
)

type MCPServerConfigProcessor struct {
	client.Client
}

func NewMCPServerConfigProcessor(client client.Client) *MCPServerConfigProcessor {
	return &MCPServerConfigProcessor{
		client,
	}
}

// +kubebuilder:object:generate=false
type MCPServerConfig struct {
	DefaultDeploymentMode string `json:"deployment_mode"`
	Transport             string `json:"transport"`
}

func (p *MCPServerConfigProcessor) LoadMCPServerConfig(ctx context.Context, logger logr.Logger) (*MCPServerConfig, error) {
	configMap := &v1.ConfigMap{}
	err := p.Client.Get(ctx, types.NamespacedName{Name: MCPServerConfigMap, Namespace: OperatorNamespace}, configMap)
	if err != nil {
		logger.Info("Unable to load the MCPServer ConfigMap")
		return nil, err
	}

	jsonData, ok := configMap.Data[DefaultConfigDataKey]
	if !ok {
		err := fmt.Errorf("configmap %s does not contain key '%s'", MCPServerConfigMap, DefaultConfigDataKey)
		logger.Error(err, "ConfigMap data key not found")
		return nil, err
	}

	if jsonData == "" {
		err := fmt.Errorf("configmap %s key '%s' is empty", MCPServerConfigMap, DefaultConfigDataKey)
		logger.Error(err, "ConfigMap data key is empty")
		return nil, err
	}

	logger.Info("Successfully retrieved JSON data from ConfigMap key", "key", DefaultConfigDataKey)

	var config MCPServerConfig
	if err := json.Unmarshal([]byte(jsonData), &config); err != nil {
		logger.Error(err, "Failed to unmarshal JSON data from ConfigMap into MCPServerConfig struct")
		return nil, fmt.Errorf("failed to unmarshal json data from configmap %s key '%s': %w", MCPServerConfigMap, DefaultConfigDataKey, err)
	}

	logger.Info("Successfully loaded and unmarshalled MCPServerConfig")
	return &config, nil
}
