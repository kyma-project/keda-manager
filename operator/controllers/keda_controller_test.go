package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	rtypes "github.com/kyma-project/module-manager/operator/pkg/types"

	"github.com/kyma-project/keda-manager/operator/api/v1alpha1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Keda controller", func() {
	Context("When creating fresh instance", func() {
		const (
			namespaceName = "keda"
			kedaName      = "test"
			saName        = "keda-manager"
		)

		var (
			emptyState = rtypes.State("")
		)

		It("The status should be Success", func() {

			ctx := context.Background()

			By(fmt.Sprintf("Creating namespace: %s", namespaceName))
			namespace := corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: namespaceName,
				},
			}
			Expect(k8sClient.Create(ctx, &namespace)).Should(Succeed())

			By(fmt.Sprintf("Creating crd: %s", kedaName))
			keda := v1alpha1.Keda{
				ObjectMeta: metav1.ObjectMeta{
					Name:      kedaName,
					Namespace: namespaceName,
				},
				Spec: v1alpha1.KedaSpec{},
			}

			Expect(k8sClient.Create(ctx, &keda)).Should(Succeed())

			var sa corev1.ServiceAccount
			Eventually(func() (bool, error) {
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      saName,
					Namespace: namespaceName,
				}, &sa)

				if err != nil {
					return false, err
				}

				return true, err
			}).
				WithPolling(time.Second * 2).
				WithTimeout(time.Second * 10).
				Should(BeTrue())

			labelLen := len(sa.Labels)
			Expect(labelLen).Should(Equal(7))

			Eventually(func() (rtypes.State, error) {
				key := types.NamespacedName{
					Name:      kedaName,
					Namespace: namespaceName,
				}

				var result v1alpha1.Keda
				if err := k8sClient.Get(ctx, key, &result); err != nil {
					return emptyState, err
				}

				data, err := json.MarshalIndent(&result, "", "  ")
				if err != nil {
					return emptyState, err
				}

				fmt.Fprintf(GinkgoWriter, "%s\n", string(data))

				return result.Status.State, nil
			}).
				WithPolling(time.Second * 5).
				WithTimeout(time.Second * 20).
				Should(Equal(rtypes.StateReady))
		})
	})
})
