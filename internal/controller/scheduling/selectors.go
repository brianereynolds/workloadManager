package scheduling

import (
	k8smanagersv1 "greyridge.com/workloadManager/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// HasSelector checks if the resource has a Selector defined
func HasSelector(resource interface{}) bool {
	switch obj := resource.(type) {
	case *appsv1.Deployment:
		return hasDeploymentSelector(obj)
	case *appsv1.StatefulSet:
		return hasStatefulSetSelector(obj)
	default:
		return false
	}
}

func CheckNodeSelector(selector *metav1.LabelSelector, procedure k8smanagersv1.Procedure, wlName string) error {

	l := log.Log

	if selector == nil {
		return nil
	}

	checkOk := false

	if val, exists := selector.MatchLabels[procedure.Selector.Key]; exists {
		if val == procedure.Selector.Initial {
			checkOk = true
		}
	}

	if checkOk == false {
		l.Info("resource does not have the expected node selector", "workload name", wlName, "affinity key", procedure.Selector.Key, "expected value", procedure.Selector.Initial)
		l.Info("Continuing...")
	}
	return nil
}

func CreateNodeSelector(key, value string) map[string]string {
	return map[string]string{
		key: value,
	}
}

// Helper function to check Selector for Deployment
func hasDeploymentSelector(deployment *appsv1.Deployment) bool {
	return hasNodeSelector(deployment.Spec.Template.Spec.NodeSelector)
}

// Helper function to check Selector for StatefulSet
func hasStatefulSetSelector(statefulset *appsv1.StatefulSet) bool {
	return hasNodeSelector(statefulset.Spec.Template.Spec.NodeSelector)
}

func hasNodeSelector(nodeSelector map[string]string) bool {
	if len(nodeSelector) > 0 {
		return true
	}
	return false
}
