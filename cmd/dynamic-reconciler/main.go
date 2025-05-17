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
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/pflag"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/external/client"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
)

// ResourceType defines a resource type to be reconciled.
type ResourceType struct {
	// APIVersion is the API version of the resource.
	APIVersion string `json:"apiVersion"`

	// Kind is the kind of the resource.
	Kind string `json:"kind"`
}

// ProviderConfig defines the configuration for a dynamic provider.
type ProviderConfig struct {
	// Name is a unique name for this provider.
	Name string `json:"name"`

	// Endpoint is the gRPC endpoint of the provider.
	Endpoint string `json:"endpoint"`

	// UseSSL indicates whether to use SSL for the connection.
	UseSSL bool `json:"useSSL,omitempty"`

	// ResourceTypes is a list of resource types this provider supports.
	ResourceTypes []ResourceType `json:"resourceTypes"`
}

// DynamicControllerConfig defines the configuration for the dynamic reconciler.
type DynamicControllerConfig struct {
	// Providers is a list of provider configurations.
	Providers []ProviderConfig `json:"providers"`
}

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
	zapLogger := logging.NewLogrLogger(ctrl.Log.WithName("dynamic-reconciler"))

	// Load configuration
	var config DynamicControllerConfig
	if configPath != "" {
		configData, err := os.ReadFile(configPath)
		if err != nil {
			setupLog.Error(err, "unable to read config file")
			os.Exit(1)
		}
		if err := json.Unmarshal(configData, &config); err != nil {
			setupLog.Error(err, "unable to parse config file")
			os.Exit(1)
		}
	} else if providerEndpoint != "" {
		// Use the command-line endpoint as a single provider
		config = DynamicControllerConfig{
			Providers: []ProviderConfig{
				{
					Name:     "default",
					Endpoint: providerEndpoint,
				},
			},
		}
	} else {
		setupLog.Error(nil, "either --config or --provider-endpoint must be specified")
		os.Exit(1)
	}

	// Initialize a new scheme
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)

	// Create a manager
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
		Metrics: metricsserver.Options{
			BindAddress: metricsAddr,
		},
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         leaderElection,
		LeaderElectionID:       "dynamic-reconciler-leader-election",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// Setup health checks
	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	// Set up controllers for each provider
	for _, provider := range config.Providers {
		if err := setupProvider(mgr, provider, pollInterval, maxReconcileRate, zapLogger); err != nil {
			setupLog.Error(err, "unable to set up provider", "provider", provider.Name)
			os.Exit(1)
		}
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

func setupProvider(mgr ctrl.Manager, config ProviderConfig, pollInterval time.Duration, maxReconcileRate int, log logging.Logger) error {
	log = log.WithValues("provider", config.Name)

	var gvks []schema.GroupVersionKind
	for _, rt := range config.ResourceTypes {
		gv, err := schema.ParseGroupVersion(rt.APIVersion)
		if err != nil {
			return errors.Wrapf(err, "invalid API version %s", rt.APIVersion)
		}
		gvk := gv.WithKind(rt.Kind)
		gvks = append(gvks, gvk)
	}

	// Create the streaming connector
	var creds credentials.TransportCredentials
	if config.UseSSL {
		// In a real implementation, we'd load proper TLS credentials
		// This is just a placeholder
		log.Debug("SSL is enabled, but insecure credentials are being used for demonstration")
		creds = insecure.NewCredentials()
	} else {
		creds = insecure.NewCredentials()
	}

	connector := client.NewStreamingConnector(
		config.Endpoint,
		creds,
		client.WithClientLogger(log),
		client.WithResourceTypes(gvks...),
	)

	// Register a cleanup function to close the connector when the manager stops
	mgr.Add(manager.RunnableFunc(func(ctx context.Context) error {
		<-ctx.Done() // Wait for context to be cancelled (manager is stopping)
		return connector.Close()
	}))

	// Set up a controller for each resource type
	for _, rt := range config.ResourceTypes {
		gv, _ := schema.ParseGroupVersion(rt.APIVersion)
		gvk := gv.WithKind(rt.Kind)

		// Ensure the schema knows about this type
		u := &unstructured.Unstructured{}
		u.SetGroupVersionKind(gvk)

		// Add the type to the scheme
		mgr.GetScheme().AddKnownTypeWithName(gvk, u.DeepCopyObject())

		// Set up the controller
		name := fmt.Sprintf("%s.%s.%s", rt.Kind, gv.Group, config.Name)
		r := managed.NewReconciler(mgr,
			resource.ManagedKind(gvk),
			managed.WithLogger(log.WithValues("controller", name)),
			managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))),
			managed.WithPollInterval(pollInterval),
			managed.WithExternalConnecter(connector),
		)

		// Create the unstructured object with the correct GVK
		obj := &unstructured.Unstructured{}
		obj.SetGroupVersionKind(gvk)

		if err := ctrl.NewControllerManagedBy(mgr).
			Named(name).
			For(obj).
			WithOptions(controller.Options{
				MaxConcurrentReconciles: maxReconcileRate,
				//RateLimiter:             ratelimiter.NewGlobal(maxReconcileRate), TODO: this is not working.
			}).
			Complete(r); err != nil {
			return errors.Wrapf(err, "cannot set up controller for %s", gvk)
		}

		log.Debug("Set up controller", "gvk", gvk.String())
	}

	return nil
}
