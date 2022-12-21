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
	"time"

	"github.com/kyma-project/keda-manager/api/v1alpha1"
	"go.uber.org/zap"
	apixtv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type KedaReconciler interface {
	reconcile.Reconciler
	SetupWithManager(mgr ctrl.Manager) error
}

// kedaReconciler reconciles a Keda object
type kedaReconciler struct {
	fn  stateFn
	log *zap.SugaredLogger
	cfg
	k8s
}

type K8sObjects struct {
	CRDs []unstructured.Unstructured
}

func NewKedaReconciler(c client.Client, log *zap.SugaredLogger, o K8sObjects) KedaReconciler {
	return &kedaReconciler{
		fn:  sFnInitialize,
		log: log,
		cfg: cfg{
			finalizer: "keda-manager.kyma-project.io/deletion-hook",
			crds:      o.CRDs,
		},
		k8s: k8s{
			client: c,
		},
	}
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
func (r *kedaReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Keda{}).
		Complete(r)
}

func sFnRemoveFinalizer(ctx context.Context, r *reconciler, s *systemState, out *out) stateFn {
	controllerutil.RemoveFinalizer(&s.instance, r.finalizer)
	if out.err = r.client.Update(ctx, &s.instance); out.err != nil {
		// stop state machine
		return nil
	}

	r.log.Debug("finalizer removed")
	return nil
}

func sFnDeleteResources(ctx context.Context, r *reconciler, s *systemState, out *out) stateFn {
	r.log.Debug("nothing to remove")
	return sFnRemoveFinalizer
}

func sFnInitialize(ctx context.Context, r *reconciler, s *systemState, out *out) stateFn {
	instanceIsBeingDeleted := !s.instance.GetDeletionTimestamp().IsZero()
	instanceHasFinalizer := controllerutil.ContainsFinalizer(&s.instance, r.finalizer)

	// in case instance does not have finalizer - add it and update instance
	if !instanceIsBeingDeleted && !instanceHasFinalizer {
		controllerutil.AddFinalizer(&s.instance, r.finalizer)
		out.err = r.client.Update(ctx, &s.instance)
		// stop state machine with potential error
		return nil
	}

	// in case instance has no finalizer and instance is being deleted - end reconciliation
	if instanceIsBeingDeleted && !controllerutil.ContainsFinalizer(&s.instance, r.finalizer) {
		// stop state machine
		return nil
	}

	// in case instance is being deleted and has finalizer - delete all resources
	if instanceIsBeingDeleted {
		return sFnDeleteResources
	}

	return sFnApplyCRDs
}

func sFnApplyCRDs(ctx context.Context, r *reconciler, s *systemState, out *out) stateFn {
	var applied bool
	applied, out.err = applyCRDs(ctx, r.client, r.cfg.crds)

	if out.err != nil {
		newCondition := cHelper.Installed().False(v1alpha1.ConditionReasonCrdError, out.err.Error())
		meta.SetStatusCondition(&s.instance.Status.Conditions, newCondition)

		if err := r.client.Status().Update(ctx, &s.instance); err != nil {
			r.log.Warn("unable to change state")
		}
		return nil
	}
	// all CRDs already exist - goto applyObj
	if !applied {
		return sFnApplyObj
	}
	// all CRDs applied
	return nil
}

func applyCRDs(ctx context.Context, c client.Client, crds []unstructured.Unstructured) (bool, error) {
	var installed bool

	for _, obj := range crds {
		var crd apixtv1.CustomResourceDefinition
		keyObj := client.ObjectKeyFromObject(&obj)

		err := c.Get(ctx, keyObj, &crd)

		// error while getting crd
		if client.IgnoreNotFound(err) != nil {
			return false, err
		}

		// crd exists - continue with crds installation
		if err == nil {
			continue
		}

		// crd does not exit - create it
		if err = c.Create(ctx, &obj); err != nil {
			return false, err
		}

		installed = true
	}

	return installed, nil
}

func (r *kedaReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var instance v1alpha1.Keda
	if err := r.client.Get(ctx, req.NamespacedName, &instance); err != nil {
		return ctrl.Result{
			RequeueAfter: time.Second * 30,
		}, client.IgnoreNotFound(err)
	}

	reconciler := reconciler{
		fn:  r.fn,
		log: r.log,
		k8s: k8s{
			client: r.client,
		},
		cfg: cfg{
			finalizer: r.finalizer,
			crds:      r.crds,
		},
	}
	return reconciler.reconcile(ctx, instance)
}
