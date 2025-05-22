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
	mcpv1alpha1 "github.com/opendatahub-io/mcp-operator/api/v1alpha1"
	"github.com/opendatahub-io/mcp-operator/internal"
	"github.com/opendatahub-io/mcp-operator/internal/processor"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// MCPServerReconciler reconciles a MCPServer object
type MCPServerReconciler struct {
	client.Client
	Scheme                     *runtime.Scheme
	mcpServerConfigProcessor   *internal.MCPServerConfigProcessor
	mcpServerTemplateProcessor processor.MCPServerTemplateProcessor
	rawKubeReconciler          internal.RawKubeReconciler
}

func NewMCPServerReconciler(client client.Client, scheme *runtime.Scheme) *MCPServerReconciler {
	return &MCPServerReconciler{
		Client:                   client,
		Scheme:                   scheme,
		mcpServerConfigProcessor: internal.NewMCPServerConfigProcessor(client),
		rawKubeReconciler:        internal.NewRawKubeReconciler(client),
	}
}

// +kubebuilder:rbac:groups=mcp.mcp.opendatahub.io,resources=mcpservers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=mcp.mcp.opendatahub.io,resources=mcpservers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=mcp.mcp.opendatahub.io,resources=mcpservers/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// the MCPServer object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.0/pkg/reconcile
func (r *MCPServerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// Initialize logger format
	logger := log.FromContext(ctx).WithValues("ModelSever", req.Name, "namespace", req.Namespace)
	logger.Info("Reconciling for ModelSever")

	// Get the ModelServer object when a reconciliation event is triggered (create, update, delete)
	mcpServer := &mcpv1alpha1.MCPServer{}
	err := r.Client.Get(ctx, req.NamespacedName, mcpServer)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Stop MCPServer reconciliation")
		return ctrl.Result{}, nil
	} else if err != nil {
		logger.Error(err, "Unable to fetch the MCPServer")
		return ctrl.Result{}, err
	}

	mcpServerConfig, err := r.mcpServerConfigProcessor.LoadMCPServerConfig(ctx, logger)
	if err != nil {
		return reconcile.Result{}, err
	}

	mcpServerTemplate, err := r.mcpServerTemplateProcessor.FetchMCPServerTemplate(ctx, logger,
		types.NamespacedName{Name: mcpServer.Spec.Template, Namespace: mcpServer.Namespace})

	mergedContainer, err := internal.MergeTemplateAndMCPServerSpecs(mcpServerTemplate, mcpServer)
	var newPodSpecContainers []corev1.Container
	for _, container := range mcpServerTemplate.Spec.Containers {
		if container.Name == internal.MCPServerContainerName {
			newPodSpecContainers = append(newPodSpecContainers, *mergedContainer)
		} else {
			newPodSpecContainers = append(newPodSpecContainers, container)
		}
	}

	deploymentMode := internal.GetDeploymentMode(mcpServer.Annotations, mcpServerConfig)
	logger.Info("MCPServer deployment mode ", "deployment mode ", deploymentMode)

	if deploymentMode == internal.RawDeployment {
		r.rawKubeReconciler.Reconcile(mcpServer)
	} else {

	}

	return ctrl.Result{}, nil
}

func (r *MCPServerReconciler) mapConfigMapToMCPServers(ctx context.Context, configMap client.Object) []reconcile.Request {
	logger := log.FromContext(ctx)
	requests := []reconcile.Request{}

	// List all MCPServer instances in all namespaces
	mcpServerList := &mcpv1alpha1.MCPServerList{} // <--- CHANGE THIS
	if err := r.Client.List(ctx, mcpServerList, &client.ListOptions{}); err != nil {
		logger.Error(err, "Failed to list MCPServer instances for ConfigMap change", "ConfigMap", configMap.GetName())
		return requests
	}

	logger.Info("ConfigMap changed, triggering reconciliation for all MCPServer instances", "ConfigMap", configMap.GetName(), "TotalMCPServers", len(mcpServerList.Items))
	for _, mcpServer := range mcpServerList.Items {
		requests = append(requests, reconcile.Request{
			NamespacedName: client.ObjectKey{
				Name:      mcpServer.Name,
				Namespace: mcpServer.Namespace,
			},
		})
	}
	return requests
}

func (r *MCPServerReconciler) createConfigMapPredicate() predicate.Predicate {
	operatorNamespace := internal.OperatorNamespace
	watchedConfigMapName := internal.MCPServerConfigMap

	// Predicate to filter for the specific ConfigMap in the operator's namespace
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return e.Object.GetName() == watchedConfigMapName && e.Object.GetNamespace() == operatorNamespace
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			return e.ObjectNew.GetName() == watchedConfigMapName && e.ObjectNew.GetNamespace() == operatorNamespace
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			// Decide if you want to reconcile on deletion.
			// If the config is gone, your MCPServer instances might need to react.
			return e.Object.GetName() == watchedConfigMapName && e.Object.GetNamespace() == operatorNamespace
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return e.Object.GetName() == watchedConfigMapName && e.Object.GetNamespace() == operatorNamespace
		},
	}
}

func (r *MCPServerReconciler) SetupWithManager(mgr ctrl.Manager) error {

	return ctrl.NewControllerManagedBy(mgr).
		For(&mcpv1alpha1.MCPServer{}). // Primary resource
		Watches(                       // Watch for changes to the ConfigMap
			&corev1.ConfigMap{},
			handler.EnqueueRequestsFromMapFunc(r.mapConfigMapToMCPServers),
			builder.WithPredicates(r.createConfigMapPredicate()),
		).
		Complete(r)
}
