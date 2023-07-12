package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/kyma-project/keda-manager/test/deployment"
	"github.com/kyma-project/keda-manager/test/hpa"
	"github.com/kyma-project/keda-manager/test/logger"
	"github.com/kyma-project/keda-manager/test/namespace"
	"github.com/kyma-project/keda-manager/test/scaledobject"
	"github.com/kyma-project/keda-manager/test/utils"
)

var (
	testTimeout = time.Second * 30
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	log, err := logger.New()
	if err != nil {
		fmt.Printf("%s: %s\n", "unable to setup logger", err)
		os.Exit(1)
	}

	log.Info("Configuring test essentials")
	client, err := utils.GetKuberentesClient()
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}

	log.Info("Start scenario")
	err = runScenario(&utils.TestUtils{
		Namespace:        fmt.Sprintf("keda-operator-test-%s", uuid.New().String()),
		DeploymentName:   "test-deployment",
		ScaledObjectName: "test-scaledobject",
		Ctx:              ctx,
		Client:           client,
		Logger:           log,
	})
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}
}

func runScenario(testutil *utils.TestUtils) error {
	// create test namespac
	testutil.Logger.Infof("Creating namespace '%s'", testutil.Namespace)
	if err := namespace.Create(testutil); err != nil {
		return err
	}

	// create test deployment
	testutil.Logger.Infof("Creating deployment '%s'", testutil.DeploymentName)
	if err := deployment.Create(testutil); err != nil {
		return err
	}

	// deploy a ScaledObject CR targetting a deployment
	testutil.Logger.Infof("Creating scaledobject '%s'", testutil.ScaledObjectName)
	if err := scaledobject.Create(testutil); err != nil {
		return err
	}

	// verify the status of the ScaledObject CR
	testutil.Logger.Infof("Verifying scaledobject '%s'", testutil.ScaledObjectName)
	if err := utils.WithRetry(testutil, scaledobject.Verify); err != nil {
		return err
	}

	// verify if the underlying HPA
	testutil.Logger.Infof("Verifying scaledobjects '%s' hpa", testutil.ScaledObjectName)
	if err := utils.WithRetry(testutil, hpa.Verify); err != nil {
		return err
	}

	// delete scaledobject
	testutil.Logger.Infof("Deleting scaledobjects '%s'", testutil.ScaledObjectName)
	if err := scaledobject.Delete(testutil); err != nil {
		return err
	}

	// verify deletion without orphan resources
	testutil.Logger.Infof("Verifying scaledobjects '%s' hpa deletion", testutil.ScaledObjectName)
	if err := utils.WithRetry(testutil, hpa.VerifyDeletion); err != nil {
		return err
	}

	// verify deletion without orphan resources
	testutil.Logger.Infof("Verifying scaledobject '%s' deletion", testutil.ScaledObjectName)
	if err := utils.WithRetry(testutil, scaledobject.VerifyDeletion); err != nil {
		return err
	}

	// cleanup
	testutil.Logger.Infof("Deleting namespace '%s'", testutil.Namespace)
	return namespace.Delete(testutil)
}
