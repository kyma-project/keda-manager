package logging

import (
	"context"

	"github.com/kyma-project/manager-toolkit/logging/config"
	"go.uber.org/zap"
)

// ReconfigureOnConfigChange monitors config changes and updates log level dynamically.
// This is a thin wrapper around the manager-toolkit implementation.
func ReconfigureOnConfigChange(ctx context.Context, log *zap.SugaredLogger, atomic zap.AtomicLevel, cfgPath string) {
	config.ReconfigureOnConfigChange(ctx, log, atomic, cfgPath)
}
