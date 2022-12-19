/*
Copyright 2022.

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

package controllers

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"

	"github.com/kyma-project/keda-manager/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	chartNs = "kyma-system"
)

// KedaReconciler reconciles a Keda object
type KedaReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	*rest.Config
}

//+kubebuilder:rbac:groups="*",resources="*",verbs=get
//+kubebuilder:rbac:groups=external.metrics.k8s.io,resources="*",verbs="*"
//+kubebuilder:rbac:groups="",resources=configmaps;configmaps/status;events;services,verbs="*"
//+kubebuilder:rbac:groups="",resources=external;pods;secrets;serviceaccounts,verbs=list;watch;create;delete;update;patch
//+kubebuilder:rbac:groups="",resources=namespaces,verbs=create;delete
//+kubebuilder:rbac:groups=apiregistration.k8s.io,resources=apiservices,verbs=create;delete;update;patch
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterrolebindings;clusterroles;rolebindings,verbs=create;delete;update;patch
//+kubebuilder:rbac:groups=apiextensions.k8s.io,resources=customresourcedefinitions,verbs=create;delete;update;patch
//+kubebuilder:rbac:groups="*",resources="*/scale",verbs="*"
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=list;watch;create;delete;update;patch
//+kubebuilder:rbac:groups=apps,resources=statefulsets;replicasets,verbs=list;watch
//+kubebuilder:rbac:groups=batch,resources=jobs,verbs="*"
//+kubebuilder:rbac:groups=coordination.k8s.io,resources=leases,verbs="*"
//+kubebuilder:rbac:groups="keda.sh",resources=clustertriggerauthentications;clustertriggerauthentications/status;scaledjobs;scaledjobs/finalizers;scaledjobs/status;scaledobjects;scaledobjects/finalizers;scaledobjects/status;triggerauthentications;triggerauthentications/status,verbs="*"
//+kubebuilder:rbac:groups=autoscaling,resources=horizontalpodautoscalers,verbs="*"

//+kubebuilder:rbac:groups=operator.kyma-project.io,resources=kedas,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=operator.kyma-project.io,resources=kedas/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=operator.kyma-project.io,resources=kedas/finalizers,verbs=update;patch

// SetupWithManager sets up the controller with the Manager.
func (r *KedaReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Config = mgr.GetConfig()

	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Keda{}).
		Complete(r)
}

type shouldContinue = bool

type ReconciliationAction func(context.Context, client.Client, v1alpha1.Keda) (shouldContinue, ctrl.Result, error)

type ReconciliationActions []ReconciliationAction

func (a ReconciliationActions) reconcileAll(ctx context.Context, c client.Client, v v1alpha1.Keda) (ctrl.Result, error) {
	for _, f := range a {
		shouldContinue, result, err := f(ctx, c, v)

		if !shouldContinue {
			return result, err
		}

		if err != nil {
			return defaultResult, err
		}
	}
	return ctrl.Result{RequeueAfter: time.Minute}, nil
}

func deleteFinalizer(ctx context.Context, c client.Client, v v1alpha1.Keda) (shouldContinue, ctrl.Result, error) {
	controllerutil.RemoveFinalizer(&v, finalizer)
	if err := c.Update(ctx, &v); err != nil {
		return false, defaultResult, err
	}
	fmt.Println("finalizer removed")
	return false, defaultResult, nil
}

var ErrMultipleInstancesInNamespace = fmt.Errorf("namespace must not contain multiple module instances")

func buildStopAndError(err error) ReconciliationAction {
	return func(ctx context.Context, c client.Client, v v1alpha1.Keda) (shouldContinue, ctrl.Result, error) {
		v.Status.State = "Error"
		meta.SetStatusCondition(&v.Status.Conditions, metav1.Condition{
			Type:               "Installed",
			Status:             "False",
			LastTransitionTime: metav1.Now(),
			Message:            "instance of a module already found on cluster",
			Reason:             "CrdCreationErr",
		})

		if err := c.Status().Update(ctx, &v); err != nil {
			fmt.Println(err)
		}

		return false, ctrl.Result{
			RequeueAfter: time.Minute,
		}, err
	}
}

func isNamespaceSingleton(ctx context.Context, c client.Client, v v1alpha1.Keda) (shouldContinue, ctrl.Result, error) {
	var list v1alpha1.KedaList
	err := c.List(ctx, &list, &client.ListOptions{})
	// stop reconciliation on any error
	if err != nil {
		return buildStopAndError(err)(ctx, c, v)
	}

	if len(list.Items) > 1 {
		return buildStopAndError(err)(ctx, c, v)
	}

	return true, defaultResult, nil
}

func defaultConditionsForFirstGeneration(ctx context.Context, c client.Client, v v1alpha1.Keda) (shouldContinue, ctrl.Result, error) {
	if len(v.Status.Conditions) > 0 {
		return true, defaultResult, nil
	}

	fmt.Println("defaulting conditions")

	msg := "custom resource created"

	meta.SetStatusCondition(&v.Status.Conditions, metav1.Condition{
		Type:               "Installed",
		Status:             "Unknown",
		Reason:             "Created",
		Message:            msg,
		LastTransitionTime: v.GetCreationTimestamp(),
	})

	meta.SetStatusCondition(&v.Status.Conditions, metav1.Condition{
		Type:               "KedaRdy",
		Status:             "Unknown",
		Reason:             "Created",
		Message:            msg,
		LastTransitionTime: v.GetCreationTimestamp(),
	})

	meta.SetStatusCondition(&v.Status.Conditions, metav1.Condition{
		Type:               "MetricsRdy",
		Status:             "Unknown",
		Reason:             "Created",
		Message:            msg,
		LastTransitionTime: v.GetCreationTimestamp(),
	})

	v.Status.State = "Processing"

	if err := c.Status().Update(ctx, &v); err != nil {
		fmt.Println(err)
	}
	return false, defaultResult, nil
}

var (
	finalizer = "keda-manager.kyma-project.io/deletion-hook"

	defaultResult = ctrl.Result{
		Requeue: false,
	}

	kedaObjKey = client.ObjectKey{
		Name:      "keda",
		Namespace: "kyma-system",
	}

	main = ReconciliationActions{
		defaultConditionsForFirstGeneration,
		isNamespaceSingleton,
	}

	deletion = ReconciliationActions{
		deleteFinalizer,
	}
)

func (r *KedaReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var instance v1alpha1.Keda
	err := r.Get(ctx, req.NamespacedName, &instance)

	if err != nil {
		return ctrl.Result{
			RequeueAfter: time.Second * 30,
		}, client.IgnoreNotFound(err)
	}

	instanceIsBeingDeleted := !instance.DeletionTimestamp.IsZero()
	instanceHasFinalizer := controllerutil.ContainsFinalizer(&instance, finalizer)

	// in case instance does not have finalizer - add it and update instance
	if !instanceIsBeingDeleted && !instanceHasFinalizer {
		controllerutil.AddFinalizer(&instance, finalizer)
		if err := r.Update(ctx, &instance); err != nil {
			return defaultResult, err
		}
		return defaultResult, nil
	}

	// in case instance has no finalizer and instance is being deleted - end reconciliation
	if instanceIsBeingDeleted && !controllerutil.ContainsFinalizer(&instance, finalizer) {
		return defaultResult, nil
	}

	// in case instance is being deleted and has finalizer - delete all resources
	if instanceIsBeingDeleted {
		return deletion.reconcileAll(ctx, r.Client, instance)
	}

	return main.reconcileAll(ctx, r.Client, instance)
}
