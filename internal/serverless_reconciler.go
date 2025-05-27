package internal

import (
	"context"
	"github.com/go-logr/logr"
	mcpv1alpha1 "github.com/opendatahub-io/mcp-operator/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ServerlessReconciler struct {
	client                      client.Client
	ksvcReconciler              *KSVCReconciler
	serviceMeshMemberReconciler *ServiceMeshMemberReconciler
}

func NewServerlessReconciler(client client.Client) *ServerlessReconciler {
	return &ServerlessReconciler{
		client:                      client,
		ksvcReconciler:              NewKSVCReconciler(client),
		serviceMeshMemberReconciler: NewServiceMeshMemberReconciler(client),
	}
}

func (s *ServerlessReconciler) Reconcile(ctx context.Context, logger logr.Logger, mspServer *mcpv1alpha1.MCPServer, mcpServerTemplate *mcpv1alpha1.MCPServerTemplate, mcpServerConfig *MCPServerConfig) error {

	err := s.serviceMeshMemberReconciler.Reconcile(ctx, logger, mspServer, mcpServerConfig)
	if err != nil {
		return err
	}

	err = s.ksvcReconciler.Reconcile(ctx, logger, mspServer, mcpServerTemplate, mcpServerConfig)
	if err != nil {
		return err
	}
	return nil
}
