/*
Copyright 2025 The Crossplane Authors.
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
	"flag"
	"os"
	"time"

	"github.com/spf13/pflag"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/crossplane/crossplane-runtime/pkg/controller/managed"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
)

func main() {
	var (
		configPath        string
		providerEndpoint  string
		leaderElection    bool
		restartOnProvider bool
		maxReconcileRate  int
		pollInterval      time.Duration
		metricsAddr       string
		probeAddr         string
		certDir           string
	)

	pflag.StringVar(&configPath, "config", "", "Path to the configuration file")
	pflag.StringVar(&providerEndpoint, "provider-endpoint", "", "gRPC endpoint for the provider (overrides config file)")
	pflag.BoolVar(&leaderElection, "leader-election", true, "Use leader election for the controller")
	pflag.BoolVar(&restartOnProvider, "restart-on-provider-disconnect", true, "Restart the reconciler if the provider connection is lost")
	pflag.IntVar(&maxReconcileRate, "max-reconcile-rate", 10, "The maximum number of concurrent reconciliations per controller")
	pflag.DurationVar(&pollInterval, "poll-interval", 1*time.Minute, "How often a managed resource should be polled when in a steady state")
	pflag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to")
	pflag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to")
	pflag.StringVar(&certDir, "cert-dir", "", "The directory containing TLS certificates")

	// Add controller-runtime flags
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()

	// Setup logging
	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))
	setupLog := ctrl.Log.WithName("setup")
	zapLogger := logging.NewLogrLogger(ctrl.Log.WithName("managed-reconciler"))

	// Load configuration
	var config managed.ControllerConfig
	var err error

	if configPath != "" {
		// Load config from file
		config, err = managed.LoadConfigFromFile(configPath)
		if err != nil {
			setupLog.Error(err, "unable to load configuration from file")
			os.Exit(1)
		}
	} else if providerEndpoint != "" {
		// Create config from endpoint
		config = managed.CreateConfigFromEndpoint(providerEndpoint)
	} else {
		setupLog.Error(nil, "either --config or --provider-endpoint must be specified")
		os.Exit(1)
	}

	// Validate config
	if err := managed.ValidateConfig(config); err != nil {
		setupLog.Error(err, "invalid configuration")
		os.Exit(1)
	}

	// Create controller builder
	controller := managed.NewController(config,
		managed.WithLogger(zapLogger),
		managed.WithMetricsAddress(metricsAddr),
		managed.WithHealthProbeAddress(probeAddr),
		managed.WithLeaderElection(leaderElection),
		managed.WithPollInterval(pollInterval),
		managed.WithMaxReconcileRate(maxReconcileRate),
	)

	ctx := ctrl.SetupSignalHandler()

	// Start the controller
	if err := controller.Start(ctx); err != nil {
		setupLog.Error(err, "problem running controller")
		os.Exit(1)
	}
}
