package utils

import (
	"context"

	"go.uber.org/zap"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type TestUtils struct {
	Ctx    context.Context
	Logger *zap.SugaredLogger
	Client client.Client

	Namespace             string
	DeploymentName        string
	MetricsServerName     string
	MetricsServerPort     int32
	MetricsServerEndpoint string
	ScaledObjectName      string
	ScaleDeploymentTo     int32
}
