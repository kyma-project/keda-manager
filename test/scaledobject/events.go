package scaledobject

import (
	"fmt"

	eventsv1 "k8s.io/api/events/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kyma-project/keda-manager/test/utils"
)

// VerifyEvents checks that KEDA emitted at least one event for the ScaledObject via
// the events.k8s.io API group. A missing event indicates that the RBAC rule granting
// create/patch on events.k8s.io/events is absent from the keda-operator ClusterRole.
func VerifyEvents(testutil *utils.TestUtils) error {
	eventList := &eventsv1.EventList{}
	if err := testutil.Client.List(testutil.Ctx, eventList, client.InNamespace(testutil.Namespace)); err != nil {
		return fmt.Errorf("listing events.k8s.io/v1 events: %w", err)
	}

	for i := range eventList.Items {
		e := &eventList.Items[i]
		if e.Regarding.Name == testutil.ScaledObjectName {
			return nil
		}
	}

	return fmt.Errorf("no events.k8s.io/v1 events found for ScaledObject %q in namespace %q — "+
		"KEDA may be missing create/patch permission on events.k8s.io/events", testutil.ScaledObjectName, testutil.Namespace)
}
