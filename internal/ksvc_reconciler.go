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
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	knservingv1 "knative.dev/serving/pkg/apis/serving/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type KSVCReconciler struct {
	client         client.Client
	deltaProcessor *DeltaProcessor
}

func NewKSVCReconciler(client client.Client) *KSVCReconciler {
	return &KSVCReconciler{
		client:         client,
		deltaProcessor: NewDeltaProcessor(),
	}
}

func (k *KSVCReconciler) Reconcile(ctx context.Context, logger logr.Logger, mspServer *mcpv1alpha1.MCPServer, mcpServerTemplate *mcpv1alpha1.MCPServerTemplate) error {

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

func (k *KSVCReconciler) createDesiredResource(logger logr.Logger, mcpServer *mcpv1alpha1.MCPServer, mcpServerTemplate *mcpv1alpha1.MCPServerTemplate) (*knservingv1.Service, error) {

	podSpec, err := GetCommonPodSpec(mcpServer, mcpServerTemplate)
	if err != nil {
		return nil, err
	}
	componentMeta := GetCommonMeta(mcpServer, mcpServerTemplate)

	podMetadata := componentMeta
	podMetadata.Name = mcpServer.Name + "-" + "service"
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

func (k *KSVCReconciler) getExistingResource(ctx context.Context, logger logr.Logger, mspServer *mcpv1alpha1.MCPServer) (*knservingv1.Service, error) {
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

func (d *KSVCReconciler) processDelta(ctx context.Context, logger logr.Logger, desiredService *knservingv1.Service, existingService *knservingv1.Service) (err error) {
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
