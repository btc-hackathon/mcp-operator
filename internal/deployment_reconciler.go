package internal

import (
	"context"
	"github.com/go-logr/logr"
	mcpv1alpha1 "github.com/opendatahub-io/mcp-operator/api/v1alpha1"
	"github.com/opendatahub-io/mcp-operator/internal/comparators"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type DeploymentReconciler interface {
	Reconcile(ctx context.Context, logger logr.Logger, mspServer *mcpv1alpha1.MCPServer, podSpec mcpv1alpha1.PodSpec) error
}

type deploymentReconciler struct {
	client         client.Client
	deltaProcessor DeltaProcessor
}

func NewDeploymentReconciler(client client.Client) DeploymentReconciler {
	return &deploymentReconciler{
		client:         client,
		deltaProcessor: NewDeltaProcessor(),
	}
}

func (d *deploymentReconciler) Reconcile(ctx context.Context, logger logr.Logger, mspServer *mcpv1alpha1.MCPServer, podSpec mcpv1alpha1.PodSpec) error {

	logger.Info("Reconciling Deployment for RAW Deployment")
	// Create Desired resource
	desiredResource, err := d.createDesiredResource(logger, mspServer, podSpec)
	if err != nil {
		return err
	}

	// Get Existing resource
	existingResource, err := d.getExistingResource(ctx, logger, mspServer)
	if err != nil {
		return err
	}

	// Process Delta
	if err = d.processDelta(ctx, logger, desiredResource, existingResource); err != nil {
		return err
	}
	return nil
}

func (d *deploymentReconciler) createDesiredResource(logger logr.Logger, mspServer *mcpv1alpha1.MCPServer, podSpec mcpv1alpha1.PodSpec) (*v1.Deployment, error) {
	deployment := &v1.Deployment{
		ObjectMeta: componentMeta,
		Spec: v1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": constants.GetRawServiceLabel(componentMeta.Name),
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: podMetadata,
				Spec:       *podSpec,
			},
		},
	}
	if err := ctrl.SetControllerReference(mspServer, deployment, d.client.Scheme()); err != nil {
		logger.Error(err, "Unable to add OwnerReference to the Raw Deployment")
		return nil, err
	}
	return deployment, nil
}

func (d *deploymentReconciler) getExistingResource(ctx context.Context, logger logr.Logger, mspServer *mcpv1alpha1.MCPServer) (*v1.Deployment, error) {

	key := types.NamespacedName{
		Name:      mspServer.Name,
		Namespace: mspServer.Namespace,
	}
	deployment := &v1.Deployment{}
	err := d.client.Get(ctx, key, deployment)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Deployment not found.")
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	logger.Info("Successfully fetch deployed Deployment")
	return deployment, nil
}

func (d *deploymentReconciler) processDelta(ctx context.Context, logger logr.Logger, desiredDeployment *v1.Deployment, existingDeployment *v1.Deployment) (err error) {
	comparator := comparators.GetDeploymentComparator()
	delta := d.deltaProcessor.ComputeDelta(comparator, desiredDeployment, existingDeployment)

	if !delta.HasChanges() {
		logger.Info("No delta found")
		return nil
	}

	if delta.IsAdded() {
		logger.Info("Delta found", "create", desiredDeployment.GetName())
		if err = d.client.Create(ctx, desiredDeployment); err != nil {
			return err
		}
	}
	if delta.IsUpdated() {
		logger.Info("Delta found", "update", existingDeployment.GetName())
		rp := existingDeployment.DeepCopy()
		rp.Labels = desiredDeployment.Labels
		rp.Annotations = desiredDeployment.Annotations
		rp.Spec = desiredDeployment.Spec

		if err = d.client.Update(ctx, rp); err != nil {
			return err
		}
	}
	if delta.IsRemoved() {
		logger.Info("Delta found", "delete", existingDeployment.GetName())
		if err = d.client.Delete(ctx, existingDeployment); err != nil {
			return err
		}
	}
	return nil
}
