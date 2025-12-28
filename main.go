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

package main

import (
	"crypto/fips140"
	"flag"
	"os"

	"github.com/go-logr/zapr"
	"go.uber.org/zap"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	ctrlwebhook "sigs.k8s.io/controller-runtime/pkg/webhook"

	"github.com/kyma-project/manager-toolkit/logging/config"
	"github.com/kyma-project/manager-toolkit/logging/logger"

	operatorv1alpha1 "github.com/kyma-project/keda-manager/api/v1alpha1"
	"github.com/kyma-project/keda-manager/controllers"
	"github.com/kyma-project/keda-manager/pkg/resources"
	//+kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(apiextensionsv1.AddToScheme(scheme))
	utilruntime.Must(operatorv1alpha1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	//if !isFIPS140Only() {
	//	setupLog.Error(errors.New("FIPS not enforced"), "FIPS 140 exclusive mode is not enabled. Check GODEBUG flags.")
	//	panic("FIPS 140 exclusive mode is not enabled. Check GODEBUG flags.")
	//}

	var metricsAddr string
	var probeAddr string
	var configPath string
	var enableLeaderElection bool

	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.StringVar(&configPath, "config-path", "", "Path to config file for dynamic reconfiguration.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.Parse()

	// Load configuration - from file if provided, otherwise from environment
	var cfg config.Config
	var err error
	if configPath != "" {
		cfg, err = config.LoadConfig(configPath)
		if err != nil {
			os.Exit(1)
		}
	} else {
		cfg, err = config.GetConfig("")
		if err != nil {
			os.Exit(1)
		}
	}

	// Setup logging with atomic level for dynamic reconfiguration
	atomicLevel := zap.NewAtomicLevel()
	parsedLogLevel, err := logger.MapLevel(cfg.LogLevel)
	if err != nil {
		setupLog.Error(err, "unable to parse logging level")
		os.Exit(1)
	}

	format, err := logger.MapFormat(cfg.LogFormat)
	if err != nil {
		setupLog.Error(err, "unable to set logging format")
		os.Exit(1)
	}

	log, err := logger.NewWithAtomicLevel(format, atomicLevel)
	if err != nil {
		setupLog.Error(err, "unable to set logger")
		os.Exit(1)
	}

	if err := logger.InitKlog(log, parsedLogLevel); err != nil {
		setupLog.Error(err, "unable to init Klog")
		os.Exit(1)
	}

	zapLog := log.WithContext()
	zapLog.Infof("loaded config: logLevel=%s, logFormat=%s", cfg.LogLevel, cfg.LogFormat)

	// Set controller-runtime logger to use our zap logger (respects format from config)
	ctrl.SetLogger(zapr.NewLogger(zapLog.Desugar()))

	// Setup signal handler once - used for both manager and dynamic config
	signalCtx := ctrl.SetupSignalHandler()

	// Start dynamic reconfiguration in background if config path is provided
	if configPath != "" {
		go config.ReconfigureOnConfigChange(signalCtx, zapLog, atomicLevel, configPath)
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
		Metrics: server.Options{
			BindAddress: metricsAddr,
		},
		WebhookServer: ctrlwebhook.NewServer(ctrlwebhook.Options{
			Port: 9443,
		}),
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "4123c01c.operator.kyma-project.io",
		Client: ctrlclient.Options{
			Cache: &ctrlclient.CacheOptions{
				DisableFor: []ctrlclient.Object{
					&corev1.Secret{},
					&corev1.ConfigMap{},
				},
			},
		},
	})
	if err != nil {
		zapLog.Error("unable to start manager", "error", err)
		os.Exit(1)
	}

	data, err := resources.LoadFromPaths("keda-networkpolicies.yaml", "keda.yaml")
	if err != nil {
		zapLog.Error("unable to load k8s data", "error", err)
		os.Exit(1)
	}

	kedaReconciler := controllers.NewKedaReconciler(
		mgr.GetClient(),
		mgr.GetEventRecorderFor("keda-manager"),
		zapLog,
		data,
	)
	if err = kedaReconciler.SetupWithManager(mgr); err != nil {
		zapLog.Error("unable to create controller", "controller", "Keda", "error", err)
		os.Exit(1)
	}
	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		zapLog.Error("unable to set up health check", "error", err)
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		zapLog.Error("unable to set up ready check", "error", err)
		os.Exit(1)
	}

	zapLog.Info("starting manager")
	if err := mgr.Start(signalCtx); err != nil {
		zapLog.Error("problem running manager", "error", err)
		os.Exit(1)
	}
}

// isFIPS140Only checks if the application is running in FIPS 140 exclusive mode.
func isFIPS140Only() bool {
	return fips140.Enabled() && os.Getenv("GODEBUG") == "fips140=only,tlsmlkem=0"
}
