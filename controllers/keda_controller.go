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
	"errors"
	"time"

	"github.com/kyma-project/keda-manager/api/v1alpha1"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/api/core/v1"
	apixtv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/pointer"
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

func NewKedaReconciler(c client.Client, log *zap.SugaredLogger, o []unstructured.Unstructured) KedaReconciler {
	return &kedaReconciler{
		fn:  sFnInitialize,
		log: log,
		cfg: cfg{
			finalizer: "keda-manager.kyma-project.io/deletion-hook",
			objs:      o,
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
	for _, obj := range r.objs {
		if obj.GetObjectKind().GroupVersionKind().Kind == "CustomResourceDefinition" {
			continue
		}

		r.log.With("objName", obj.GetName()).
			With("gvk", obj.GroupVersionKind()).
			Debug("deleting")

		if out.err = r.client.Delete(ctx, &obj); client.IgnoreNotFound(out.err) != nil {
			r.log.Error(out.err)
		}
	}

	if out.err != nil {
		s.instance.Status.State = "Error"
		// stop state machine
		return nil
	}

	return sFnRemoveFinalizer
}

func sFnInitialize(ctx context.Context, r *reconciler, s *systemState, out *out) stateFn {
	instanceIsBeingDeleted := !s.instance.GetDeletionTimestamp().IsZero()
	instanceHasFinalizer := controllerutil.ContainsFinalizer(&s.instance, r.finalizer)

	// in case instance does not have finalizer - add it and update instance
	if !instanceIsBeingDeleted && !instanceHasFinalizer {
		r.log.Debug("adding finalizer")
		controllerutil.AddFinalizer(&s.instance, r.finalizer)
		out.err = r.client.Update(ctx, &s.instance)
		// stop state machine with potential error
		return nil
	}
	// in case instance has no finalizer and instance is being deleted - end reconciliation
	if instanceIsBeingDeleted && !controllerutil.ContainsFinalizer(&s.instance, r.finalizer) {
		r.log.Debug("instance is being deleted")
		// stop state machine
		return nil
	}
	// in case instance is being deleted and has finalizer - delete all resources
	if instanceIsBeingDeleted {
		return sFnDeleteResources
	}

	return sFnApply
}

func sFnVerify(ctx context.Context, r *reconciler, s *systemState, out *out) stateFn {
	var count int
	for _, obj := range s.objs {
		if obj.GetKind() != "Deployment" {
			continue
		}

		var deployment appsv1.Deployment
		if out.err = runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &deployment); out.err != nil {
			return nil
		}

		for _, cond := range deployment.Status.Conditions {
			if cond.Type == appsv1.DeploymentAvailable && cond.Status == v1.ConditionTrue {
				count++
			}
		}
	}

	if count == 2 {
		s.instance.Status.State = "Ready"
		out.result.RequeueAfter = time.Second * 60
	} else {
		s.instance.Status.State = "Pending"
		out.result.RequeueAfter = time.Second * 5
	}

	condition := cHelper.Installed().True(v1alpha1.ConditionReasonVerification, "verification started")
	meta.SetStatusCondition(&s.instance.Status.Conditions, condition)

	out.err = r.client.Status().Update(ctx, &s.instance)
	return nil
}

func sFnApply(ctx context.Context, r *reconciler, s *systemState, out *out) stateFn {
	var isError bool
	for _, obj := range r.objs {
		isCRD := obj.GetKind() == "CustomResourceDefinition" && obj.GetAPIVersion() == "apiextensions.k8s.io/v1"
		applyFn := applyObj

		if isCRD {
			applyFn = applyCRD
		}

		r.log.With("gvk", obj.GetObjectKind().GroupVersionKind()).
			With("objKey", client.ObjectKeyFromObject(&obj)).
			Debug("applying")

		if err := applyFn(ctx, r.client, r.log, &obj); err != nil {
			r.log.With("err", err).Debug("apply result")
			isError = true
		}

		s.objs = append(s.objs, obj)
	}
	// no errors
	if !isError {
		return sFnVerify
	}

	out.err = errors.New("installation error")
	newCondition := cHelper.Installed().False(v1alpha1.ConditionReasonCrdError, out.err.Error())
	meta.SetStatusCondition(&s.instance.Status.Conditions, newCondition)

	out.err = r.client.Status().Update(ctx, &s.instance)
	out.result.RequeueAfter = 30 * time.Second
	return nil
}

func applyObj(ctx context.Context, c client.Client, log *zap.SugaredLogger, obj *unstructured.Unstructured) error {
	err := c.Patch(ctx, obj, client.Apply, &client.PatchOptions{
		Force:        pointer.Bool(true),
		FieldManager: "m00g3n",
	})
	return err
}

func applyCRD(ctx context.Context, c client.Client, log *zap.SugaredLogger, crd *unstructured.Unstructured) error {
	var freshCRD apixtv1.CustomResourceDefinition
	keyObj := client.ObjectKeyFromObject(crd)
	// check if CRD is already applied
	err := c.Get(ctx, keyObj, &freshCRD)
	// crd exists - continue with crds installation
	if err == nil {
		log.Debug("CRD already exists")
		return nil
	}
	// error while getting crd
	if client.IgnoreNotFound(err) != nil {
		return err
	}
	// crd does not exit - create it
	return c.Create(ctx, crd)
}

func (r *kedaReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var instance v1alpha1.Keda
	if err := r.client.Get(ctx, req.NamespacedName, &instance); err != nil {
		return ctrl.Result{
			RequeueAfter: time.Second * 5,
		}, client.IgnoreNotFound(err)
	}

	reconciler := reconciler{
		fn:  r.fn,
		log: r.log,
		k8s: k8s{
			client: r.client,
		},
		cfg: r.cfg,
	}
	return reconciler.reconcile(ctx, instance)
}

func sFnApplyObj(ctx context.Context, r *reconciler, s *systemState, out *out) stateFn {
	r.log.Info("not implemented yet")
	return nil
}
