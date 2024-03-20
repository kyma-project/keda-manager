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

	"github.com/kyma-project/keda-manager/api/v1alpha1"
	"github.com/kyma-project/keda-manager/pkg/reconciler"
	"go.uber.org/zap"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type KedaReconciler interface {
	reconcile.Reconciler
	SetupWithManager(mgr ctrl.Manager) error
}

// kedaReconciler reconciles a Keda object
type kedaReconciler struct {
	log *zap.SugaredLogger
	reconciler.Cfg
	reconciler.K8s
}

func (r *kedaReconciler) mapFunction(ctx context.Context, object client.Object) []reconcile.Request {
	var kedas v1alpha1.KedaList
	err := r.List(ctx, &kedas)

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

	r.log.
		With("name", object.GetName()).
		With("ns", object.GetNamespace()).
		With("gvk", object.GetObjectKind().GroupVersionKind()).
		With("rscVer", object.GetResourceVersion()).
		With("kedaRscVer", kedas.Items[0].ResourceVersion).
		Debug("redirecting")

	// make sure only 1 controller will handle change
	return []ctrl.Request{
		{
			NamespacedName: types.NamespacedName{
				Namespace: kedas.Items[0].Namespace,
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

// SetupWithManager sets up the controller with the Manager.
func (r *kedaReconciler) SetupWithManager(mgr ctrl.Manager) error {
	labelSelectorPredicate, err := predicate.LabelSelectorPredicate(
		metav1.LabelSelector{
			MatchLabels: map[string]string{
				"app.kubernetes.io/part-of": "keda-manager",
			},
		},
	)
	if err != nil {
		return err
	}

	b := ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Keda{}, builder.WithPredicates(ommitStatusChanged))

	// create functtion to register wached objects
	watchFn := func(u unstructured.Unstructured) {
		r.log.With("gvk", u.GroupVersionKind().String()).Infoln("adding watcher")
		b = b.Watches(&u,
			handler.EnqueueRequestsFromMapFunc(r.mapFunction),
			builder.WithPredicates(
				predicate.And(
					predicate.ResourceVersionChangedPredicate{},
					labelSelectorPredicate,
				),
			),
		)
	}

	if err := registerWatchDistinct(r.Objs, watchFn); err != nil {
		return err
	}

	return b.Complete(r)
}

func (r *kedaReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var instance v1alpha1.Keda
	if err := r.Get(ctx, req.NamespacedName, &instance); err != nil {
		return ctrl.Result{
			Requeue: true,
		}, client.IgnoreNotFound(err)
	}

	stateFSM := reconciler.NewFsm(r.log, r.Cfg, reconciler.K8s{
		Client:        r.Client,
		EventRecorder: r.EventRecorder,
	})
	return stateFSM.Run(ctx, instance)
}

func NewKedaReconciler(c client.Client, r record.EventRecorder, log *zap.SugaredLogger, o []unstructured.Unstructured) KedaReconciler {
	return &kedaReconciler{
		log: log,
		Cfg: reconciler.Cfg{
			Finalizer: v1alpha1.Finalizer,
			Objs:      o,
		},
		K8s: reconciler.K8s{
			Client:        c,
			EventRecorder: r,
		},
	}
}
