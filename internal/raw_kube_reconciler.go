package internal

import (
	"context"
	"github.com/go-logr/logr"
	mcpv1alpha1 "github.com/opendatahub-io/mcp-operator/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type RawKubeReconciler interface {
	Reconcile(ctx context.Context, logger logr.Logger, mspServer *mcpv1alpha1.MCPServer, podSpec mcpv1alpha1.PodSpec) error
}

type rawKubeReconciler struct {
	client               client.Client
	deploymentReconciler DeploymentReconciler
	serviceReconciler    ServiceReconciler
}

func NewRawKubeReconciler(client client.Client) RawKubeReconciler {
	return &rawKubeReconciler{
		client:               client,
		deploymentReconciler: NewDeploymentReconciler(client),
		serviceReconciler:    NewServiceReconciler(client),
	}
}

func (r *rawKubeReconciler) Reconcile(ctx context.Context, logger logr.Logger, mspServer *mcpv1alpha1.MCPServer, podSpec mcpv1alpha1.PodSpec) error {

	err := r.deploymentReconciler.Reconcile(ctx, logger, mspServer, podSpec)
	if err != nil {
		return err
	}

	err = r.serviceReconciler.Reconcile(ctx, logger, mspServer, podSpec)
	if err != nil {
		return err
	}
	return nil
}
