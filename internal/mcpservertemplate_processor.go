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
	"context"
	"github.com/go-logr/logr"
	"github.com/opendatahub-io/mcp-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type MCPServerTemplateProcessor struct {
	client.Client
}

func NewMCPServerTemplateProcessor(client client.Client) *MCPServerTemplateProcessor {
	return &MCPServerTemplateProcessor{
		client,
	}
}

func (m *MCPServerTemplateProcessor) FetchMCPServerTemplate(ctx context.Context, logger logr.Logger, key types.NamespacedName) (*v1alpha1.MCPServerTemplate, error) {

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
