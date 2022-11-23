package keda

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	kedaCoreLabels = map[string]string{"app": "keda-operator", "app.kubernetes.io/name": "keda-operator"}
)

func IsInstalled(config *rest.Config, logger logr.Logger) (bool, error) {
	k8sClient, err := client.New(config, client.Options{})
	if err != nil {
		return false, fmt.Errorf("failed to create Kubernetes Client: %v", err)
	}

	return isInstalledWithClient(k8sClient, logger)
}

func isInstalledWithClient(c client.Client, logger logr.Logger) (bool, error) {
	// use multiple label matches to be sure.
	matchingLabels := client.MatchingLabels(kedaCoreLabels)
	listOpts := &client.ListOptions{}
	matchingLabels.ApplyToList(listOpts)

	deployList := &appsv1.DeploymentList{}
	if err := c.List(context.Background(), deployList, listOpts); err != nil {
		return false, fmt.Errorf("failed to list deployments: %v", err)
	}

	if len(deployList.Items) > 0 {
		logger.Info(fmt.Sprintf("found [%d] deployments with matchingLabels: %v", len(deployList.Items), matchingLabels))
		return true, nil
	}
	return false, nil
}
