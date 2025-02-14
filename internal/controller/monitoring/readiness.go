package monitoring

import (
	"context"
	k8smanagersv1 "greyridge.com/workloadManager/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"time"
)

func IsResourceReady(ctx context.Context, wlType string) bool {
	namespace := ctx.Value("namespace").(string)
	clientset := ctx.Value("clientset").(kubernetes.Interface)

	if wlType == k8smanagersv1.Deployment {
		deployment := ctx.Value("resource").(*appsv1.Deployment)
		return isDeploymentReady(clientset, namespace, deployment)
	}
	if wlType == k8smanagersv1.StatefulSet {
		statefulset := ctx.Value("resource").(*appsv1.StatefulSet)
		return isStatefulSetReady(clientset, namespace, statefulset)
	}
	return false
}

func isDeploymentReady(clientset kubernetes.Interface, namespace string, deployment *appsv1.Deployment) bool {
	l := log.Log
	l.Info("Waiting to start...", "name", deployment.Name)

	mondeployment, err := clientset.AppsV1().Deployments(namespace).Get(context.Background(), deployment.Name, metav1.GetOptions{})
	if err != nil {
		l.Error(err, "Could not monitor")
	}

	labelSelector := metav1.FormatLabelSelector(mondeployment.Spec.Selector)
	pods, err := getPodFromLabel(clientset, namespace, labelSelector)

	var podname string = "UNKNOWN"
	if len(pods.Items) > 0 {
		podname = pods.Items[0].Name
	}

	// Get the pod
	pod, err := clientset.CoreV1().Pods(namespace).Get(context.Background(), podname, metav1.GetOptions{})
	if err != nil {
		l.Error(err, "Could not find pod named", "podname", podname)
	}

	for _, condition := range pod.Status.Conditions {
		if condition.Type == "Ready" && condition.Status == "True" {
			creationTimestamp := pod.ObjectMeta.CreationTimestamp.Time
			currentTime := time.Now()

			l.Info("Pod started.", "name", deployment.Name, "start-up time", currentTime.Sub(creationTimestamp))
			return true
		}
	}

	return false
}

func isStatefulSetReady(clientset kubernetes.Interface, namespace string, statefulset *appsv1.StatefulSet) bool {
	l := log.Log
	l.Info("Waiting to start...", "name", statefulset.Name)

	monstatefulset, err := clientset.AppsV1().StatefulSets(namespace).Get(context.Background(), statefulset.Name, metav1.GetOptions{})
	if err != nil {
		l.Error(err, "Could not monitor")
	}

	labelSelector := metav1.FormatLabelSelector(monstatefulset.Spec.Selector)
	pods, err := getPodFromLabel(clientset, namespace, labelSelector)

	// Print the status of each pod
	for _, pod := range pods.Items {
		if pod.DeletionTimestamp != nil {
			// One of the pods in the stateful set is term
			l.Info("Pod is terminating", "name", pod.Name)
			return false
		}
	}

	expectedReplicas := *monstatefulset.Spec.Replicas
	readyReplicas := monstatefulset.Status.ReadyReplicas
	l.Info("Monitoring replicas", "expected", expectedReplicas, "ready", readyReplicas)
	if readyReplicas == expectedReplicas {
		l.Info("Statefulset ready.", "name", statefulset.Name)
		return true
	}
	return false
}

func getPodFromLabel(clientset kubernetes.Interface, namespace string, labelSelector string) (*v1.PodList, error) {
	// List the pods matching the label selector
	pods, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, err
	}
	return pods, nil
}
