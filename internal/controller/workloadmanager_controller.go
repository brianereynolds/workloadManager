/*
Copyright 2024.

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

package controller

import (
	"context"
	"errors"
	"github.com/brianereynolds/k8smanagers_utils"
	k8smanagersv1 "greyridge.com/workloadManager/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"os"
	"os/exec"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"strings"
	"time"
)

// WorkloadManagerReconciler reconciles a WorkloadManager object
type WorkloadManagerReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	clientset *kubernetes.Clientset
}

func (r *WorkloadManagerReconciler) getClientSet(ctx context.Context, wlManager *k8smanagersv1.WorkloadManager) (*kubernetes.Clientset, error) {
	l := log.Log

	if r.clientset != nil {
		return r.clientset, nil
	}

	l.Info("getClientSet using " + wlManager.Spec.SPNLoginType)

	aksClient, err := k8smanagers_utils.GetManagedClusterClient(ctx, wlManager.Spec.SubscriptionID)
	var kubeconfig []byte

	if wlManager.Spec.SPNLoginType == k8smanagersv1.ListClusterAdminCredentials {
		kubeConfigResp, err := aksClient.ListClusterAdminCredentials(ctx, wlManager.Spec.ResourceGroup,
			wlManager.Spec.ClusterName, nil)
		if err != nil {
			l.Error(err, "failed to get AKS credentials using "+k8smanagersv1.ListClusterAdminCredentials)
			return nil, err
		}

		kubeconfig = kubeConfigResp.Kubeconfigs[0].Value
	}

	if wlManager.Spec.SPNLoginType == k8smanagersv1.ListClusterUserCredentials {
		kubeConfigResp, err := aksClient.ListClusterUserCredentials(ctx, wlManager.Spec.ResourceGroup, wlManager.Spec.ClusterName, nil)
		if err != nil {
			l.Error(err, "failed to get AKS credentials using "+k8smanagersv1.ListClusterUserCredentials)
			return nil, err
		}

		kubeconfig = kubeConfigResp.Kubeconfigs[0].Value
	}

	var kubeconfigpath = "/.kube/config"

	if kubeconfig != nil {
		_, err = k8smanagers_utils.WriteKubeFile(kubeconfig)
		if err != nil {
			l.Error(err, "cannot write kubeconfig")
			return nil, err
		}
	}

	if wlManager.Spec.SPNLoginType == k8smanagersv1.AzCli {
		cmd := exec.Command("az", "login", "--service-principal",
			"--username", os.Getenv("AZURE_CLIENT_ID"),
			"--tenant", os.Getenv("AZURE_TENANT_ID"),
			"--password", os.Getenv("AZURE_CLIENT_SECRET"))
		l.Info("az login: ", "cmd", strings.Replace(cmd.String(), os.Getenv("AZURE_CLIENT_SECRET"), "*********", -1))
		result, err := cmd.CombinedOutput()
		if err != nil {
			l.Error(err, "failed to az login using Azure CLI", "error", string(result))
			return nil, err
		}

		cmd = exec.Command("az", "account", "set", "--subscription", wlManager.Spec.SubscriptionID)
		l.V(1).Info("az account set sub", "cmd", cmd)
		result, err = cmd.CombinedOutput()
		if err != nil {
			l.Error(err, "failed to set subscription using Azure CLI", "error", string(result))
			return nil, err
		}

		cmd = exec.Command("az", "aks", "get-credentials", "--resource-group", wlManager.Spec.ResourceGroup,
			"--name", wlManager.Spec.ClusterName, "--overwrite-existing")
		l.V(1).Info("az aks get creds", "cmd", cmd)
		result, err = cmd.CombinedOutput()
		if err != nil {
			l.Error(err, "Failed to get kubeconfig using Azure CLI", "output", string(result))
			return nil, err
		}

		cmd = exec.Command("kubelogin", "convert-kubeconfig", "-l", "azurecli")
		l.V(1).Info("kubelogin", "cmd", cmd)
		result, err = cmd.CombinedOutput()
		if err != nil {
			l.Error(err, "Failed to kubelogin", "output", string(result))
			return nil, err
		}
	}

	clientset, err := k8smanagers_utils.GetClientSet(ctx, kubeconfigpath)
	if err != nil {
		l.Error(err, "Cannot GetClientSet")
		return nil, err
	}
	r.clientset = clientset

	return r.clientset, nil

}

// validate will check the contents of the Workload Manager configuration
func (r *WorkloadManagerReconciler) validate(ctx context.Context, wlManager *k8smanagersv1.WorkloadManager) error {
	clientset, err := r.getClientSet(ctx, wlManager)
	if err != nil {
		return err
	}

	for _, procedure := range wlManager.Spec.Procedures {
		if procedure.Type == k8smanagersv1.StatefulSet {
			err = r.validateProcedures(ctx, clientset, procedure, k8smanagersv1.StatefulSet)
		}

		if procedure.Type == k8smanagersv1.Deployment {
			err = r.validateProcedures(ctx, clientset, procedure, k8smanagersv1.Deployment)
		}
	}

	return nil
}

func (r *WorkloadManagerReconciler) validateProcedures(ctx context.Context, clientset *kubernetes.Clientset, procedure k8smanagersv1.Procedure, wlType string) error {
	l := log.Log

	for _, workload := range procedure.Workloads {

		var affinity *v1.NodeAffinity

		if wlType == k8smanagersv1.StatefulSet {
			statefulset, err := clientset.AppsV1().StatefulSets(procedure.Namespace).Get(ctx, workload, metav1.GetOptions{})
			if err != nil {
				l.Error(err, "Stateful not found", "namespace", procedure.Namespace, "name", workload)
				return err
			}
			affinity = statefulset.Spec.Template.Spec.Affinity.NodeAffinity
		}
		if wlType == k8smanagersv1.Deployment {
			deployment, err := clientset.AppsV1().Deployments(procedure.Namespace).Get(ctx, workload, metav1.GetOptions{})
			if err != nil {
				l.Error(err, "Deployment not found", "namespace", procedure.Namespace, "name", workload)
				return err
			}
			affinity = deployment.Spec.Template.Spec.Affinity.NodeAffinity
		}

		err := r.checkNodeAffinity(affinity, procedure, workload)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *WorkloadManagerReconciler) checkNodeAffinity(affinity *v1.NodeAffinity, procedure k8smanagersv1.Procedure, wlName string) error {
	l := log.Log

	if affinity == nil {
		err := errors.New("could not find any any node affinity")
		l.Error(err, "Validation failed")
		return err
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

func (r *WorkloadManagerReconciler) apply(ctx context.Context, wlManager *k8smanagersv1.WorkloadManager) error {
	l := log.Log

	clientset, err := r.getClientSet(ctx, wlManager)
	if err != nil {
		return err
	}

	for _, procedure := range wlManager.Spec.Procedures {

		if wlManager.Spec.TestMode {
			l.Info("TEST MODE: The controller will try to set the affinity", "workloads", procedure.Workloads, "key", procedure.Affinity.Key, "from", procedure.Affinity.Initial, "to", procedure.Affinity.Target)
			continue
		}

		if procedure.Type == k8smanagersv1.StatefulSet {
			err = r.updateAffinity(ctx, clientset, procedure, k8smanagersv1.StatefulSet)
		}

		if procedure.Type == k8smanagersv1.Deployment {
			err = r.updateAffinity(ctx, clientset, procedure, k8smanagersv1.Deployment)
		}

		if err != nil {
			return err
		}
	}

	return nil
}

func (r *WorkloadManagerReconciler) updateAffinity(ctx context.Context, clientset *kubernetes.Clientset, procedure k8smanagersv1.Procedure, wlType string) error {
	l := log.Log

	var deployment *appsv1.Deployment
	var statefulset *appsv1.StatefulSet
	var err error
	var interval time.Duration

	for _, workload := range procedure.Workloads {
		if wlType == k8smanagersv1.StatefulSet {
			statefulset, err = clientset.AppsV1().StatefulSets(procedure.Namespace).Get(ctx, workload, metav1.GetOptions{})
			if err != nil {
				l.Error(err, "Stateful not found", "namespace", procedure.Namespace, "name", workload)
				return err
			}

			statefulset.Spec.Template.Spec.Affinity.NodeAffinity = r.createNodeAffinity(procedure.Affinity.Key, procedure.Affinity.Target)

			_, err = clientset.AppsV1().StatefulSets(procedure.Namespace).Update(ctx, statefulset, metav1.UpdateOptions{})
			if err != nil {
				l.Error(err, "Error updating statefulset", "namespace", procedure.Namespace, "name", workload)
				return err
			}
			interval = 30 * time.Second
			time.Sleep(30 * time.Second) // Pause to allow affinity injection to take
		}
		if wlType == k8smanagersv1.Deployment {
			deployment, err = clientset.AppsV1().Deployments(procedure.Namespace).Get(ctx, workload, metav1.GetOptions{})
			if err != nil {
				l.Error(err, "Deployment not found", "namespace", procedure.Namespace, "name", workload)
				return err
			}
			deployment.Spec.Template.Spec.Affinity.NodeAffinity = r.createNodeAffinity(procedure.Affinity.Key, procedure.Affinity.Target)

			deployment, err = clientset.AppsV1().Deployments(procedure.Namespace).Update(ctx, deployment, metav1.UpdateOptions{})
			if err != nil {
				l.Error(err, "Error updating deployment", "namespace", procedure.Namespace, "name", workload)
				return err
			}
			time.Sleep(5 * time.Second) // Pause to allow affinity injection to take
			interval = 5 * time.Second
		}

		ctx = context.WithValue(ctx, "namespace", procedure.Namespace)
		ctx = context.WithValue(ctx, "clientset", clientset)
		if wlType == k8smanagersv1.Deployment {
			ctx = context.WithValue(ctx, "resource", deployment)
		}
		if wlType == k8smanagersv1.StatefulSet {
			ctx = context.WithValue(ctx, "resource", statefulset)
		}

		if procedure.Timeout == 0 {
			procedure.Timeout = 600
		}
		timeout := time.Duration(procedure.Timeout) * time.Second
		l.Info("Starting to wait", "name", workload, "timeout", timeout)
		waitForConditionWithTimeout(func() bool {
			return isResourceReady(ctx, wlType)
		}, interval, timeout)
	}

	return nil
}

func (r *WorkloadManagerReconciler) createNodeAffinity(key string, value string) *v1.NodeAffinity {
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

func isResourceReady(ctx context.Context, wlType string) bool {
	namespace := ctx.Value("namespace").(string)
	clientset := ctx.Value("clientset").(*kubernetes.Clientset)

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

func isDeploymentReady(clientset *kubernetes.Clientset, namespace string, deployment *appsv1.Deployment) bool {
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
			l.Info("Pod started.", "name", deployment.Name)
			return true
		}
	}

	return false
}

func isStatefulSetReady(clientset *kubernetes.Clientset, namespace string, statefulset *appsv1.StatefulSet) bool {
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

func getPodFromLabel(clientset *kubernetes.Clientset, namespace string, labelSelector string) (*v1.PodList, error) {
	// List the pods matching the label selector
	pods, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, err
	}
	return pods, nil
}
func waitForConditionWithTimeout(condFunc func() bool, interval, timeout time.Duration) bool {
	l := log.Log

	timeoutChan := time.After(timeout) // Set the timeout period
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-timeoutChan:
			l.Info("Waiting time exceeded", "timeout", timeout.String())
			return false
		case <-ticker.C:
			if condFunc() {
				return true
			}
		}
	}
}

// +kubebuilder:rbac:groups=k8smanagers.greyridge.com,resources=workloadmanagers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=k8smanagers.greyridge.com,resources=workloadmanagers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=k8smanagers.greyridge.com,resources=workloadmanagers/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// the WorkloadManager object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.0/pkg/reconcile
func (r *WorkloadManagerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := log.Log
	l.Info("Enter Reconcile")

	var wlManager k8smanagersv1.WorkloadManager

	// Defaults
	wlManager.Spec.SPNLoginType = k8smanagersv1.ListClusterAdminCredentials

	if err := r.Get(ctx, req.NamespacedName, &wlManager); err != nil {
		panic(err.Error())
	}

	requeue := wlManager.Spec.RetryOnError

	if err := r.validate(ctx, &wlManager); err != nil {
		l.Error(err, "Error during validate")
		return ctrl.Result{Requeue: requeue}, nil
	}

	if err := r.apply(ctx, &wlManager); err != nil {
		l.Error(err, "Error during apply")
		return ctrl.Result{Requeue: requeue}, nil
	}

	l.Info("Exit Reconcile")
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *WorkloadManagerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8smanagersv1.WorkloadManager{}).
		Complete(r)
}
