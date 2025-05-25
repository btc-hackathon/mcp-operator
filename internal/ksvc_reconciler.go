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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	knservingv1 "knative.dev/serving/pkg/apis/serving/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type KSVCReconciler interface {
	Reconcile(ctx context.Context, logger logr.Logger, mspServer *mcpv1alpha1.MCPServer, mcpServerTemplate *mcpv1alpha1.MCPServerTemplate) error
}

type ksvcReconciler struct {
	client         client.Client
	deltaProcessor DeltaProcessor
}

func NewKSVCReconciler(client client.Client) KSVCReconciler {
	return &ksvcReconciler{
		client:         client,
		deltaProcessor: NewDeltaProcessor(),
	}
}

func (k *ksvcReconciler) Reconcile(ctx context.Context, logger logr.Logger, mspServer *mcpv1alpha1.MCPServer, mcpServerTemplate *mcpv1alpha1.MCPServerTemplate) error {

	logger.Info("Reconciling Knative service for Serverless")
	// Create Desired resource
	desiredResource, err := k.createDesiredResource(logger, mspServer, mcpServerTemplate)
	if err != nil {
		return err
	}

	// Get Existing resource
	existingResource, err := k.getExistingResource(ctx, logger, mspServer)
	if err != nil {
		return err
	}

	// Process Delta
	if err = k.processDelta(ctx, logger, desiredResource, existingResource); err != nil {
		return err
	}
	return nil
}

func (k *ksvcReconciler) createDesiredResource(logger logr.Logger, mcpServer *mcpv1alpha1.MCPServer, mcpServerTemplate *mcpv1alpha1.MCPServerTemplate) (*knservingv1.Service, error) {

	mergedContainer, err := GetUnifiedMCPServerContainer(mcpServerTemplate, mcpServer)
	if err != nil {
		return nil, err
	}
	var newPodSpecContainers []corev1.Container
	for _, container := range mcpServerTemplate.Spec.Containers {
		if container.Name == MCPServerContainerName {
			newPodSpecContainers = append(newPodSpecContainers, *mergedContainer)
		} else {
			newPodSpecContainers = append(newPodSpecContainers, container)
		}
	}

	podSpec := &corev1.PodSpec{
		Containers:         newPodSpecContainers,
		ImagePullSecrets:   append(mcpServerTemplate.Spec.ImagePullSecrets, mcpServer.Spec.ImagePullSecrets...),
		ServiceAccountName: mcpServer.Spec.ServiceAccountName,
	}

	componentMeta := metav1.ObjectMeta{
		Name:      mcpServer.Name,
		Namespace: mcpServer.Namespace,
		Labels: Union(
			mcpServerTemplate.Labels,
			mcpServer.Labels,
			map[string]string{
				MCPServerPodLabelKey: mcpServer.Name,
			},
		),
		Annotations: Union(
			mcpServerTemplate.Annotations,
			mcpServer.Annotations,
		),
	}

	podMetadata := componentMeta
	podMetadata.Labels["app"] = mcpServer.Name

	service := &knservingv1.Service{
		ObjectMeta: componentMeta,
		Spec: knservingv1.ServiceSpec{
			ConfigurationSpec: knservingv1.ConfigurationSpec{
				Template: knservingv1.RevisionTemplateSpec{
					ObjectMeta: podMetadata,
					Spec: knservingv1.RevisionSpec{
						PodSpec: *podSpec,
					},
				},
			},
		},
	}
	if err := ctrl.SetControllerReference(mcpServer, service, k.client.Scheme()); err != nil {
		logger.Error(err, "Unable to add OwnerReference to the Knative Service")
		return nil, err
	}
	return service, nil
}

func (k *ksvcReconciler) getExistingResource(ctx context.Context, logger logr.Logger, mspServer *mcpv1alpha1.MCPServer) (*knservingv1.Service, error) {
	key := types.NamespacedName{
		Name:      mspServer.Name,
		Namespace: mspServer.Namespace,
	}
	deployment := &knservingv1.Service{}
	err := k.client.Get(ctx, key, deployment)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Knative Service not found.")
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	logger.Info("Successfully fetch deployed Knative Service")
	return deployment, nil
}

func (d *ksvcReconciler) processDelta(ctx context.Context, logger logr.Logger, desiredService *knservingv1.Service, existingService *knservingv1.Service) (err error) {
	comparator := GetKSVCComparator()
	delta := d.deltaProcessor.ComputeDelta(comparator, desiredService, existingService)

	if !delta.HasChanges() {
		logger.Info("No delta found")
		return nil
	}

	if delta.IsAdded() {
		logger.Info("Delta found", "create", desiredService.GetName())
		if err = d.client.Create(ctx, desiredService); err != nil {
			return err
		}
	}
	if delta.IsUpdated() {
		logger.Info("Delta found", "update", existingService.GetName())
		rp := existingService.DeepCopy()
		rp.Labels = desiredService.Labels
		rp.Annotations = desiredService.Annotations
		rp.Spec = desiredService.Spec

		if err = d.client.Update(ctx, rp); err != nil {
			return err
		}
	}
	if delta.IsRemoved() {
		logger.Info("Delta found", "delete", existingService.GetName())
		if err = d.client.Delete(ctx, existingService); err != nil {
			return err
		}
	}
	return nil
}
