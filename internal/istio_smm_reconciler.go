package internal

import (
	"context"
	"errors"
	mcpv1alpha1 "github.com/opendatahub-io/mcp-operator/api/v1alpha1"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "maistra.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ServiceMeshMemberReconciler struct {
	client         client.Client
	smmHandler     *ServiceMeshMemberHandler
	deltaProcessor *DeltaProcessor
}

func NewServiceMeshMemberReconciler(client client.Client) *ServiceMeshMemberReconciler {
	return &ServiceMeshMemberReconciler{
		client:         client,
		smmHandler:     NewServiceMeshMember(client),
		deltaProcessor: NewDeltaProcessor(),
	}
}

func (r *ServiceMeshMemberReconciler) Reconcile(ctx context.Context, logger logr.Logger, mcpServer *mcpv1alpha1.MCPServer, mcpServerConfig *MCPServerConfig) error {
	logger.Info("Verifying that the namespace is enrolled to the mesh")

	deploymentMode := GetDeploymentMode(logger, mcpServer.Annotations, mcpServerConfig)
	if deploymentMode != Serverless {
		logger.Info("Deployment mode is not Serverless. Skipping to create ServiceMeshMember resource")
		return nil
	}

	// Create Desired resource
	desiredResource := r.createDesiredResource(mcpServer)

	// Get Existing resource
	existingResource, err := r.getExistingResource(ctx, logger, mcpServer.Namespace)
	if err != nil {
		return err
	}

	// Process Delta
	if err = r.processDelta(ctx, logger, desiredResource, existingResource); err != nil {
		return err
	}
	return nil
}

func (r *ServiceMeshMemberReconciler) createDesiredResource(mcpServer *mcpv1alpha1.MCPServer) *v1.ServiceMeshMember {

	return &v1.ServiceMeshMember{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ServiceMeshMemberName,
			Namespace: mcpServer.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "mcp-operator",
			},
			Annotations: nil,
		},
		Spec: v1.ServiceMeshMemberSpec{
			ControlPlaneRef: v1.ServiceMeshControlPlaneRef{
				Name:      ServiceMeshControlPlanName,
				Namespace: ServiceMeshControlPlanNamespace,
			},
		},
	}
}

func (r *ServiceMeshMemberReconciler) getExistingResource(ctx context.Context, logger logr.Logger, namespace string) (*v1.ServiceMeshMember, error) {
	return r.smmHandler.Fetch(ctx, logger, namespace, ServiceMeshMemberName)
}

func (r *ServiceMeshMemberReconciler) processDelta(ctx context.Context, logger logr.Logger, desiredSMM *v1.ServiceMeshMember, existingSMM *v1.ServiceMeshMember) error {
	comparator := GetServiceMeshMemberComparator()
	delta := r.deltaProcessor.ComputeDelta(comparator, desiredSMM, existingSMM)

	if !delta.HasChanges() {
		logger.Info("No delta found")
		return nil
	}

	if delta.IsAdded() {
		logger.Info("Delta found", "create", desiredSMM.GetName())
		if err := r.client.Create(ctx, desiredSMM); err != nil {
			return err
		}
	}

	if delta.IsUpdated() {
		// Don't update a resource that we don't own.
		if existingSMM.Labels["app.kubernetes.io/managed-by"] != "mcp-operator" {
			logger.Error(errors.New("non suitable ServiceMeshMember resource"),
				"There is a user-owned ServiceMeshMember resource that needs to be updated. MCP Server may not work properly.",
				"smm.name", existingSMM.GetName(),
				"smm.desired.smcp_namespace", desiredSMM.Spec.ControlPlaneRef.Namespace,
				"smm.desired.smcp_name", desiredSMM.Spec.ControlPlaneRef.Name,
				"smm.current.smcp_namespace", existingSMM.Spec.ControlPlaneRef.Namespace,
				"smm.current.smcp_name", existingSMM.Spec.ControlPlaneRef.Name)
			// Don't return error because it is not recoverable. It does not make sense
			// to keep trying. It needs user intervention.
			return nil
		}

		logger.Info("Delta found", "update", existingSMM.GetName())
		rp := existingSMM.DeepCopy()
		rp.Spec = desiredSMM.Spec

		if err := r.client.Update(ctx, rp); err != nil {
			return err
		}
	}

	if delta.IsRemoved() {
		// Don't delete a resource that we don't own.
		if existingSMM.Labels["app.kubernetes.io/managed-by"] != "mcp-operator" {
			logger.Info("Model Serving no longer needs the namespace enrolled to the mesh. The ServiceMeshMember resource is not removed, because it is user-owned.")
			return nil
		}

		logger.Info("Delta found", "delete", existingSMM.GetName())
		if err := r.client.Delete(ctx, existingSMM); err != nil {
			return err
		}
	}
	return nil
}
