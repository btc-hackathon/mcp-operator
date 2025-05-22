package comparators

import (
	v1 "k8s.io/api/apps/v1"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetDeploymentComparator() ResourceComparator {
	return func(deployed client.Object, requested client.Object) bool {
		deployedDeployment := deployed.(*v1.Deployment)
		requestedDeployment := requested.(*v1.Deployment)
		return reflect.DeepEqual(deployedDeployment.Spec.Template, requestedDeployment.Spec.Template) &&
			reflect.DeepEqual(deployedDeployment.Labels, requestedDeployment.Labels) &&
			reflect.DeepEqual(deployedDeployment.Annotations, requestedDeployment.Annotations)
	}
}
