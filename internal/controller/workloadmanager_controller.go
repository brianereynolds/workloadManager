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
	"greyridge.com/workloadManager/internal/controller/monitoring"
	"greyridge.com/workloadManager/internal/controller/scheduling"
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
	"strconv"
	"strings"
	"time"
)

// WorkloadManagerReconciler reconciles a WorkloadManager object
type WorkloadManagerReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	clientset *kubernetes.Clientset
}

func isRunningInDocker() bool {
	return os.Getenv("container") == "docker"
}

func (r *WorkloadManagerReconciler) getClientSet(ctx context.Context, wlManager *k8smanagersv1.WorkloadManager) (*kubernetes.Clientset, error) {
	l := log.Log

	if r.clientset != nil {
		l.V(1).Info("Returning cached clientset")
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
	if isRunningInDocker() == false {
		kubeconfigpath = os.Getenv("HOME") + "/.kube/config"
	}

	if wlManager.Spec.SPNLoginType == k8smanagersv1.ListClusterUserCredentials ||
		wlManager.Spec.SPNLoginType == k8smanagersv1.ListClusterAdminCredentials {

		// We should have a kubeconfig at this point
		if kubeconfig == nil {
			err = errors.New("Login has failed using " + wlManager.Spec.SPNLoginType)
			return nil, err
		}

		_, err = k8smanagers_utils.WriteKubeFile(kubeconfig)
		if err != nil {
			l.Error(err, "cannot write kubeconfig")
			return nil, err
		}
	}

	if wlManager.Spec.SPNLoginType == k8smanagersv1.AzCli {
		azureClientId := os.Getenv("AZURE_CLIENT_ID")
		azureTenantId := os.Getenv("AZURE_TENANT_ID")
		azureClientSecret := os.Getenv("AZURE_CLIENT_SECRET")
		cmd := exec.Command("az", "login", "--service-principal",
			"--username", azureClientId,
			"--tenant", azureTenantId,
			"--password", azureClientSecret)
		if azureClientId == "" {
			return nil, errors.New("AZURE_CLIENT_ID is mandatory when using " + k8smanagersv1.AzCli)
		}
		if azureTenantId == "" {
			return nil, errors.New("AZURE_TENANT_ID is mandatory when using " + k8smanagersv1.AzCli)
		}
		if azureClientSecret == "" {
			return nil, errors.New("AZURE_CLIENT_SECRET is mandatory when using " + k8smanagersv1.AzCli)
		}

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
		var selector *metav1.LabelSelector

		if wlType == k8smanagersv1.StatefulSet {
			statefulset, err := clientset.AppsV1().StatefulSets(procedure.Namespace).Get(ctx, workload, metav1.GetOptions{})
			if err != nil {
				l.Error(err, "Stateful not found", "namespace", procedure.Namespace, "name", workload)
				return err
			}
			if scheduling.HasAffinity(statefulset) {
				affinity = statefulset.Spec.Template.Spec.Affinity.NodeAffinity
			}
			if scheduling.HasSelector(statefulset) {
				selector = statefulset.Spec.Selector
			}
		}
		if wlType == k8smanagersv1.Deployment {
			deployment, err := clientset.AppsV1().Deployments(procedure.Namespace).Get(ctx, workload, metav1.GetOptions{})
			if err != nil {
				l.Error(err, "Deployment not found", "namespace", procedure.Namespace, "name", workload)
				return err
			}

			if scheduling.HasAffinity(deployment) {
				affinity = deployment.Spec.Template.Spec.Affinity.NodeAffinity
			}
			if scheduling.HasSelector(deployment) {
				selector = deployment.Spec.Selector
			}
		}

		if affinity == nil && selector == nil {
			err := errors.New("could not find any any node affinity or node selector")
			l.Error(err, "Validation failed")
			return err
		}

		err := scheduling.CheckNodeAffinity(affinity, procedure, workload)
		if err != nil {
			return err
		}
		err = scheduling.CheckNodeSelector(selector, procedure, workload)
		if err != nil {
			return err
		}
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
			err = r.updateScheduling(ctx, clientset, procedure, k8smanagersv1.StatefulSet)
		}

		if procedure.Type == k8smanagersv1.Deployment {
			err = r.updateScheduling(ctx, clientset, procedure, k8smanagersv1.Deployment)
		}

		if err != nil {
			return err
		}
	}

	return nil
}

