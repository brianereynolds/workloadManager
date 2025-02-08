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
	"go.uber.org/zap/zapcore"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	k8smanagersv1 "greyridge.com/workloadManager/api/v1"
)

var _ = Describe("WorkloadManager Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		zapLogger := zap.New(zap.UseDevMode(true), zap.Level(zapcore.DebugLevel))
		log.SetLogger(zapLogger)

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default",
		}
		workloadmanager := &k8smanagersv1.WorkloadManager{}

		BeforeEach(func() {
			By("creating the custom resource for the Kind WorkloadManager")
			err := k8sClient.Get(ctx, typeNamespacedName, workloadmanager)
			if err != nil && errors.IsNotFound(err) {
				resource := &k8smanagersv1.WorkloadManager{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					Spec: k8smanagersv1.WorkloadManagerSpec{
						SubscriptionID: "3e54eb54-946e-4ff4-a430-d7b190cd45cf",
						ResourceGroup:  "node-upgrader",
						ClusterName:    "lm-cluster",
						SPNLoginType:   "azCli",
						RetryOnError:   false,
						TestMode:       false,
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			resource := &k8smanagersv1.WorkloadManager{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance WorkloadManager")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})

		It("should successfully reconcile the resource", func() {
			controllerReconciler := &WorkloadManagerReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})

			Expect(err).NotTo(HaveOccurred())
		})

		It("Validate affinity", func() {
			resource := &k8smanagersv1.WorkloadManager{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			procedure := k8smanagersv1.Procedure{
				Affinity: k8smanagersv1.Affinity{
					Key:     "agentpool",
					Initial: "initialaffinity",
					Target:  "targetaffinity",
				},
			}

			resource.Spec.Procedures = append(resource.Spec.Procedures, procedure)

			Expect(k8sClient.Update(ctx, resource)).To(Succeed())

			controllerReconciler := &WorkloadManagerReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})

			Expect(err).NotTo(HaveOccurred())
		})

		It("Validate selector", func() {
			resource := &k8smanagersv1.WorkloadManager{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			procedure := k8smanagersv1.Procedure{
				Selector: k8smanagersv1.Selector{
					Key:     "app/node",
					Initial: "initialselector",
					Target:  "initialselector",
				},
			}

			resource.Spec.Procedures = append(resource.Spec.Procedures, procedure)

			Expect(k8sClient.Update(ctx, resource)).To(Succeed())

			controllerReconciler := &WorkloadManagerReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})

			Expect(err).NotTo(HaveOccurred())
		})
	})
})
