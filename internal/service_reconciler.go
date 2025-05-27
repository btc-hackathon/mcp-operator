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
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strconv"
)

type ServiceReconciler struct {
	client           client.Client
	serviceProcessor *ServiceProcessor
	deltaProcessor   *DeltaProcessor
}

func NewServiceReconciler(client client.Client) *ServiceReconciler {
	return &ServiceReconciler{
		client:           client,
		serviceProcessor: NewServiceProcessor(client),
		deltaProcessor:   NewDeltaProcessor(),
	}
}

func (s *ServiceReconciler) Reconcile(ctx context.Context, logger logr.Logger, mcpServer *mcpv1alpha1.MCPServer, mcpServerTemplate *mcpv1alpha1.MCPServerTemplate, mcpServerConfig *MCPServerConfig) error {

	logger.Info("Reconciling Service for RAW Deployment")
	// Create Desired resource
	desiredResource, err := s.createDesiredResource(logger, mcpServer, mcpServerTemplate, mcpServerConfig)
	if err != nil {
		return err
	}

	// Get Existing resource
	existingResource, err := s.getExistingResource(ctx, logger, mcpServer)
	if err != nil {
		return err
	}

	// Process Delta
	if err = s.processDelta(ctx, logger, desiredResource, existingResource); err != nil {
		return err
	}
	return nil
}

func (s *ServiceReconciler) createDesiredResource(logger logr.Logger, mcpServer *mcpv1alpha1.MCPServer, mcpServerTemplate *mcpv1alpha1.MCPServerTemplate, mcpServerConfig *MCPServerConfig) (*corev1.Service, error) {

	deploymentMode := GetDeploymentMode(logger, mcpServer.Annotations, mcpServerConfig)
	if deploymentMode != RawDeployment {
		logger.Info("Deployment mode is not RAWDeployment. Skipping Service creation")
		return nil, nil
	}

	container, err := GetUnifiedMCPServerContainer(mcpServerTemplate, mcpServer)
	if err != nil {
		return nil, err
	}

	var servicePorts []corev1.ServicePort
	if len(container.Ports) > 0 {
		var servicePort corev1.ServicePort
		servicePort = corev1.ServicePort{
			Name: container.Ports[0].Name,
			Port: CommonDefaultHttpPort,
			TargetPort: intstr.IntOrString{
				Type:   intstr.Int,
				IntVal: container.Ports[0].ContainerPort,
			},
			Protocol: container.Ports[0].Protocol,
		}
		if len(servicePort.Name) == 0 {
			servicePort.Name = "http"
		}
		servicePorts = append(servicePorts, servicePort)

		for i := 1; i < len(container.Ports); i++ {
			port := container.Ports[i]
			if port.Protocol == "" {
				port.Protocol = corev1.ProtocolTCP
			}
			servicePort = corev1.ServicePort{
				Name: port.Name,
				Port: port.ContainerPort,
				TargetPort: intstr.IntOrString{
					Type:   intstr.Int,
					IntVal: port.ContainerPort,
				},
				Protocol: port.Protocol,
			}
			servicePorts = append(servicePorts, servicePort)
		}
	} else {
		port, _ := strconv.Atoi(MCPServerDefaultHttpPort)
		portInt32 := int32(port) // nolint  #nosec G109
		servicePorts = append(servicePorts, corev1.ServicePort{
			Name: mcpServer.Name,
			Port: CommonDefaultHttpPort,
			TargetPort: intstr.IntOrString{
				Type:   intstr.Int,
				IntVal: portInt32, // #nosec G109
			},
			Protocol: corev1.ProtocolTCP,
		})
	}

	componentMeta := GetCommonMeta(mcpServer, mcpServerTemplate)

	service := &corev1.Service{
		ObjectMeta: componentMeta,
		Spec: corev1.ServiceSpec{
			Ports: servicePorts,
			Type:  corev1.ServiceTypeClusterIP,
			Selector: map[string]string{
				"app": mcpServer.Name,
			},
		},
	}
	if err := ctrl.SetControllerReference(mcpServer, service, s.client.Scheme()); err != nil {
		logger.Error(err, "Unable to add OwnerReference to the Raw Deployment Service")
		return nil, err
	}
	return service, nil
}

func (s *ServiceReconciler) getExistingResource(ctx context.Context, logger logr.Logger, mcpServer *mcpv1alpha1.MCPServer) (*corev1.Service, error) {

	key := types.NamespacedName{
		Name:      mcpServer.Name,
		Namespace: mcpServer.Namespace,
	}
	return s.serviceProcessor.FetchService(ctx, logger, key)
}

func (s *ServiceReconciler) processDelta(ctx context.Context, logger logr.Logger, desiredService *corev1.Service, existingService *corev1.Service) (err error) {
	comparator := GetServiceComparator()
	delta := s.deltaProcessor.ComputeDelta(comparator, desiredService, existingService)

	if !delta.HasChanges() {
		logger.Info("No delta found")
		return nil
	}

	if delta.IsAdded() {
		logger.Info("Delta found", "create", desiredService.GetName())
		if err = s.client.Create(ctx, desiredService); err != nil {
			return err
		}
	}
	if delta.IsUpdated() {
		logger.Info("Delta found", "update", existingService.GetName())
		rp := existingService.DeepCopy()
		rp.Labels = desiredService.Labels
		rp.Annotations = desiredService.Annotations
		rp.Spec = desiredService.Spec

		if err = s.client.Update(ctx, rp); err != nil {
			return err
		}
	}
	if delta.IsRemoved() {
		logger.Info("Delta found", "delete", existingService.GetName())
		if err = s.client.Delete(ctx, existingService); err != nil {
			return err
		}
	}
	return nil
}
