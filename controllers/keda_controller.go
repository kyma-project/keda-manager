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
	"reflect"
	"time"

	"github.com/kyma-project/keda-manager/api/v1alpha1"
	"github.com/kyma-project/keda-manager/pkg/crypto/sha256"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
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

func (r *kedaReconciler) mapFunction(object client.Object) []reconcile.Request {
	var kedas v1alpha1.KedaList
	err := r.client.List(context.Background(), &kedas)

	if apierrors.IsNotFound(err) {
		return nil
	}

	if err != nil {
		r.log.Error(err)
		return nil
	}

	if len(kedas.Items) < 1 {
		return nil
	}

	// instance is being deleted, do not notify it about changes
	instanceIsBeingDeleted := !kedas.Items[0].GetDeletionTimestamp().IsZero()
	if instanceIsBeingDeleted {
		return nil
	}

	r.log.With("gen", object.GetGeneration()).
		With("rscVer", object.GetResourceVersion()).
		With("name", object.GetName()).
		With("gvk", object.GetObjectKind().GroupVersionKind()).
		With("kedaRscVer", kedas.Items[0].ResourceVersion).
		Debug("redirecting")

	// make sure only 1 controller will handle change
	return []ctrl.Request{
		{
			NamespacedName: types.NamespacedName{
				Namespace: object.GetNamespace(),
				Name:      kedas.Items[0].Name,
			},
		},
	}
}

var ommitStatusChanged = predicate.Or(
	predicate.LabelChangedPredicate{},
	predicate.AnnotationChangedPredicate{},
	predicate.GenerationChangedPredicate{},
)

func (r *kedaReconciler) watchAll(b *builder.Builder, objs []unstructured.Unstructured, p predicate.Predicate) error {
	visited := map[string]struct{}{}

	for _, obj := range objs {
		u := unstructured.Unstructured{}
		u.SetGroupVersionKind(schema.GroupVersionKind{
			Kind:    obj.GetKind(),
			Group:   obj.GetObjectKind().GroupVersionKind().Group,
			Version: obj.GetObjectKind().GroupVersionKind().Version,
		})
		u.SetName(obj.GetName())

		shaStr, err := sha256.DefaultWriterSumerBuilder.CalculateSHA256(u)
		if err != nil {
			return err
		}

		if _, found := visited[shaStr]; found {
			continue
		}

		r.log.With("obj", u).Debug("watching")

		b = b.Watches(
			&source.Kind{Type: &u},
			handler.EnqueueRequestsFromMapFunc(r.mapFunction),
			builder.WithPredicates(
				predicate.And(
					predicate.ResourceVersionChangedPredicate{},
					p,
				),
			),
		)

		visited[shaStr] = struct{}{}
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *kedaReconciler) SetupWithManager(mgr ctrl.Manager) error {
	labelSelectorPredicate, err := predicate.LabelSelectorPredicate(
		metav1.LabelSelector{
			MatchLabels: map[string]string{
				"app.kubernetes.io/name": "keda-manager",
			},
		},
	)
	if err != nil {
		return err
	}

	b := ctrl.NewControllerManagedBy(mgr).For(&v1alpha1.Keda{}, builder.WithPredicates(predicate.GenerationChangedPredicate{}))

	if err := r.watchAll(b, r.objs, labelSelectorPredicate); err != nil {
		return err
	}

	return b.Complete(r)
}

func (r *kedaReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var instance v1alpha1.Keda
	if err := r.client.Get(ctx, req.NamespacedName, &instance); err != nil {
		return ctrl.Result{
			RequeueAfter: time.Minute,
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

	err := r.client.Update(ctx, &s.instance)
	if client.IgnoreNotFound(err) != nil {
		return nil, &ctrl.Result{RequeueAfter: time.Second}, err
	}

	r.log.Debug("finalizer removed")
	return stop()
}

func sFnDeleteResources(ctx context.Context, r *reconciler, s *systemState) (stateFn, *ctrl.Result, error) {
	// TODO add CR cleanup before objs cleanup
	// TODO reconcile - event base (on edit) - watch, label
	var err error
	for _, obj := range r.objs {
		r.log.
			With("objName", obj.GetName()).
			With("gvk", obj.GroupVersionKind()).
			Debug("deleting")

		err = r.client.Delete(ctx, &obj)
		err = client.IgnoreNotFound(err)

		if err != nil {
			r.log.With("deleting resource").Error(err)
		}
	}

	if err != nil {
		s.instance.Status.State = "Error"
		if err := r.client.Status().Update(ctx, &s.instance); err != nil {
			r.log.Error(err)
		}
		return nil, &ctrl.Result{RequeueAfter: time.Second}, err
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
		if err != nil {
			return stopWithError(err)
		}

		instance := s.instance.DeepCopy()
		instance.Status.State = "Initialized"
		err = r.k8s.client.Status().Update(ctx, instance)

		// stop state machine with potential error
		return nil, &ctrl.Result{RequeueAfter: time.Second}, err
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

	result := ctrl.Result{}

	instance := s.instance.DeepCopy()
	if count == 2 {
		if instance.Status.State == "Ready" {
			return nil, nil, nil
		}

		instance.Status.State = "Ready"
		r.log.Debug("deployment rdy")
	} else {
		if instance.Status.State == "Ready" {
			return nil, nil, nil
		}

		r.log.With("status", instance.Status).Debug("current status")
		instance.Status.State = "Processing"
		//condition := cHelper.Installed().True(v1alpha1.ConditionReasonVerification, "verification started")
		//meta.SetStatusCondition(&instance.Status.Conditions, condition)
	}

	r.log.With("rscVersion", instance.ResourceVersion).
		With("generation", instance.Generation).
		Debug("changing status")

	err := r.client.Status().Update(ctx, instance)
	if err != nil {
		result = ctrl.Result{
			RequeueAfter: time.Second,
		}
	}
	return nil, &result, err
}

func sFnApply(ctx context.Context, r *reconciler, s *systemState) (stateFn, *ctrl.Result, error) {
	var isError bool
	for _, obj := range r.objs {
		r.log.With("gvk", obj.GetObjectKind().GroupVersionKind()).
			With("name", obj.GetName()).
			Debug("applying")

		err := r.client.Patch(ctx, &obj, client.Apply, &client.PatchOptions{
			Force:        pointer.Bool(true),
			FieldManager: "m00g3n",
		})

		if err != nil {
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
	instance := s.instance.DeepCopy()
	//newCondition := cHelper.Installed().False(v1alpha1.ConditionReasonCrdError, err.Error())
	//meta.SetStatusCondition(&s.instance.Status.Conditions, newCondition)

	if reflect.DeepEqual(s.instance.Status, instance.Status) {
		return nil, nil, nil
	}

	if err := r.client.Status().Update(ctx, instance); err != nil {
		r.log.Error(err)
	}

	return nil, &ctrl.Result{RequeueAfter: time.Second}, err
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
