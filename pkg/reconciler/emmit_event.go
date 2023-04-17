package reconciler

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

func sFnEmitStrictEventFunc(next stateFn, result *ctrl.Result, err error, eventType, eventReason, eventMessage string) stateFn {
	return func(ctx context.Context, m *fsm, s *systemState) (stateFn, *ctrl.Result, error) {
		m.Event(
			&s.instance,
			eventType,
			eventReason,
			eventMessage,
		)

		return next, result, err
	}
}

func sFnEmmitEventfunc(next stateFn, result *ctrl.Result, err error) stateFn {
	return func(_ context.Context, m *fsm, s *systemState) (stateFn, *ctrl.Result, error) {
		// compare if any condition change
		for _, condition := range s.instance.Status.Conditions {
			// check if condition exists in memento status
			memorizedCondition := meta.FindStatusCondition(s.snapshot.Conditions, condition.Type)
			// ignore unchanged conditions
			if memorizedCondition != nil &&
				memorizedCondition.Status == condition.Status &&
				memorizedCondition.Reason == condition.Reason &&
				memorizedCondition.Message == condition.Message {
				continue
			}
			m.Event(
				&s.instance,
				eventType(condition),
				condition.Reason,
				fmt.Sprintf("%s: %s/%s", condition.Message, s.instance.Namespace, s.instance.Name),
			)
		}
		return next, result, err
	}
}

func eventType(condition metav1.Condition) string {
	eventType := "Normal"
	if condition.Status == metav1.ConditionFalse {
		eventType = "Warning"
	}
	return eventType
}
