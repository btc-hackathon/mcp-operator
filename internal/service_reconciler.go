package internal

import (
	"context"
	"github.com/go-logr/logr"
	mcpv1alpha1 "github.com/opendatahub-io/mcp-operator/api/v1alpha1"
	"github.com/opendatahub-io/mcp-operator/internal/comparators"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strconv"
)

type ServiceReconciler interface {
	Reconcile(ctx context.Context, logger logr.Logger, mspServer *mcpv1alpha1.MCPServer, podSpec mcpv1alpha1.PodSpec) error
}

type serviceReconciler struct {
	client         client.Client
	deltaProcessor DeltaProcessor
}

func NewServiceReconciler(client client.Client) ServiceReconciler {
	return &serviceReconciler{
		client:         client,
		deltaProcessor: NewDeltaProcessor(),
	}
}

func (s *serviceReconciler) Reconcile(ctx context.Context, logger logr.Logger, mspServer *mcpv1alpha1.MCPServer, podSpec mcpv1alpha1.PodSpec) error {

	logger.Info("Reconciling Service for RAW Deployment")
	// Create Desired resource
	desiredResource, err := s.createDesiredResource(logger, mspServer, podSpec)
	if err != nil {
		return err
	}

	// Get Existing resource
	existingResource, err := s.getExistingResource(ctx, logger, mspServer)
	if err != nil {
		return err
	}

	// Process Delta
	if err = s.processDelta(ctx, logger, desiredResource, existingResource); err != nil {
		return err
	}
	return nil
}

func (s *serviceReconciler) createDesiredResource(logger logr.Logger, mspServer *mcpv1alpha1.MCPServer, podSpec mcpv1alpha1.PodSpec) (*corev1.Service, error) {

	container, err := FetchMSPServerContainer(podSpec.Containers)
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
			Name: mspServer.Name,
			Port: CommonDefaultHttpPort,
			TargetPort: intstr.IntOrString{
				Type:   intstr.Int,
				IntVal: portInt32, // #nosec G109
			},
			Protocol: corev1.ProtocolTCP,
		})
	}

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mspServer.Name,
			Namespace: mspServer.Namespace,
			Labels: map[string]string{
				"name": mspServer.Name,
			},
		},
		Spec: corev1.ServiceSpec{
			Ports: servicePorts,
			Type:  corev1.ServiceTypeClusterIP,
			Selector: map[string]string{
				"app": mspServer.Name,
			},
		},
	}
	if err := ctrl.SetControllerReference(mspServer, service, s.client.Scheme()); err != nil {
		logger.Error(err, "Unable to add OwnerReference to the Raw Deployment Service")
		return nil, err
	}
	return service, nil
}

func (s *serviceReconciler) getExistingResource(ctx context.Context, logger logr.Logger, mspServer *mcpv1alpha1.MCPServer) (*corev1.Service, error) {

	key := types.NamespacedName{
		Name:      mspServer.Name,
		Namespace: mspServer.Namespace,
	}
	service := &corev1.Service{}
	err := s.client.Get(ctx, key, service)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Service not found.")
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	logger.Info("Successfully fetch deployed Service")
	return service, nil
}

func (s *serviceReconciler) processDelta(ctx context.Context, logger logr.Logger, desiredService *corev1.Service, existingService *corev1.Service) (err error) {
	comparator := comparators.GetServiceComparator()
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
