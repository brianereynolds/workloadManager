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

//     "sigs.k8s.io/controller-runtime/pkg/client"
// "github.com/davecgh/go-spew/spew"

package controller

import (
	"context"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap/zapcore"
	k8smanagersv1 "greyridge.com/workloadManager/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"time"
)

var _ = Describe("WorkloadManager Controller", func() {
	Context("When reconciling a resource", func() {
		ctx := context.Background()

		zapLogger := zap.New(zap.UseDevMode(true), zap.Level(zapcore.DebugLevel))
		log.SetLogger(zapLogger)

		var deployment *appsv1.Deployment
		var resource *k8smanagersv1.WorkloadManager

		BeforeEach(func() {
			deployment = newDeployment()
			resource = newResource()
		})

		AfterEach(func() {
			_ = k8sClient.Delete(ctx, deployment)
			_ = k8sClient.Delete(ctx, resource)

			time.Sleep(5 * time.Second)
		})

		It("should successfully reconcile the resource with default config", func() {
			Expect(createResource(resource)).To(Succeed())

			Expect(triggerReconcile(ctx, k8sClient, typeNamespacedName)).To(Succeed())
		})

		It("Test affinity on Deployment", func() {
			procedure := k8smanagersv1.Procedure{
				Type:      "deployment",
				Namespace: "default",
				Workloads: []string{deployment.Name},
				Affinity: k8smanagersv1.Affinity{
					Key:     "agentpool",
					Initial: "initialaffinity",
					Target:  "targetaffinity",
				},
				Timeout: 5,
			}

			resource.Spec.Procedures = append(resource.Spec.Procedures, procedure)
			Expect(createResource(resource)).To(Succeed())

			nodeAffinity := &corev1.NodeAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
					NodeSelectorTerms: []corev1.NodeSelectorTerm{
						{
							MatchExpressions: []corev1.NodeSelectorRequirement{
								{
									Key:      "agentpool",
									Operator: corev1.NodeSelectorOpIn,
									Values:   []string{"initialaffinity"},
								},
							},
						},
					},
				},
			}

			affinity := &corev1.Affinity{
				NodeAffinity: nodeAffinity,
			}

			deployment.Spec.Template.Spec.Affinity = affinity

			Expect(createDeployment(deployment)).To(Succeed())

			Expect(triggerReconcile(ctx, k8sClient, typeNamespacedName)).To(Succeed())

			// Get deployment and check affinity, it should be updated with the target affinity
			actualDeployment := &appsv1.Deployment{}

			_ = k8sClient.Get(ctx, client.ObjectKey{
				Namespace: deployment.ObjectMeta.Namespace,
				Name:      deployment.ObjectMeta.Name,
			}, actualDeployment)

			actualAffinity := actualDeployment.Spec.Template.Spec.Affinity.NodeAffinity
			Expect(actualAffinity).NotTo(BeNil())
			Expect(actualAffinity.RequiredDuringSchedulingIgnoredDuringExecution).NotTo(BeNil())
			Expect(actualAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms).To(HaveLen(1))
			Expect(actualAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions).To(HaveLen(1))
			Expect(actualAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[0].Key).To(Equal("agentpool"))
			Expect(actualAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[0].Operator).To(Equal(corev1.NodeSelectorOpIn))
			Expect(actualAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[0].Values).To(ContainElement("targetaffinity"))
		})

		It("Test selector on Deployment ", func() {
			procedure := k8smanagersv1.Procedure{
				Type:      "deployment",
				Namespace: "default",
				Workloads: []string{deployment.Name},
				Selector: k8smanagersv1.Selector{
					Key:     "pasx/node",
					Initial: "miscblue",
					Target:  "miscgreen",
				},
				Timeout: 5,
			}

			resource.Spec.Procedures = append(resource.Spec.Procedures, procedure)
			Expect(createResource(resource)).To(Succeed())

			nodeSelector := map[string]string{
				"pasx/node": "miscblue",
			}

			deployment.Spec.Template.Spec.NodeSelector = nodeSelector

			Expect(createDeployment(deployment)).To(Succeed())

			Expect(triggerReconcile(ctx, k8sClient, typeNamespacedName)).To(Succeed())

			// Get deployment and check node selector, it should be updated with the target selector
			actualDeployment := &appsv1.Deployment{}

			_ = k8sClient.Get(ctx, client.ObjectKey{
				Namespace: deployment.ObjectMeta.Namespace,
				Name:      deployment.ObjectMeta.Name,
			}, actualDeployment)

			actualNodeSelector := actualDeployment.Spec.Template.Spec.NodeSelector
			Expect(actualNodeSelector).To(HaveKeyWithValue("pasx/node", "miscgreen"))
		})

		It("Test affinity on Statefulset", func() {}) // TODO

		It("Test affinity on Statefulset", func() {}) // TODO
	})
})

func triggerReconcile(ctx context.Context, k8sClient client.Client, typeNamespacedName types.NamespacedName) error {
	controllerReconciler := &WorkloadManagerReconciler{
		Client: k8sClient,
		Scheme: k8sClient.Scheme(),
	}

	_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
		NamespacedName: typeNamespacedName,
	})

	return err
}

func createResource(resource *k8smanagersv1.WorkloadManager) error {
	return k8sClient.Create(ctx, resource)
}

func createDeployment(deployment *appsv1.Deployment) error {
	return k8sClient.Create(ctx, deployment)

}
func int32Ptr(i int32) *int32 { return &i }
