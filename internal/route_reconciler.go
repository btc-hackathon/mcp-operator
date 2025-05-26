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
	"fmt"
	"github.com/go-logr/logr"
	mcpv1alpha1 "github.com/opendatahub-io/mcp-operator/api/v1alpha1"
	v1 "github.com/openshift/api/route/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type RouteReconciler struct {
	client           client.Client
	serviceProcessor *ServiceProcessor
	deltaProcessor   *DeltaProcessor
}

func NewRouteReconciler(client client.Client) *RouteReconciler {
	return &RouteReconciler{
		client:           client,
		serviceProcessor: NewServiceProcessor(client),
		deltaProcessor:   NewDeltaProcessor(),
	}
}

func (s *RouteReconciler) Reconcile(ctx context.Context, logger logr.Logger, mcpServer *mcpv1alpha1.MCPServer, mcpServerTemplate *mcpv1alpha1.MCPServerTemplate, networkVisibility NetworkVisibility) error {

	logger.Info("Reconciling Route for RAW Deployment")
	// Create Desired resource
	desiredResource, err := s.createDesiredResource(ctx, logger, mcpServer, mcpServerTemplate, networkVisibility)
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

func (s *RouteReconciler) createDesiredResource(ctx context.Context, logger logr.Logger, mcpServer *mcpv1alpha1.MCPServer, mcpServerTemplate *mcpv1alpha1.MCPServerTemplate, networkVisibility NetworkVisibility) (*v1.Route, error) {

	if networkVisibility != Exposed {
		logger.Info("MCPServer does not have annotation '" + NetworkVisibilityAnnotation + "' " +
			" set to '" + string(Exposed) + "'. Skipping route creation")
		return nil, nil
	}

	componentMeta := GetCommonMeta(mcpServer, mcpServerTemplate)
	targetService, err := s.serviceProcessor.FetchService(ctx, logger, types.NamespacedName{Name: mcpServer.Name, Namespace: mcpServer.Namespace})
	if err != nil {
		return nil, err
	} else if targetService == nil {
		return nil, fmt.Errorf("service(%s) not found", mcpServer.Name)
	}

	var targetPort intstr.IntOrString
	for _, port := range targetService.Spec.Ports {
		if port.Name == "http" || port.Name == mcpServer.Name {
			targetPort = intstr.FromString(port.Name)
		}
	}

	route := &v1.Route{
		ObjectMeta: componentMeta,
		Spec: v1.RouteSpec{
			To: v1.RouteTargetReference{
				Kind:   "Service",
				Name:   targetService.Name,
				Weight: ptr.To(int32(100)),
			},
			Port: &v1.RoutePort{
				TargetPort: targetPort,
			},
			WildcardPolicy: v1.WildcardPolicyNone,
		},
		Status: v1.RouteStatus{
			Ingress: []v1.RouteIngress{},
		},
	}
	if err := ctrl.SetControllerReference(mcpServer, route, s.client.Scheme()); err != nil {
		logger.Error(err, "Unable to add OwnerReference to the Raw Deployment Route")
		return nil, err
	}
	return route, nil
}

func (s *RouteReconciler) getExistingResource(ctx context.Context, logger logr.Logger, mcpServer *mcpv1alpha1.MCPServer) (*v1.Route, error) {

	key := types.NamespacedName{
		Name:      mcpServer.Name,
		Namespace: mcpServer.Namespace,
	}
	service := &v1.Route{}
	err := s.client.Get(ctx, key, service)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Route not found.")
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	logger.Info("Successfully fetch deployed Route")
	return service, nil
}

func (s *RouteReconciler) processDelta(ctx context.Context, logger logr.Logger, desiredRoute *v1.Route, existingRoute *v1.Route) (err error) {
	comparator := GetRouteComparator()
	delta := s.deltaProcessor.ComputeDelta(comparator, desiredRoute, existingRoute)

	if !delta.HasChanges() {
		logger.Info("No delta found")
		return nil
	}

	if delta.IsAdded() {
		logger.Info("Delta found", "create", desiredRoute.GetName())
		if err = s.client.Create(ctx, desiredRoute); err != nil {
			return err
		}
	}
	if delta.IsUpdated() {
		logger.Info("Delta found", "update", existingRoute.GetName())
		rp := existingRoute.DeepCopy()
		rp.Labels = desiredRoute.Labels
		rp.Annotations = desiredRoute.Annotations
		rp.Spec = desiredRoute.Spec

		if err = s.client.Update(ctx, rp); err != nil {
			return err
		}
	}
	if delta.IsRemoved() {
		logger.Info("Delta found", "delete", existingRoute.GetName())
		if err = s.client.Delete(ctx, existingRoute); err != nil {
			return err
		}
	}
	return nil
}
