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
	mcpv1alpha1 "github.com/opendatahub-io/mcp-operator/api/v1alpha1"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type DeploymentReconciler struct {
	client              client.Client
	deploymentProcessor *DeploymentProcessor
	deltaProcessor      *DeltaProcessor
}

func NewDeploymentReconciler(client client.Client) *DeploymentReconciler {
	return &DeploymentReconciler{
		client:              client,
		deploymentProcessor: NewDeploymentProcessor(client),
		deltaProcessor:      NewDeltaProcessor(),
	}
}

func (d *DeploymentReconciler) Reconcile(ctx context.Context, logger logr.Logger, mspServer *mcpv1alpha1.MCPServer, mcpServerTemplate *mcpv1alpha1.MCPServerTemplate) error {

	logger.Info("Reconciling Deployment for RAW Deployment")
	// Create Desired resource
	desiredResource, err := d.createDesiredResource(logger, mspServer, mcpServerTemplate)
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

	available, err := d.deploymentProcessor.IsDeploymentAvailable(ctx, logger, types.NamespacedName{Name: mspServer.GetName(), Namespace: mspServer.GetNamespace()})
	if err != nil {
		logger.Error(err, "failed to check deployment status")
		return err
	}
	if !available {
		logger.Info("deployment not available")
		return ErrorForDeploymentNotReachable(mspServer.GetName())
	}

	return nil
}

func (d *DeploymentReconciler) createDesiredResource(logger logr.Logger, mcpServer *mcpv1alpha1.MCPServer, mcpServerTemplate *mcpv1alpha1.MCPServerTemplate) (*v1.Deployment, error) {

	podSpec, err := GetCommonPodSpec(mcpServer, mcpServerTemplate)
	if err != nil {
		return nil, err
	}
	componentMeta := GetCommonMeta(mcpServer, mcpServerTemplate)

	podMetadata := componentMeta
	podMetadata.Labels["app"] = mcpServer.Name

	deployment := &v1.Deployment{
		ObjectMeta: componentMeta,
		Spec: v1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": mcpServer.Name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: podMetadata,
				Spec:       *podSpec,
			},
		},
	}
	if err := ctrl.SetControllerReference(mcpServer, deployment, d.client.Scheme()); err != nil {
		logger.Error(err, "Unable to add OwnerReference to the Raw Deployment")
		return nil, err
	}
	return deployment, nil
}

func (d *DeploymentReconciler) getExistingResource(ctx context.Context, logger logr.Logger, mspServer *mcpv1alpha1.MCPServer) (*v1.Deployment, error) {
	key := types.NamespacedName{
		Name:      mspServer.Name,
		Namespace: mspServer.Namespace,
	}
	return d.deploymentProcessor.FetchDeployment(ctx, logger, key)
}

func (d *DeploymentReconciler) processDelta(ctx context.Context, logger logr.Logger, desiredDeployment *v1.Deployment, existingDeployment *v1.Deployment) (err error) {
	comparator := GetDeploymentComparator(logger)
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
