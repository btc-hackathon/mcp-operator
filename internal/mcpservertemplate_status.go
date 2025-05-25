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
	"fmt"
	"github.com/go-logr/logr"
	"github.com/opendatahub-io/mcp-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	templateConditionReady = "Ready"
)

// MCPServerTemplateStatusHandler ...
type MCPServerTemplateStatusHandler interface {
	HandleStatusChange(ctx context.Context, logger logr.Logger, instance *v1alpha1.MCPServerTemplate, err error)
}

type mcpServerTemplateStatusHandler struct {
	client.Client
}

func NewMCPServerTemplateStatusHandler(client client.Client) MCPServerTemplateStatusHandler {
	return &mcpServerTemplateStatusHandler{
		client,
	}
}

func (s *mcpServerTemplateStatusHandler) HandleStatusChange(ctx context.Context, logger logr.Logger, mcpServerTemplate *v1alpha1.MCPServerTemplate, err error) {

	if err != nil {
		meta.SetStatusCondition(&mcpServerTemplate.Status.Conditions, metav1.Condition{
			Type:    templateConditionReady,
			Status:  metav1.ConditionFalse,
			Reason:  "InvalidTemplateSpec",
			Message: err.Error(),
		})
	} else {
		meta.SetStatusCondition(&mcpServerTemplate.Status.Conditions, metav1.Condition{
			Type:    templateConditionReady,
			Status:  metav1.ConditionTrue,
			Reason:  "ValidTemplate",
			Message: fmt.Sprintf("MCPServerTemplate is valid"),
		})
	}
	if err := s.Status().Update(ctx, mcpServerTemplate); err != nil {
		logger.Error(err, "Failed to update MCPServerTemplate status")
	}
}
