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

package controller

import (
	"context"
	"fmt"
	"github.com/opendatahub-io/mcp-operator/internal"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	mcpv1alpha1 "github.com/opendatahub-io/mcp-operator/api/v1alpha1"
)

// MCPServerTemplateReconciler reconciles a MCPServerTemplate object
type MCPServerTemplateReconciler struct {
	client.Client
	Scheme                     *runtime.Scheme
	mcpServerTemplateProcessor *internal.MCPServerTemplateProcessor
	statusHandler              *internal.MCPServerTemplateStatusHandler
}

func NewMCPServerTemplateReconciler(client client.Client, scheme *runtime.Scheme) *MCPServerTemplateReconciler {
	return &MCPServerTemplateReconciler{
		Client:                     client,
		Scheme:                     scheme,
		mcpServerTemplateProcessor: internal.NewMCPServerTemplateProcessor(client),
		statusHandler:              internal.NewMCPServerTemplateStatusHandler(client),
	}
}

// +kubebuilder:rbac:groups=mcp.opendatahub.io,resources=mcpservertemplates,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=mcp.opendatahub.io,resources=mcpservertemplates/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=mcp.opendatahub.io,resources=mcpservertemplates/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// the MCPServerTemplate object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.0/pkg/reconcile
func (r *MCPServerTemplateReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithValues("MCPServerTemplate", req.Name, "namespace", req.Namespace)
	logger.Info("Reconciling for MCPServerTemplate")

	// Get the ModelServerTemplate object when a reconciliation event is triggered (create, update, delete)
	mcpServerTemplate, err := r.mcpServerTemplateProcessor.FetchMCPServerTemplate(ctx, logger, req.NamespacedName)
	if err != nil || mcpServerTemplate == nil {
		return ctrl.Result{}, err
	}

	defer r.statusHandler.HandleStatusChange(ctx, logger, mcpServerTemplate, err)

	if len(mcpServerTemplate.Spec.Containers) == 0 {
		return ctrl.Result{}, fmt.Errorf("no container configuration found in MCPServerTemplate")
	}

	containerFound := false
	for _, container := range mcpServerTemplate.Spec.Containers {
		if container.Name == internal.MCPServerContainerName {
			containerFound = true
			break
		}
	}
	if !containerFound {
		return ctrl.Result{}, fmt.Errorf("no container with name %s found in MCPServerTemplate", internal.MCPServerContainerName)
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *MCPServerTemplateReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&mcpv1alpha1.MCPServerTemplate{}).
		Complete(r)
}
