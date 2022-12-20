package reconciler

import (
	"context"
	"fmt"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"time"
)

type Config struct {
	Finalizer    string
	Installation []ReconciliationAction
	Deletion     []ReconciliationAction
	Prototype    client.Object
}

type Reconciler struct {
	client.Client
	Config
}

type shouldContinue = bool

type ReconciliationAction func(context.Context, client.Client, ctrl.Request) (shouldContinue, ctrl.Result, error)

type ReconciliationActions []ReconciliationAction

func (a ReconciliationActions) reconcileAll(ctx context.Context, c client.Client, req ctrl.Request) (ctrl.Result, error) {
	for _, f := range a {
		shouldContinue, result, err := f(ctx, c, req)

		if !shouldContinue {
			return result, err
		}

		if err != nil {
			return defaultResult, err
		}
	}
	return ctrl.Result{RequeueAfter: time.Minute}, nil
}

var (
	_ reconcile.Reconciler = &Reconciler{}

	defaultResult = ctrl.Result{}
)

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	instance, ok := r.Prototype.DeepCopyObject().(client.Object)
	if !ok {
		return ctrl.Result{}, fmt.Errorf("invalid custom resource object type for reconciliation %s", req.String())
	}

	if err := r.Get(ctx, req.NamespacedName, instance); err != nil {
		return ctrl.Result{
			RequeueAfter: time.Second * 30,
		}, client.IgnoreNotFound(err)
	}

	instanceIsBeingDeleted := !instance.GetDeletionTimestamp().IsZero()
	instanceHasFinalizer := controllerutil.ContainsFinalizer(instance, r.Finalizer)

	// in case instance does not have finalizer - add it and update instance
	if !instanceIsBeingDeleted && !instanceHasFinalizer {
		controllerutil.AddFinalizer(instance, r.Finalizer)
		if err := r.Update(ctx, instance); err != nil {
			return defaultResult, err
		}
		return defaultResult, nil
	}

	// in case instance has no finalizer and instance is being deleted - end reconciliation
	if instanceIsBeingDeleted && !controllerutil.ContainsFinalizer(instance, r.Finalizer) {
		return defaultResult, nil
	}

	// in case instance is being deleted and has finalizer - delete all resources
	if instanceIsBeingDeleted {
		deletion := append(r.Deletion, r.buildDeleteFinalizer())
		return ReconciliationActions(deletion).reconcileAll(ctx, r.Client, req)
	}

	return ReconciliationActions(r.Installation).reconcileAll(ctx, r.Client, req)
}

func (r *Reconciler) buildDeleteFinalizer() ReconciliationAction {
	return func(ctx context.Context, c client.Client, req ctrl.Request) (shouldContinue, ctrl.Result, error) {
		instance, ok := r.Prototype.DeepCopyObject().(client.Object)
		if !ok {
			return false, defaultResult, fmt.Errorf("invalid custom resource object type  %v", instance)
		}

		if err := r.Get(ctx, req.NamespacedName, instance); err != nil {
			return false, ctrl.Result{}, client.IgnoreNotFound(err)
		}

		controllerutil.RemoveFinalizer(instance, r.Finalizer)
		if err := c.Update(ctx, instance); err != nil {
			return false, defaultResult, err
		}
		fmt.Println("finalizer removed")
		return false, defaultResult, nil
	}
}
