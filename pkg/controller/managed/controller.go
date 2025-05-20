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

package managed

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	"github.com/crossplane/crossplane-runtime/pkg/engine"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
)

// ControllerOption configures a Controller.
type ControllerOption func(*Controller)

// WithMetricsAddress sets the address for the metrics server.
func WithMetricsAddress(addr string) ControllerOption {
	return func(b *Controller) {
		b.metricsAddr = addr
	}
}

// WithHealthProbeAddress sets the address for the health probe server.
func WithHealthProbeAddress(addr string) ControllerOption {
	return func(b *Controller) {
		b.probeAddr = addr
	}
}

// WithLeaderElection enables or disables leader election.
func WithLeaderElection(enabled bool) ControllerOption {
	return func(b *Controller) {
		b.leaderElection = enabled
	}
}

// WithPollInterval sets the polling interval for all reconcilers.
func WithPollInterval(d time.Duration) ControllerOption {
	return func(b *Controller) {
		b.pollInterval = d
	}
}

// WithMaxReconcileRate sets the maximum reconciliation rate.
func WithMaxReconcileRate(rate int) ControllerOption {
	return func(b *Controller) {
		b.maxReconcileRate = rate
	}
}

// WithLogger sets the logger for the controller.
func WithLogger(log logging.Logger) ControllerOption {
	return func(b *Controller) {
		b.log = log
	}
}

// Controller builds a DynamicControllerManager.
type Controller struct {
	config           DynamicControllerConfig
	log              logging.Logger
	metricsAddr      string
	probeAddr        string
	leaderElection   bool
	pollInterval     time.Duration
	maxReconcileRate int
	engine           engine.IControllerEngine
	manager          ctrl.Manager
	providers        []*Provider
}

// NewController creates a new Controller.
func NewController(config DynamicControllerConfig, opts ...ControllerOption) *Controller {
	b := &Controller{
		config:           config,
		log:              logging.NewNopLogger(),
		metricsAddr:      ":8080",
		probeAddr:        ":8081",
		leaderElection:   true,
		pollInterval:     1 * time.Minute,
		maxReconcileRate: 10,
	}

	for _, opt := range opts {
		opt(b)
	}

	return b
}

// Start creates and starts all the providers and a controller runtime manager.
func (b *Controller) Start(ctx context.Context) error {
	// Initialize a new scheme
	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		return errors.Wrap(err, "cannot add kubernetes client-go scheme")
	}

	// Create a manager
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
		Metrics: metricsserver.Options{
			BindAddress: b.metricsAddr,
		},
		HealthProbeBindAddress: b.probeAddr,
		LeaderElection:         b.leaderElection,
		LeaderElectionID:       "managed-reconciler-leader-election",
	})
	if err != nil {
		return errors.Wrap(err, "unable to create controller manager")
	}

	// Setup health checks
	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		return errors.Wrap(err, "unable to set up health check")
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		return errors.Wrap(err, "unable to set up ready check")
	}

	if b.engine, err = engine.NewEngineFromManager(ctx, mgr, b.log); err != nil {
		return errors.Wrap(err, "unable to create engine")
	}

	// Create providers for each provider config
	b.providers = make([]*Provider, 0, len(b.config.Providers))
	for _, pc := range b.config.Providers {
		provider, err := NewProvider(pc,
			WithProviderLogger(b.log.WithValues("provider", pc.Name)),
			WithProviderPollInterval(b.pollInterval),
			WithProviderMaxReconcileRate(b.maxReconcileRate),
		)
		if err != nil {
			return errors.Wrapf(err, "cannot create provider %s", pc.Name)
		}
		b.providers = append(b.providers, provider)
	}

	// Store the manager for provider setup
	b.manager = mgr

	// Set up providers with the manager and engine
	for _, provider := range b.providers {
		if err := provider.Setup(ctx, b.engine, b.manager); err != nil {
			return errors.Wrapf(err, "cannot set up provider %s", provider.config.Name)
		}
	}

	setupLog := log.Log.WithName("setup")
	setupLog.Info("starting manager")
	return b.manager.Start(ctx)
}
