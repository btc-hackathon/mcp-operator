package processor

import (
	"context"
	"github.com/go-logr/logr"
	"github.com/opendatahub-io/mcp-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type MCPServerTemplateProcessor interface {
	FetchMCPServerTemplate(ctx context.Context, logger logr.Logger, key types.NamespacedName) (*v1alpha1.MCPServerTemplate, error)
}

type mcpServerTemplateProcessor struct {
	client.Client
}

func NewMCPServerTemplateProcessor(client client.Client) MCPServerTemplateProcessor {
	return &mcpServerTemplateProcessor{
		client,
	}
}

func (m *mcpServerTemplateProcessor) FetchMCPServerTemplate(ctx context.Context, logger logr.Logger, key types.NamespacedName) (*v1alpha1.MCPServerTemplate, error) {
	mcpServerTemplate := &v1alpha1.MCPServerTemplate{}
	err := m.Client.Get(ctx, key, mcpServerTemplate)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("MCPServerTemplate not found")
		return nil, nil
	} else if err != nil {
		logger.Error(err, "Unable to fetch the MCPServerTemplate")
		return nil, err
	}
	return mcpServerTemplate, nil
}
