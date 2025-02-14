package monitoring

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	k8smanagersv1 "greyridge.com/workloadManager/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestIsResourceReady(t *testing.T) {
	namespace := "test-namespace"
	deploymentName := "test-deployment"
	statefulsetName := "test-statefulset"

	clientset := fake.NewClientset(
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      deploymentName,
				Namespace: namespace,
			},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"app": "test"},
				},
			},
		},
		&appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      statefulsetName,
				Namespace: namespace,
			},
			Spec: appsv1.StatefulSetSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"app": "test"},
				},
				Replicas: int32Ptr(1),
			},
			Status: appsv1.StatefulSetStatus{
				ReadyReplicas: 1,
			},
		},
		&v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pod",
				Namespace: namespace,
				Labels:    map[string]string{"app": "test"},
			},
			Status: v1.PodStatus{
				Conditions: []v1.PodCondition{
					{
						Type:   v1.PodReady,
						Status: v1.ConditionTrue,
					},
				},
			},
		},
	)

	ctx := context.Background()
	ctx = context.WithValue(ctx, "namespace", namespace)
	ctx = context.WithValue(ctx, "clientset", clientset)

	// Test Deployment readiness
	ctx = context.WithValue(ctx, "resource", &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: deploymentName},
	})
	assert.True(t, IsResourceReady(ctx, k8smanagersv1.Deployment))

	// Test StatefulSet readiness
	ctx = context.WithValue(ctx, "resource", &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{Name: statefulsetName},
	})
	assert.True(t, IsResourceReady(ctx, k8smanagersv1.StatefulSet))
}

func int32Ptr(i int32) *int32 {
	return &i
}