func (r *WorkloadManagerReconciler) updateScheduling(ctx context.Context, clientset *kubernetes.Clientset, procedure k8smanagersv1.Procedure, wlType string) error {
	l := log.Log

	var deployment *appsv1.Deployment
	var statefulset *appsv1.StatefulSet
	var err error
	var interval time.Duration

	if procedure.Timeout == 0 {
		procedure.Timeout = 600
	}

	for _, workload := range procedure.Workloads {
		if wlType == k8smanagersv1.StatefulSet {
			statefulset, err = clientset.AppsV1().StatefulSets(procedure.Namespace).Get(ctx, workload, metav1.GetOptions{})
			if err != nil {
				l.Error(err, "Stateful not found", "namespace", procedure.Namespace, "name", workload)
				return err
			}

			if scheduling.HasAffinity(statefulset) {
				l.V(1).Info("Statefulset has Affinity", "Key", procedure.Affinity.Key, "Target", procedure.Affinity.Target)
				statefulset.Spec.Template.Spec.Affinity.NodeAffinity = scheduling.CreateNodeAffinity(procedure.Affinity.Key, procedure.Affinity.Target)
			}
			if scheduling.HasSelector(statefulset) {
				l.V(1).Info("Statefulset has Selector", "Key", procedure.Selector.Key, "Target", procedure.Selector.Target)
				statefulset.Spec.Template.Spec.NodeSelector = scheduling.CreateNodeSelector(procedure.Selector.Key, procedure.Selector.Target)
			}

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
			if scheduling.HasAffinity(deployment) {
				l.V(1).Info("Deployment has Affinity", "Key", procedure.Affinity.Key, "Target", procedure.Affinity.Target)
				deployment.Spec.Template.Spec.Affinity.NodeAffinity = scheduling.CreateNodeAffinity(procedure.Affinity.Key, procedure.Affinity.Target)
			}
			if scheduling.HasSelector(deployment) {
				l.V(1).Info("Deployment has Selector", "Key", procedure.Selector.Key, "Target", procedure.Selector.Target)
				deployment.Spec.Template.Spec.NodeSelector = scheduling.CreateNodeSelector(procedure.Selector.Key, procedure.Selector.Target)
			}

			deployment, err = clientset.AppsV1().Deployments(procedure.Namespace).Update(ctx, deployment, metav1.UpdateOptions{})
			if err != nil {
				l.Error(err, "Error updating deployment", "namespace", procedure.Namespace, "name", workload)
				return err
			}
			if procedure.Timeout > 10 {
				time.Sleep(10 * time.Second) // Pause to allow affinity injection to take
			}
			interval = 10 * time.Second
		}

		ctx = context.WithValue(ctx, "namespace", procedure.Namespace)
		ctx = context.WithValue(ctx, "clientset", clientset)
		if wlType == k8smanagersv1.Deployment {
			ctx = context.WithValue(ctx, "resource", deployment)
		}
		if wlType == k8smanagersv1.StatefulSet {
			ctx = context.WithValue(ctx, "resource", statefulset)
		}

		timeout := time.Duration(procedure.Timeout) * time.Second
		l.Info("Starting to wait", "name", workload, "timeout", timeout)
		waitForConditionWithTimeout(func() bool {
			return monitoring.IsResourceReady(ctx, wlType)
		}, interval, timeout)
	}

	return nil
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
				ticker.Stop() // Explicitly stop ticker
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
	l.V(1).Info("Debug logging enabled!")

	var wlManager k8smanagersv1.WorkloadManager

	if err := r.Get(ctx, req.NamespacedName, &wlManager); err != nil {
		panic(err.Error())
	}

	// Defaults
	if wlManager.Spec.SPNLoginType == "" {
		l.V(1).Info("Setting default SPNLoginType " + k8smanagersv1.ListClusterAdminCredentials)
		wlManager.Spec.SPNLoginType = k8smanagersv1.ListClusterAdminCredentials
	}

	requeue := wlManager.Spec.RetryOnError
	l.V(1).Info("Retry on error " + strconv.FormatBool(wlManager.Spec.RetryOnError))

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
