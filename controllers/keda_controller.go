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

// SetupWithManager sets up the controller with the Manager.
func (r *kedaReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Keda{}).
		Complete(r)
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

func sFnRemoveFinalizer(ctx context.Context, r *reconciler, s *systemState) (stateFn, *ctrl.Result, error) {
	controllerutil.RemoveFinalizer(&s.instance, r.finalizer)
	if err := r.client.Update(ctx, &s.instance); err != nil {
		return stopWithError(err)
	}

	r.log.Debug("finalizer removed")
	return stop()
}

func sFnDeleteResources(ctx context.Context, r *reconciler, s *systemState) (stateFn, *ctrl.Result, error) {
	// TODO add CR cleanup before objs cleanup
	// TODO reconcile - event base (on edit) - watch, label
	var err error
	for _, obj := range r.objs {
		r.log.With("objName", obj.GetName()).With("gvk", obj.GroupVersionKind()).
			Debug("deleting")

		err = r.client.Delete(ctx, &obj)
		err = client.IgnoreNotFound(err)

		if err != nil {
			r.log.Error(err)
		}
	}

	if err != nil {
		s.instance.Status.State = "Error"
		return stopWithError(err)
	}

	return switchState(sFnRemoveFinalizer)
}

func sFnInitialize(ctx context.Context, r *reconciler, s *systemState) (stateFn, *ctrl.Result, error) {
	instanceIsBeingDeleted := !s.instance.GetDeletionTimestamp().IsZero()
	instanceHasFinalizer := controllerutil.ContainsFinalizer(&s.instance, r.finalizer)

	// in case instance does not have finalizer - add it and update instance
	if !instanceIsBeingDeleted && !instanceHasFinalizer {
		r.log.Debug("adding finalizer")
		controllerutil.AddFinalizer(&s.instance, r.finalizer)
		err := r.client.Update(ctx, &s.instance)
		// stop state machine with potential error
		return stopWithError(err)
	}
	// in case instance has no finalizer and instance is being deleted - end reconciliation
	if instanceIsBeingDeleted && !controllerutil.ContainsFinalizer(&s.instance, r.finalizer) {
		r.log.Debug("instance is being deleted")
		// stop state machine
		return stop()
	}
	// in case instance is being deleted and has finalizer - delete all resources
	if instanceIsBeingDeleted {
		return switchState(sFnDeleteResources)
	}

	return switchState(sFnApply)
}

func sFnVerify(ctx context.Context, r *reconciler, s *systemState) (stateFn, *ctrl.Result, error) {
	var count int
	for _, obj := range s.objs {
		if obj.GetKind() != "Deployment" {
			continue
		}

		var deployment appsv1.Deployment
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &deployment); err != nil {
			return stopWithError(err)
		}

		for _, cond := range deployment.Status.Conditions {
			if cond.Type == appsv1.DeploymentAvailable && cond.Status == v1.ConditionTrue {
				count++
			}
		}
	}

	var result ctrl.Result
	if count == 2 {
		s.instance.Status.State = "Ready"
		result.RequeueAfter = time.Second * 60
	} else {
		s.instance.Status.State = "Pending"
		result.RequeueAfter = time.Second * 5
	}

	condition := cHelper.Installed().True(v1alpha1.ConditionReasonVerification, "verification started")
	meta.SetStatusCondition(&s.instance.Status.Conditions, condition)

	err := r.client.Status().Update(ctx, &s.instance)
	return nil, &result, err
}

func sFnApply(ctx context.Context, r *reconciler, s *systemState) (stateFn, *ctrl.Result, error) {
	var isError bool
	for _, obj := range r.objs {
		r.log.With("gvk", obj.GetObjectKind().GroupVersionKind()).
			With("objKey", client.ObjectKeyFromObject(&obj)).
			Debug("applying")

		if err := applyObj(ctx, r.client, r.log, &obj); err != nil {
			r.log.With("err", err).Debug("apply result")
			isError = true
		}

		s.objs = append(s.objs, obj)
	}
	// no errors
	if !isError {
		return switchState(sFnVerify)
	}

	err := errors.New("installation error")
	newCondition := cHelper.Installed().False(v1alpha1.ConditionReasonCrdError, err.Error())
	meta.SetStatusCondition(&s.instance.Status.Conditions, newCondition)

	r.client.Status().Update(ctx, &s.instance)
	return nil, &ctrl.Result{RequeueAfter: 30 * time.Second}, err
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

func NewKedaReconciler(c client.Client, log *zap.SugaredLogger, o []unstructured.Unstructured) KedaReconciler {
	return &kedaReconciler{
		fn:  sFnInitialize,
		log: log,
		cfg: cfg{
			finalizer: v1alpha1.Finalizer,
			objs:      o,
		},
		k8s: k8s{
			client: c,
		},
	}
}
