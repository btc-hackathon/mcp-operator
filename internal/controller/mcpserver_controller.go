package controller

import (
	"context"
	"fmt"
	mcpv1alpha1 "github.com/opendatahub-io/mcp-operator/api/v1alpha1"
	"github.com/opendatahub-io/mcp-operator/internal"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	knservingv1 "knative.dev/serving/pkg/apis/serving/v1"
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
	mcpServerProcessor         *internal.MCPServerProcessor
	mcpServerConfigProcessor   *internal.MCPServerConfigProcessor
	mcpServerTemplateProcessor *internal.MCPServerTemplateProcessor
	rawKubeReconciler          *internal.RawKubeReconciler
	serverlessReconciler       *internal.ServerlessReconciler
	statusHandler              *internal.MCPServerStatusHandler
}

func NewMCPServerReconciler(client client.Client, scheme *runtime.Scheme) *MCPServerReconciler {
	return &MCPServerReconciler{
		Client:                     client,
		Scheme:                     scheme,
		mcpServerProcessor:         internal.NewMCPServerProcessor(client),
		mcpServerTemplateProcessor: internal.NewMCPServerTemplateProcessor(client),
		mcpServerConfigProcessor:   internal.NewMCPServerConfigProcessor(client),
		rawKubeReconciler:          internal.NewRawKubeReconciler(client),
		serverlessReconciler:       internal.NewServerlessReconciler(client),
		statusHandler:              internal.NewMCPServerStatusHandler(client),
	}
}

// +kubebuilder:rbac:groups=mcp.opendatahub.io,resources=mcpservers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=mcp.opendatahub.io,resources=mcpservers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=mcp.opendatahub.io,resources=mcpservers/finalizers,verbs=update
// +kubebuilder:rbac:groups=serving.knative.dev,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=serving.knative.dev,resources=services/status,verbs=get
// +kubebuilder:rbac:groups=apps,resources=deployments;,verbs=create;delete;get;list;patch;update;watch
// +kubebuilder:rbac:groups=core,resources=services,verbs=create;delete;get;list;patch;update;watch
// +kubebuilder:rbac:groups=core,resources=services,verbs=create;delete;get;list;patch;update;watch
// +kubebuilder:rbac:groups=route.openshift.io,resources=routes,verbs=create;delete;get;list;patch;update;watch
// +kubebuilder:rbac:groups=maistra.io,resources=servicemeshmembers,verbs=create;delete;get;list;patch;update;watch
// +kubebuilder:rbac:groups=maistra.io,resources=servicemeshcontrolplanes,verbs=get;list;watch;use
// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch

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
	mcpServer, err := r.mcpServerProcessor.FetchMCPServer(ctx, logger, req.NamespacedName)
	if err != nil || mcpServer == nil {
		return ctrl.Result{}, err
	}

	defer r.statusHandler.HandleStatusChange(ctx, logger, mcpServer, err)

	templateName, ok := mcpServer.Annotations[internal.MCPServerTemplateAnnotation]
	if !ok || templateName == "" {
		return ctrl.Result{}, fmt.Errorf("no template name found in MCPServer annotations")
	}

	mcpServerTemplate, err := r.mcpServerTemplateProcessor.FetchMCPServerTemplate(ctx, logger, types.NamespacedName{Name: templateName, Namespace: internal.OperatorNamespace})
	if err != nil {
		return ctrl.Result{}, err
	}

	mcpServerConfig, err := r.mcpServerConfigProcessor.LoadMCPServerConfig(ctx, logger)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = r.rawKubeReconciler.Reconcile(ctx, logger, mcpServer, mcpServerTemplate, mcpServerConfig)
	if err != nil {
		return internal.NewReconciliationErrorHandler().GetReconcileResultFor(err)
	}

	err = r.serverlessReconciler.Reconcile(ctx, logger, mcpServer, mcpServerTemplate, mcpServerConfig)
	if err != nil {
		return internal.NewReconciliationErrorHandler().GetReconcileResultFor(err)
	}

	return ctrl.Result{}, nil
}

func (r *MCPServerReconciler) SetupWithManager(mgr ctrl.Manager) error {

	return ctrl.NewControllerManagedBy(mgr).
		For(&mcpv1alpha1.MCPServer{}).
		Owns(&v1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&knservingv1.Service{}).
		Watches( // Watch for changes to the ConfigMap
			&corev1.ConfigMap{},
			handler.EnqueueRequestsFromMapFunc(r.mapConfigMapToMCPServers),
			builder.WithPredicates(r.createConfigMapPredicate(), predicate.GenerationChangedPredicate{}),
		).
		Complete(r)
}

func (r *MCPServerReconciler) mapConfigMapToMCPServers(ctx context.Context, configMap client.Object) []reconcile.Request {
	logger := log.FromContext(ctx)
	var requests []reconcile.Request

	// List all MCPServer instances in all namespaces
	mcpServerList := &mcpv1alpha1.MCPServerList{}
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
