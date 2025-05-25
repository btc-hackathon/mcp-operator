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
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// MCPServerStatusHandler ...
type MCPServerStatusHandler interface {
	HandleStatusChange(ctx context.Context, logger logr.Logger, instance *v1alpha1.MCPServer, err error)
}

type mcpServerStatusHandler struct {
	client       client.Client
	errorHandler ReconciliationErrorHandler
}

func NewMCPServerStatusHandler(client client.Client) MCPServerStatusHandler {
	return &mcpServerStatusHandler{
		client:       client,
		errorHandler: NewReconciliationErrorHandler(),
	}
}

func (s *mcpServerStatusHandler) HandleStatusChange(ctx context.Context, logger logr.Logger, mcpServer *v1alpha1.MCPServer, err error) {
	logger.Info("Updating status for MCPServer", "err", err)

	if err != nil {
		s.setFailedConditions(&mcpServer.Status.Conditions, err)
	} else {
		s.setSuccessfulConditions(&mcpServer.Status.Conditions)
	}
	if err := s.client.Status().Update(ctx, mcpServer); err != nil {
		logger.Error(err, "Failed to update MCPServer status")
	}
}

func (s *mcpServerStatusHandler) setFailedConditions(conditions *[]metav1.Condition, err error) {
	reason := s.errorHandler.GetReasonForError(err)
	message := err.Error()
	meta.SetStatusCondition(conditions, metav1.Condition{
		Type:    string(Ready),
		Status:  metav1.ConditionFalse,
		Reason:  string(reason),
		Message: message,
	})
}

func (s *mcpServerStatusHandler) setSuccessfulConditions(conditions *[]metav1.Condition) {
	meta.SetStatusCondition(conditions, metav1.Condition{
		Type:    string(Ready),
		Status:  metav1.ConditionTrue,
		Reason:  string(MCPServerDeployed),
		Message: "",
	})
}
