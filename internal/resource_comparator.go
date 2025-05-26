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
	"github.com/go-logr/logr"
	v12 "github.com/openshift/api/route/v1"
	v1 "k8s.io/api/apps/v1"

	corev1 "k8s.io/api/core/v1"
	knservingv1 "knative.dev/serving/pkg/apis/serving/v1"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sort"
)

// ResourceComparator would compare deployed & requested resource. It would retrun `true` if both resource are same else it would return `false`
type ResourceComparator func(deployed client.Object, requested client.Object) bool

func GetDeploymentComparator(logger logr.Logger) ResourceComparator {
	return func(deployed client.Object, requested client.Object) bool {
		deployedDeployment := deployed.(*v1.Deployment)
		requestedDeployment := requested.(*v1.Deployment)
		return comparePodTemplateSpec(&deployedDeployment.Spec.Template.Spec, &requestedDeployment.Spec.Template.Spec)
	}
}

func comparePodTemplateSpec(deployed *corev1.PodSpec, requested *corev1.PodSpec) bool {

	result := reflect.DeepEqual(deployed.ImagePullSecrets, requested.ImagePullSecrets) &&
		reflect.DeepEqual(deployed.ServiceAccountName, requested.ServiceAccountName)
	if !result {
		return false
	}

	if len(deployed.Containers) != len(requested.Containers) {
		return false
	}
	sortContainersByName(deployed)
	sortContainersByName(requested)
	for i := range deployed.Containers {
		result = reflect.DeepEqual(deployed.Containers[i].Image, requested.Containers[i].Image) &&
			reflect.DeepEqual(deployed.Containers[i].Ports, requested.Containers[i].Ports) &&
			reflect.DeepEqual(deployed.Containers[i].Args, requested.Containers[i].Args) &&
			reflect.DeepEqual(deployed.Containers[i].Env, requested.Containers[i].Env)
		if !result {
			return false
		}
	}
	return true
}

func sortContainersByName(pod *corev1.PodSpec) {
	sort.Slice(pod.Containers, func(i, j int) bool {
		return pod.Containers[i].Name < pod.Containers[j].Name
	})
}

func GetServiceComparator() ResourceComparator {
	return func(deployed client.Object, requested client.Object) bool {
		deployedService := deployed.(*corev1.Service)
		requestedService := requested.(*corev1.Service)
		return reflect.DeepEqual(deployedService.Spec.Ports, requestedService.Spec.Ports) &&
			reflect.DeepEqual(deployedService.Spec.Type, requestedService.Spec.Type) &&
			reflect.DeepEqual(deployedService.Spec.Selector, requestedService.Spec.Selector) &&
			reflect.DeepEqual(deployedService.Annotations, requestedService.Annotations) &&
			reflect.DeepEqual(deployedService.Labels, requestedService.Labels)
	}
}

func GetKSVCComparator() ResourceComparator {
	return func(deployed client.Object, requested client.Object) bool {
		deployedService := deployed.(*knservingv1.Service)
		requestedService := requested.(*knservingv1.Service)
		return comparePodTemplateSpec(&deployedService.Spec.Template.Spec.PodSpec, &requestedService.Spec.Template.Spec.PodSpec)
	}
}

func GetRouteComparator() ResourceComparator {
	return func(deployed client.Object, requested client.Object) bool {
		deployedRoute := deployed.(*v12.Route)
		requestedRoute := requested.(*v12.Route)
		return reflect.DeepEqual(deployedRoute.Spec.Host, requestedRoute.Spec.Host) &&
			reflect.DeepEqual(deployedRoute.Spec.To, requestedRoute.Spec.To) &&
			reflect.DeepEqual(deployedRoute.Spec.Port, requestedRoute.Spec.Port) &&
			reflect.DeepEqual(deployedRoute.Spec.TLS, requestedRoute.Spec.TLS) &&
			reflect.DeepEqual(deployedRoute.Spec.WildcardPolicy, requestedRoute.Spec.WildcardPolicy) &&
			reflect.DeepEqual(deployedRoute.ObjectMeta.Labels, requestedRoute.ObjectMeta.Labels)
	}
}
