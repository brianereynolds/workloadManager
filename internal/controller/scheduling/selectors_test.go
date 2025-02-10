package scheduling

import (
	"testing"

	k8smanagersv1 "greyridge.com/workloadManager/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestHasSelector tests the HasSelector function
func TestHasSelector(t *testing.T) {
	deployment := &appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					NodeSelector: map[string]string{
						"disktype": "ssd",
					},
				},
			},
		},
	}
	if !HasSelector(deployment) {
		t.Errorf("Expected deployment to have a selector")
	}

	statefulSet := &appsv1.StatefulSet{
		Spec: appsv1.StatefulSetSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					NodeSelector: map[string]string{
						"disktype": "ssd",
					},
				},
			},
		},
	}
	if !HasSelector(statefulSet) {
		t.Errorf("Expected statefulset to have a selector")
	}

	otherResource := "non-k8s-resource"
	if HasSelector(otherResource) {
		t.Errorf("Expected non-k8s resource to not have a selector")
	}
}

// TestCheckNodeSelector tests the CheckNodeSelector function
func TestCheckNodeSelector(t *testing.T) {
	selector := &metav1.LabelSelector{
		MatchLabels: map[string]string{
			"disktype": "ssd",
		},
	}
	procedure := k8smanagersv1.Procedure{
		Selector: k8smanagersv1.Selector{
			Key:     "disktype",
			Initial: "ssd",
		},
	}
	err := CheckNodeSelector(selector, procedure, "test-workload")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	procedure.Selector.Initial = "hdd"
	err = CheckNodeSelector(selector, procedure, "test-workload")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

// TestCreateNodeSelector tests the CreateNodeSelector function
func TestCreateNodeSelector(t *testing.T) {
	key := "disktype"
	value := "ssd"
	nodeSelector := CreateNodeSelector(key, value)
	if nodeSelector[key] != value {
		t.Errorf("Expected nodeSelector[%s] to be %s, got %s", key, value, nodeSelector[key])
	}
}

// TestHasDeploymentSelector tests the hasDeploymentSelector helper function
func TestHasDeploymentSelector(t *testing.T) {
	deployment := &appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					NodeSelector: map[string]string{
						"disktype": "ssd",
					},
				},
			},
		},
	}
	if !hasDeploymentSelector(deployment) {
		t.Errorf("Expected deployment to have a selector")
	}
}

// TestHasStatefulSetSelector tests the hasStatefulSetSelector helper function
func TestHasStatefulSetSelector(t *testing.T) {
	statefulSet := &appsv1.StatefulSet{
		Spec: appsv1.StatefulSetSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					NodeSelector: map[string]string{
						"disktype": "ssd",
					},
				},
			},
		},
	}
	if !hasStatefulSetSelector(statefulSet) {
		t.Errorf("Expected statefulset to have a selector")
	}
}

// TestHasNodeSelector tests the hasNodeSelector helper function
func TestHasNodeSelector(t *testing.T) {
	nodeSelector := map[string]string{
		"disktype": "ssd",
	}
	if !hasNodeSelector(nodeSelector) {
		t.Errorf("Expected nodeSelector to be true")
	}

	emptyNodeSelector := map[string]string{}
	if hasNodeSelector(emptyNodeSelector) {
		t.Errorf("Expected empty nodeSelector to be false")
	}
}
