package comparators

import (
	knservingv1 "knative.dev/serving/pkg/apis/serving/v1"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetKSVCComparator() ResourceComparator {
	return func(deployed client.Object, requested client.Object) bool {
		deployedService := deployed.(*knservingv1.Service)
		requestedService := requested.(*knservingv1.Service)
		return reflect.DeepEqual(deployedService.Spec.Template, requestedService.Spec.Template) &&
			reflect.DeepEqual(deployedService.Labels, requestedService.Labels) &&
			reflect.DeepEqual(deployedService.Annotations, requestedService.Annotations)
	}
}
