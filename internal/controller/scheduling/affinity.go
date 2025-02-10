package scheduling

import (
	k8smanagersv1 "greyridge.com/workloadManager/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// HasAffinity checks if the resource has a Selector defined
func HasAffinity(resource interface{}) bool {
	switch obj := resource.(type) {
	case *appsv1.Deployment:
		return hasDeploymentAffinity(obj)
	case *appsv1.StatefulSet:
		return hasStatefulSetAffinity(obj)
	default:
		return false
	}
}

func CheckNodeAffinity(affinity *v1.NodeAffinity, procedure k8smanagersv1.Procedure, wlName string) error {
	l := log.Log

	if affinity == nil {
		return nil
	}

	checkOk := false
	for _, terms := range affinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms {
		for _, expressions := range terms.MatchExpressions {
			if expressions.Key == procedure.Affinity.Key {
				for _, value := range expressions.Values {
					if value == procedure.Affinity.Initial {
						checkOk = true
					}
				}
			}
		}
	}

	if checkOk == false {
		l.Info("resource does not have the expected node affinity", "workload name", wlName, "affinity key", procedure.Affinity.Key, "expected value", procedure.Affinity.Initial)
		l.Info("Continuing...")
	}
	return nil
}

// CreateNodeAffinity Helper function to create a Node Affinity
func CreateNodeAffinity(key string, value string) *v1.NodeAffinity {
	nodeAffinity := &v1.NodeAffinity{
		RequiredDuringSchedulingIgnoredDuringExecution: &v1.NodeSelector{
			NodeSelectorTerms: []v1.NodeSelectorTerm{
				{
					MatchExpressions: []v1.NodeSelectorRequirement{
						{
							Key:      key,
							Operator: v1.NodeSelectorOpIn,
							Values:   []string{value},
						},
					},
				},
			},
		},
	}
	return nodeAffinity
}

// Helper function to check Selector for Deployment
func hasDeploymentAffinity(deployment *appsv1.Deployment) bool {
	return hasPodSpecNodeAffinity(&deployment.Spec.Template.Spec)
}

// Helper function to check Selector for StatefulSet
func hasStatefulSetAffinity(statefulset *appsv1.StatefulSet) bool {
	return hasPodSpecNodeAffinity(&statefulset.Spec.Template.Spec)
}

func hasPodSpecNodeAffinity(podSpec *corev1.PodSpec) bool {
	if podSpec.Affinity == nil {
		return false
	}

	nodeAffinity := podSpec.Affinity.NodeAffinity
	if nodeAffinity == nil {
		return false
	}

	if nodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution != nil ||
		nodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution != nil {
		return true
	}

	return false
}
